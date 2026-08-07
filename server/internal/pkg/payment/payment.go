// Package payment 定义支付驱动抽象与注册表，隔离外部差异点。
// 一期实现易支付（epay，彩虹易支付兼容协议），method_code 如 "epay_alipay" / "epay_wxpay"。
package payment

import (
	"context"
	"net/http"
)

// Payment 创建支付单请求。
type Payment struct {
	OrderNo   string // 商户订单号
	Method    string // 渠道码（如 epay_alipay）
	Amount    int64  // 金额（分）
	Subject   string // 商品名称
	NotifyURL string // 异步通知地址
	ReturnURL string // 同步跳转地址
}

// CreateResult 创建结果。
type CreateResult struct {
	Type     string // url / qrcode / paid
	Content  string // 跳转 URL 或二维码内容
	ExpireIn int    // 秒
}

// NotifyResult 回调解析结果（验签后）。
type NotifyResult struct {
	OrderNo string // 商户订单号
	TradeNo string // 网关流水号
	Amount  int64  // 实收（分）
	Paid    bool
}

// QueryResult 主动查单结果。
type QueryResult struct {
	TradeNo string
	Paid    bool
}

// Driver 支付驱动接口。
type Driver interface {
	Name() string
	CreatePayment(ctx context.Context, p *Payment) (*CreateResult, error)
	VerifyNotify(r *http.Request) (*NotifyResult, error)
	Query(ctx context.Context, tradeNo string) (*QueryResult, error)
}

var registry = map[string]Driver{}

// Register 注册驱动；codes 为该驱动对应的渠道码（如 epay_alipay / epay_wxpay），
// 缺省使用驱动 Name()。
func Register(d Driver, codes ...string) {
	if len(codes) == 0 {
		codes = []string{d.Name()}
	}
	for _, c := range codes {
		registry[c] = d
	}
}

// Get 按渠道码取驱动；未注册返回 nil。
func Get(name string) Driver { return registry[name] }

// Available 已注册渠道码列表。
func Available() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}
