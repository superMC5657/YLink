package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// doCORS 构造一次经 CORS 中间件的请求，返回响应头与状态码。
func doCORS(t *testing.T, allowOrigins []string, origin string, method string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(allowOrigins))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/ping", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestCORS(t *testing.T) {
	allow := []string{
		"http://localhost",
		"https://localhost",
		"http://localhost:5174",
		"https://localhost:5174",
		"https://panel.example.com",
	}

	t.Run("https origin 放行", func(t *testing.T) {
		for _, origin := range []string{
			"https://localhost",
			"https://localhost:5174",
			"https://panel.example.com",
		} {
			w := doCORS(t, allow, origin, http.MethodGet)
			assert.Equal(t, http.StatusOK, w.Code, origin)
			assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"), origin)
			assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"), origin)
		}
	})

	t.Run("http origin 放行", func(t *testing.T) {
		w := doCORS(t, allow, "http://localhost:5174", http.MethodGet)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:5174", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("非白名单 origin 不放行(浏览器将拦截)", func(t *testing.T) {
		w := doCORS(t, allow, "http://evil.example.com", http.MethodGet)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("OPTIONS 预检返回 204 且放行头完整", func(t *testing.T) {
		w := doCORS(t, allow, "https://panel.example.com", http.MethodOptions)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "https://panel.example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	})

	t.Run("无 Origin 头(同源/服务端调用)不受影响", func(t *testing.T) {
		w := doCORS(t, allow, "", http.MethodGet)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
}
