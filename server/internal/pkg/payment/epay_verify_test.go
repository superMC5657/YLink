package payment

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// verify 与 sign 单一实现等价（sign/sign_type 排除语义一致）且常量时间比较不影响判定。
func TestEpayVerifySign(t *testing.T) {
	e := &Epay{cfg: EpayConfig{PID: "1001", Key: "testkey"}}
	form := url.Values{
		"pid": {"1001"}, "trade_no": {"20260825001"}, "out_trade_no": {"ORD1"},
		"type": {"alipay"}, "money": {"10.00"}, "trade_status": {"TRADE_SUCCESS"},
	}
	sign := e.sign(form)
	assert.NotEmpty(t, sign)

	// form 携带 sign/sign_type（回调真实形态）不影响重算
	form.Set("sign", sign)
	form.Set("sign_type", "MD5")
	assert.True(t, e.verify(form, sign))

	// 篡改任一参数 / 空签名 → false
	form.Set("money", "0.01")
	assert.False(t, e.verify(form, sign))
	assert.False(t, e.verify(form, ""))
}
