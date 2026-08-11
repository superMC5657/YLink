package service

import (
	"context"

	"github.com/redis/go-redis/v9"

	redispkg "ylink/internal/pkg/redis"
)

// sessionVersion 读取用户当前会话版本号；Key 不存在视为 0。
// 中间件通过同一 Key 与 access token 快照比对，见 pkg/redis.SessionVersionKey。
func sessionVersion(ctx context.Context, rdb *redis.Client, userID int64) int64 {
	n, err := rdb.Get(ctx, redispkg.SessionVersionKey(userID)).Int64()
	if err != nil {
		return 0
	}
	return n
}

// bumpSessionVersion 使该用户全部已签发 access token 立即失效（401）。
// 触发点：封禁/解封、角色变更、代理审批通过、代理商降级、找回密码、登出。
func bumpSessionVersion(ctx context.Context, rdb *redis.Client, userID int64) {
	// 失败不阻塞主流程：Redis 异常时退化为 access TTL 自然过期（原行为）
	_ = rdb.Incr(ctx, redispkg.SessionVersionKey(userID)).Err()
}
