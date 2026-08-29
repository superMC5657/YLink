package service

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/pkg/subscribe"
	"ylink-backend/internal/repo"
)

// ---- 订阅模板（F10） ----

func subTemplateRows(name, content string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"name", "content", "updated_at"}).
		AddRow(name, content, time.Now())
}

// 默认输出与内置生成器逐字节一致（自定义缺失 → 回退内置，无告警路径）。
func TestSubscriptionRenderFallsBackToBuiltin(t *testing.T) {
	e := newTestEnv(t)
	svc := NewSubscribeService(e.db, e.rdb, &repo.Repos{}, &config.Config{})

	u := sampleSubUser()
	nodes := sampleSubNodes()
	builtin, err := subscribe.Clash{}.Build(u, nodes)
	require.NoError(t, err)
	// 与重构前逐字节一致：非空节点列表的输出以换行结尾
	assert.True(t, strings.HasSuffix(string(builtin), "\n"))

	// 无自定义行 → 查询报错回退
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscription_templates"`)).
		WillReturnError(assert.AnError)
	out, err := svc.renderSubscription(subscribe.Clash{}, u, nodes, "YLink", "ui")
	require.NoError(t, err)
	assert.Equal(t, string(builtin), string(out))
}

// 自定义模板可用时渲染结果生效；节点块经预渲染变量注入。
func TestSubscriptionRenderCustom(t *testing.T) {
	e := newTestEnv(t)
	svc := NewSubscribeService(e.db, e.rdb, &repo.Repos{}, &config.Config{})

	u := sampleSubUser()
	nodes := sampleSubNodes()
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscription_templates"`)).
		WillReturnRows(subTemplateRows("clash", "# custom header\nproxies:\n{{.NodeBlock}}"))
	out, err := svc.renderSubscription(subscribe.Clash{}, u, nodes, "YLink", "ui")
	require.NoError(t, err)
	s := string(out)
	assert.True(t, len(s) > 0)
	assert.Contains(t, s, "# custom header")
	assert.Contains(t, s, "type: trojan") // 节点块预渲染注入完整
}

// 坏模板（语法错误/变量缺失）渲染失败 → 回退内置生成器，订阅不 500（spec F10 验收要点）。
func TestSubscriptionRenderBrokenCustomFallsBack(t *testing.T) {
	e := newTestEnv(t)
	svc := NewSubscribeService(e.db, e.rdb, &repo.Repos{}, &config.Config{})

	u := sampleSubUser()
	nodes := sampleSubNodes()
	builtin, err := subscribe.V2ray{}.Build(u, nodes)
	require.NoError(t, err)

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscription_templates"`)).
		WillReturnRows(subTemplateRows("v2ray", "{{.Links"))
	out, err := svc.renderSubscription(subscribe.V2ray{}, u, nodes, "YLink", "ui")
	require.NoError(t, err)
	assert.Equal(t, string(builtin), string(out))
}

// 保存前校验：语法错误拒绝（40000），不写库。
func TestSaveSubscriptionTemplateBadSyntax(t *testing.T) {
	e := newTestEnv(t)
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, &config.Config{App: config.AppConfig{Name: "YLink"}}, nil, nil)

	err := svc.SaveSubscriptionTemplate(context.Background(), 1, "clash", "{{.NodeBlock", "1.2.3.4")
	assert.Equal(t, 40000, codeOf(err))

	err = svc.SaveSubscriptionTemplate(context.Background(), 1, "unknown-format", "x", "1.2.3.4")
	assert.Equal(t, 40400, codeOf(err))
}

