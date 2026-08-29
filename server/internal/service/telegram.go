package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/logger"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

// ---- Telegram 机器人（F12 最小闭环） ----

// telegramSettings settings 表 telegram 键（管理端 JSON 编辑）。
type telegramSettings struct {
	BotToken      string `json:"bot_token"`
	BotUsername   string `json:"bot_username"`
	WebhookSecret string `json:"webhook_secret"`
	Enabled       bool   `json:"enabled"`
}

const (
	tgBindCodeTTL    = 10 * time.Minute
	tgBindResendGap  = 60 * time.Second
	tgBindRateDaily  = 20
	tgHTTPTimeout    = 10 * time.Second
	tgWebhookAPIPath = "/api/v1/telegram/webhook"
)

// TelegramService Telegram 绑定 / webhook / 通知推送。
type TelegramService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	cfg   *config.Config
	http  *http.Client
	// sendFn 可替换的 sendMessage 实现（测试注入，避免单测外呼真实 API）。
	sendFn func(ctx context.Context, botToken string, chatID int64, text string) error
}

func NewTelegramService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config) *TelegramService {
	return &TelegramService{db: db, rdb: rdb, repos: repos, cfg: cfg, http: &http.Client{Timeout: tgHTTPTimeout}}
}

// doSend 发送前分流：测试注入 sendFn 时走注入实现。
func (s *TelegramService) doSend(ctx context.Context, botToken string, chatID int64, text string) error {
	if s.sendFn != nil {
		return s.sendFn(ctx, botToken, chatID, text)
	}
	return s.send(ctx, botToken, chatID, text)
}

func (s *TelegramService) settings(ctx context.Context) telegramSettings {
	var st telegramSettings
	if raw, err := s.repos.Setting.Get(s.db, "telegram"); err == nil {
		_ = json.Unmarshal([]byte(raw), &st)
	}
	return st
}

// BindCode POST /user/telegram/bind-code：签发绑定验证码（Redis 10min，60s 重发间隔 + 每日上限）。
func (s *TelegramService) BindCode(ctx context.Context, userID int64) (*model.TelegramBindCodeResp, error) {
	st := s.settings(ctx)
	if !st.Enabled || st.BotToken == "" {
		return nil, errs.New(40000, "Telegram 绑定未启用")
	}
	resendKey := redispkg.Key("tg", "bind", "rate", fmt.Sprint(userID))
	ok, _ := s.rdb.SetNX(ctx, resendKey, "1", tgBindResendGap).Result()
	if !ok {
		return nil, &errs.Error{Code: 42900, Message: "请求过于频繁，请稍后再试", HTTP: 429}
	}
	dayKey := redispkg.Key("tg", "bind", "daily", fmt.Sprint(userID))
	n, _ := s.rdb.Incr(ctx, dayKey).Result()
	if n == 1 {
		s.rdb.Expire(ctx, dayKey, 24*time.Hour)
	}
	if n > tgBindRateDaily {
		return nil, &errs.Error{Code: 42900, Message: "今日绑定验证码次数已达上限", HTTP: 429}
	}

	code := randomCode(6)
	s.rdb.Set(ctx, redispkg.Key("tg", "bind", "code", code), userID, tgBindCodeTTL)
	return &model.TelegramBindCodeResp{Code: code, BotUsername: st.BotUsername, TTLMinutes: 10}, nil
}

// Unbind POST /user/telegram/unbind：解绑后立即停止推送（验收要点）。
// 置空走条件更新——UserRepo.Update 的 Updates(struct) 跳过零值字段，无法清列。
func (s *TelegramService) Unbind(ctx context.Context, userID int64) error {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return errs.ErrNotFound
	}
	if user.TelegramID == nil {
		return errs.New(40000, "尚未绑定 Telegram")
	}
	return s.db.Model(&model.User{}).Where("id = ?", userID).Update("telegram_id", nil).Error
}

// telegramUpdate webhook 请求体（仅解析所需字段）。
type TelegramUpdate struct {
	Message *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

// Webhook POST /telegram/webhook：secret 头校验 → /bind <code> 写 users.telegram_id /
// /unbind 清除 → bot sendMessage 回执。命令之外的更新静默 200（由 handler 保证）。
func (s *TelegramService) Webhook(ctx context.Context, secret string, up *TelegramUpdate) error {
	st := s.settings(ctx)
	if st.BotToken == "" || st.WebhookSecret == "" || secret != st.WebhookSecret {
		return errs.New(40300, "forbidden")
	}
	if up.Message == nil || up.Message.Text == "" {
		return nil
	}
	chatID := up.Message.Chat.ID
	text := strings.TrimSpace(up.Message.Text)
	switch {
	case text == "/unbind":
		// 解绑始终放行：总开关关闭后允许用户切断推送。
		s.handleUnbindCommand(ctx, st.BotToken, chatID)
	case !st.Enabled:
		// enabled 是总开关：关闭后 /bind、/start 等静默忽略（webhook 仍 200）。
	case text == "/start" || text == "/help":
		s.reply(ctx, st.BotToken, chatID, "发送 /bind <验证码> 绑定账号（验证码在网站个人信息页获取）；/unbind 解绑。")
	case strings.HasPrefix(text, "/bind"):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/bind"))
		s.handleBind(ctx, st.BotToken, chatID, code)
	}
	return nil
}

// handleBind 校验验证码并绑定；同一 chat 仅可绑定一个账号（0008 部分唯一索引兜底）。
func (s *TelegramService) handleBind(ctx context.Context, botToken string, chatID int64, code string) {
	if code == "" {
		s.reply(ctx, botToken, chatID, "用法：/bind <验证码>（验证码在网站个人信息页获取）")
		return
	}
	uidAny, err := s.rdb.GetDel(ctx, redispkg.Key("tg", "bind", "code", code)).Result()
	if err != nil {
		s.reply(ctx, botToken, chatID, "验证码无效或已过期，请在网站重新获取。")
		return
	}
	var userID int64
	if _, err := fmt.Sscan(uidAny, &userID); err != nil {
		s.reply(ctx, botToken, chatID, "验证码无效，请在网站重新获取。")
		return
	}
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		s.reply(ctx, botToken, chatID, "账号不存在，绑定失败。")
		return
	}
	if user.TelegramID != nil && *user.TelegramID == chatID {
		s.reply(ctx, botToken, chatID, "该账号已绑定此 Telegram，无需重复绑定。")
		return
	}
	if user.IsBanned {
		s.reply(ctx, botToken, chatID, "账号已被封禁，绑定失败。")
		return
	}
	// 条件更新只写 telegram_id，避免 stale 快照覆盖并发修改的余额等字段；
	// 0008 唯一索引冲突在此落库报错：该 chat 已绑定其他账号。
	res := s.db.Model(&model.User{}).
		Where("id = ? AND is_banned = ?", userID, false).
		Update("telegram_id", chatID)
	if res.Error != nil {
		logger.L().Warn("telegram bind save", zapS("user_id", fmt.Sprint(userID)), zapE(res.Error))
		s.reply(ctx, botToken, chatID, "绑定失败：该 Telegram 可能已绑定其他账号。")
		return
	}
	if res.RowsAffected == 0 {
		s.reply(ctx, botToken, chatID, "账号状态已变更，绑定失败。")
		return
	}
	s.reply(ctx, botToken, chatID, "绑定成功 ✅ 到期/流量提醒将同步推送到此处。")
}

