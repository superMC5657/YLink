package model

import "time"

// ---- 工单 ----

type TicketListItem struct {
	ID          int64      `json:"id"`
	Subject     string     `json:"subject"`
	Level       int        `json:"level"`
	Status      int        `json:"status"`
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
	ID        int64           `json:"id"`
	Subject   string          `json:"subject"`
	Level     int             `json:"level"`
	Status    int             `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	Messages  []TicketMsgResp `json:"messages"`
}

type TicketMsgResp struct {
	ID         int64     `json:"id"`
	SenderType int       `json:"sender_type"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}
