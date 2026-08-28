package handler

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
)

// ---- 管理端 · 第三批（xboard-gap-fill）端点：F02 提现审核 / F15 内容排序与分类 / F11 邮件模板 / F20 版本 ----

// ---- 佣金提现审核（F02） ----

// WithdrawPay 确认提现打款
// @Summary 确认提现打款（线下打款由管理员线下执行，系统内记账并关闭工单）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "提现工单 ID"
// @Param body body model.AdminApproveReq false "备注"
// @Success 200 {object} resp.Body
// @Failure 409 {object} resp.Body
// @Router /admin/tickets/{id}/withdraw/pay [post]
func (h *Admin) WithdrawPay(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminApproveReq
	_ = c.ShouldBindJSON(&req) // body 可选
	if err := h.svc.ReviewWithdraw(c.Request.Context(), h.adminID(c), id, true, req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// WithdrawReject 拒绝提现
// @Summary 拒绝提现（自动退回佣金并关闭工单，写流水与审计）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "提现工单 ID"
// @Param body body model.AdminApproveReq false "备注"
// @Success 200 {object} resp.Body
// @Failure 409 {object} resp.Body
// @Router /admin/tickets/{id}/withdraw/reject [post]
func (h *Admin) WithdrawReject(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminApproveReq
	_ = c.ShouldBindJSON(&req) // body 可选
	if err := h.svc.ReviewWithdraw(c.Request.Context(), h.adminID(c), id, false, req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 内容排序（F15） ----

// SortNotices 公告排序
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

// SortKnowledges 知识库排序
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
