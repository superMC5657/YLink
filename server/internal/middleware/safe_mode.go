package middleware

import (
	"net"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/pkg/errs"
)

// SafeMode F22 域名白名单：开启后，请求 Host（去端口）不在白名单内一律 403。
// 用于防止用户端/订阅端被绑定到仿冒域名；管理端、订阅下发、支付回调同受保护。
// allowedHosts 应包含 App.BaseURL 的 host 与部署方追加的 SafeDomains（router 层负责汇总）。
func SafeMode(enabled bool, allowedHosts []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, h := range allowedHosts {
		h = normalizeHost(h)
		if h != "" {
			allowed[h] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		if !enabled || len(allowed) == 0 {
			c.Next()
			return
		}
		host := normalizeHost(c.Request.Host)
		if _, ok := allowed[host]; ok {
			c.Next()
			return
		}
		Fail(c, errs.ErrForbidden)
	}
}

// normalizeHost 提取主机名：去端口、去 scheme 前缀、转小写（IPv6 保留方括号形式归一）。
func normalizeHost(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	// 允许直接配置 URL（如 https://panel.example.com）
	if i := strings.Index(raw, "://"); i >= 0 {
		u, err := url.Parse(raw)
		if err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
		raw = raw[i+3:]
	}
	if host, _, err := net.SplitHostPort(raw); err == nil && host != "" {
		return host
	}
	return strings.Trim(raw, "[]")
}
