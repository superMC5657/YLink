// Package validate 提供参数校验辅助：binding 错误统一转换为 40000 + 字段级文案。
package validate

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var v = binding.Validator.Engine().(*validator.Validate)

// Messages 将 binding/validator 错误转换为用户可读文案（无内部细节）。
func Messages(err error) string {
	if err == nil {
		return ""
	}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		parts := make([]string, 0, len(ve))
		for _, fe := range ve {
			parts = append(parts, fieldMessage(fe))
		}
		return strings.Join(parts, "; ")
	}
	return "请求参数不合法"
}

func fieldMessage(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " 不能为空"
	case "email":
		return field + " 格式不正确"
	case "min":
		return field + " 长度/数值不符合要求"
	case "max":
		return field + " 长度/数值超出限制"
	case "oneof":
		return field + " 取值不合法"
	default:
		return field + " 校验失败(" + fe.Tag() + ")"
	}
}

// Engine 暴露底层 validator，供自定义校验注册。
func Engine() *validator.Validate { return v }
