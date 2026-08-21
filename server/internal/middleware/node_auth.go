package middleware

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"ylink-backend/internal/pkg/errs"
	redispkg "ylink-backend/internal/pkg/redis"
)

// ErrNodeKeyNotFound 未知节点密钥(lookup 返回本错误 → 401)。
var ErrNodeKeyNotFound = errors.New("node key not found")

// NodeAuth 解析 X-Node-Key(节点上报,契约 §17),注入 server_id。
// 密钥 → 节点 id 映射经 Redis 缓存 60s(node:key:{k});重置密钥由服务层删除旧缓存即刻失效。
// lookup 由路由层注入(service.NodeService.ServerIDByKey,纯 DB 查询),中间件不直接依赖数据层。
func NodeAuth(lookup func(ctx context.Context, key string) (int64, error), rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Node-Key")
		if key == "" {
			Fail(c, errs.ErrUnauthorized)
			return
		}
		ctx := c.Request.Context()
		cacheKey := NodeKeyCacheKey(key)
		if v, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
			if id, perr := strconv.ParseInt(v, 10, 64); perr == nil {
				c.Set("server_id", id)
				c.Next()
				return
			}
		}
		serverID, err := lookup(ctx, key)
		if err != nil {
			Fail(c, errs.ErrUnauthorized)
			return
		}
		rdb.Set(ctx, cacheKey, serverID, 60*time.Second)
		c.Set("server_id", serverID)
		c.Next()
	}
}

// ServerID 取 NodeAuth 注入的节点 id(与 UserID/UserRole 命名一致)。
func ServerID(c *gin.Context) int64 { return c.GetInt64("server_id") }

// NodeKeyCacheKey node:key:{k} 缓存键(重置密钥时删除)。
func NodeKeyCacheKey(key string) string { return redispkg.Key("node", "key", key) }
