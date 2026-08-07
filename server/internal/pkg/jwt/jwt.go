// Package jwt 封装 golang-jwt/v5 的签发与解析。
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 为业务自定义声明。
type Claims struct {
	UserID    int64  `json:"uid"`
	Role      int    `json:"role"`
	JTI       string `json:"jti"`   // refresh 白名单索引
	TokenType string `json:"ttype"` // access / refresh，防止两类令牌混用
	jwt.RegisteredClaims
}

// Manager 负责签发/解析 access 与 refresh token。
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// Generate 签发 access/refresh 令牌对；jti 用于 refresh 白名单（Redis）。
// 两类令牌带不同 TokenType，防止混淆使用。
func (m *Manager) Generate(userID int64, role int, jti string) (access, refresh string, err error) {
	now := time.Now()
	access, err = m.sign(&Claims{
		UserID:    userID,
		Role:      role,
		JTI:       jti,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	})
	if err != nil {
		return "", "", err
	}
	refresh, err = m.sign(&Claims{
		UserID:    userID,
		Role:      role,
		JTI:       jti,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
	})
	return access, refresh, err
}

func (m *Manager) sign(c *Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// Parse 校验签名与有效期并返回声明。
func (m *Manager) Parse(token string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(token, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	return c, nil
}

// AccessTTL 返回 access token 有效期（秒），用于响应 expires_in。
func (m *Manager) AccessTTL() time.Duration { return m.accessTTL }
