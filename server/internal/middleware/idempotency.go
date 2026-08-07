package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

// Idempotency 幂等中间件：POST 请求携带 Idempotency-Key 时，
// 24h 内同 Key 直接返回首次成功响应（Redis 缓存），防止重复提交。
// 仅缓存 2xx 响应；业务失败不缓存。
func Idempotency(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" || c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		ctx := context.Background()
		// 幂等键按用户隔离，防止跨用户回放
		ck := idempotencyKey(middlewareUserID(c), key)

		if cached, err := rdb.Get(ctx, ck).Bytes(); err == nil {
			// 命中：直接回放首次响应（HTTP 200 + 相同 body）
			c.Header("X-Idempotent-Replay", "true")
			c.Data(http.StatusOK, "application/json; charset=utf-8", cached)
			c.Abort()
			return
		}

		w := &recorder{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = w
		c.Next()

		// 仅成功响应可缓存；若业务失败（信封 code!=0）也回放，避免重复建单的判断交给 handler
		if w.Status() >= 200 && w.Status() < 300 && w.buf.Len() > 0 {
			rdb.Set(ctx, ck, w.buf.Bytes(), idempotencyTTL)
		}
	}
}

type recorder struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

// idempotencyKey 构造带用户维度的幂等缓存 key。
func idempotencyKey(userID int64, key string) string {
	return fmt.Sprintf("idem:%d:%s", userID, key)
}

// middlewareUserID 读取当前用户 ID（Auth 中间件注入）。
func middlewareUserID(c *gin.Context) int64 {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

func (r *recorder) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}

//nolint:unused // 保留给未来需要 WriteString 的调用方
func (r *recorder) WriteString(s string) (int, error) {
	r.buf.WriteString(s)
	return r.ResponseWriter.WriteString(s)
}
