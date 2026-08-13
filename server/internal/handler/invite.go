package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
	"ylink-backend/internal/service"
)

// Invite 营销端点。
type Invite struct {
	svc *service.InviteService
}

func NewInvite(svc *service.InviteService) *Invite { return &Invite{svc: svc} }

// Summary 邀请总览
// @Summary 邀请总览
// @Tags 营销
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.InviteSummaryResp}
// @Router /invite/summary [get]
func (h *Invite) Summary(c *gin.Context) {
	data, err := h.svc.Summary(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Codes 邀请码列表
// @Summary 邀请码列表
// @Tags 营销
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.InviteCodesResp}
// @Router /invite/codes [get]
func (h *Invite) Codes(c *gin.Context) {
	data, err := h.svc.Codes(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// CreateCode 新增邀请码
// @Summary 新增邀请码
// @Tags 营销
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.InviteCodeItem}
// @Failure 400 {object} resp.Body
// @Router /invite/codes [post]
func (h *Invite) CreateCode(c *gin.Context) {
	data, err := h.svc.CreateCode(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// DeleteCode 删除邀请码
// @Summary 删除邀请码
// @Tags 营销
// @Security BearerAuth
// @Produce json
// @Param code path string true "邀请码"
// @Success 200 {object} resp.Body
// @Failure 400 {object} resp.Body
// @Router /invite/codes/{code} [delete]
func (h *Invite) DeleteCode(c *gin.Context) {
	if err := h.svc.DeleteCode(c.Request.Context(), middleware.UserID(c), c.Param("code")); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// Records 佣金发放记录
// @Summary 佣金发放记录
// @Tags 营销
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} resp.Body{data=resp.Page}
// @Router /invite/records [get]
func (h *Invite) Records(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	list, total, err := h.svc.Records(c.Request.Context(), middleware.UserID(c), page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

// Transfer 佣金划转余额
// @Summary 佣金划转余额
// @Tags 营销
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.TransferReq true "请求"
// @Success 200 {object} resp.Body{data=model.TransferResp}
// @Router /invite/transfer [post]
func (h *Invite) Transfer(c *gin.Context) {
	var req model.TransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.Transfer(c.Request.Context(), middleware.UserID(c), model.YuanToFen(req.Amount))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// AgentStatus 代理状态
// @Summary 代理状态
// @Tags 营销
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.AgentStatusResp}
// @Router /agent/status [get]
func (h *Invite) AgentStatus(c *gin.Context) {
	data, err := h.svc.AgentStatus(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// ApplyAgent 提交代理申请
// @Summary 提交代理申请
// @Tags 营销
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=object{apply_status=string}}
// @Router /agent/apply [post]
func (h *Invite) ApplyAgent(c *gin.Context) {
	status, err := h.svc.ApplyAgent(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"apply_status": status})
}
