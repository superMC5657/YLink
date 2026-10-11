package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/payment"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/service"
)

// Order 交易端点。
type Order struct {
	svc *service.OrderService
}

func NewOrder(svc *service.OrderService) *Order { return &Order{svc: svc} }

// Plans 套餐列表
// @Summary 套餐列表
// @Tags 交易
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=object{list=[]model.PlanResp}}
// @Router /plans [get]
func (h *Order) Plans(c *gin.Context) {
	data, err := h.svc.Plans(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// CouponCheck 优惠券试算
// @Summary 优惠券试算
// @Tags 交易
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.CouponCheckReq true "请求"
// @Success 200 {object} resp.Body{data=model.CouponCheckResp}
// @Failure 400 {object} resp.Body
// @Router /coupons/check [post]
func (h *Order) CouponCheck(c *gin.Context) {
	var req model.CouponCheckReq
	if !resp.Bind(c, &req) {
		return
	}
	data, err := h.svc.CouponCheck(c.Request.Context(), middleware.UserID(c), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// CouponsAvailable 可用优惠券列表（用户端可见）
// @Summary 可用优惠券列表
// @Tags 交易
// @Security BearerAuth
// @Produce json
// @Param plan_id query int false "套餐 ID（可选过滤）"
// @Param period query string false "周期（可选过滤）"
// @Success 200 {object} resp.Body{data=model.CouponAvailableResp}
// @Router /coupons/available [get]
func (h *Order) CouponsAvailable(c *gin.Context) {
	planID, _ := strconv.ParseInt(c.Query("plan_id"), 10, 64)
	list, err := h.svc.AvailableCoupons(c.Request.Context(), middleware.UserID(c), planID, c.Query("period"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// Create 创建订单（支持 Idempotency-Key）
// @Summary 创建订单
// @Tags 交易
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param Idempotency-Key header string false "幂等键"
// @Param body body model.CreateOrderReq true "请求"
// @Success 200 {object} resp.Body{data=model.OrderResp}
// @Router /orders [post]
func (h *Order) Create(c *gin.Context) {
	var req model.CreateOrderReq
	if !resp.Bind(c, &req) {
		return
	}
	data, err := h.svc.CreateOrder(c.Request.Context(), middleware.UserID(c), c.GetHeader("Idempotency-Key"), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// List 订单列表
// @Summary 订单列表
// @Tags 交易
// @Security BearerAuth
// @Produce json
// @Param status query int false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} resp.Body{data=resp.Page}
// @Router /orders [get]
func (h *Order) List(c *gin.Context) {
	page, pageSize := pageParams(c)
	var status *int
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 && v <= 3 {
			status = &v
		}
	}
	list, total, err := h.svc.ListOrders(c.Request.Context(), middleware.UserID(c), status, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

// Detail 订单详情（兼支付轮询）
// @Summary 订单详情
// @Tags 交易
// @Security BearerAuth
// @Produce json
// @Param order_no path string true "订单号"
// @Success 200 {object} resp.Body{data=model.OrderDetailResp}
// @Router /orders/{order_no} [get]
func (h *Order) Detail(c *gin.Context) {
	data, err := h.svc.GetOrder(c.Request.Context(), middleware.UserID(c), c.Param("order_no"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Cancel 取消订单
// @Summary 取消订单
// @Tags 交易
// @Security BearerAuth
// @Produce json
// @Param order_no path string true "订单号"
// @Success 200 {object} resp.Body{data=model.OrderResp}
// @Failure 409 {object} resp.Body
// @Router /orders/{order_no}/cancel [post]
func (h *Order) Cancel(c *gin.Context) {
	data, err := h.svc.CancelOrder(c.Request.Context(), middleware.UserID(c), c.Param("order_no"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Checkout 收银台拉起支付
// @Summary 收银台
// @Tags 交易
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param order_no path string true "订单号"
// @Param body body model.CheckoutReq true "请求"
// @Success 200 {object} resp.Body{data=model.CheckoutResp}
// @Router /orders/{order_no}/checkout [post]
func (h *Order) Checkout(c *gin.Context) {
	var req model.CheckoutReq
	if !resp.Bind(c, &req) {
		return
	}
	data, err := h.svc.Checkout(c.Request.Context(), middleware.UserID(c), c.Param("order_no"), req.Method)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Notify POST /payment/notify/{method}（服务端间，免鉴权）。
// 应答语义：验签失败/账务失败 → fail(400) 由网关重试；验签通过的非成功终态
// （如 trade_status=TRADE_CLOSED）→ success 确认并告警，业务终态无需网关重试。
func (h *Order) Notify(c *gin.Context) {
	method := c.Param("method")
	driver := payment.Get(method)
	if driver == nil {
		c.String(400, "fail")
		return
	}
	nr, err := driver.VerifyNotify(c.Request)
	if err != nil || nr == nil {
		// 验签失败:可能是伪造或传输损坏,fail 让网关重试
		c.String(400, "fail")
		return
	}
	if !nr.Paid {
		// 验签通过但非成功终态:ack 防重试风暴,原文落库后告警人工跟进
		logger.L().Warn("payment notify non-success terminal state",
			zap.String("method", method), zap.String("order_no", nr.OrderNo),
			zap.String("trade_no", nr.TradeNo))
		if err := h.svc.HandleNotify(c.Request.Context(), method, nr); err != nil {
			logger.L().Error("payment notify persist failed",
				zap.String("method", method), zap.Error(err))
			c.String(400, "fail")
			return
		}
		c.String(200, "success")
		return
	}
	if err := h.svc.HandleNotify(c.Request.Context(), method, nr); err != nil {
		c.String(400, "fail")
		return
	}
	// 易支付要求纯文本 success
	c.String(200, "success")
}