// 内置模板预览：示例数据渲染，clash 含节点块、v2ray 返回 base64 前文本。
func TestPreviewSubscriptionTemplateBuiltin(t *testing.T) {
	e := newTestEnv(t)
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, &config.Config{App: config.AppConfig{Name: "YLink"}}, nil, nil)

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscription_templates"`)).
		WillReturnError(assert.AnError)
	clash, err := svc.PreviewSubscriptionTemplate(context.Background(), "clash")
	require.NoError(t, err)
	assert.Contains(t, clash, "proxies:")
	assert.Contains(t, clash, "type: trojan")

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "subscription_templates"`)).
		WillReturnError(assert.AnError)
	v2, err := svc.PreviewSubscriptionTemplate(context.Background(), "v2ray")
	require.NoError(t, err)
	assert.Contains(t, v2, "trojan://") // base64 解码后可读
	if _, err := base64.StdEncoding.DecodeString(v2); err == nil {
		t.Fatal("v2ray 预览应为 base64 前的可读文本")
	}
}

// ---- Telegram 绑定 / webhook / 推送（F12） ----

func tgSettingsRow(raw string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"value"}).AddRow(raw)
}

const tgSettingsJSON = `{"bot_token":"123:ABC","bot_username":"ylink_bot","webhook_secret":"s3cret","enabled":true}`

func (e *testEnv) expectSettings(raw string) {
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnRows(tgSettingsRow(raw))
}

func newTgSvc(t *testing.T, e *testEnv) *TelegramService {
	t.Helper()
	return NewTelegramService(e.db, e.rdb, &repo.Repos{}, &config.Config{App: config.AppConfig{Name: "YLink"}})
}

// 未启用（settings 缺失）→ 拒绝签发验证码。
func TestTelegramBindCodeDisabled(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnError(assert.AnError)

	_, err := svc.BindCode(context.Background(), 1)
	assert.Equal(t, 40000, codeOf(err))
}

// 正常签发：60s 重发间隔拦截第二次；验证码写入 Redis 且 10min TTL。
func TestTelegramBindCodeFlow(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)

	e.expectSettings(tgSettingsJSON)
	resp, err := svc.BindCode(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, resp.Code, 6)
	assert.Equal(t, "ylink_bot", resp.BotUsername)

	// 验证码已在 Redis
	val, err := e.rdb.Get(context.Background(), redispkg.Key("tg", "bind", "code", resp.Code)).Result()
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	// 重发间隔内第二次 → 429
	e.expectSettings(tgSettingsJSON)
	_, err = svc.BindCode(context.Background(), 1)
	assert.Equal(t, 42900, codeOf(err))
}

// webhook /bind：secret 通过、验证码有效 → 写 users.telegram_id 并回执。
func TestTelegramWebhookBind(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)

	uid := int64(7)
	tgID := int64(555)
	u := &model.User{ID: uid, Email: "a@b.com", TelegramID: nil, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	e.expectSettings(tgSettingsJSON)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(u))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	var replied string
	svc.sendFn = func(_ context.Context, _ string, _ int64, text string) error {
		replied = text
		return nil
	}

	// 预置验证码
	code := "654321"
	e.rdb.Set(context.Background(), redispkg.Key("tg", "bind", "code", code), uid, time.Minute)

	up := &TelegramUpdate{Message: tgMessage(tgID, "/bind "+code)}
	err := svc.Webhook(context.Background(), "s3cret", up)
	require.NoError(t, err)
	assert.Contains(t, replied, "绑定成功")
}

// secret 不匹配 → 403（webhook 防伪造，验收要点）。
func TestTelegramWebhookBadSecret(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)
	e.expectSettings(tgSettingsJSON)

	up := &TelegramUpdate{Message: tgMessage(555, "/bind 111111")}
	err := svc.Webhook(context.Background(), "wrong-secret", up)
	assert.Equal(t, 40300, codeOf(err))
}

// 用户端解绑：telegram_id 置空（条件更新，验收要点：解绑后立即停止推送）。
func TestTelegramUnbind(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)

	tgID := int64(555)
	u := &model.User{ID: 1, Email: "a@b.com", TelegramID: &tgID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(u))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	require.NoError(t, svc.Unbind(context.Background(), 1))

	// 未绑定 → 40000
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(&model.User{ID: 1, Email: "a@b.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	err := svc.Unbind(context.Background(), 1)
	assert.Equal(t, 40000, codeOf(err))
}

// 推送降级：未绑定 / 未启用均不外呼；启用时发送且失败不影响调用方。
func TestTelegramNotifyUser(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)

	called := 0
	svc.sendFn = func(_ context.Context, _ string, _ int64, _ string) error {
		called++
		return assert.AnError // 发送失败也不向上传播
	}

	// 未绑定 → 不发送
	svc.NotifyUser(context.Background(), model.User{ID: 1}, "x")
	assert.Equal(t, 0, called)

	// 已绑定但未启用 → 不发送
	tgID := int64(555)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnError(assert.AnError)
	svc.NotifyUser(context.Background(), model.User{ID: 1, TelegramID: &tgID}, "x")
	assert.Equal(t, 0, called)

	// 启用 → 发送（前缀站点名）
	e.expectSettings(tgSettingsJSON)
	svc.NotifyUser(context.Background(), model.User{ID: 1, TelegramID: &tgID}, "到期提醒")
	assert.Equal(t, 1, called)
}

// tgMessage 构造 webhook 消息体。
func tgMessage(chatID int64, text string) *struct {
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text string `json:"text"`
} {
	m := &struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	}{}
	m.Chat.ID = chatID
	m.Text = text
	return m
}

// rtFunc 可注入的 RoundTripper（mock Telegram Bot API 响应）。
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// enabled=false 总开关：/bind 被拦截且不消费验证码（webhook 仍 200）。
func TestTelegramWebhookDisabledBlocksBind(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)
	e.expectSettings(`{"bot_token":"123:ABC","bot_username":"ylink_bot","webhook_secret":"s3cret","enabled":false}`)

	var replied string
	svc.sendFn = func(_ context.Context, _ string, _ int64, text string) error {
		replied = text
		return nil
	}

	code := "654321"
	e.rdb.Set(context.Background(), redispkg.Key("tg", "bind", "code", code), int64(7), time.Minute)

	up := &TelegramUpdate{Message: tgMessage(555, "/bind "+code)}
	err := svc.Webhook(context.Background(), "s3cret", up)
	require.NoError(t, err)
	assert.Empty(t, replied)

	// 验证码未被消费
	val, err := e.rdb.Get(context.Background(), redispkg.Key("tg", "bind", "code", code)).Result()
	require.NoError(t, err)
	assert.Equal(t, "7", val)
}

// enabled=false 总开关：/unbind 仍放行（解绑不被锁死）。
func TestTelegramWebhookDisabledAllowsUnbind(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)
	e.expectSettings(`{"bot_token":"123:ABC","bot_username":"ylink_bot","webhook_secret":"s3cret","enabled":false}`)
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	var replied string
	svc.sendFn = func(_ context.Context, _ string, _ int64, text string) error {
		replied = text
		return nil
	}

	up := &TelegramUpdate{Message: tgMessage(555, "/unbind")}
	err := svc.Webhook(context.Background(), "s3cret", up)
	require.NoError(t, err)
	assert.Contains(t, replied, "已解绑")
}

// Telegram API 返回 ok=false → 保留审计记录但返回错误（不再假成功）。
func TestTelegramWebhookSetupRejected(t *testing.T) {
	e := newTestEnv(t)
	svc := newTgSvc(t, e)
	e.expectSettings(tgSettingsJSON)
	svc.http = &http.Client{Transport: rtFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"description":"invalid webhook URL"}`)),
			Header:     http.Header{},
		}, nil
	})}
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	_, err := svc.SetupTelegramWebhook(context.Background(), 1, "127.0.0.1")
	require.Error(t, err)
	assert.Equal(t, 50000, codeOf(err))
	assert.Contains(t, err.Error(), "invalid webhook URL")
}
