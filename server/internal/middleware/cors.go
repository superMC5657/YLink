package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"ylink/internal/pkg/errs"
	"ylink/internal/pkg/resp"
)

// CORS 白名单域名；订阅端点由路由层单独放行任意来源。
func CORS(allowOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if slices.Contains(allowOrigins, origin) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, Accept-Language, X-Client")
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// CORSAny 允许任意来源（仅用于订阅端点，代理客户端直连）。
func CORSAny() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// UserID 从上下文取当前用户 ID（由 Auth 注入）。
func UserID(c *gin.Context) int64 { return c.GetInt64("user_id") }

// UserRole 从上下文取当前用户角色。
func UserRole(c *gin.Context) int {
	if v, ok := c.Get("role"); ok {
		if r, ok := v.(int); ok {
			return r
		}
	}
	return 0
}

// Fail 统一业务错误响应（中间件内使用）。
func Fail(c *gin.Context, e *errs.Error) {
	c.AbortWithStatusJSON(e.HTTP, resp.Body{Code: e.Code, Message: e.Message, Data: nil})
}
