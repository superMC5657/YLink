// Package model 定义表结构体、枚举与 DTO。
package model

import "time"

// 角色
const (
	RoleUser  = 0
	RoleAdmin = 1
	RoleAgent = 2
)

// 订单状态
const (
	OrderPending   = 0 // 待支付
	OrderCompleted = 1 // 已完成
	OrderCanceled  = 2 // 已取消
	OrderRefunded  = 3 // 已退款
)

// 支付单状态
const (
	PayPending = 0
	PaySuccess = 1
	PayClosed  = 2
)

// 佣金状态
const (
	CommissionPending = 0 // 确认中
	CommissionGranted = 1 // 已发放
	CommissionRevoked = 2 // 已撤销
)

// 订阅周期
var Periods = []string{"month", "quarter", "half_year", "year", "onetime"}

// PeriodDuration 返回周期对应时长。
func PeriodDuration(period string) time.Duration {
	switch period {
	case "month":
		return 30 * 24 * time.Hour
	case "quarter":
		return 90 * 24 * time.Hour
	case "half_year":
		return 180 * 24 * time.Hour
	case "year":
		return 365 * 24 * time.Hour
	case "onetime":
		return 0
	}
	return 0
}

// User 对应 users 表。
type User struct {
	ID                int64      `gorm:"primaryKey" json:"id"`
	Email             string     `gorm:"size:190;uniqueIndex" json:"email"`
	PasswordHash      string     `gorm:"size:255" json:"-"`
	Role              int        `gorm:"default:0" json:"role"`
	Balance           int64      `gorm:"default:0" json:"balance"`
	CommissionBalance int64      `gorm:"default:0" json:"commission_balance"`
	InviteByID        *int64     `gorm:"index" json:"invite_by_id,omitempty"`
	IsBanned          bool       `gorm:"default:false" json:"is_banned"`
	RemindExpire      bool       `gorm:"default:true" json:"remind_expire"`
	RemindTraffic     bool       `gorm:"default:false" json:"remind_traffic"`
	TelegramID        *int64     `json:"telegram_id,omitempty"`
	PlanID            *int64     `json:"plan_id,omitempty"`
	ExpiredAt         *time.Time `json:"expired_at,omitempty"`
	TransferEnable    int64      `gorm:"default:0" json:"transfer_enable"`
	U                 int64      `gorm:"default:0" json:"u"`
	D                 int64      `gorm:"default:0" json:"d"`
	SpeedLimit        *int       `json:"speed_limit,omitempty"`
	DeviceLimit       *int       `json:"device_limit,omitempty"`
	SubToken          string     `gorm:"size:36;uniqueIndex" json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TableName 指定表名。
func (User) TableName() string { return "users" }

// Plan 对应 plans 表。
type Plan struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"size:64" json:"name"`
	Content       string    `gorm:"type:text" json:"content"`
	MonthPrice    *int64    `json:"month_price"`
	QuarterPrice  *int64    `json:"quarter_price"`
	HalfYearPrice *int64    `json:"half_year_price"`
	YearPrice     *int64    `json:"year_price"`
	OnetimePrice  *int64    `json:"onetime_price"`
	TrafficGB     int       `json:"traffic_gb"`
	SpeedLimit    *int      `json:"speed_limit"`
	DeviceLimit   *int      `json:"device_limit"`
	GroupIDs      string    `gorm:"type:json" json:"-"`
	IsShow        bool      `gorm:"default:true" json:"-"`
	Sort          int       `gorm:"default:0" json:"sort"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
}

func (Plan) TableName() string { return "plans" }

// Order 对应 orders 表。
type Order struct {
	ID             int64      `gorm:"primaryKey" json:"-"`
	OrderNo        string     `gorm:"size:32;uniqueIndex" json:"order_no"`
	UserID         int64      `gorm:"index" json:"-"`
	PlanID         int64      `json:"-"`
	Period         string     `gorm:"size:16" json:"period"`
	Amount         int64      `json:"-"`
	DiscountAmount int64      `gorm:"default:0" json:"-"`
	BalanceUsed    int64      `gorm:"default:0" json:"-"`
	PayAmount      int64      `json:"-"`
	CouponID       *int64     `json:"-"`
	Status         int        `gorm:"default:0" json:"status"`
	PayMethod      *string    `gorm:"size:32" json:"-"`
	PaidAt         *time.Time `json:"-"`
	IdempotencyKey *string    `gorm:"size:64;uniqueIndex" json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"-"`
}

func (Order) TableName() string { return "orders" }

// Payment 对应 payments 表。
type Payment struct {
	ID            int64      `gorm:"primaryKey" json:"-"`
	OrderNo       string     `gorm:"size:32;index" json:"order_no"`
	UserID        int64      `json:"-"`
	Method        string     `gorm:"size:32" json:"-"`
	Amount        int64      `json:"-"`
	TradeNo       *string    `gorm:"size:64;uniqueIndex" json:"-"`
	Status        int        `gorm:"default:0" json:"-"`
	NotifyPayload string     `gorm:"type:text" json:"-"`
	PaidAt        *time.Time `json:"-"`
	CreatedAt     time.Time  `json:"-"`
	UpdatedAt     time.Time  `json:"-"`
}

func (Payment) TableName() string { return "payments" }

// Coupon 对应 coupons 表。
type Coupon struct {
	ID           int64      `gorm:"primaryKey" json:"-"`
	Code         string     `gorm:"size:64;uniqueIndex" json:"code"`
	Type         int        `json:"-"` // 1=固定金额 2=百分比
	Value        int64      `json:"-"`
	MinSpend     int64      `gorm:"default:0" json:"-"`
	LimitPerUser int        `gorm:"default:0" json:"-"`
	TotalLimit   int        `gorm:"default:0" json:"-"`
	UsedCount    int        `gorm:"default:0" json:"-"`
	ValidPeriods *string    `gorm:"type:json" json:"-"`
	PlanIDs      *string    `gorm:"type:json" json:"-"`
	StartedAt    *time.Time `json:"-"`
	EndedAt      *time.Time `json:"-"`
	IsEnable     bool       `gorm:"default:true" json:"-"`
	CreatedAt    time.Time  `json:"-"`
	UpdatedAt    time.Time  `json:"-"`
}

func (Coupon) TableName() string { return "coupons" }

// CouponUsage 对应 coupon_usages 表。
type CouponUsage struct {
	ID       int64  `gorm:"primaryKey" json:"-"`
	CouponID int64  `json:"-"`
	UserID   int64  `json:"-"`
	OrderNo  string `gorm:"size:32" json:"-"`
}

func (CouponUsage) TableName() string { return "coupon_usages" }

// InviteCode 对应 invite_codes 表。
type InviteCode struct {
	ID        int64     `gorm:"primaryKey" json:"-"`
	UserID    int64     `gorm:"index" json:"-"`
	Code      string    `gorm:"size:32;uniqueIndex" json:"code"`
	Status    int       `gorm:"default:1" json:"-"`
	UsedCount int       `gorm:"default:0" json:"used_count"`
	CreatedAt time.Time `json:"created_at"`
}

func (InviteCode) TableName() string { return "invite_codes" }

// CommissionLog 对应 commission_logs 表。
type CommissionLog struct {
	ID           int64      `gorm:"primaryKey" json:"-"`
	InviteUserID int64      `gorm:"index" json:"-"`
	FromUserID   int64      `json:"-"`
	OrderNo      string     `gorm:"size:32;uniqueIndex" json:"order_no"`
	OrderAmount  int64      `json:"-"`
	Rate         int        `json:"rate"`
	Amount       int64      `json:"-"`
	Status       int        `gorm:"default:0" json:"status"`
	ConfirmedAt  *time.Time `json:"confirmed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (CommissionLog) TableName() string { return "commission_logs" }

// ServerGroup 对应 server_groups 表。
type ServerGroup struct {
	ID   int64  `gorm:"primaryKey" json:"id"`
	Name string `gorm:"size:64" json:"name"`
	Sort int    `gorm:"default:0" json:"sort"`
}

func (ServerGroup) TableName() string { return "server_groups" }

// Server 对应 servers 表。
type Server struct {
	ID      int64   `gorm:"primaryKey" json:"id"`
	GroupID int64   `gorm:"index" json:"-"`
	Name    string  `gorm:"size:64" json:"name"`
	Type    string  `gorm:"size:32" json:"type"`
	Host    string  `gorm:"size:255" json:"-"`
	Port    int     `json:"-"`
	Config  string  `gorm:"type:json" json:"-"`
	Rate    float64 `gorm:"type:decimal(3,1);default:1.0" json:"rate"`
	Tags    *string `gorm:"type:json" json:"tags"`
	Status  int     `gorm:"default:1" json:"status"`
	IsShow  bool    `gorm:"default:true" json:"-"`
	Sort    int     `gorm:"default:0" json:"-"`
}

func (Server) TableName() string { return "servers" }

// Notice 对应 notices 表。
type Notice struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:128" json:"title"`
	Content   string    `gorm:"type:mediumtext" json:"content"`
	IsShow    bool      `gorm:"default:true" json:"-"`
	Sort      int       `gorm:"default:0" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

func (Notice) TableName() string { return "notices" }

// Knowledge 对应 knowledges 表。
type Knowledge struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Category  string    `gorm:"size:64" json:"category"`
	Title     string    `gorm:"size:128" json:"title"`
	Body      string    `gorm:"type:mediumtext" json:"body,omitempty"`
	Language  string    `gorm:"size:10;default:zh-CN" json:"language,omitempty"`
	IsShow    bool      `gorm:"default:true" json:"-"`
	Sort      int       `gorm:"default:0" json:"-"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Knowledge) TableName() string { return "knowledges" }

