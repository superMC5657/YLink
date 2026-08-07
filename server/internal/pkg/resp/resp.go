// Package resp 提供统一响应信封 {code, message, data}。
package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ylink/internal/pkg/errs"
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

// Page 为分页响应数据结构。
type Page struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// PageOK 返回分页成功响应。
func PageOK(c *gin.Context, list any, total int64, page, pageSize int) {
	OK(c, Page{List: list, Total: total, Page: page, PageSize: pageSize})
}
