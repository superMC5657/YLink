package service

import (
	"context"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
)

// telegram adapter：委托底层实现，late 注入收敛于此 seam。

// SetTelegram 注入 telegram 能力（router 装配期调用，仅触及本 adapter）。
func (m *adminTelegramAdapter) SetTelegram(tg *TelegramService) { m.tg = tg }

// SetupTelegramWebhook POST /admin/telegram/webhook/setup（F12，委托底层实现）。
func (m *adminTelegramAdapter) SetupTelegramWebhook(ctx context.Context, adminID int64, ip string) (*model.AdminTelegramWebhookSetupResp, error) {
	if m.tg == nil {
		return nil, errs.New(50000, "Telegram 服务未初始化")
	}
	return m.tg.SetupTelegramWebhook(ctx, adminID, ip)
}
