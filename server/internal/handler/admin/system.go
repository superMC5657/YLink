package admin

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
)

// ---- 设置 ----

func (h *Admin) ListSettings(c *gin.Context) {
	data, err := h.svc.ListSettings(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// SaveSetting 保存站点设置
// @Summary 保存站点设置
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminSettingsReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/settings [put]
func (h *Admin) SaveSetting(c *gin.Context) {
	var req model.AdminSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.SaveSetting(c.Request.Context(), req.Key, req.Value); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 版本检查（F20） ----

// Version 版本信息
// @Summary 版本检查 + 变更日志（配置 update.manifest_url 时远端拉取最新版本）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.AdminVersionResp}
// @Router /admin/version [get]
func (h *Admin) Version(c *gin.Context) {
	data, err := h.svc.VersionInfo(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// ---- Telegram（F12 管理端） ----

// SetupTelegramWebhook 注册 Telegram webhook
// @Summary 注册 Telegram webhook（调 setWebhook；webhook_secret 缺失时自动生成并保存）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.AdminTelegramWebhookSetupResp}
// @Router /admin/telegram/webhook/setup [post]
func (h *Admin) SetupTelegramWebhook(c *gin.Context) {
	data, err := h.svc.SetupTelegramWebhook(c.Request.Context(), h.adminID(c), c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}
