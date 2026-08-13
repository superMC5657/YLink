package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jwtpkg "ylink-backend/internal/pkg/jwt"
	redispkg "ylink-backend/internal/pkg/redis"
)

func newAuthEnv(t *testing.T) (*jwtpkg.Manager, *redis.Client) {
	mgr := jwtpkg.NewManager("test-secret-key-0123456789abcdef0123456789", 2*time.Hour, 336*time.Hour)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mgr, rdb
}

// doAuth 构造一次经 Auth 中间件的请求，返回响应码与注入的 user_id（无则 -1）。
func doAuth(t *testing.T, mgr *jwtpkg.Manager, rdb *redis.Client, bearer string) (int, int64) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(mgr, rdb))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"uid": c.GetInt64("user_id")})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	r.ServeHTTP(w, req)
	uid := int64(-1)
	if w.Code == http.StatusOK {
		var body struct {
			UID int64 `json:"uid"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		uid = body.UID
	}
	return w.Code, uid
}

func TestAuthMiddleware(t *testing.T) {
	mgr, rdb := newAuthEnv(t)
	ctx := context.Background()

	t.Run("无 Authorization 头 → 401", func(t *testing.T) {
		code, _ := doAuth(t, mgr, rdb, "")
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("无效 token → 401", func(t *testing.T) {
		code, _ := doAuth(t, mgr, rdb, "not-a-token")
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("refresh token 不可当 access 用 → 401", func(t *testing.T) {
		_, refresh, err := mgr.Generate(7, 0, "jti-r", 0)
		require.NoError(t, err)
		code, _ := doAuth(t, mgr, rdb, refresh)
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("有效 access 且 SV 匹配 → 200 注入 user_id", func(t *testing.T) {
		access, _, err := mgr.Generate(7, 0, "jti-a", 0)
		require.NoError(t, err)
		code, uid := doAuth(t, mgr, rdb, access)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, int64(7), uid)
	})

	t.Run("Redis 无版本 Key 视为 0，sv=0 通过", func(t *testing.T) {
		access, _, err := mgr.Generate(8, 1, "jti-b", 0)
		require.NoError(t, err)
		code, uid := doAuth(t, mgr, rdb, access)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, int64(8), uid)
	})

	t.Run("封禁/降级 bump 后旧 access 立即失效 → 401", func(t *testing.T) {
		access, _, err := mgr.Generate(9, 0, "jti-c", 0)
		require.NoError(t, err)
		code, uid := doAuth(t, mgr, rdb, access)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, int64(9), uid)

		// 模拟封禁：bump 会话版本号（admin UpdateUser 内部同样调用）
		require.NoError(t, rdb.Incr(ctx, redispkg.SessionVersionKey(9)).Err())

		code, uid = doAuth(t, mgr, rdb, access)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Equal(t, int64(-1), uid)

		// 重新登录（重新签发）后 sv=1 匹配，恢复可用
		access2, _, err := mgr.Generate(9, 0, "jti-d", 1)
		require.NoError(t, err)
		code, uid = doAuth(t, mgr, rdb, access2)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, int64(9), uid)
	})
}
