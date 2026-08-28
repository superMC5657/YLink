package model

import "time"

// ---- 邀请与佣金 ----

type InviteSummaryResp struct {
	CommissionBalance float64 `json:"commission_balance"`
	CommissionRate    int     `json:"commission_rate"`
	RegisteredCount   int64   `json:"registered_count"`
	TotalCommission   float64 `json:"total_commission"`
	PendingCommission float64 `json:"pending_commission"`
}

type InviteCodeItem struct {
	Code      string    `json:"code"`
	UsedCount int       `json:"used_count"`
	CreatedAt time.Time `json:"created_at"`
}

type InviteCodesResp struct {
	List  []InviteCodeItem `json:"list"`
	Limit int              `json:"limit"`
	// RegisterURLPrefix 注册链接路径后缀(如 /#/register?code=):不含域名,
	// 完整链接由前端按当前页面 origin 拼接;本字段仅为契约占位,前端不消费其值。
	RegisterURLPrefix string `json:"register_url_prefix"`
}

type CommissionRecordResp struct {
	OrderNo     string     `json:"order_no"`
	Amount      float64    `json:"amount"`
	Rate        int        `json:"rate"`
	Status      int        `json:"status"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type TransferReq struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

type TransferResp struct {
	Balance           float64 `json:"balance"`
	CommissionBalance float64 `json:"commission_balance"`
}

// ---- 佣金提现（F02，仅代理商） ----

// WithdrawCreateReq POST /invite/withdraw：提交佣金提现工单（提现方式 + 收款账号 + 金额）。
type WithdrawCreateReq struct {
	Amount  float64 `json:"amount" binding:"required,gt=0"`
	Method  string  `json:"method" binding:"required,max=32"`
	Account string  `json:"account" binding:"required,max=255"`
}

// WithdrawItem GET /invite/withdraws 提现记录（仅本人）。
type WithdrawItem struct {
	ID           int64      `json:"id"`
	Amount       float64    `json:"amount"`
	Method       string     `json:"method"`
	Account      string     `json:"account"`
	Status       int        `json:"status"` // 0=处理中 1=已发放 2=已退回
	ReviewRemark *string    `json:"review_remark"`
	TicketID     int64      `json:"ticket_id"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ---- 代理商 ----

type AgentCondition struct {
	Met  bool   `json:"met"`
	Text string `json:"text"`
}

type AgentStatusResp struct {
	IsAgent              bool             `json:"is_agent"`
	ApplyStatus          string           `json:"apply_status"` // none/pending/approved/rejected
	Qualified            bool             `json:"qualified"`
	ValidInvites         int64            `json:"valid_invites"`
	RequiredValidInvites int              `json:"required_valid_invites"`
	Conditions           []AgentCondition `json:"conditions"`
}
