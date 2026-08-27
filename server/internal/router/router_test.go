package router

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
)

// TestAdminUserRouteCollisions F05：gin 同层静态(users/batch、users/mail、users/export)与
// 参数(users/:id/...)混合注册不得 panic（路由冲突会在 New 注册时 panic），且目标路由全部存在。
func TestAdminUserRouteCollisions(t *testing.T) {
	r := New(Deps{Cfg: newTestCfg(), DB: &gorm.DB{}})

	set := make(map[string]bool)
	for _, ri := range r.Routes() {
		set[ri.Method+" "+ri.Path] = true
	}
	want := []string{
		http.MethodGet + " /api/v1/admin/users/export",
		http.MethodPost + " /api/v1/admin/users/batch",
		http.MethodPost + " /api/v1/admin/users/mail",
		http.MethodPost + " /api/v1/admin/users/:id/sub-token/reset",
		http.MethodGet + " /api/v1/admin/audit-logs",
		http.MethodGet + " /api/v1/client/subscribe/:token",
		http.MethodGet + " /api/v1/admin/users",
	}
	for _, w := range want {
		assert.True(t, set[w], "路由未注册: %s", w)
	}
}

func newTestCfg() *config.Config {
	cfg := &config.Config{}
	cfg.App.Env = "development"
	cfg.App.BaseURL = "http://localhost:8081"
	cfg.JWT.Secret = "test-secret-key-0123456789abcdef0123456789"
	cfg.JWT.AccessTTL = 3600
	cfg.JWT.RefreshTTL = 7200
	return cfg
}
