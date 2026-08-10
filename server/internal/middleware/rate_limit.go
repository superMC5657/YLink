package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"ylink/internal/pkg/errs"
)

// RateLimit 基于 Redis 的固定窗口限流（多实例共享计数）。
// scope 用于区分登录/注册（严格）、订阅端点、全局（宽松）等。
// 窗口内超过 limit 次 → 42900。
func RateLimit(rdb *redis.Client, scope string, limit int, window time.Duration) gin.HandlerFunc {
	key := func(c *gin.Context) string {
		return "rl:" + scope + ":" + c.ClientIP()
	}
	return func(c *gin.Context) {
		ctx := context.Background()
		k := key(c)
		// Lua 原子递增：窗口内计数，超过返回 -1
		const script = `
local n = redis.call("INCR", KEYS[1])
if n == 1 then redis.call("EXPIRE", KEYS[1], ARGV[1]) end
return n
`
		n, err := rdb.Eval(ctx, script, []string{k}, int64(window.Seconds())).Int64()
		if err == nil && n > int64(limit) {
			Fail(c, errs.ErrTooManyReq)
			return
		}
		c.Next()
	}
}

// StrictLimiter 便捷构造：登录/注册/验证码等严格限流。
func StrictLimiter(rdb *redis.Client, scope string) gin.HandlerFunc {
	return RateLimit(rdb, scope, 5, time.Minute)
}

// GlobalLimiter 全局宽松限流。
func GlobalLimiter(rdb *redis.Client) gin.HandlerFunc {
	return RateLimit(rdb, "global", 300, time.Minute)
}
