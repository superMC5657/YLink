package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
)

// ---- 订单 ----

func (h *Admin) ListOrders(c *gin.Context) {
	page, pageSize := pageParams(c)
	var status *int
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 && v <= 3 {
			status = &v
		}
	}
	list, total, err := h.svc.ListOrders(c.Request.Context(), status, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

// CloseOrder 关闭订单（待支付）
// @Summary 关闭订单
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param order_no path string true "订单号"
// @Param body body model.RefundReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/orders/{order_no}/close [post]
func (h *Admin) CloseOrder(c *gin.Context) {
	var req model.RefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.CloseOrder(c.Request.Context(), h.adminID(c), c.Param("order_no"), req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Refund 订单退款（佣金回滚 + 审计）
// @Summary 订单退款
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param order_no path string true "订单号"
// @Param body body model.RefundReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/orders/{order_no}/refund [post]
func (h *Admin) Refund(c *gin.Context) {
	var req model.RefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.Refund(c.Request.Context(), h.adminID(c), c.Param("order_no"), req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

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
