package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"ylink-backend/internal/pkg/errs"
)

func safeModeRouter(enabled bool, hosts []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SafeMode(enabled, hosts))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestSafeModeDisabled(t *testing.T) {
	r := safeModeRouter(false, []string{"panel.example.com"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	assert.Equal(t, http.StatusOK, w.Code, "关闭 safe_mode 时任意 Host 放行")
}

func TestSafeModeAllowedHosts(t *testing.T) {
	cases := []struct {
		name string
		host string // 请求 Host 头
		code int
	}{
		{"白名单裸域名", "panel.example.com", http.StatusOK},
		{"白名单域名带端口", "panel.example.com:8443", http.StatusOK},
		{"白名单大小写不敏感", "PANEL.EXAMPLE.COM", http.StatusOK},
		{"非白名单域名", "evil.example.net", http.StatusForbidden},
		{"非白名单域名带端口", "evil.example.net:80", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := safeModeRouter(true, []string{"https://panel.example.com", "sub.example.com"})
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.Host = tc.host
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tc.code, w.Code)
		})
	}

	// 403 走统一信封
	r := safeModeRouter(true, []string{"panel.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Host = "evil.example.net"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), `40300`)
	e := errs.ErrForbidden
	assert.Equal(t, 403, e.HTTP)
}

func TestNormalizeHost(t *testing.T) {
	assert.Equal(t, "panel.example.com", normalizeHost("Panel.Example.com:443"))
	assert.Equal(t, "panel.example.com", normalizeHost("https://panel.example.com"))
	assert.Equal(t, "panel.example.com", normalizeHost("http://panel.example.com:8080/api"))
	assert.Equal(t, "", normalizeHost("  "))
	assert.Equal(t, "2001-db8--1.ipv6-literal.net", normalizeHost("2001-DB8--1.ipv6-literal.NET"))
}
