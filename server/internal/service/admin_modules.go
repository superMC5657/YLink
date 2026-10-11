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

// admin 深模块总装：按域拆分的 module 集合与小 interface 缝合点。
//
// 术语说明：下文用 module 指代按域内聚的实现单元，用 interface/seam 指代
// module 对外暴露的窄契约。每个 module 只持有本域所需的最小依赖；跨域复用
// 经由 seam 显式声明。ica-A facade（AdminService）已退役：装配唯一入口为
// NewAdminSeams，handler 与测试直取所需域，不再经转发层（ADR-0005）。
// 审计 target 摘要保持 batch:<count> 写法；流量模式 clear_usage/reset_quota
// 语义不变；节点上报走模式 A（X-Node-Key 鉴权），手工导入走模式 B。

// adminKernel 装配期共享的依赖收敛点，仅在构造 module 时使用，不对外暴露。
type adminKernel struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	cfg   *config.Config
	set   *SettingService
	ml    *mailer.Mailer
}

// adminAuditModule 审计域 module：写审计 + 目标可读化 + 批量摘要。
type adminAuditModule struct {
	db    *gorm.DB
	repos *repo.Repos
}

// adminUserModule 用户域 module：查询/封禁/余额/批量/邮件/订阅密钥。
type adminUserModule struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	cfg   *config.Config
	ml    *mailer.Mailer
	audit *adminAuditModule
}

// adminAgentModule 代理与佣金域 module：申请审批 + 佣金列表。
type adminAgentModule struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	audit *adminAuditModule
}

// adminOrderModule 订单域 module：列表/关闭/退款 + 订阅收回。
type adminOrderModule struct {
	db    *gorm.DB
	repos *repo.Repos
	audit *adminAuditModule
}

// adminCatalogModule 台账域 module（crud 向量）：套餐/节点基础/分组/优惠券。
type adminCatalogModule struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
}

// adminContentModule 内容域 module：公告/知识/分类/排序 + 模板。
type adminContentModule struct {
	db    *gorm.DB
	repos *repo.Repos
	cfg   *config.Config
	ml    *mailer.Mailer
	audit *adminAuditModule
}

// adminNodeBatchModule 节点批量域 module：批量/复制/排序/密钥轮换。
type adminNodeBatchModule struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	audit *adminAuditModule
}

// adminStatsModule 统计域 module：总览 + F04 三张报表。
type adminStatsModule struct {
	db    *gorm.DB
	repos *repo.Repos
}

// adminTrafficModule 流量域 module：模式 B 导入 + 重置 + 重置记录。
type adminTrafficModule struct {
	db    *gorm.DB
	repos *repo.Repos
	audit *adminAuditModule
}

// adminTicketModule 工单域 module：列表/详情/回复/关闭。
// 提现附带信息经由 withdraw 缝合读取，收敛 ticket-withdraw 散落。
type adminTicketModule struct {
	db       *gorm.DB
	repos    *repo.Repos
	withdraw *adminWithdrawModule
}

// adminWithdrawModule 提现域 module：审核打款/拒绝退回 + 工单附带信息。
// 本 module 为 ticket-withdraw 唯一的写 seam 与读 seam。
type adminWithdrawModule struct {
	db    *gorm.DB
	repos *repo.Repos
	audit *adminAuditModule
}

// adminSystemModule 配置域 module：站点配置读写。
type adminSystemModule struct {
	db    *gorm.DB
	repos *repo.Repos
	set   *SettingService
}

// adminVersionModule 版本域 module：当前版本 + 远端清单。
type adminVersionModule struct {
	rdb *redis.Client
	cfg *config.Config
}

// newAdminKernel 收敛装配入参为 kernel。
func newAdminKernel(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, set *SettingService, ml *mailer.Mailer) *adminKernel {
	return &adminKernel{db: db, rdb: rdb, repos: repos, cfg: cfg, set: set, ml: ml}
}

// AdminSeams 管理端域 seam 总装：各域 module 的唯一装配点。
// handler 按需取用对应域；测试同样直取所需域（转发层已随 facade 退役）。
type AdminSeams struct {
	Audit    AdminAuditSeam
	User     AdminUserSeam
	Agent    AdminAgentSeam
	Order    AdminOrderSeam
	Catalog  AdminCatalogSeam
	Content  AdminContentSeam
	Node     AdminNodeBatchSeam
	Stats    AdminStatsSeam
	Traffic  AdminTrafficSeam
	Ticket   AdminTicketSeam
	Withdraw AdminWithdrawSeam
	System   AdminSystemSeam
	Version  AdminVersionSeam
}

