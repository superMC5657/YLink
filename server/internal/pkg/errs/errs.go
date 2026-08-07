// Package errs 定义业务错误类型与错误码（与 docs/api/README.md 第 2 节保持一致）。
package errs

import (
	"errors"
	"net/http"
)

// Error 为携带业务码的错误，HTTP 状态由 Code 推导（特殊码显式覆盖）。
type Error struct {
	Code    int
	Message string
	HTTP    int
}

func (e *Error) Error() string { return e.Message }

// New 创建业务错误；HTTP 状态自动映射。
func New(code int, message string) *Error {
	return &Error{Code: code, Message: message, HTTP: httpStatus(code)}
}

// httpStatus 依据契约文档第 2 节推导 HTTP 状态。
func httpStatus(code int) int {
	switch code {
	case 0:
		return http.StatusOK
	case 10003:
		return http.StatusTooManyRequests
	case 11003, 14001, 15002:
		return http.StatusConflict
	}
	switch code / 100 {
	case 400:
		return http.StatusBadRequest
	case 401:
		return http.StatusUnauthorized
	case 403:
		return http.StatusForbidden
	case 404:
		return http.StatusNotFound
	case 409:
		return http.StatusConflict
	case 429:
		return http.StatusTooManyRequests
	case 500:
		return http.StatusInternalServerError
	default: // 1xxxx 业务码默认 400
		return http.StatusBadRequest
	}
}

// From 将任意 error 归一化为 *Error；未知错误统一 50000（不外泄内部细节）。
func From(err error) *Error {
	if err == nil {
		return &Error{Code: 0, Message: "ok", HTTP: http.StatusOK}
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Code: 50000, Message: "服务器内部错误", HTTP: http.StatusInternalServerError}
}

// Resolve 以业务码构造错误；code=0 视为成功。
func Resolve(code int, message string) *Error {
	if code == 0 {
		return &Error{Code: 0, Message: message, HTTP: http.StatusOK}
	}
	return New(code, message)
}

// ---- 常用错误常量（契约第 2 节） ----

var (
	ErrParam          = New(40000, "参数校验失败")
	ErrUnauthorized   = New(40100, "未登录或登录已过期")
	ErrBadCredentials = New(40101, "邮箱或密码错误")
	ErrForbidden      = New(40300, "无权限操作")
	ErrNotFound       = New(40400, "资源不存在")
	ErrConflict       = New(40900, "状态冲突")
	ErrTooManyReq     = New(42900, "请求过于频繁，请稍后再试")
	ErrInternal       = New(50000, "服务器内部错误")

	ErrEmailTaken      = New(10001, "该邮箱已注册")
	ErrCaptcha         = New(10002, "验证码错误或已过期")
	ErrCaptchaFrequent = New(10003, "验证码发送过于频繁")
	ErrInviteCode      = New(10004, "邀请码无效")

	ErrPlanNotAvailable    = New(11001, "套餐不存在或未上架")
	ErrPlanPeriod          = New(11002, "套餐不支持所选周期")
	ErrOrderStatus         = New(11003, "订单状态不允许该操作")
	ErrBalanceInsufficient = New(11004, "余额不足")
	ErrPayMethod           = New(11005, "支付渠道不可用")
	ErrPlanInUse           = New(11006, "该套餐存在关联订单，无法删除")

	ErrCoupon                 = New(12001, "优惠券无效或已过期")
	ErrInviteMax              = New(13001, "邀请码数量已达上限")
	ErrCommissionInsufficient = New(13002, "可划转佣金不足")

	ErrTicketClosed = New(14001, "工单已关闭")

	ErrAgentNotQualified = New(15001, "暂不满足代理申请条件")
	ErrAgentDuplicated   = New(15002, "代理申请审核中，请勿重复提交")
)
