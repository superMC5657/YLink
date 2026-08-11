package model

import "time"

// ---- 管理端 · 仪表盘 ----

type AdminOverviewResp struct {
	UserCount       int64   `json:"user_count"`
	AgentCount      int64   `json:"agent_count"`
	OrderCount      int64   `json:"order_count"`
	CompletedOrders int64   `json:"completed_orders"`
	TotalRevenue    float64 `json:"total_revenue"` // 已完成订单实收合计（元）
	TodayRevenue    float64 `json:"today_revenue"` // 今日实收（元）
	PlanCount       int64   `json:"plan_count"`
}

// AdminPlanView 管理端套餐视图:价格统一为元,展开 group_ids/is_show(用户端 Plan 隐藏)。
type AdminPlanView struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Content       string   `json:"content"`
	MonthPrice    *float64 `json:"month_price"`
	QuarterPrice  *float64 `json:"quarter_price"`
	HalfYearPrice *float64 `json:"half_year_price"`
	YearPrice     *float64 `json:"year_price"`
	OnetimePrice  *float64 `json:"onetime_price"`
	TrafficGB     int      `json:"traffic_gb"`
	SpeedLimit    *int     `json:"speed_limit"`
	DeviceLimit   *int     `json:"device_limit"`
	GroupIDs      []int64  `json:"group_ids"`
	IsShow        bool     `json:"is_show"`
	Sort          int      `json:"sort"`
}

// AdminServerView 管理端节点视图:展开 group_id/host/port/config(用户端隐藏)。
type AdminServerView struct {
	ID      int64    `json:"id"`
	GroupID int64    `json:"group_id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Host    string   `json:"host"`
	Port    int      `json:"port"`
	Config  string   `json:"config"`
	Rate    float64  `json:"rate"`
	Tags    []string `json:"tags"`
	Status  int      `json:"status"`
	IsShow  bool     `json:"is_show"`
	Sort    int      `json:"sort"`
}

// ---- 管理端 · 用户 ----

type AdminUserItem struct {
	ID                int64      `json:"id"`
	Email             string     `json:"email"`
	Role              int        `json:"role"`
	Balance           float64    `json:"balance"`
	CommissionBalance float64    `json:"commission_balance"`
	IsBanned          bool       `json:"is_banned"`
	InviteByID        *int64     `json:"invite_by_id"`
	PlanID            *int64     `json:"plan_id"`
	ExpiredAt         *time.Time `json:"expired_at"`
	TransferEnable    int64      `json:"transfer_enable"`
	U                 int64      `json:"u"`
	D                 int64      `json:"d"`
	CreatedAt         time.Time  `json:"created_at"`
}

type AdminUpdateUserReq struct {
	Role   *int  `json:"role" binding:"omitempty,oneof=0 1 2"`
	Banned *bool `json:"banned"`
}

type AdjustBalanceReq struct {
	Amount float64 `json:"amount" binding:"required"` // 元，可正可负
	Remark string  `json:"remark"`
}

// ---- 管理端 · 套餐 ----

type AdminPlanReq struct {
	Name          string   `json:"name" binding:"required"`
	Content       string   `json:"content"`
	MonthPrice    *float64 `json:"month_price"`
	QuarterPrice  *float64 `json:"quarter_price"`
	HalfYearPrice *float64 `json:"half_year_price"`
	YearPrice     *float64 `json:"year_price"`
	OnetimePrice  *float64 `json:"onetime_price"`
	TrafficGB     int      `json:"traffic_gb" binding:"required"`
	SpeedLimit    *int     `json:"speed_limit"`
	DeviceLimit   *int     `json:"device_limit"`
	GroupIDs      *[]int64 `json:"group_ids"` // nil=不更新;空数组=清空(该套餐不关联任何分组)
	IsShow        *bool    `json:"is_show"`
	Sort          int      `json:"sort"`
}

// ---- 管理端 · 节点 ----

type AdminServerGroupReq struct {
	Name string `json:"name" binding:"required"`
	Sort int    `json:"sort"`
}

type AdminServerReq struct {
	GroupID int64     `json:"group_id" binding:"required"`
	Name    string    `json:"name" binding:"required"`
	Type    string    `json:"type" binding:"required,oneof=shadowsocks vmess vless trojan hysteria2 tuic"`
	Host    string    `json:"host" binding:"required"`
	Port    int       `json:"port" binding:"required"`
	Config  string    `json:"config" binding:"required"` // 协议私有参数 JSON
	Rate    float64   `json:"rate"`
	Tags    *[]string `json:"tags"` // nil=不更新;空数组=清空
	Status  int       `json:"status" binding:"omitempty,oneof=1 2 3"`
	IsShow  *bool     `json:"is_show"`
	Sort    int       `json:"sort"`
}

// ---- 管理端 · 订单 ----

type AdminOrderItem struct {
	OrderNo        string     `json:"order_no"`
	UserID         int64      `json:"user_id"`
	UserEmail      string     `json:"user_email"`
	PlanName       string     `json:"plan_name"`
	Period         string     `json:"period"`
	Amount         float64    `json:"amount"`
	DiscountAmount float64    `json:"discount_amount"`
	BalanceUsed    float64    `json:"balance_used"`
	PayAmount      float64    `json:"pay_amount"`
	Status         int        `json:"status"`
	PayMethod      *string    `json:"pay_method"`
	PaidAt         *time.Time `json:"paid_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type RefundReq struct {
	Remark string `json:"remark"`
}

