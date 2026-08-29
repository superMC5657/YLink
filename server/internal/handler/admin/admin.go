// Package admin 管理端 handler（role=admin），按业务域拆分文件。
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/service"
)

// Admin 管理端端点（role=admin）。
type Admin struct {
	svc *service.AdminService
}

func NewAdmin(svc *service.AdminService) *Admin { return &Admin{svc: svc} }

func (h *Admin) adminID(c *gin.Context) int64 { return middleware.UserID(c) }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return 0, false
	}
	return id, true
}

func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return page, pageSize
}

func statDays(c *gin.Context) int {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	return days
}
