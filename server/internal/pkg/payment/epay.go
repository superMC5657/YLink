// 易支付（彩虹易支付兼容协议）驱动。
// 下单：GET {gateway}/submit.php?pid=&type=&out_trade_no=&notify_url=&return_url=&name=&money=&sign=
// 异步通知：POST form 表单（pid/trade_no/out_trade_no/type/name/money/trade_status/sign）
// 签名：除 sign、sign_type 外参数按 key 升序拼接 k=v&...，末尾追加 &key={商户密钥}，MD5 小写。
package payment

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// EpayConfig 易支付配置。
type EpayConfig struct {
	Gateway string
	PID     string
	Key     string
}

// Epay 易支付驱动（方法名 "epay"；渠道由 method 后缀区分，如 epay_alipay / epay_wxpay）。
type Epay struct {
	cfg EpayConfig
}

func NewEpay(cfg EpayConfig) *Epay { return &Epay{cfg: cfg} }

// Name 注册名。
func (e *Epay) Name() string { return "epay" }

// CreatePayment 拉起支付：跳转型。
func (e *Epay) CreatePayment(ctx context.Context, p *Payment) (*CreateResult, error) {
	channel := strings.TrimPrefix(p.Method, "epay_")
	if channel == "" || channel == p.Method {
		channel = "alipay"
	}
	params := url.Values{}
	params.Set("pid", e.cfg.PID)
	params.Set("type", channel)
	params.Set("out_trade_no", p.OrderNo)
	params.Set("notify_url", p.NotifyURL)
	params.Set("return_url", p.ReturnURL)
	params.Set("name", p.Subject)
	params.Set("money", fenToYuanStr(p.Amount))
	params.Set("sign", e.sign(params))
	params.Set("sign_type", "MD5")

	u := strings.TrimRight(e.cfg.Gateway, "/") + "/submit.php?" + params.Encode()
	return &CreateResult{Type: "url", Content: u, ExpireIn: 1800}, nil
}

// VerifyNotify 验签并解析回调。
func (e *Epay) VerifyNotify(r *http.Request) (*NotifyResult, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}
	form := r.Form
	gotSign := form.Get("sign")
	if gotSign == "" || !e.verify(form, gotSign) {
		return nil, fmt.Errorf("epay sign verify failed")
	}
	amountFen, err := yuanStrToFen(form.Get("money"))
	if err != nil {
		return nil, fmt.Errorf("bad money: %w", err)
	}
	paid := form.Get("trade_status") == "TRADE_SUCCESS"
	return &NotifyResult{
		OrderNo: form.Get("out_trade_no"),
		TradeNo: form.Get("trade_no"),
		Amount:  amountFen,
		Paid:    paid,
	}, nil
}

// Query 主动查单（彩虹协议 act=order）。
func (e *Epay) Query(ctx context.Context, orderNo string) (*QueryResult, error) {
	params := url.Values{}
	params.Set("act", "order")
	params.Set("pid", e.cfg.PID)
	params.Set("key", e.cfg.Key)
	params.Set("out_trade_no", orderNo)
	u := strings.TrimRight(e.cfg.Gateway, "/") + "/api.php?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	// 彩虹协议返回 JSON：{"code":1,"msg":"查询订单号成功!","trade_no":...,"money":...,"status":1}
	var r struct {
		Code    int    `json:"code"`
		TradeNo string `json:"trade_no"`
		Status  int    `json:"status"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("epay query bad response: %w", err)
	}
	return &QueryResult{TradeNo: r.TradeNo, Paid: r.Status == 1}, nil
}

// ---- 签名 ----

func (e *Epay) sign(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params.Get(k))
	}
	sb.WriteString("&key=")
	sb.WriteString(e.cfg.Key)
	sum := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(sum[:])
}

func (e *Epay) verify(form url.Values, sign string) bool {
	keys := make([]string, 0, len(form))
	for k := range form {
		if k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(form.Get(k))
	}
	sb.WriteString("&key=")
	sb.WriteString(e.cfg.Key)
	sum := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(sum[:]) == sign
}

// ---- 金额工具 ----

func fenToYuanStr(fen int64) string {
	return strconv.FormatFloat(float64(fen)/100, 'f', 2, 64)
}

func yuanStrToFen(s string) (int64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(f*100 + 0.5), nil
}
