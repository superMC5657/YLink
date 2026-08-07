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
	List              []InviteCodeItem `json:"list"`
	Limit             int              `json:"limit"`
	RegisterURLPrefix string           `json:"register_url_prefix"`
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