// handleUnbindCommand bot 侧 /unbind：按 chatID 反查解绑。
func (s *TelegramService) handleUnbindCommand(ctx context.Context, botToken string, chatID int64) {
	res := s.db.Model(&model.User{}).Where("telegram_id = ?", chatID).Update("telegram_id", nil)
	if res.Error != nil || res.RowsAffected == 0 {
		s.reply(ctx, botToken, chatID, "该 Telegram 未绑定任何账号。")
		return
	}
	s.reply(ctx, botToken, chatID, "已解绑，将不再推送通知。")
}

// reply bot sendMessage 回执；失败仅记日志（webhook 对 Telegram 始终 200）。
func (s *TelegramService) reply(ctx context.Context, botToken string, chatID int64, text string) {
	if err := s.doSend(ctx, botToken, chatID, text); err != nil {
		logger.L().Warn("telegram reply", zapS("chat_id", fmt.Sprint(chatID)), zapE(err))
	}
}

// send 调 Telegram Bot API sendMessage（不引 SDK）。
func (s *TelegramService) send(ctx context.Context, botToken string, chatID int64, text string) error {
	body, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": text})
	url := "https://api.telegram.org/bot" + botToken + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api status %d", resp.StatusCode)
	}
	return nil
}

// NotifyUser 通知推送：用户已绑定且 bot 启用时发送，失败仅记日志不阻断主流程
// （spec F12 验收要点：推送失败不影响主流程，解绑后立即停止推送）。
func (s *TelegramService) NotifyUser(ctx context.Context, u model.User, text string) {
	if u.TelegramID == nil {
		return
	}
	st := s.settings(ctx)
	if !st.Enabled || st.BotToken == "" {
		return
	}
	if err := s.doSend(ctx, st.BotToken, *u.TelegramID, "【"+s.cfg.App.Name+"】"+text); err != nil {
		logger.L().Warn("telegram notify", zapS("user_id", fmt.Sprint(u.ID)), zapE(err))
	}
}

// SetupTelegramWebhook POST /admin/telegram/webhook/setup：调 setWebhook 注册回调地址；
// webhook_secret 缺失时自动生成并回写 settings，写审计。
func (s *TelegramService) SetupTelegramWebhook(ctx context.Context, adminID int64, ip string) (*model.AdminTelegramWebhookSetupResp, error) {
	st := s.settings(ctx)
	if st.BotToken == "" {
		return nil, errs.New(40000, "请先在设置中配置 telegram.bot_token")
	}
	if st.WebhookSecret == "" {
		st.WebhookSecret = randomHex(32)
		raw, _ := json.Marshal(st)
		if err := s.repos.Setting.Set(s.db, "telegram", string(raw)); err != nil {
			return nil, err
		}
	}
	webhookURL := strings.TrimSuffix(s.cfg.App.BaseURL, "/") + tgWebhookAPIPath
	payload, _ := json.Marshal(map[string]any{
		"url":                  webhookURL,
		"secret_token":         st.WebhookSecret,
		"allowed_updates":      []string{"message"},
		"drop_pending_updates": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.telegram.org/bot"+st.BotToken+"/setWebhook", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, errs.New(50000, "Telegram API 请求失败: "+err.Error())
	}
	defer resp.Body.Close()
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	_ = s.repos.Audit.Create(s.db, &model.AuditLog{
		AdminID: adminID, Action: "telegram_webhook_setup", Target: strp("telegram"),
		Detail: strp(fmt.Sprintf(`{"ok":%v,"url":%q}`, out.OK, webhookURL)), IP: strp(ip),
	})
	if !out.OK {
		return nil, errs.New(50000, "Telegram API 返回失败: "+out.Description)
	}
	return &model.AdminTelegramWebhookSetupResp{WebhookURL: webhookURL, Message: "webhook 注册成功"}, nil
}

func strp(s string) *string { return &s }

// randomHex 生成 n 字节十六进制随机串（webhook secret）。
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
