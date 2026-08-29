package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
)

// ---- 代理 ----

func (h *Admin) ListAgentApplies(c *gin.Context) {
	page, pageSize := pageParams(c)
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))
	list, total, err := h.svc.ListAgentApplies(c.Request.Context(), status, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

func (h *Admin) ApproveAgent(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminApproveReq
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.ReviewAgentApply(c.Request.Context(), h.adminID(c), id, true, req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) RejectAgent(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminApproveReq
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.ReviewAgentApply(c.Request.Context(), h.adminID(c), id, false, req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 佣金 ----

func (h *Admin) ListCommissions(c *gin.Context) {
	page, pageSize := pageParams(c)
	var status *int
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = &v
		}
	}
	list, total, err := h.svc.ListCommissions(c.Request.Context(), status, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}
