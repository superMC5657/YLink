package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"nanocloud/internal/middleware"
	"nanocloud/internal/model"
	"nanocloud/internal/pkg/payment"
	"nanocloud/internal/pkg/resp"
	"nanocloud/internal/pkg/validate"
	"nanocloud/internal/service"
)

// Order 交易端点。
type Order struct {
	svc *service.OrderService
}

func NewOrder(svc *service.OrderService) *Order { return &Order{svc: svc} }

// Plans GET /plans
func (h *Order) Plans(c *gin.Context) {
	data, err := h.svc.Plans(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// CouponCheck POST /coupons/check
func (h *Order) CouponCheck(c *gin.Context) {
	var req model.CouponCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CouponCheck(c.Request.Context(), middleware.UserID(c), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Create POST /orders（幂等：Idempotency-Key）
func (h *Order) Create(c *gin.Context) {
	var req model.CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateOrder(c.Request.Context(), middleware.UserID(c), c.GetHeader("Idempotency-Key"), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// List GET /orders
func (h *Order) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
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

// Detail GET /orders/{order_no}（兼支付轮询）
func (h *Order) Detail(c *gin.Context) {
	data, err := h.svc.GetOrder(c.Request.Context(), middleware.UserID(c), c.Param("order_no"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Cancel POST /orders/{order_no}/cancel
func (h *Order) Cancel(c *gin.Context) {
	data, err := h.svc.CancelOrder(c.Request.Context(), middleware.UserID(c), c.Param("order_no"))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Checkout POST /orders/{order_no}/checkout
func (h *Order) Checkout(c *gin.Context) {
	var req model.CheckoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.Checkout(c.Request.Context(), middleware.UserID(c), c.Param("order_no"), req.Method, c.Request)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Notify POST /payment/notify/{method}（服务端间，免鉴权）
func (h *Order) Notify(c *gin.Context) {
	method := c.Param("method")
	driver := payment.Get(method)
	if driver == nil {
		c.String(400, "fail")
		return
	}
	nr, err := driver.VerifyNotify(c.Request)
	if err != nil || nr == nil || !nr.Paid {
		c.String(400, "fail")
		return
	}
	if err := h.svc.HandleNotify(c.Request.Context(), method, nr); err != nil {
		c.String(400, "fail")
		return
	}
	// 易支付要求纯文本 success
	c.String(200, "success")
}
