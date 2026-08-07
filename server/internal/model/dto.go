package model

import (
	"math"
	"time"
)

// ---- 通用 ----

// FenToYuan 分 → 元（number，两位小数）。
func FenToYuan(fen int64) float64 { return math.Round(float64(fen)) / 100 }

// YuanToFen 元 → 分（四舍五入到分）。
func YuanToFen(yuan float64) int64 { return int64(math.Round(yuan * 100)) }

// ---- 认证 ----

type CaptchaReq struct {
	Email string `json:"email" binding:"required,email"`
	Type  string `json:"type" binding:"required,oneof=register forgot"`
}

type CaptchaResp struct {
	ExpireIn    int `json:"expire_in"`
	ResendAfter int `json:"resend_after"`
}

type AuthRegisterReq struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8,max=72"`
	EmailCode  string `json:"email_code" binding:"required,len=6"`
	InviteCode string `json:"invite_code"`
}

type AuthLoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgotReq struct {
	Email     string `json:"email" binding:"required,email"`
	EmailCode string `json:"email_code" binding:"required,len=6"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

type UserBrief struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  int    `json:"role"`
}

type TokenResp struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	User         UserBrief `json:"user"`
}

// ---- 用户 ----

type UpdateProfileReq struct {
	RemindExpire  *bool `json:"remind_expire"`
	RemindTraffic *bool `json:"remind_traffic"`
}

type UserProfileResp struct {
	RemindExpire  bool `json:"remind_expire"`
	RemindTraffic bool `json:"remind_traffic"`
}

type UserStatResp struct {
	Email             string  `json:"email"`
	Balance           float64 `json:"balance"`
	CommissionBalance float64 `json:"commission_balance"`
	PendingOrderCount int64   `json:"pending_order_count"`
	OpenTicketCount   int64   `json:"open_ticket_count"`
	InvitedCount      int64   `json:"invited_count"`
	IsAgent           bool    `json:"is_agent"`
}

// ---- 通用时间 ----

// Time 契约时间格式（RFC3339 带时区）。
type Time = time.Time
