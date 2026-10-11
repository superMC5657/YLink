package service

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/repo"
)

// AdminService 管理端 module 总装（facade）：按域深模块装配，对外保持原有窄缝合点。
// 每个域 module 只持有最小依赖；telegram late 注入收敛于 adapter seam。
// 本 facade 仅做委托，便于存量调用方逐步迁移到各域 seam。
type AdminService struct {
	audit    *adminAuditModule
	user     *adminUserModule
	agent    *adminAgentModule
	order    *adminOrderModule
	catalog  *adminCatalogModule
	content  *adminContentModule
	node     *adminNodeBatchModule
	stats    *adminStatsModule
	traffic  *adminTrafficModule
	ticket   *adminTicketModule
	withdraw *adminWithdrawModule
	system   *adminSystemModule
	version  *adminVersionModule
	telegram *adminTelegramAdapter
}

func NewAdminService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, set *SettingService, ml *mailer.Mailer) *AdminService {
	k := newAdminKernel(db, rdb, repos, cfg, set, ml)
	audit := &adminAuditModule{db: k.db, repos: k.repos}
	user := &adminUserModule{db: k.db, rdb: k.rdb, repos: k.repos, cfg: k.cfg, ml: k.ml, audit: audit}
	agent := &adminAgentModule{db: k.db, rdb: k.rdb, repos: k.repos, audit: audit}
	order := &adminOrderModule{db: k.db, repos: k.repos, audit: audit}
	catalog := &adminCatalogModule{db: k.db, rdb: k.rdb, repos: k.repos}
	content := &adminContentModule{db: k.db, repos: k.repos, cfg: k.cfg, ml: k.ml, audit: audit}
	node := &adminNodeBatchModule{db: k.db, rdb: k.rdb, repos: k.repos, audit: audit}
	stats := &adminStatsModule{db: k.db, repos: k.repos}
	traffic := &adminTrafficModule{db: k.db, repos: k.repos, audit: audit}
	withdraw := &adminWithdrawModule{db: k.db, repos: k.repos, audit: audit}
	ticket := &adminTicketModule{db: k.db, repos: k.repos, withdraw: withdraw}
	system := &adminSystemModule{db: k.db, repos: k.repos, set: k.set}
	version := &adminVersionModule{rdb: k.rdb, cfg: k.cfg}
	telegram := &adminTelegramAdapter{}
	return &AdminService{
		audit: audit, user: user, agent: agent, order: order,
		catalog: catalog, content: content, node: node, stats: stats,
		traffic: traffic, ticket: ticket, withdraw: withdraw,
		system: system, version: version, telegram: telegram,
	}
}

// SetTelegram 注入 telegram 能力（router 装配期调用，仅触及 adapter seam）。
func (s *AdminService) SetTelegram(tg *TelegramService) { s.telegram.SetTelegram(tg) }

// SetupTelegramWebhook POST /admin/telegram/webhook/setup（F12，委托 adapter）。
func (s *AdminService) SetupTelegramWebhook(ctx context.Context, adminID int64, ip string) (*model.AdminTelegramWebhookSetupResp, error) {
	return s.telegram.SetupTelegramWebhook(ctx, adminID, ip)
}

// ---- 审计 ----

func (s *AdminService) ListAuditLogs(ctx context.Context, f AuditLogFilter, page, pageSize int) ([]model.AdminAuditLogItem, int64, []string, error) {
	return s.audit.ListAuditLogs(ctx, f, page, pageSize)
}

// ---- 用户 ----

func (s *AdminService) ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]model.AdminUserItem, int64, error) {
	return s.user.ListUsers(ctx, keyword, page, pageSize)
}

func (s *AdminService) UpdateUser(ctx context.Context, adminID, userID int64, req *model.AdminUpdateUserReq, ip string) error {
	return s.user.UpdateUser(ctx, adminID, userID, req, ip)
}

func (s *AdminService) AdjustBalance(ctx context.Context, adminID, userID int64, amountFen int64, remark, ip string) error {
	return s.user.AdjustBalance(ctx, adminID, userID, amountFen, remark, ip)
}

func (s *AdminService) ExportUsers(ctx context.Context, keyword string, fn func(rows [][]string) error) error {
	return s.user.ExportUsers(ctx, keyword, fn)
}

