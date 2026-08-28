package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"ylink-backend/internal/pkg/errs"
	jwtpkg "ylink-backend/internal/pkg/jwt"
	redispkg "ylink-backend/internal/pkg/redis"
)

const (
	roleUser  = 0
	roleAdmin = 1
	roleAgent = 2
)

// Auth 解析 Bearer access_token，注入 user_id / role / jti。
// 除签名与有效期外，还校验会话版本号（sv）：封禁/降级/登出等操作 bump 版本号后，
// 旧 access token 立即失效（401），无需等待 access TTL 自然过期。
func Auth(mgr *jwtpkg.Manager, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			Fail(c, errs.ErrUnauthorized)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := mgr.Parse(token)
		if err != nil || claims.UserID <= 0 || claims.TokenType != "access" {
			Fail(c, errs.ErrUnauthorized)
			return
		}
		// 会话版本号比对：Key 不存在视为 0（Redis 重启安全）；Redis 异常不阻断（退化为 TTL）。
		// 与踢下线标记（F14 auth:kill:{uid} Hash field=jti）合并为一次 Pipeline，控制鉴权 RTT。
		pipe := rdb.Pipeline()
		svCmd := pipe.Get(c.Request.Context(), redispkg.SessionVersionKey(claims.UserID))
		killCmd := pipe.HExists(c.Request.Context(), redispkg.SessionKillKey(claims.UserID), claims.JTI)
		_, _ = pipe.Exec(c.Request.Context())
		if cur, err := svCmd.Int64(); err == nil && claims.SV != cur {
			Fail(c, errs.ErrUnauthorized)
			return
		}
		if killed, _ := killCmd.Result(); killed {
			Fail(c, errs.ErrUnauthorized)
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("jti", claims.JTI)
		c.Next()
	}
}

// RequireRole 校验角色（用于 admin 分组：role=admin）。
func RequireRole(role int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if UserRole(c) != role {
			Fail(c, errs.ErrForbidden)
			return
		}
		c.Next()
	}
}
