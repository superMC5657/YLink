package model

import "math"

// 资金金额深 Module：分/元转换与溢出防护的唯一拥有者。
//
// 危险：YuanToFen 对超大金额会回绕为负数（负 fen 绕过「余额 < 金额」校验后
// 反而增加余额造成资损）或精度丢失。非法金额必须在入口即不可表达：
// 所有动账路径（赠送 Transfer、提现 SubmitWithdraw、手工调账 AdjustBalance、
// 批量发放 BatchUsers、订单/优惠券金额）一律经由本 Module 的 checked 入口，
// 转换与校验不分离。展示侧只读 FenToYuan（分→元恒安全）可直接用。
//
// Depth 所在：安全整数域 + 回绕拦截 + 精度语义只活在一处。

// MaxMoneyFen 资金金额安全上限（分）= 2^52（约 4.5e13 元）。
// 仅作 float64 元→分转换的溢出/精度防护，非业务限额。
const MaxMoneyFen = int64(1) << 52

func moneyFenToYuan(fen int64) float64 { return math.Round(float64(fen)) / 100 }

func moneyYuanToFen(yuan float64) int64 { return int64(math.Round(yuan * 100)) }

// ValidMoneyFen 校验动账金额（分）：> 0（拦截回绕出的负值/0）且在安全整数域内。
// 用于 Transfer / SubmitWithdraw 等“金额必须为正”的入口。
func ValidMoneyFen(fen int64) bool { return fen > 0 && fen <= MaxMoneyFen }

// ValidAdjustFen 校验手工调账金额（分）：可正可负（加款/扣款），但绝对值必须在
// 安全域内，拦截回绕出的超大绝对值。比较式写法（不对 MinInt64 取反，防溢出）。
func ValidAdjustFen(fen int64) bool {
	if fen == 0 {
		return false
	}
	return fen <= MaxMoneyFen && fen >= -MaxMoneyFen
}

// CheckedYuanToFen 元→分 checked 入口：转换 + 正金额校验一次完成。
// ok=false 时调用方一律按非法金额拒绝（ErrAmountInvalid），不得再用裸 YuanToFen。
func CheckedYuanToFen(yuan float64) (fen int64, ok bool) {
	fen = moneyYuanToFen(yuan)
	if !ValidMoneyFen(fen) {
		return 0, false
	}
	return fen, true
}

// CheckedAdjustYuanToFen 调账元→分 checked 入口：允许负数，绝对值限域。
func CheckedAdjustYuanToFen(yuan float64) (fen int64, ok bool) {
	fen = moneyYuanToFen(yuan)
	if !ValidAdjustFen(fen) {
		return 0, false
	}
	return fen, true
}

// BalanceAfterValid 调账后余额校验：调整后 balance 不允许为负。
func BalanceAfterValid(balance, deltaFen int64) bool { return balance+deltaFen >= 0 }

// ValidBoundedFen 台账金额界校验：展示/配置类金额允许 0，只拦截回绕超限。
func ValidBoundedFen(fen int64) bool { return fen <= MaxMoneyFen && fen >= -MaxMoneyFen }

// CheckedBoundedYuanToFen 台账元→分 checked 入口：允许 0，仅拒超限回绕。
func CheckedBoundedYuanToFen(yuan float64) (fen int64, ok bool) {
	fen = moneyYuanToFen(yuan)
	if !ValidBoundedFen(fen) {
		return 0, false
	}
	return fen, true
}
