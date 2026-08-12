package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink/internal/middleware"
	"ylink/internal/model"
	"ylink/internal/pkg/resp"
	"ylink/internal/pkg/validate"
	"ylink/internal/service"
)

// Ticket 工单端点。
type Ticket struct {
	svc *service.TicketService
}

func NewTicket(svc *service.TicketService) *Ticket { return &Ticket{svc: svc} }

// List 工单列表
// @Summary 工单列表
// @Tags 工单
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} resp.Body{data=resp.Page}
// @Router /tickets [get]
func (h *Ticket) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	list, total, err := h.svc.List(c.Request.Context(), middleware.UserID(c), page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

// Create 创建工单
// @Summary 创建工单
// @Tags 工单
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.CreateTicketReq true "请求"
// @Success 200 {object} resp.Body{data=model.TicketListItem}
// @Router /tickets [post]
func (h *Ticket) Create(c *gin.Context) {
	var req model.CreateTicketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.Create(c.Request.Context(), middleware.UserID(c), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Detail 工单详情
// @Summary 工单详情
// @Tags 工单
// @Security BearerAuth
// @Produce json
// @Param id path int true "工单 ID"
// @Success 200 {object} resp.Body{data=model.TicketDetailResp}
// @Router /tickets/{id} [get]
func (h *Ticket) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Reply 回复工单
// @Summary 回复工单
// @Tags 工单
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "工单 ID"
// @Param body body model.ReplyTicketReq true "请求"
// @Success 200 {object} resp.Body
// @Failure 409 {object} resp.Body
// @Router /tickets/{id}/reply [post]
func (h *Ticket) Reply(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return
	}
	var req model.ReplyTicketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.Reply(c.Request.Context(), middleware.UserID(c), id, req.Message); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Close 关闭工单
// @Summary 关闭工单
// @Tags 工单
// @Security BearerAuth
// @Produce json
// @Param id path int true "工单 ID"
// @Success 200 {object} resp.Body{data=model.TicketListItem}
// @Router /tickets/{id}/close [post]
func (h *Ticket) Close(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return
	}
	data, err := h.svc.Close(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Reopen 重新打开工单(关闭后最多一次)
// @Summary 重新打开工单
// @Tags 工单
// @Security BearerAuth
// @Produce json
// @Param id path int true "工单 ID"
// @Success 200 {object} resp.Body{data=model.TicketListItem}
// @Failure 409 {object} resp.Body
// @Router /tickets/{id}/reopen [post]
func (h *Ticket) Reopen(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return
	}
	data, err := h.svc.Reopen(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}
