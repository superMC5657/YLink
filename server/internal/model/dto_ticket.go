package model

import "time"

// ---- 工单 ----

type TicketListItem struct {
	ID          int64      `json:"id"`
	Subject     string     `json:"subject"`
	Level       int        `json:"level"`
	Type        int        `json:"type"` // 0=普通 1=佣金提现(F02)
	Status      int        `json:"status"`
	ReopenCount int        `json:"reopen_count"`
	LastReplyAt *time.Time `json:"last_reply_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateTicketReq struct {
	Subject string `json:"subject" binding:"required,min=1,max=128"`
	Level   int    `json:"level" binding:"oneof=0 1 2"`
	Message string `json:"message" binding:"required"`
}

type ReplyTicketReq struct {
	Message string `json:"message" binding:"required"`
}

type TicketDetailResp struct {
	ID          int64           `json:"id"`
	Subject     string          `json:"subject"`
	Level       int             `json:"level"`
	Type        int             `json:"type"`
	Status      int             `json:"status"`
	ReopenCount int             `json:"reopen_count"`
	CreatedAt   time.Time       `json:"created_at"`
	Messages    []TicketMsgResp `json:"messages"`
	// Withdraw 提现工单附带的结构化提现信息（type=1 时非空，F02）
	Withdraw *TicketWithdrawInfo `json:"withdraw,omitempty"`
}

// TicketWithdrawInfo 提现工单详情（F02）：管理端据此展示提现方式/账号并执行确认/拒绝。
type TicketWithdrawInfo struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Amount       float64    `json:"amount"`
	Method       string     `json:"method"`
	Account      string     `json:"account"`
	Status       int        `json:"status"` // 0=处理中 1=已发放 2=已退回
	ReviewRemark *string    `json:"review_remark"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type TicketMsgResp struct {
	ID         int64     `json:"id"`
	SenderType int       `json:"sender_type"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}