// Ticket 对应 tickets 表。
type Ticket struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	UserID      int64      `gorm:"index" json:"-"`
	Subject     string     `gorm:"size:128" json:"subject"`
	Level       int        `json:"level"`
	Status      int        `gorm:"default:0" json:"status"`
	ReopenCount int        `gorm:"default:0" json:"reopen_count"` // 已重开次数(最多一次)
	LastReplyAt *time.Time `json:"last_reply_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (Ticket) TableName() string { return "tickets" }

// TicketMessage 对应 ticket_messages 表。
type TicketMessage struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	TicketID   int64     `gorm:"index" json:"-"`
	SenderType int       `json:"sender_type"`
	SenderID   int64     `json:"-"`
	Message    string    `gorm:"type:text" json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

func (TicketMessage) TableName() string { return "ticket_messages" }

// TrafficLog 对应 traffic_logs 表。
type TrafficLog struct {
	ID     int64  `gorm:"primaryKey" json:"-"`
	UserID int64  `gorm:"index" json:"-"`
	Date   string `gorm:"type:date" json:"date"`
	U      int64  `json:"u"`
	D      int64  `json:"d"`
}

func (TrafficLog) TableName() string { return "traffic_logs" }

// Setting 对应 settings 表。
type Setting struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"type:json" json:"value"`
}

func (Setting) TableName() string { return "settings" }

// AuditLog 对应 audit_logs 表。
type AuditLog struct {
	ID        int64     `gorm:"primaryKey" json:"-"`
	AdminID   int64     `json:"-"`
	Action    string    `gorm:"size:64" json:"-"`
	Target    *string   `gorm:"size:128" json:"-"`
	Detail    *string   `gorm:"type:json" json:"-"`
	IP        *string   `gorm:"size:64" json:"-"`
	CreatedAt time.Time `json:"-"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// AgentApply 对应 agent_applies 表。
type AgentApply struct {
	ID         int64      `gorm:"primaryKey" json:"id"`
	UserID     int64      `gorm:"uniqueIndex" json:"-"`
	Status     int        `gorm:"default:0" json:"status"` // 0=待审核 1=通过 2=拒绝
	Remark     *string    `gorm:"size:255" json:"-"`
	ReviewedAt *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (AgentApply) TableName() string { return "agent_applies" }
