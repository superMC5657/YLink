package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// claim ica2-2：行为变更=金额转换+溢出防护收敛 money 深 Module，非法金额入口即不可表达；
// 必须不变=正常金额分元互转精度、余额为负拒绝、提现/划转正金额语义；
// 最怕的失败=超大金额回绕为负造成资损、调账负值被误杀。
func TestMoneyModule(t *testing.T) {
	// 正常互转
	assert.Equal(t, 16.0, FenToYuan(1600))
	assert.Equal(t, int64(1600), YuanToFen(16.0))
	assert.InEpsilon(t, 0.1, FenToYuan(10), 1e-9)

	// 正金额 checked：0/负/超限拒绝
	_, ok := CheckedYuanToFen(0)
	assert.False(t, ok)
	_, ok = CheckedYuanToFen(-5)
	assert.False(t, ok)
	_, ok = CheckedYuanToFen(1e19)
	assert.False(t, ok)
	_, ok = CheckedYuanToFen(1e14) // 1e16 分超 MaxMoneyFen
	assert.False(t, ok)
	fen, ok := CheckedYuanToFen(16.0)
	assert.True(t, ok)
	assert.Equal(t, int64(1600), fen)

	// 调账 checked：负值允许，0 与超限拒绝
	dfen, ok := CheckedAdjustYuanToFen(-5.0)
	assert.True(t, ok)
	assert.Equal(t, int64(-500), dfen)
	_, ok = CheckedAdjustYuanToFen(0)
	assert.False(t, ok)
	_, ok = CheckedAdjustYuanToFen(1e19)
	assert.False(t, ok)

	// 台账界：0 允许，超限拒绝
	_, ok = CheckedBoundedYuanToFen(0)
	assert.True(t, ok)
	_, ok = CheckedBoundedYuanToFen(1e19)
	assert.False(t, ok)

	// 余额语义
	assert.True(t, BalanceAfterValid(100, -100))
	assert.False(t, BalanceAfterValid(0, -100))
	assert.True(t, ValidMoneyFen(1))
	assert.False(t, ValidMoneyFen(0))
	assert.False(t, ValidMoneyFen(-1))
	assert.False(t, ValidMoneyFen(MaxMoneyFen+1))
}
