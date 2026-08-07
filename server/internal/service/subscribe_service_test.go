package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nanocloud/internal/config"
	"nanocloud/internal/model"
	"nanocloud/internal/pkg/passwd"
	"nanocloud/internal/pkg/subscribe"
	"nanocloud/internal/repo"
)

func TestClashBuild(t *testing.T) {
	gen := subscribe.Clash{}
	out, err := gen.Build(&subscribe.User{Name: "u", SpeedLimit: intPtr(100)}, []subscribe.Node{
		{Name: "香港 01", Type: "trojan", Host: "1.2.3.4", Port: 443, Password: "pw", SNI: "hk.example.com", Rate: 1},
		{Name: "东京 01", Type: "shadowsocks", Host: "2.3.4.5", Port: 8388, Password: "sspw", Method: "aes-256-gcm", Rate: 1.5},
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "type: trojan")
	assert.Contains(t, s, "server: 1.2.3.4")
	assert.Contains(t, s, "cipher: aes-256-gcm")
	assert.Contains(t, s, "limit-speed:")
}

func TestV2rayBuild(t *testing.T) {
	gen := subscribe.V2ray{}
	out, err := gen.Build(&subscribe.User{}, []subscribe.Node{
		{Name: "香港 01", Type: "trojan", Host: "1.2.3.4", Port: 443, Password: "pw", SNI: "hk.example.com"},
	})
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(string(out))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "trojan://pw@1.2.3.4:443")
}

func TestSingBoxBuild(t *testing.T) {
	gen := subscribe.SingBox{}
	out, err := gen.Build(&subscribe.User{}, []subscribe.Node{
		{Name: "香港 01", Type: "vmess", Host: "1.2.3.4", Port: 443, Password: "uuid-1", Network: "ws", Path: "/path", Security: "tls", SNI: "hk.example.com"},
	})
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(out, &doc))
	outbounds := doc["outbounds"].([]any)
	assert.Len(t, outbounds, 1)
}

func TestGenerateInvalidToken(t *testing.T) {
	e := newTestEnv(t)
	svc := NewSubscribeService(e.db, e.rdb, &repo.Repos{}, &config.Config{})
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users`")).WillReturnError(assert.AnError)

	_, err := svc.Generate(context.Background(), "bad-token", "", "")
	assert.Equal(t, 40100, codeOf(err))
}

func TestGenerateNoSubscription(t *testing.T) {
	e := newTestEnv(t)
	svc := NewSubscribeService(e.db, e.rdb, &repo.Repos{}, &config.Config{App: config.AppConfig{BaseURL: "https://api.example.com"}})

	// 用户：无订阅
	now := time.Now()
	u := &model.User{ID: 1, Email: "a@b.com", SubToken: "tok", CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users`")).WillReturnRows(userRow(u))

	res, err := svc.Generate(context.Background(), "tok", "clash", "")
	require.NoError(t, err)
	assert.Contains(t, res.ContentType, "text/yaml")
	// 无订阅 → 提示节点
	assert.Contains(t, string(res.Content), "未购买套餐")
	assert.Contains(t, res.UserInfo, "expire=0")
}

func TestResetSubscribe(t *testing.T) {
	e := newTestEnv(t)
	svc := NewSubscribeService(e.db, e.rdb, &repo.Repos{}, &config.Config{App: config.AppConfig{BaseURL: "https://api.example.com"}})
	hash, _ := hashPassword("Passw0rd!")
	u := &model.User{ID: 1, Email: "a@b.com", PasswordHash: hash, SubToken: "old-token", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	// 查用户 + 更新
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users`")).WillReturnRows(userRow(u))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `users`")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	url, err := svc.ResetSubscribe(context.Background(), 1, "Passw0rd!")
	require.NoError(t, err)
	assert.True(t, strings.Contains(*url, "https://api.example.com/api/v1/client/subscribe/"))
	assert.False(t, strings.Contains(*url, "old-token"))
}

func hashPassword(p string) (string, error) {
	return passwd.Hash(p)
}

func intPtr(i int) *int { return &i }
