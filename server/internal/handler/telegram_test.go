package handler

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/repo"
	"ylink-backend/internal/service"
)

// 缝测试：Telegram Webhook 的 secret 校验先于 body 读取（docs/api §19——
// 坏/缺 secret 的任意 body 一律 403；正确 secret + 坏 JSON 维持静默 200）。
// 最怕的失败=伪造请求借 bind 错误的 200 掩盖 403 语义。

const tgWebhookSettings = `{"bot_token":"123:ABC","bot_username":"b","webhook_secret":"s3cret","enabled":true}`

func newTelegramHandler(t *testing.T) (*Telegram, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	// settings 经 SettingService 60s 缓存读取，需要真实（mini）Redis 客户端
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := service.NewTelegramService(db, rdb, &repo.Repos{}, &config.Config{App: config.AppConfig{Name: "YLink"}}, nil)
	return NewTelegram(svc), mock
}

// expectSettingsRows 预置 n 次 settings 读取（缓存未命中时查库一次，命中不查）。
func expectSettingsRows(mock sqlmock.Sqlmock, n int, raw string) {
	for i := 0; i < n; i++ {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(raw))
	}
}

func callWebhook(h *Telegram, secretHeader, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/telegram/webhook", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if secretHeader != "" {
		c.Request.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretHeader)
	}
	h.Webhook(c)
	return w
}

// 坏/缺 secret 的任意 body（含非法 JSON）→ 403，且不再读 body 之后的链路。
func TestWebhookRejectsBadSecretBeforeBody(t *testing.T) {
	cases := []struct {
		name   string
		secret string
	}{
		{"缺失 secret", ""},
		{"错误 secret", "wrong"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, mock := newTelegramHandler(t)
			expectSettingsRows(mock, 1, tgWebhookSettings)
			w := callWebhook(h, tc.secret, `{"not": valid json`)
			assert.Equal(t, 403, w.Code)
			assert.Equal(t, "forbidden", w.Body.String())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// secret 未配置（settings 读取失败/为空）→ 同样 403，不受 body 影响。
func TestWebhookRejectsWhenSecretUnconfigured(t *testing.T) {
	h, mock := newTelegramHandler(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnError(assert.AnError)
	w := callWebhook(h, "whatever", `{"not": valid json`)
	assert.Equal(t, 403, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 正确 secret + 坏 JSON → 维持原契约：静默 200（不触发 Telegram 重试风暴）。
func TestWebhookValidSecretBadJSONStillOK(t *testing.T) {
	h, mock := newTelegramHandler(t)
	expectSettingsRows(mock, 1, tgWebhookSettings)
	w := callWebhook(h, "s3cret", `{"not": valid json`)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "ok", w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

// 正确 secret + 合法更新（空 text，无命令分支）→ 200，不外呼 bot API。
// settings 仅查库 1 次：handler 预检装入 60s 缓存，Webhook 内部读取命中缓存。
func TestWebhookValidSecretValidBodyOK(t *testing.T) {
	h, mock := newTelegramHandler(t)
	expectSettingsRows(mock, 1, tgWebhookSettings)
	w := callWebhook(h, "s3cret", `{"message":{"chat":{"id":1},"text":""}}`)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "ok", w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}