// NewAdminSeams 装配全部域 module（签名与退役的 NewAdminService 一致，调用方平移）。
func NewAdminSeams(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, set *SettingService, ml *mailer.Mailer) *AdminSeams {
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
	return &AdminSeams{
		Audit: audit, User: user, Agent: agent, Order: order,
		Catalog: catalog, Content: content, Node: node, Stats: stats,
		Traffic: traffic, Ticket: ticket, Withdraw: withdraw,
		System: system, Version: version,
	}
}

// ---- 对外暴露的窄 interface（handler 仅依赖所需 seam） ----

// AdminAuditSeam 审计读 seam。
type AdminAuditSeam interface {
	ListAuditLogs(ctx context.Context, f AuditLogFilter, page, pageSize int) ([]model.AdminAuditLogItem, int64, []string, error)
}

// AdminUserSeam 用户域 seam。
type AdminUserSeam interface {
	ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]model.AdminUserItem, int64, error)
	UpdateUser(ctx context.Context, adminID, userID int64, req *model.AdminUpdateUserReq, ip string) error
	AdjustBalance(ctx context.Context, adminID, userID int64, amountFen int64, remark, ip string) error
	ExportUsers(ctx context.Context, keyword string, fn func(rows [][]string) error) error
	BatchUsers(ctx context.Context, adminID int64, req *model.AdminBatchUserReq, ip string) (*model.AdminBatchUserResp, error)
	SendMail(ctx context.Context, adminID int64, req *model.AdminSendMailReq, ip string) (*model.AdminSendMailResp, error)
	ResetUserSubToken(ctx context.Context, adminID, userID int64, ip string) (string, error)
}

// AdminAgentSeam 代理与佣金域 seam。
type AdminAgentSeam interface {
	ListAgentApplies(ctx context.Context, status int, page, pageSize int) ([]model.AdminAgentApplyItem, int64, error)
	ReviewAgentApply(ctx context.Context, adminID, applyID int64, approve bool, remark, ip string) error
	ListCommissions(ctx context.Context, status *int, page, pageSize int) ([]model.AdminCommissionItem, int64, error)
}

// AdminOrderSeam 订单域 seam。
type AdminOrderSeam interface {
	ListOrders(ctx context.Context, status *int, page, pageSize int) ([]model.AdminOrderItem, int64, error)
	CloseOrder(ctx context.Context, adminID int64, orderNo, remark, ip string) error
	Refund(ctx context.Context, adminID int64, orderNo, remark, gatewayRefundID, ip string) error
}

// AdminCatalogSeam 台账域 seam（crud 向量）。
type AdminCatalogSeam interface {
	ListAllPlans(ctx context.Context) ([]model.AdminPlanView, error)
	CreatePlan(ctx context.Context, req *model.AdminPlanReq) (*model.Plan, error)
	UpdatePlan(ctx context.Context, id int64, req *model.AdminPlanReq) error
	DeletePlan(ctx context.Context, id int64) error
	ListAllServers(ctx context.Context) ([]model.AdminServerView, error)
	ListAllServerGroups(ctx context.Context) ([]model.ServerGroup, error)
	CreateServerGroup(ctx context.Context, req *model.AdminServerGroupReq) (*model.ServerGroup, error)
	UpdateServerGroup(ctx context.Context, id int64, req *model.AdminServerGroupReq) error
	DeleteServerGroup(ctx context.Context, id int64) error
	CreateServer(ctx context.Context, req *model.AdminServerReq) (*model.Server, error)
	UpdateServer(ctx context.Context, id int64, req *model.AdminServerReq) error
	DeleteServer(ctx context.Context, id int64) error
	ListAllCoupons(ctx context.Context) ([]model.AdminCouponView, error)
	CreateCoupon(ctx context.Context, req *model.AdminCouponReq) (*model.Coupon, error)
	UpdateCoupon(ctx context.Context, id int64, req *model.AdminCouponReq) error
	DeleteCoupon(ctx context.Context, id int64) error
}

