// Package middleware 实现请求中间件链。
// 顺序：RequestID → Recovery → AccessLog → CORS → RateLimit → [Auth] → [Idempotency] → handler
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"ylink/internal/pkg/logger"
	"ylink/internal/pkg/resp"
)

// RequestID 生成/透传 X-Request-Id。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-Id")
		if rid == "" {
			rid = newRequestID()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-Id", rid)
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Recovery 捕获 panic → 500 envelope + 堆栈日志（不泄漏内部细节）。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.L().Error("panic recovered",
					zap.Any("panic", r),
					zap.Any("request_id", c.GetString("request_id")),
					zap.Any("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, resp.Body{Code: 50000, Message: "服务器内部错误", Data: nil})
			}
		}()
		c.Next()
	}
}

// AccessLog 记录方法/路径/状态/耗时/用户 ID，不记 body 与 token。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.L().Info("access",
			zap.Any("method", c.Request.Method),
			zap.Any("path", c.Request.URL.Path),
			zap.Any("status", c.Writer.Status()),
			zap.Any("latency_ms", time.Since(start).Milliseconds()),
			zap.Any("user_id", c.GetInt64("user_id")),
			zap.Any("request_id", c.GetString("request_id")),
			zap.Any("ip", c.ClientIP()),
		)
	}
}
