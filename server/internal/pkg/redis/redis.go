// Package redis 提供 go-redis 客户端初始化。
package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"ylink-backend/internal/config"
)

// 会话吊销深 Module 的 Key 拥有者（写 seam 见 service/session_revocation.go，
// 读 seam 见本文件的 SessionAccessAllowed；middleware 只经由此 seam 提问，
// 不直接拼 Key）：白名单 refresh:{uid}:{jti}、会话版本号SV auth:ver:{uid}、
// kill-Hash auth:kill:{uid} 三套语义归一，不外泄字段含义。

// KillTTL 踢下线标记保留时长：随 refresh 白名单 TTL（默认 14d），调用方传入 0 即用此值。
const KillTTL = 14 * 24 * time.Hour

// New 创建 Redis 客户端并探测连通性。
func New(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}

// Key 统一构造带命名空间的 Key。
func Key(parts ...string) string {
	const sep = ":"
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

// RefreshKey 白名单 Key：`refresh:{uid}:{jti}`，值为 `{ip,ua,ts}` JSON，TTL 14d。
// 由会话吊销深 Module 拥有，调用方不得手拼 "refresh:" 前缀。
func RefreshKey(userID int64, jti string) string {
	return Key("refresh", strconv.FormatInt(userID, 10), jti)
}

// SessionVersionKey 会话版本号 Key：`auth:ver:{userID}`。
// 封禁/降级/登出等操作 INCR 该值，Auth 中间件比对 access token 中快照的 sv，
// 不一致即 401——实现封禁/降级对已签发 JWT 的实时失效（access TTL 内无需等过期）。
// Key 不存在视为版本 0（Redis 重启/首次签发均安全）。
// 由会话吊销深 Module 拥有。
func SessionVersionKey(userID int64) string {
	return Key("auth", "ver", strconv.FormatInt(userID, 10))
}

// SessionKillKey 会话踢下线标记 Key（F14）：`auth:kill:{userID}` Hash，field=jti。
// 用户端踢下线指定会话时写入 field；Auth 中间件 HExists 命中即 401——
// 单会话 access 立即失效且不影响其余会话（区别于 bump 全局会话版本号）。
// 由会话吊销深 Module 拥有。
func SessionKillKey(userID int64) string {
	return Key("auth", "kill", strconv.FormatInt(userID, 10))
}

// SessionAccessAllowed 会话准入读 seam：middleware 唯一提问口。
// 返回 ok=false 时 reason 为 "sv"（版本号漂移）或 "kill"（单会话踢下线）。
// Redis 异常退化为放行（TTL 自然过期兜底），与历史行为一致。
func SessionAccessAllowed(ctx context.Context, rdb *redis.Client, userID int64, jti string, sv int64) (ok bool, reason string) {
	pipe := rdb.Pipeline()
	svCmd := pipe.Get(ctx, SessionVersionKey(userID))
	killCmd := pipe.HExists(ctx, SessionKillKey(userID), jti)
	_, _ = pipe.Exec(ctx)
	if cur, err := svCmd.Int64(); err == nil && sv != cur {
		return false, "sv"
	}
	if killed, _ := killCmd.Result(); killed {
		return false, "kill"
	}
	return true, ""
}