// AdminContentSeam 内容域 seam。
type AdminContentSeam interface {
	SortNotices(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error
	SortKnowledges(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error
	ListKnowledgeCategories(ctx context.Context, language string) ([]model.AdminKnowledgeCategoryItem, error)
	CreateKnowledgeCategory(ctx context.Context, adminID int64, req *model.AdminKnowledgeCategoryReq, ip string) (*model.KnowledgeCategory, error)
	UpdateKnowledgeCategory(ctx context.Context, adminID, id int64, req *model.AdminKnowledgeCategoryUpdateReq, ip string) error
	DeleteKnowledgeCategory(ctx context.Context, adminID, id int64, ip string) error
	CreateNotice(ctx context.Context, req *model.AdminNoticeReq) (*model.Notice, error)
	UpdateNotice(ctx context.Context, id int64, req *model.AdminNoticeReq) error
	DeleteNotice(ctx context.Context, id int64) error
	ListAllNotices(ctx context.Context) ([]model.AdminNoticeItem, error)
	CreateKnowledge(ctx context.Context, req *model.AdminKnowledgeReq) (*model.Knowledge, error)
	UpdateKnowledge(ctx context.Context, id int64, req *model.AdminKnowledgeReq) error
	DeleteKnowledge(ctx context.Context, id int64) error
	ListAllKnowledges(ctx context.Context) ([]model.AdminKnowledgeItem, error)
	ListMailTemplates(ctx context.Context) ([]model.AdminMailTemplateItem, error)
	SaveMailTemplate(ctx context.Context, adminID int64, name string, req *model.AdminMailTemplateReq, ip string) error
	ResetMailTemplate(ctx context.Context, adminID int64, name, ip string) error
	TestMailTemplate(ctx context.Context, adminID int64, name, toEmail, ip string) error
	ListSubscriptionTemplates(ctx context.Context) ([]model.AdminSubscriptionTemplateItem, error)
	SaveSubscriptionTemplate(ctx context.Context, adminID int64, name, content, ip string) error
	ResetSubscriptionTemplate(ctx context.Context, adminID int64, name, ip string) error
	PreviewSubscriptionTemplate(ctx context.Context, name string) (string, error)
}

// AdminNodeBatchSeam 节点批量域 seam。
type AdminNodeBatchSeam interface {
	BatchServers(ctx context.Context, adminID int64, req *model.AdminBatchServerReq, ip string) (*model.AdminBatchServerResp, error)
	CopyServer(ctx context.Context, adminID, serverID int64, ip string) (*model.AdminServerView, error)
	SortServers(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error
	ResetNodeKey(ctx context.Context, adminID, serverID int64, ip string) (string, error)
}

// AdminStatsSeam 统计域 seam。
type AdminStatsSeam interface {
	Overview(ctx context.Context) (*model.AdminOverviewResp, error)
	StatOrders(ctx context.Context, days int) (*model.AdminStatOrdersResp, error)
	StatUsers(ctx context.Context, days int) (*model.AdminStatUsersResp, error)
	StatTraffic(ctx context.Context, days int) (*model.AdminStatTrafficResp, error)
}

// AdminTrafficSeam 流量域 seam。
type AdminTrafficSeam interface {
	ImportTraffic(ctx context.Context, adminID int64, req *model.TrafficImportReq, ip string) error
	ResetTraffic(ctx context.Context, adminID int64, req *model.AdminTrafficResetReq, ip string) (*model.AdminTrafficResetResp, error)
	ListTrafficResets(ctx context.Context, userID *int64, page, pageSize int) ([]model.AdminTrafficResetLogItem, int64, error)
}

// AdminTicketSeam 工单域 seam。
type AdminTicketSeam interface {
	ListTickets(ctx context.Context, page, pageSize int) ([]model.AdminTicketItem, int64, error)
	GetTicketDetail(ctx context.Context, id int64) (*model.TicketDetailResp, error)
	ReplyTicket(ctx context.Context, adminID, id int64, message string) error
	CloseTicket(ctx context.Context, id int64) error
}

// AdminWithdrawSeam 提现域 seam（ticket-withdraw 唯一写入口）。
type AdminWithdrawSeam interface {
	ReviewWithdraw(ctx context.Context, adminID, ticketID int64, approve bool, remark, ip string) error
}

// AdminSystemSeam 配置域 seam。
type AdminSystemSeam interface {
	ListSettings(ctx context.Context) ([]model.AdminSettingsResp, error)
	SaveSetting(ctx context.Context, key, value string) error
}

// AdminVersionSeam 版本域 seam。
type AdminVersionSeam interface {
	VersionInfo(ctx context.Context) (*model.AdminVersionResp, error)
}
