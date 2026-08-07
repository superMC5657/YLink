package model

import "time"

// ---- 套餐 GET /plans ----

type PlanPrices struct {
	Month    *float64 `json:"month,omitempty"`
	Quarter  *float64 `json:"quarter,omitempty"`
	HalfYear *float64 `json:"half_year,omitempty"`
	Year     *float64 `json:"year,omitempty"`
	Onetime  *float64 `json:"onetime,omitempty"`
}

type PlanResp struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Prices      PlanPrices `json:"prices"`
	TrafficGB   int        `json:"traffic_gb"`
	SpeedLimit  *int       `json:"speed_limit"`
	DeviceLimit *int       `json:"device_limit"`
	Content     string     `json:"content"`
	Sort        int        `json:"sort"`
}

// ---- 优惠券 ----

type CouponCheckReq struct {
	Code   string `json:"code" binding:"required"`
	PlanID int64  `json:"plan_id" binding:"required"`
	Period string `json:"period" binding:"required,oneof=month quarter half_year year onetime"`
}

type CouponCheckResp struct {
	Valid          bool    `json:"valid"`
	DiscountAmount float64 `json:"discount_amount"`
	PayAmount      float64 `json:"pay_amount"`
}

// ---- 订单 ----

type CreateOrderReq struct {
	PlanID     int64  `json:"plan_id" binding:"required"`
	Period     string `json:"period" binding:"required,oneof=month quarter half_year year onetime"`
	CouponCode string `json:"coupon_code"`
}

type OrderResp struct {
	OrderNo        string    `json:"order_no"`
	PlanName       string    `json:"plan_name"`
	Period         string    `json:"period"`
	Amount         float64   `json:"amount"`
	DiscountAmount float64   `json:"discount_amount"`
	PayAmount      float64   `json:"pay_amount"`
	Status         int       `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type OrderDetailResp struct {
	OrderNo        string     `json:"order_no"`
	PlanName       string     `json:"plan_name"`
	Period         string     `json:"period"`
	Amount         float64    `json:"amount"`
	DiscountAmount float64    `json:"discount_amount"`
	BalanceUsed    float64    `json:"balance_used"`
	PayAmount      float64    `json:"pay_amount"`
	CouponCode     *string    `json:"coupon_code"`
	Status         int        `json:"status"`
	PayMethod      *string    `json:"pay_method"`
	PaidAt         *time.Time `json:"paid_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type CheckoutReq struct {
	Method string `json:"method" binding:"required"`
}

type CheckoutResp struct {
	Type     string  `json:"type"`
	Content  *string `json:"content"`
	ExpireIn int     `json:"expire_in"`
}
