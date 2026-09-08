package payment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestYuanStrToFenExact 金额字符串分位精确解析:float*100 差 1 分会导致整单对账失败。
func TestYuanStrToFenExact(t *testing.T) {
	cases := []struct {
		in    string
		want  int64
		okErr bool
	}{
		{"10.29", 1029, true}, // float: 10.29*100=1028.9999… 经典差 1 分值
		{"4.35", 435, true},   // float 二进制近似值的又一经典值
		{"8.65", 865, true},
		{"0.10", 10, true},
		{"10", 1000, true},
		{"10.5", 1050, true},
		{"10.", 1000, true},
		{"  6.00  ", 600, true},
		{"1.234", 0, false}, // 超 2 位小数
		{"-1.00", 0, false}, // 负金额
		{"abc", 0, false},   // 非数字
		{"1.2.3", 0, false}, // 多点
		{"", 0, false},      // 空
		{"1e3", 0, false},   // 科学计数法拒绝
		{"0x10", 0, false},  // 进制前缀拒绝
	}
	for _, c := range cases {
		got, err := yuanStrToFen(c.in)
		if c.okErr {
			require.NoError(t, err, "input %q", c.in)
			assert.Equal(t, c.want, got, "input %q", c.in)
		} else {
			assert.Error(t, err, "input %q", c.in)
		}
	}
}

func TestFenToYuanStr(t *testing.T) {
	assert.Equal(t, "10.00", fenToYuanStr(1000))
	assert.Equal(t, "0.05", fenToYuanStr(5))
	// 大额不经 float:float64 精度在 2^53 分(约 9 亿亿元)内安全,但格式化走纯整数更稳
	assert.Equal(t, "12345678901.23", fenToYuanStr(1234567890123))
}
