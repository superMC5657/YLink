package handler

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/service"
)

// Telegram F12 用户端绑定 + 公开 webhook 端点。
type Telegram struct {
	svc *service.TelegramService
}

func NewTelegram(svc *service.TelegramService) *Telegram { return &Telegram{svc: svc} }

// BindCode 获取绑定验证码
// @Summary 获取 Telegram 绑定验证码（10 分钟内发送 /bind <code> 给 bot）
// @Tags 用户
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.TelegramBindCodeResp}
// @Router /user/telegram/bind-code [post]
func (h *Telegram) BindCode(c *gin.Context) {
	data, err := h.svc.BindCode(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Unbind 解绑 Telegram
// @Summary 解绑 Telegram（解绑后立即停止推送）
// @Tags 用户
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body
// @Router /user/telegram/unbind [post]
func (h *Telegram) Unbind(c *gin.Context) {
	if err := h.svc.Unbind(c.Request.Context(), middleware.UserID(c)); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Webhook Telegram 回调
// @Summary Telegram bot webhook（X-Telegram-Bot-Api-Secret-Token 头校验；/bind、/unbind 命令）
// @Tags Webhook
// @Accept json
// @Produce plain
// @Success 200 {string} string "ok"
// @Failure 403 {string} string "forbidden"
// @Router /telegram/webhook [post]
func (h *Telegram) Webhook(c *gin.Context) {
	var up service.TelegramUpdate
	if err := c.ShouldBindJSON(&up); err != nil {
		// 非 Telegram 更新体：静默 200，避免对端重试风暴
		c.String(200, "ok")
		return
	}
	secret := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	if err := h.svc.Webhook(c.Request.Context(), secret, &up); err != nil {
		c.String(403, "forbidden")
		return
	}
	// 命令处理结果经 bot sendMessage 回执；对 Telegram 始终 200
	c.String(200, "ok")
}
