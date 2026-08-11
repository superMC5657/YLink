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

// CouponItem 用户可见的可用优惠券（GET /coupons/available）。
// 仅暴露展示所需字段；total_limit/used_count/limit_per_user 属运营内部信息不返回。
type CouponItem struct {
	Code         string     `json:"code"`
	Type         int        `json:"type"`          // 1=固定金额 2=百分比
	Value        float64    `json:"value"`         // type=1 为元；type=2 为百分比数值（如 10 表示 10%）
	MinSpend     float64    `json:"min_spend"`     // 元
	ValidPeriods []string   `json:"valid_periods"` // 空=不限周期
	PlanIDs      []int64    `json:"plan_ids"`      // 空=全部套餐
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
}

// CouponAvailableResp 可用优惠券列表信封。
type CouponAvailableResp struct {
	List []CouponItem `json:"list"`
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
