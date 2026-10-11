// Package page 分页钳制唯一 owner（ADR-0004，ADR-0001 的对称收敛：
// PageOK 拥有 nil→[]，本包拥有缺省/越界回落）。
//
// 缺省与上限成对出现（Limit），杜绝“缺省 10、上限 50”散落在各 List 手写。
// 标准 Limit（缺省 10/上限 50）覆盖全部列表；审计类 Limit（缺省 20/上限 100）
// 仅用于审计日志与流量重置记录（管理端翻阅场景，契约 §16 有载）。
package page

// Limit 分页边界：缺省与上限成对，调用方只选档，不手写数字。
type Limit struct {
	Default int
	Max     int
}

// Standard 标准列表档：缺省 10，上限 50（契约 §1.3）。
var Standard = Limit{Default: 10, Max: 50}

// Audit 审计类档：缺省 20，上限 100（审计日志/重置记录翻阅场景）。
var Audit = Limit{Default: 20, Max: 100}

// ClampPage 页码钳制：<1 回落 1。
func ClampPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// ClampPageSize 标准档条数钳制：<1 或 >50 回落 10。
func ClampPageSize(pageSize int) int {
	return Standard.ClampSize(pageSize)
}

// Clamp 标准档一对钳制。
func Clamp(page, pageSize int) (int, int) {
	return ClampPage(page), ClampPageSize(pageSize)
}

// ClampSize 按档钳制条数：<1 或 >Max 回落 Default。
func (l Limit) ClampSize(pageSize int) int {
	if pageSize < 1 || pageSize > l.Max {
		return l.Default
	}
	return pageSize
}
