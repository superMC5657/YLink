package handler

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
)

// ---- 管理端 · 第四批（xboard-gap-fill）端点：F10 订阅模板 / F12 Telegram ----

// ListSubscriptionTemplates 订阅模板列表
// @Summary 订阅模板列表（内置生成器模板 + 自定义覆盖合并）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=object{list=[]model.AdminSubscriptionTemplateItem}}
// @Router /admin/subscription-templates [get]
func (h *Admin) ListSubscriptionTemplates(c *gin.Context) {
	data, err := h.svc.ListSubscriptionTemplates(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// SaveSubscriptionTemplate 保存订阅模板
// @Summary 保存订阅模板（Go template 语法，保存前用示例数据渲染校验）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param name path string true "客户端类型（clash/sing-box/v2ray）"
// @Param body body model.AdminSubscriptionTemplateReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/subscription-templates/{name} [put]
func (h *Admin) SaveSubscriptionTemplate(c *gin.Context) {
	var req model.AdminSubscriptionTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.SaveSubscriptionTemplate(c.Request.Context(), h.adminID(c), c.Param("name"), req.Content, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ResetSubscriptionTemplate 恢复默认订阅模板
// @Summary 恢复订阅模板内置生成器（删除自定义行）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param name path string true "客户端类型"
// @Success 200 {object} resp.Body
// @Router /admin/subscription-templates/{name} [delete]
func (h *Admin) ResetSubscriptionTemplate(c *gin.Context) {
	if err := h.svc.ResetSubscriptionTemplate(c.Request.Context(), h.adminID(c), c.Param("name"), c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// PreviewSubscriptionTemplate 预览订阅模板
// @Summary 预览订阅模板（按当前模板用示例数据渲染；v2ray 返回 base64 前文本）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param name path string true "客户端类型"
// @Success 200 {object} resp.Body{data=model.AdminSubscriptionTemplatePreviewResp}
// @Router /admin/subscription-templates/{name}/preview [post]
func (h *Admin) PreviewSubscriptionTemplate(c *gin.Context) {
	content, err := h.svc.PreviewSubscriptionTemplate(c.Request.Context(), c.Param("name"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, model.AdminSubscriptionTemplatePreviewResp{Name: c.Param("name"), Content: content})
}

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
