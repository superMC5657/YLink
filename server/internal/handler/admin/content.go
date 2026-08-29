package admin

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
)

// ---- 公告 ----

// ListNotices 公告列表（含隐藏）
// @Summary 公告列表
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=[]model.AdminNoticeItem}
// @Router /admin/notices [get]
func (h *Admin) ListNotices(c *gin.Context) {
	data, err := h.svc.ListAllNotices(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateNotice(c *gin.Context) {
	var req model.AdminNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateNotice(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateNotice(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateNotice(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteNotice(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteNotice(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// SortNotices 公告排序（F15）
// @Summary 公告排序（items 按 sort 值更新，单事务）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminSortReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/notices/sort [post]
func (h *Admin) SortNotices(c *gin.Context) {
	var req model.AdminSortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.SortNotices(c.Request.Context(), h.adminID(c), req.Items, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 知识库 ----

// ListKnowledges 知识库列表（含隐藏）
// @Summary 知识库列表
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=[]model.AdminKnowledgeItem}
// @Router /admin/knowledges [get]
func (h *Admin) ListKnowledges(c *gin.Context) {
	data, err := h.svc.ListAllKnowledges(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateKnowledge(c *gin.Context) {
	var req model.AdminKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateKnowledge(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateKnowledge(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateKnowledge(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteKnowledge(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteKnowledge(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// SortKnowledges 知识库排序（F15）
// @Summary 知识库排序（items 按 sort 值更新，单事务）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminSortReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/knowledges/sort [post]
func (h *Admin) SortKnowledges(c *gin.Context) {
	var req model.AdminSortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.SortKnowledges(c.Request.Context(), h.adminID(c), req.Items, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 知识库分类（F15） ----

// ListKnowledgeCategories 分类列表
// @Summary 知识库分类列表（language 为空返回全部，含文档计数）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param language query string false "语言（zh-CN/en-US，空=全部）"
// @Success 200 {object} resp.Body{data=object{list=[]model.AdminKnowledgeCategoryItem}}
// @Router /admin/knowledge-categories [get]
func (h *Admin) ListKnowledgeCategories(c *gin.Context) {
	data, err := h.svc.ListKnowledgeCategories(c.Request.Context(), c.Query("language"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// CreateKnowledgeCategory 新建分类
// @Summary 新建知识库分类
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminKnowledgeCategoryReq true "请求"
// @Success 200 {object} resp.Body{data=model.KnowledgeCategory}
// @Router /admin/knowledge-categories [post]
func (h *Admin) CreateKnowledgeCategory(c *gin.Context) {
	var req model.AdminKnowledgeCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateKnowledgeCategory(c.Request.Context(), h.adminID(c), &req, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// UpdateKnowledgeCategory 更新分类
// @Summary 更新知识库分类（改名级联同步知识文档展示分类）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "分类 ID"
// @Param body body model.AdminKnowledgeCategoryUpdateReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/knowledge-categories/{id} [put]
func (h *Admin) UpdateKnowledgeCategory(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminKnowledgeCategoryUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateKnowledgeCategory(c.Request.Context(), h.adminID(c), id, &req, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// DeleteKnowledgeCategory 删除分类
// @Summary 删除知识库分类（分类下仍有文档时拒绝）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param id path int true "分类 ID"
// @Success 200 {object} resp.Body
// @Router /admin/knowledge-categories/{id} [delete]
func (h *Admin) DeleteKnowledgeCategory(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteKnowledgeCategory(c.Request.Context(), h.adminID(c), id, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 邮件模板（F11） ----

// ListMailTemplates 邮件模板列表
// @Summary 邮件模板列表（内置默认 + 自定义覆盖合并）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=object{list=[]model.AdminMailTemplateItem}}
// @Router /admin/mail-templates [get]
func (h *Admin) ListMailTemplates(c *gin.Context) {
	data, err := h.svc.ListMailTemplates(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// SaveMailTemplate 保存邮件模板
// @Summary 保存邮件模板（Go template 语法，保存前校验可解析）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param name path string true "模板名（captcha/expire_remind/traffic_remind）"
// @Param body body model.AdminMailTemplateReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/mail-templates/{name} [put]
func (h *Admin) SaveMailTemplate(c *gin.Context) {
	var req model.AdminMailTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.SaveMailTemplate(c.Request.Context(), h.adminID(c), c.Param("name"), &req, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ResetMailTemplate 恢复默认模板
// @Summary 恢复邮件模板默认文案（删除自定义行）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param name path string true "模板名"
// @Success 200 {object} resp.Body
// @Router /admin/mail-templates/{name} [delete]
func (h *Admin) ResetMailTemplate(c *gin.Context) {
	if err := h.svc.ResetMailTemplate(c.Request.Context(), h.adminID(c), c.Param("name"), c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// TestMailTemplate 测试发送模板邮件
// @Summary 测试发送邮件模板（渲染示例内容走真实 SMTP）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param name path string true "模板名"
// @Param body body model.AdminMailTemplateTestReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/mail-templates/{name}/test [post]
func (h *Admin) TestMailTemplate(c *gin.Context) {
	var req model.AdminMailTemplateTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.TestMailTemplate(c.Request.Context(), h.adminID(c), c.Param("name"), req.ToEmail, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 订阅模板（F10） ----

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
