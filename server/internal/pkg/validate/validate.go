// Package validate 为历史 seam，现为 resp 的薄 adapter。
// 校验文案的 canonical 实现已收敛在 resp module（Messages/Engine），此处仅委托，
// 保证既有引用路径仍可用；新 handler 直接用 resp.Bind，不再学此 interface。
// 用语：module 指内聚单元，interface/seam 指窄契约，adapter 指薄委托。
package validate

import (
	"github.com/go-playground/validator/v10"

	"ylink-backend/internal/pkg/resp"
)

// Messages 将 binding/validator 错误转换为用户可读文案，委托到 resp seam。
func Messages(err error) string { return resp.Messages(err) }

// Engine 暴露底层 validator，委托到 resp seam。
func Engine() *validator.Validate { return resp.Engine() }
