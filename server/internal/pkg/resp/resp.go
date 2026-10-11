// Package resp 是 handler 层的统一 HTTP seam：envelope、分页归一、绑定校验收敛在同一 module 内。
// handler 只学这一个窄 interface，不再分别引用 validate/errs；映射语义与分页契约保持不变。
// 用语：module 指内聚单元，interface/seam 指窄契约，adapter 指薄委托。
package resp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"ylink-backend/internal/pkg/errs"
)

// Body 为统一信封。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 返回成功响应（code=0, message=ok）。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

// OKWithMessage 返回成功响应并携带自定义 message。
func OKWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: message, Data: data})
}

// Fail 返回业务错误；错误码与 HTTP 状态由 errs.Error 自带映射决定。
func Fail(c *gin.Context, err error) {
	e := errs.From(err)
	c.JSON(e.HTTP, Body{Code: e.Code, Message: e.Message, Data: nil})
}

// FailWithCode 以业务码+文案直接返回（不依赖 error 类型）。
func FailWithCode(c *gin.Context, code int, message string) {
	e := errs.Resolve(code, message)
	c.JSON(e.HTTP, Body{Code: e.Code, Message: e.Message, Data: nil})
}

// FailPlain 以纯文本返回业务错误（订阅下发等非 envelope seam，代理客户端直连）。
func FailPlain(c *gin.Context, err error) {
	e := errs.From(err)
	c.String(e.HTTP, e.Message)
}

// Page 为分页响应数据结构。
type Page struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// PageOK 返回分页成功响应，nil 切片归一为空数组。
// Go nil slice 会被 encoding/json 序列化为 null（如空表查询），前端以 list.length
// 判空时会对 null 抛 TypeError，导致页面渲染中断（列表区转圈卡死）。
// 这里统一兜底：nil 切片返回空数组 []，保持分页契约稳定。
// 泛型 seam 在编译期固定元素类型，空值判定只需 == nil，无需 reflect 运行时分支。
func PageOK[T any](c *gin.Context, list []T, total int64, page, pageSize int) {
	if list == nil {
		list = []T{}
	}
	OK(c, Page{List: list, Total: total, Page: page, PageSize: pageSize})
}

// Bind 解析 JSON 请求体；失败时按 40000 信封直接应答并返回 false。
// 收敛前每个 handler 重复三行：ShouldBindJSON + Messages 拼接 + FailWithCode；
// 收敛后 handler 只调这一句，不再单独引用校验 seam。
func Bind(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		FailWithCode(c, 40000, "参数校验失败: "+Messages(err))
		return false
	}
	return true
}

var validatorEngine = binding.Validator.Engine().(*validator.Validate)

// Messages 将 binding/validator 错误转换为用户可读文案（无内部细节）。
// canonical 实现收敛在此 module，validate 包仅为薄 adapter 委托。
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
func Engine() *validator.Validate { return validatorEngine }
