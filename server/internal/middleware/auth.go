package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"nanocloud/internal/pkg/errs"
	jwtpkg "nanocloud/internal/pkg/jwt"
)

const (
	roleUser  = 0
	roleAdmin = 1
	roleAgent = 2
)

// Auth 解析 Bearer access_token，注入 user_id / role / jti。
func Auth(mgr *jwtpkg.Manager) gin.HandlerFunc {
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
