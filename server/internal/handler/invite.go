package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"nanocloud/internal/middleware"
	"nanocloud/internal/model"
	"nanocloud/internal/pkg/resp"
	"nanocloud/internal/pkg/validate"
	"nanocloud/internal/service"
)

// Invite 营销端点。
type Invite struct {
	svc *service.InviteService
}

func NewInvite(svc *service.InviteService) *Invite { return &Invite{svc: svc} }

// Summary GET /invite/summary
func (h *Invite) Summary(c *gin.Context) {
	data, err := h.svc.Summary(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Codes GET /invite/codes
func (h *Invite) Codes(c *gin.Context) {
	data, err := h.svc.Codes(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// CreateCode POST /invite/codes
func (h *Invite) CreateCode(c *gin.Context) {
	data, err := h.svc.CreateCode(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Records GET /invite/records
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

// Transfer POST /invite/transfer
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

// AgentStatus GET /agent/status
func (h *Invite) AgentStatus(c *gin.Context) {
	data, err := h.svc.AgentStatus(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// ApplyAgent POST /agent/apply
func (h *Invite) ApplyAgent(c *gin.Context) {
	status, err := h.svc.ApplyAgent(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"apply_status": status})
}
