package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// doNodeAuth 构造一次经 NodeAuth 中间件的请求,返回响应码与注入的 server_id(无则 -1)。
// lookupCalls 统计 DB 查询次数(验证缓存生效)。
func doNodeAuth(t *testing.T, rdb *redis.Client, lookup func(context.Context, string) (int64, error), key string, lookupCalls *int) (int, int64) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NodeAuth(lookup, rdb))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"sid": c.GetInt64("server_id")})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	if key != "" {
		req.Header.Set("X-Node-Key", key)
	}
	r.ServeHTTP(w, req)
	sid := int64(-1)
	if w.Code == http.StatusOK {
		var body struct {
			SID int64 `json:"sid"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		sid = body.SID
	}
	return w.Code, sid
}

func TestNodeAuthMiddleware(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	lookup := func(c context.Context, key string) (int64, error) {
		if key == "valid-key" {
			return 42, nil
		}
		return 0, errors.New("not found")
	}
	wrapCount := func(f func(context.Context, string) (int64, error), calls *int) func(context.Context, string) (int64, error) {
		return func(c context.Context, k string) (int64, error) {
			*calls++
			return f(c, k)
		}
	}

	t.Run("无 X-Node-Key → 401", func(t *testing.T) {
		calls := 0
		code, sid := doNodeAuth(t, rdb, wrapCount(lookup, &calls), "", &calls)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Equal(t, int64(-1), sid)
		assert.Zero(t, calls, "缺头不应触发 DB 查询")
	})

	t.Run("未知密钥 → 401", func(t *testing.T) {
		calls := 0
		code, _ := doNodeAuth(t, rdb, wrapCount(lookup, &calls), "bad-key", &calls)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Equal(t, 1, calls)
	})

	t.Run("有效密钥 → 注入 server_id 并缓存", func(t *testing.T) {
		calls := 0
		code, sid := doNodeAuth(t, rdb, wrapCount(lookup, &calls), "valid-key", &calls)
		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, int64(42), sid)
		assert.Equal(t, 1, calls)

		// 第二次请求命中缓存,不再查库
		code2, sid2 := doNodeAuth(t, rdb, wrapCount(lookup, &calls), "valid-key", &calls)
		require.Equal(t, http.StatusOK, code2)
		assert.Equal(t, int64(42), sid2)
		assert.Equal(t, 1, calls, "缓存命中不应触发 DB 查询")
		assert.True(t, mr.Exists(NodeKeyCacheKey("valid-key")))
	})

	_ = ctx
}