func (s *AdminService) BatchUsers(ctx context.Context, adminID int64, req *model.AdminBatchUserReq, ip string) (*model.AdminBatchUserResp, error) {
	return s.user.BatchUsers(ctx, adminID, req, ip)
}

func (s *AdminService) SendMail(ctx context.Context, adminID int64, req *model.AdminSendMailReq, ip string) (*model.AdminSendMailResp, error) {
	return s.user.SendMail(ctx, adminID, req, ip)
}

func (s *AdminService) ResetUserSubToken(ctx context.Context, adminID, userID int64, ip string) (string, error) {
	return s.user.ResetUserSubToken(ctx, adminID, userID, ip)
}

// ---- 代理与佣金 ----

func (s *AdminService) ListAgentApplies(ctx context.Context, status int, page, pageSize int) ([]model.AdminAgentApplyItem, int64, error) {
	return s.agent.ListAgentApplies(ctx, status, page, pageSize)
}

func (s *AdminService) ReviewAgentApply(ctx context.Context, adminID, applyID int64, approve bool, remark, ip string) error {
	return s.agent.ReviewAgentApply(ctx, adminID, applyID, approve, remark, ip)
}

func (s *AdminService) ListCommissions(ctx context.Context, status *int, page, pageSize int) ([]model.AdminCommissionItem, int64, error) {
	return s.agent.ListCommissions(ctx, status, page, pageSize)
}

// ---- 订单 ----

func (s *AdminService) ListOrders(ctx context.Context, status *int, page, pageSize int) ([]model.AdminOrderItem, int64, error) {
	return s.order.ListOrders(ctx, status, page, pageSize)
}

func (s *AdminService) CloseOrder(ctx context.Context, adminID int64, orderNo, remark, ip string) error {
	return s.order.CloseOrder(ctx, adminID, orderNo, remark, ip)
}

func (s *AdminService) Refund(ctx context.Context, adminID int64, orderNo, remark, gatewayRefundID, ip string) error {
	return s.order.Refund(ctx, adminID, orderNo, remark, gatewayRefundID, ip)
}

// ---- 台账（crud 向量） ----

func (s *AdminService) ListAllPlans(ctx context.Context) ([]model.AdminPlanView, error) {
	return s.catalog.ListAllPlans(ctx)
}

func (s *AdminService) CreatePlan(ctx context.Context, req *model.AdminPlanReq) (*model.Plan, error) {
	return s.catalog.CreatePlan(ctx, req)
}

func (s *AdminService) UpdatePlan(ctx context.Context, id int64, req *model.AdminPlanReq) error {
	return s.catalog.UpdatePlan(ctx, id, req)
}

func (s *AdminService) DeletePlan(ctx context.Context, id int64) error {
	return s.catalog.DeletePlan(ctx, id)
}

func (s *AdminService) ListAllServers(ctx context.Context) ([]model.AdminServerView, error) {
	return s.catalog.ListAllServers(ctx)
}

func (s *AdminService) ListAllServerGroups(ctx context.Context) ([]model.ServerGroup, error) {
	return s.catalog.ListAllServerGroups(ctx)
}

func (s *AdminService) CreateServerGroup(ctx context.Context, req *model.AdminServerGroupReq) (*model.ServerGroup, error) {
	return s.catalog.CreateServerGroup(ctx, req)
}

func (s *AdminService) UpdateServerGroup(ctx context.Context, id int64, req *model.AdminServerGroupReq) error {
	return s.catalog.UpdateServerGroup(ctx, id, req)
}

func (s *AdminService) DeleteServerGroup(ctx context.Context, id int64) error {
	return s.catalog.DeleteServerGroup(ctx, id)
}

func (s *AdminService) CreateServer(ctx context.Context, req *model.AdminServerReq) (*model.Server, error) {
	return s.catalog.CreateServer(ctx, req)
}

func (s *AdminService) UpdateServer(ctx context.Context, id int64, req *model.AdminServerReq) error {
	return s.catalog.UpdateServer(ctx, id, req)
}

func (s *AdminService) DeleteServer(ctx context.Context, id int64) error {
	return s.catalog.DeleteServer(ctx, id)
}

func (s *AdminService) ListAllCoupons(ctx context.Context) ([]model.AdminCouponView, error) {
	return s.catalog.ListAllCoupons(ctx)
}

