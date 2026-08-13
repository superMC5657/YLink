// Package redis 提供 go-redis 客户端初始化。
package redis

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"

	"ylink-backend/internal/config"
)

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

// SessionVersionKey 会话版本号 Key：`auth:ver:{userID}`。
// 封禁/降级/登出等操作 INCR 该值，Auth 中间件比对 access token 中快照的 sv，
// 不一致即 401——实现封禁/降级对已签发 JWT 的实时失效（access TTL 内无需等过期）。
// Key 不存在视为版本 0（Redis 重启/首次签发均安全）。
func SessionVersionKey(userID int64) string {
	return Key("auth", "ver", strconv.FormatInt(userID, 10))
}
