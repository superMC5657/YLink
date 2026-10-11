// Package admin 管理端 handler（role=admin），按业务域拆分文件。
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/service"
)

// Admin 管理端 module 总装：按域缝合的 handler 集合。
// 每个域 handler 只依赖对应窄 seam（interface），便于单域变更集中一处。
// 术语：下文用 module 指代域内聚单元，用 interface/seam 指代窄契约，用 adapter 指代 telegram 委托。
type Admin struct {
	audit    service.AdminAuditSeam
	user     service.AdminUserSeam
	agent    service.AdminAgentSeam
	order    service.AdminOrderSeam
	catalog  service.AdminCatalogSeam
	content  service.AdminContentSeam
	node     service.AdminNodeBatchSeam
	stats    service.AdminStatsSeam
	traffic  service.AdminTrafficSeam
	ticket   service.AdminTicketSeam
	withdraw service.AdminWithdrawSeam
	system   service.AdminSystemSeam
	version  service.AdminVersionSeam
	telegram service.AdminTelegramSeam
}

func NewAdmin(svc *service.AdminService) *Admin {
	return &Admin{
		audit: svc, user: svc, agent: svc, order: svc,
		catalog: svc, content: svc, node: svc, stats: svc,
		traffic: svc, ticket: svc, withdraw: svc,
		system: svc, version: svc, telegram: svc,
	}
}

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
