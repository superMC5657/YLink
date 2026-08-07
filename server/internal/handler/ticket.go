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

// Ticket 工单端点。
type Ticket struct {
	svc *service.TicketService
}

func NewTicket(svc *service.TicketService) *Ticket { return &Ticket{svc: svc} }

// List GET /tickets
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

// Create POST /tickets
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

// Detail GET /tickets/{id}
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

// Reply POST /tickets/{id}/reply
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

// Close POST /tickets/{id}/close
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
