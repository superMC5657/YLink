package admin

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
)

// ---- 工单 ----

func (h *Admin) ListTickets(c *gin.Context) {
	page, pageSize := pageParams(c)
	list, total, err := h.ticket.ListTickets(c.Request.Context(), page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

func (h *Admin) TicketDetail(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.ticket.GetTicketDetail(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) ReplyTicket(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminReplyReq
	if !resp.Bind(c, &req) {
		return
	}
	if err := h.ticket.ReplyTicket(c.Request.Context(), h.adminID(c), id, req.Message); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) CloseTicket(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.ticket.CloseTicket(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
