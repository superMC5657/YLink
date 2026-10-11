// Package admin 管理端 handler（role=admin），按业务域拆分文件。
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/handler"
	"ylink-backend/internal/middleware"
	"ylink-backend/internal/service"
)

// Admin 管理端 module 总装：按域缝合的 handler 集合。
// 每个域 handler 只依赖对应窄 seam（interface），便于单域变更集中一处。
// 装配经 service.AdminSeams 直取各域；telegram 能力由 router 直注（facade 已退役，见 ADR-0005）。
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
	telegram *service.TelegramService
}

func NewAdmin(s *service.AdminSeams, tg *service.TelegramService) *Admin {
	return &Admin{
		audit: s.Audit, user: s.User, agent: s.Agent, order: s.Order,
		catalog: s.Catalog, content: s.Content, node: s.Node, stats: s.Stats,
		traffic: s.Traffic, ticket: s.Ticket, withdraw: s.Withdraw,
		system: s.System, version: s.Version, telegram: tg,
	}
}

func (h *Admin) adminID(c *gin.Context) int64 { return middleware.UserID(c) }

func idParam(c *gin.Context) (int64, bool) {
	// 薄 adapter：实现收敛于 handler param 唯一 owner。
	return handler.IDParam(c)
}

func pageParams(c *gin.Context) (int, int) {
	// 薄 adapter：实现收敛于 handler param 唯一 owner。
	return handler.PageParams(c)
}

func pageParamsAudit(c *gin.Context) (int, int) {
	// 薄 adapter：审计类分页（缺省 20/上限 100）收敛于 handler param 唯一 owner。
	return handler.PageParamsAudit(c)
}

func statDays(c *gin.Context) int {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	return days
}