func (s *AdminService) CreateCoupon(ctx context.Context, req *model.AdminCouponReq) (*model.Coupon, error) {
	return s.catalog.CreateCoupon(ctx, req)
}

func (s *AdminService) UpdateCoupon(ctx context.Context, id int64, req *model.AdminCouponReq) error {
	return s.catalog.UpdateCoupon(ctx, id, req)
}

func (s *AdminService) DeleteCoupon(ctx context.Context, id int64) error {
	return s.catalog.DeleteCoupon(ctx, id)
}

// ---- 内容 ----

func (s *AdminService) SortNotices(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return s.content.SortNotices(ctx, adminID, items, ip)
}

func (s *AdminService) SortKnowledges(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return s.content.SortKnowledges(ctx, adminID, items, ip)
}

func (s *AdminService) ListKnowledgeCategories(ctx context.Context, language string) ([]model.AdminKnowledgeCategoryItem, error) {
	return s.content.ListKnowledgeCategories(ctx, language)
}

func (s *AdminService) CreateKnowledgeCategory(ctx context.Context, adminID int64, req *model.AdminKnowledgeCategoryReq, ip string) (*model.KnowledgeCategory, error) {
	return s.content.CreateKnowledgeCategory(ctx, adminID, req, ip)
}

func (s *AdminService) UpdateKnowledgeCategory(ctx context.Context, adminID, id int64, req *model.AdminKnowledgeCategoryUpdateReq, ip string) error {
	return s.content.UpdateKnowledgeCategory(ctx, adminID, id, req, ip)
}

func (s *AdminService) DeleteKnowledgeCategory(ctx context.Context, adminID, id int64, ip string) error {
	return s.content.DeleteKnowledgeCategory(ctx, adminID, id, ip)
}

func (s *AdminService) CreateNotice(ctx context.Context, req *model.AdminNoticeReq) (*model.Notice, error) {
	return s.content.CreateNotice(ctx, req)
}

func (s *AdminService) UpdateNotice(ctx context.Context, id int64, req *model.AdminNoticeReq) error {
	return s.content.UpdateNotice(ctx, id, req)
}

func (s *AdminService) DeleteNotice(ctx context.Context, id int64) error {
	return s.content.DeleteNotice(ctx, id)
}

func (s *AdminService) CreateKnowledge(ctx context.Context, req *model.AdminKnowledgeReq) (*model.Knowledge, error) {
	return s.content.CreateKnowledge(ctx, req)
}

func (s *AdminService) UpdateKnowledge(ctx context.Context, id int64, req *model.AdminKnowledgeReq) error {
	return s.content.UpdateKnowledge(ctx, id, req)
}

func (s *AdminService) DeleteKnowledge(ctx context.Context, id int64) error {
	return s.content.DeleteKnowledge(ctx, id)
}

func (s *AdminService) ListAllNotices(ctx context.Context) ([]model.AdminNoticeItem, error) {
	return s.content.ListAllNotices(ctx)
}

func (s *AdminService) ListAllKnowledges(ctx context.Context) ([]model.AdminKnowledgeItem, error) {
	return s.content.ListAllKnowledges(ctx)
}

func (s *AdminService) ListMailTemplates(ctx context.Context) ([]model.AdminMailTemplateItem, error) {
	return s.content.ListMailTemplates(ctx)
}

func (s *AdminService) SaveMailTemplate(ctx context.Context, adminID int64, name string, req *model.AdminMailTemplateReq, ip string) error {
	return s.content.SaveMailTemplate(ctx, adminID, name, req, ip)
}

func (s *AdminService) ResetMailTemplate(ctx context.Context, adminID int64, name, ip string) error {
	return s.content.ResetMailTemplate(ctx, adminID, name, ip)
}

func (s *AdminService) TestMailTemplate(ctx context.Context, adminID int64, name, toEmail, ip string) error {
	return s.content.TestMailTemplate(ctx, adminID, name, toEmail, ip)
}

func (s *AdminService) ListSubscriptionTemplates(ctx context.Context) ([]model.AdminSubscriptionTemplateItem, error) {
	return s.content.ListSubscriptionTemplates(ctx)
}

