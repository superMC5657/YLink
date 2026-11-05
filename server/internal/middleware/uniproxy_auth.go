package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"ylink-backend/internal/model"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/pkg/resp"
)

// UniProxyLookupFunc 节点查询函数类型。
type UniProxyLookupFunc func(ctx context.Context, nodeID int64) (*model.Server, error)

// UniProxyTokenCacheKey 生成 UniProxy 鉴权缓存键：node:uniproxy:token:{node_id}。
func UniProxyTokenCacheKey(nodeID int64) string {
	return redispkg.Key("node", "uniproxy", "token", fmt.Sprint(nodeID))
}

// GetUniProxyServerID 从上下文中获取经 UniProxy 鉴权校验的节点 ID。
func GetUniProxyServerID(c *gin.Context) (int64, bool) {
	if v, exists := c.Get("server_id"); exists {
		if id, ok := v.(int64); ok && id > 0 {
			return id, true
		}
	}
	if id := c.GetInt64("server_id"); id > 0 {
		return id, true
	}
	if id := c.GetInt64("uniproxy_server_id"); id > 0 {
		return id, true
	}
	return 0, false
}

// UniProxyServerID 从上下文中获取经 UniProxy 鉴权校验的节点 ID。
func UniProxyServerID(c *gin.Context) int64 {
	id, _ := GetUniProxyServerID(c)
	return id
}

// abortUniProxyUnauthorized 统一中止并返回标准 401 错误信封。
func abortUniProxyUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, resp.Body{
		Code:    40100,
		Message: "unauthorized",
		Data:    nil,
	})
}

// UniProxyAuth V2Board UniProxy 节点通信鉴权中间件。
// 兼容 URL Query 参数 ?node_id={id}&token={token} 以及 Header Token / X-Token / X-Node-Key。
// 具备权威 Redis 缓存加速（TTL 60s），节点不存在、密钥不匹配或状态非正常（status != 1 && status != 2）均返回 HTTP 401 标准信封。
func UniProxyAuth(lookup UniProxyLookupFunc, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 提取 node_id
		nodeIDStr := strings.TrimSpace(c.Query("node_id"))
		if nodeIDStr == "" {
			abortUniProxyUnauthorized(c)
			return
		}

		nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
		if err != nil || nodeID <= 0 {
			abortUniProxyUnauthorized(c)
			return
		}

		// 2. 提取 token
		token := strings.TrimSpace(c.Query("token"))
		if token == "" {
			token = strings.TrimSpace(c.GetHeader("Token"))
		}
		if token == "" {
			token = strings.TrimSpace(c.GetHeader("X-Token"))
		}
		if token == "" {
			token = strings.TrimSpace(c.GetHeader("X-Node-Key"))
		}
		if token == "" {
			abortUniProxyUnauthorized(c)
			return
		}

		ctx := c.Request.Context()
		cacheKey := UniProxyTokenCacheKey(nodeID)

		// 3. 权威 Redis 缓存校验：命中但 token 不匹配立即 401（不回源 DB）
		if rdb != nil {
			cachedToken, err := rdb.Get(ctx, cacheKey).Result()
			if err == nil {
				if cachedToken != token {
					abortUniProxyUnauthorized(c)
					return
				}
				c.Set("server_id", nodeID)
				c.Set("uniproxy_server_id", nodeID)
				c.Next()
				return
			}
		}

		// 4. 回源数据库查询校验
		srv, err := lookup(ctx, nodeID)
		if err != nil || srv == nil {
			abortUniProxyUnauthorized(c)
			return
		}

		// 允许 status=1(正常) 与 status=2(拥挤)
		if srv.Status != 1 && srv.Status != 2 {
			abortUniProxyUnauthorized(c)
			return
		}

		if srv.NodeKey != token {
			abortUniProxyUnauthorized(c)
			return
		}

		// 5. 鉴权通过：回写 Redis 缓存 60s
		if rdb != nil {
			_ = rdb.Set(ctx, cacheKey, srv.NodeKey, 60*time.Second).Err()
		}

		c.Set("server_id", nodeID)
		c.Set("uniproxy_server_id", nodeID)
		c.Next()
	}
}
