package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"ylink-backend/internal/model"
)

func TestUniProxyAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	nodes := map[int64]*model.Server{
		1: {
			ID:      1,
			NodeKey: "valid-token-1",
			Status:  1, // normal
		},
		2: {
			ID:      2,
			NodeKey: "valid-token-2",
			Status:  2, // crowded: allowed
		},
		3: {
			ID:      3,
			NodeKey: "disabled-token-3",
			Status:  3, // offline/maintenance: inactive
		},
	}

	lookupCalls := 0
	lookup := func(ctx context.Context, nodeID int64) (*model.Server, error) {
		lookupCalls++
		if srv, ok := nodes[nodeID]; ok {
			return srv, nil
		}
		return nil, errors.New("not found")
	}

	router := gin.New()
	router.Use(UniProxyAuth(lookup, rdb))
	router.GET("/test", func(c *gin.Context) {
		id, ok := GetUniProxyServerID(c)
		c.JSON(http.StatusOK, gin.H{
			"server_id":          id,
			"ok":                 ok,
			"uniproxy_server_id": UniProxyServerID(c),
		})
	})

	doReq := func(query string, headers map[string]string) (int, map[string]any) {
		url := "/test"
		if query != "" {
			url += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body
	}

	assertUnauthorized := func(code int, body map[string]any) {
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Equal(t, float64(40100), body["code"])
		assert.Equal(t, "unauthorized", body["message"])
		assert.Nil(t, body["data"])
	}

	// 1. 缺失 node_id
	code, body := doReq("token=valid-token-1", nil)
	assertUnauthorized(code, body)

	// 2. 非法 node_id
	code, body = doReq("node_id=abc&token=valid-token-1", nil)
	assertUnauthorized(code, body)

	// 3. 缺失 token
	code, body = doReq("node_id=1", nil)
	assertUnauthorized(code, body)

	// 4. 节点不存在
	code, body = doReq("node_id=999&token=test", nil)
	assertUnauthorized(code, body)

	// 5. 节点维护/停用 (status = 3) -> 401
	code, body = doReq("node_id=3&token=disabled-token-3", nil)
	assertUnauthorized(code, body)

	// 6. token 错误
	code, body = doReq("node_id=1&token=wrong-token", nil)
	assertUnauthorized(code, body)

	// 7. 成功：正常节点 (status = 1) 通过 Query 传递 token
	lookupCalls = 0
	code, body = doReq("node_id=1&token=valid-token-1", nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(1), body["server_id"])
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, float64(1), body["uniproxy_server_id"])
	assert.Equal(t, 1, lookupCalls)

	// 8. 成功：命中 Redis 缓存（不再调用 lookup）
	code, body = doReq("node_id=1&token=valid-token-1", nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(1), body["server_id"])
	assert.Equal(t, 1, lookupCalls, "should hit redis cache and not call lookup")

	// 9. 权威 Redis 校验：缓存命中但 token 不匹配，立即 401 且不回源 DB
	code, body = doReq("node_id=1&token=cached-mismatch-token", nil)
	assertUnauthorized(code, body)
	assert.Equal(t, 1, lookupCalls, "authoritative redis mismatch should not call lookup")

	// 10. 成功：拥挤节点 (status = 2) 允许连接
	lookupCalls = 0
	code, body = doReq("node_id=2&token=valid-token-2", nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(2), body["server_id"])
	assert.Equal(t, 1, lookupCalls)

	// 11. 成功：通过 Header Token 传递
	mr.FastForward(61 * time.Second) // 缓存过期
	lookupCalls = 0
	code, body = doReq("node_id=1", map[string]string{"Token": "valid-token-1"})
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(1), body["server_id"])
	assert.Equal(t, 1, lookupCalls)

	// 12. 成功：通过 Header X-Token 传递
	mr.FastForward(61 * time.Second)
	code, body = doReq("node_id=1", map[string]string{"X-Token": "valid-token-1"})
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(1), body["server_id"])

	// 13. 成功：通过 Header X-Node-Key 传递
	mr.FastForward(61 * time.Second)
	code, body = doReq("node_id=1", map[string]string{"X-Node-Key": "valid-token-1"})
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(1), body["server_id"])

	// 14. 缓存键生成
	assert.Equal(t, "node:uniproxy:token:1", UniProxyTokenCacheKey(1))
}

func TestGetUniProxyServerID_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// 空上下文
	id, ok := GetUniProxyServerID(c)
	assert.False(t, ok)
	assert.Equal(t, int64(0), id)

	// 仅 uniproxy_server_id
	c.Set("uniproxy_server_id", int64(42))
	id, ok = GetUniProxyServerID(c)
	assert.True(t, ok)
	assert.Equal(t, int64(42), id)

	// server_id 优先
	c.Set("server_id", int64(100))
	id, ok = GetUniProxyServerID(c)
	assert.True(t, ok)
	assert.Equal(t, int64(100), id)
	assert.Equal(t, int64(100), UniProxyServerID(c))
}