// ---- 管理端 · 优惠券 ----

type AdminCouponReq struct {
	Code         string     `json:"code" binding:"required"`
	Type         int        `json:"type" binding:"required,oneof=1 2"`
	Value        float64    `json:"value" binding:"required"`
	MinSpend     float64    `json:"min_spend"`
	LimitPerUser int        `json:"limit_per_user"`
	TotalLimit   int        `json:"total_limit"`
	ValidPeriods []string   `json:"valid_periods"`
	PlanIDs      []int64    `json:"plan_ids"`
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	IsEnable     *bool      `json:"is_enable"`
}

// AdminCouponView 管理端优惠券视图:展开 type/value/min_spend(元)/used_count/valid_periods/plan_ids。
type AdminCouponView struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	Type         int        `json:"type"`
	Value        float64    `json:"value"`
	MinSpend     float64    `json:"min_spend"`
	LimitPerUser int        `json:"limit_per_user"`
	TotalLimit   int        `json:"total_limit"`
	UsedCount    int        `json:"used_count"`
	ValidPeriods []string   `json:"valid_periods"`
	PlanIDs      []int64    `json:"plan_ids"`
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	IsEnable     bool       `json:"is_enable"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ---- 管理端 · 内容 ----

type AdminNoticeItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsShow    bool      `json:"is_show"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminKnowledgeItem struct {
	ID        int64     `json:"id"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Language  string    `json:"language"`
	IsShow    bool      `json:"is_show"`
	Sort      int       `json:"sort"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminNoticeReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	IsShow  *bool  `json:"is_show"`
	Sort    int    `json:"sort"`
}

type AdminKnowledgeReq struct {
	Category string `json:"category" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body" binding:"required"`
	Language string `json:"language" binding:"omitempty,oneof=zh-CN en-US"`
	IsShow   *bool  `json:"is_show"`
	Sort     int    `json:"sort"`
}

// ---- 管理端 · 工单 ----

type AdminTicketItem struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	UserEmail   string     `json:"user_email"`
	Subject     string     `json:"subject"`
	Level       int        `json:"level"`
	Status      int        `json:"status"`
	LastReplyAt *time.Time `json:"last_reply_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AdminReplyReq struct {
	Message string `json:"message" binding:"required"`
}

// ---- 管理端 · 代理申请 ----

type AdminAgentApplyItem struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	UserEmail    string    `json:"user_email"`
	ValidInvites int64     `json:"valid_invites"`
	Status       int       `json:"status"` // 0=待审核 1=通过 2=拒绝
	CreatedAt    time.Time `json:"created_at"`
}

type AdminApproveReq struct {
	Remark string `json:"remark"`
}

// ---- 管理端 · 佣金日志 ----

type AdminCommissionItem struct {
	ID           int64      `json:"id"`
	InviteUserID int64      `json:"invite_user_id"`
	InviteEmail  string     `json:"invite_email"`
	FromUserID   int64      `json:"from_user_id"`
	FromEmail    string     `json:"from_email"`
	OrderNo      string     `json:"order_no"`
	OrderAmount  float64    `json:"order_amount"`
	Rate         int        `json:"rate"`
	Amount       float64    `json:"amount"`
	Status       int        `json:"status"`
	ConfirmedAt  *time.Time `json:"confirmed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ---- 管理端 · 流量导入 ----

type TrafficImportItem struct {
	UserID int64  `json:"user_id" binding:"required"`
	Date   string `json:"date" binding:"required"` // YYYY-MM-DD
	U      int64  `json:"u"`
	D      int64  `json:"d"`
}

type TrafficImportReq struct {
	Items []TrafficImportItem `json:"items" binding:"required,min=1"`
}

// ---- 管理端 · 设置 ----

type AdminSettingsReq struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"` // JSON 字符串
}

type AdminSettingsResp struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
