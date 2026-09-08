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
	// 商户号校验：防止跨商户重放（若商户密钥被共用）
	if form.Get("pid") != e.cfg.PID {
		return nil, fmt.Errorf("epay pid mismatch")
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
		Raw:     form.Encode(), // Encode 按 key 升序,原文落库供对账
	}, nil
}

// Query 主动查单（彩虹协议 act=order）。key 走 POST form，避免出现在 URL/网关访问日志里。
func (e *Epay) Query(ctx context.Context, orderNo string) (*QueryResult, error) {
	form := url.Values{}
	form.Set("act", "order")
	form.Set("pid", e.cfg.PID)
	form.Set("key", e.cfg.Key)
	form.Set("out_trade_no", orderNo)
	u := strings.TrimRight(e.cfg.Gateway, "/") + "/api.php"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	// 纯整数格式化,不经 float(fen/100),避免大额浮点精度问题
	sign := ""
	if fen < 0 {
		sign = "-"
		fen = -fen
	}
	return fmt.Sprintf("%s%d.%02d", sign, fen/100, fen%100)
}

// yuanStrToFen 元字符串 → 分(字符串分位精确解析,不经 float;
// float*100+0.5 在如 8.65 等二进制近似值上会差 1 分导致整单对账失败)。
// 仅接受 0-2 位小数的非负金额,其余一律拒绝。
func yuanStrToFen(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty money")
	}
	intPart, fracPart, hasDot := strings.Cut(s, ".")
	if !isDigits(intPart) || (hasDot && !isDigits(fracPart)) {
		return 0, fmt.Errorf("invalid money %q", s)
	}
	if len(fracPart) > 2 {
		return 0, fmt.Errorf("too many decimals in %q", s)
	}
	fen, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil || fen < 0 {
		return 0, fmt.Errorf("invalid money %q", s)
	}
	fen *= 100
	if len(fracPart) >= 1 {
		fen += int64(fracPart[0]-'0') * 10
	}
	if len(fracPart) >= 2 {
		fen += int64(fracPart[1] - '0')
	}
	return fen, nil
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
