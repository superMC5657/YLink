package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAndParse(t *testing.T) {
	m := NewManager("test-secret-key-0123456789abcdef", 2*time.Hour, 336*time.Hour)
	access, refresh, err := m.Generate(10086, 1, "jti-1", 3)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)

	c, err := m.Parse(access)
	assert.NoError(t, err)
	assert.Equal(t, int64(10086), c.UserID)
	assert.Equal(t, 1, c.Role)
	assert.Equal(t, "jti-1", c.JTI)
	assert.Equal(t, "access", c.TokenType)
	assert.Equal(t, int64(3), c.SV) // 会话版本号快照往返

	rc, err := m.Parse(refresh)
	assert.NoError(t, err)
	assert.Equal(t, "refresh", rc.TokenType)
	assert.Equal(t, int64(3), rc.SV)
}

func TestParseExpired(t *testing.T) {
	m := NewManager("test-secret-key-0123456789abcdef", time.Nanosecond, time.Nanosecond)
	time.Sleep(time.Millisecond)
	access, _, _ := m.Generate(1, 0, "jti", 0)
	_, err := m.Parse(access)
	assert.Error(t, err)
}

func TestParseBadToken(t *testing.T) {
	m := NewManager("test-secret-key-0123456789abcdef", time.Hour, time.Hour)
	_, err := m.Parse("not-a-token")
	assert.Error(t, err)
}