func (s *AdminService) SaveSubscriptionTemplate(ctx context.Context, adminID int64, name, content, ip string) error {
	return s.content.SaveSubscriptionTemplate(ctx, adminID, name, content, ip)
}

func (s *AdminService) ResetSubscriptionTemplate(ctx context.Context, adminID int64, name, ip string) error {
	return s.content.ResetSubscriptionTemplate(ctx, adminID, name, ip)
}

func (s *AdminService) PreviewSubscriptionTemplate(ctx context.Context, name string) (string, error) {
	return s.content.PreviewSubscriptionTemplate(ctx, name)
}

// ---- 节点批量 ----

func (s *AdminService) BatchServers(ctx context.Context, adminID int64, req *model.AdminBatchServerReq, ip string) (*model.AdminBatchServerResp, error) {
	return s.node.BatchServers(ctx, adminID, req, ip)
}

func (s *AdminService) CopyServer(ctx context.Context, adminID, serverID int64, ip string) (*model.AdminServerView, error) {
	return s.node.CopyServer(ctx, adminID, serverID, ip)
}

func (s *AdminService) SortServers(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return s.node.SortServers(ctx, adminID, items, ip)
}

func (s *AdminService) ResetNodeKey(ctx context.Context, adminID, serverID int64, ip string) (string, error) {
	return s.node.ResetNodeKey(ctx, adminID, serverID, ip)
}

// ---- 统计 ----

func (s *AdminService) Overview(ctx context.Context) (*model.AdminOverviewResp, error) {
	return s.stats.Overview(ctx)
}

func (s *AdminService) StatOrders(ctx context.Context, days int) (*model.AdminStatOrdersResp, error) {
	return s.stats.StatOrders(ctx, days)
}

func (s *AdminService) StatUsers(ctx context.Context, days int) (*model.AdminStatUsersResp, error) {
	return s.stats.StatUsers(ctx, days)
}

func (s *AdminService) StatTraffic(ctx context.Context, days int) (*model.AdminStatTrafficResp, error) {
	return s.stats.StatTraffic(ctx, days)
}

// ---- 流量 ----

func (s *AdminService) ImportTraffic(ctx context.Context, adminID int64, req *model.TrafficImportReq, ip string) error {
	return s.traffic.ImportTraffic(ctx, adminID, req, ip)
}

func (s *AdminService) ResetTraffic(ctx context.Context, adminID int64, req *model.AdminTrafficResetReq, ip string) (*model.AdminTrafficResetResp, error) {
	return s.traffic.ResetTraffic(ctx, adminID, req, ip)
}

func (s *AdminService) ListTrafficResets(ctx context.Context, userID *int64, page, pageSize int) ([]model.AdminTrafficResetLogItem, int64, error) {
	return s.traffic.ListTrafficResets(ctx, userID, page, pageSize)
}

// ---- 工单与提现 ----

func (s *AdminService) ListTickets(ctx context.Context, page, pageSize int) ([]model.AdminTicketItem, int64, error) {
	return s.ticket.ListTickets(ctx, page, pageSize)
}

func (s *AdminService) GetTicketDetail(ctx context.Context, id int64) (*model.TicketDetailResp, error) {
	return s.ticket.GetTicketDetail(ctx, id)
}

func (s *AdminService) ReplyTicket(ctx context.Context, adminID, id int64, message string) error {
	return s.ticket.ReplyTicket(ctx, adminID, id, message)
}

func (s *AdminService) CloseTicket(ctx context.Context, id int64) error {
	return s.ticket.CloseTicket(ctx, id)
}

func (s *AdminService) ReviewWithdraw(ctx context.Context, adminID, ticketID int64, approve bool, remark, ip string) error {
	return s.withdraw.ReviewWithdraw(ctx, adminID, ticketID, approve, remark, ip)
}

// ---- 配置与版本 ----

func (s *AdminService) ListSettings(ctx context.Context) ([]model.AdminSettingsResp, error) {
	return s.system.ListSettings(ctx)
}

func (s *AdminService) SaveSetting(ctx context.Context, key, value string) error {
	return s.system.SaveSetting(ctx, key, value)
}

func (s *AdminService) VersionInfo(ctx context.Context) (*model.AdminVersionResp, error) {
	return s.version.VersionInfo(ctx)
}
