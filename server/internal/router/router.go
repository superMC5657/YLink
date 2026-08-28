// Package router 聚合全部路由，按模块分组并挂载中间件。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	_ "ylink-backend/docs"

	"ylink-backend/internal/config"
	"ylink-backend/internal/handler"
	"ylink-backend/internal/middleware"
	jwtpkg "ylink-backend/internal/pkg/jwt"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/repo"
	"ylink-backend/internal/service"
)

// Deps 为路由组装所需的全部依赖。
type Deps struct {
	DB     *gorm.DB
	Redis  *redis.Client
	JWT    *jwtpkg.Manager
	Mailer *mailer.Mailer
	Cfg    *config.Config
}

// app 聚合各层实例。
type app struct {
	repos *repo.Repos

	authSvc    *service.AuthService
	userSvc    *service.UserService
	contentSvc *service.ContentService
	serverSvc  *service.ServerService
	orderSvc   *service.OrderService
	subSvc     *service.SubscribeService
	inviteSvc  *service.InviteService
	ticketSvc  *service.TicketService
	adminSvc   *service.AdminService
	nodeSvc    *service.NodeService

	authH    *handler.Auth
	userH    *handler.User
	contentH *handler.Content
	serverH  *handler.Server
	orderH   *handler.Order
	subH     *handler.Subscribe
	inviteH  *handler.Invite
	ticketH  *handler.Ticket
	adminH   *handler.Admin
	nodeH    *handler.Node
}

// New 构建 gin 引擎（中间件链 + 分组路由）。
func New(d Deps) *gin.Engine {
	if d.Cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	a := newApp(d)

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.AccessLog())
	r.Use(middleware.CORS(d.Cfg.CORS.AllowOrigins))
	r.Use(middleware.GlobalLimiter(d.Redis))
	r.Use(middleware.Metrics())

	// 健康检查（不走 envelope）
	health := handler.NewHealth(d.DB, d.Redis)
	r.GET("/healthz", health.Liveness)
	r.GET("/readyz", health.Readiness)
	// Prometheus 指标（Grafana 看板数据源）
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	// Swagger（仅开发环境）
	if !d.Cfg.App.IsProduction() {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := r.Group("/api/v1")
	// F22 safe_mode：开启后非白名单域名（App.BaseURL host + security.safe_domains）一律 403。
	// 白名单含支付回调与订阅端点的访问域名；healthz/metrics 在 api group 之外不受影响。
	allowedHosts := append([]string{d.Cfg.App.BaseURL}, d.Cfg.Security.SafeDomains...)
	api.Use(middleware.SafeMode(d.Cfg.Security.SafeMode, allowedHosts))
	registerUser(api, d, a)    // 用户端 API
	registerAdmin(api, d, a)   // 管理端 API（role=admin，路径段可经 security.admin_path 定制）
	registerClient(api, d, a)  // 订阅下发（免登录，路径段可经 security.subscribe_path 定制）
	registerWebhook(api, d, a) // 支付回调（免登录）
	registerNode(api, d, a)    // 节点上报（X-Node-Key）

	return r
}

// newApp 组装 service 与 handler。
func newApp(d Deps) *app {
	repos := &repo.Repos{}
	settingSvc := service.NewSettingService(d.DB, d.Redis, repos)
	authSvc := service.NewAuthService(d.DB, d.Redis, repos, d.JWT, d.Mailer, d.Cfg)
	userSvc := service.NewUserService(d.DB, d.Redis, repos, authSvc)
	contentSvc := service.NewContentService(d.DB, d.Redis, repos, settingSvc)
	serverSvc := service.NewServerService(d.DB, d.Redis, repos)
	orderSvc := service.NewOrderService(d.DB, d.Redis, repos, settingSvc, d.Cfg, d.Mailer)
	subSvc := service.NewSubscribeService(d.DB, d.Redis, repos, d.Cfg)
	inviteSvc := service.NewInviteService(d.DB, d.Redis, repos, d.Cfg)
	ticketSvc := service.NewTicketService(d.DB, d.Redis, repos)
	adminSvc := service.NewAdminService(d.DB, d.Redis, repos, d.Cfg, settingSvc, d.Mailer)
	nodeSvc := service.NewNodeService(d.DB, d.Redis, repos)
	return &app{
		repos:      repos,
		authSvc:    authSvc,
		userSvc:    userSvc,
		contentSvc: contentSvc,
		serverSvc:  serverSvc,
		orderSvc:   orderSvc,
		subSvc:     subSvc,
		inviteSvc:  inviteSvc,
		ticketSvc:  ticketSvc,
		adminSvc:   adminSvc,
		nodeSvc:    nodeSvc,
		authH:      handler.NewAuth(authSvc),
		userH:      handler.NewUser(userSvc),
		contentH:   handler.NewContent(contentSvc),
		serverH:    handler.NewServer(serverSvc),
		orderH:     handler.NewOrder(orderSvc),
		subH:       handler.NewSubscribe(subSvc),
		inviteH:    handler.NewInvite(inviteSvc),
		ticketH:    handler.NewTicket(ticketSvc),
		adminH:     handler.NewAdmin(adminSvc),
		nodeH:      handler.NewNode(nodeSvc),
	}
}

// registerUser 用户端 API：公共部分（免鉴权）+ 需鉴权部分。
func registerUser(g *gin.RouterGroup, d Deps, a *app) {
	pub := g.Group("")
	// 严格限流：验证码、登录
	pub.POST("/captcha/email", middleware.StrictLimiter(d.Redis, "captcha"), a.authH.CaptchaEmail)
	pub.POST("/auth/register", middleware.StrictLimiter(d.Redis, "register"), a.authH.Register)
	pub.POST("/auth/login", middleware.StrictLimiter(d.Redis, "login"), a.authH.Login)
	pub.POST("/auth/refresh", a.authH.Refresh)
	pub.POST("/auth/forgot", middleware.StrictLimiter(d.Redis, "forgot"), a.authH.Forgot)
	pub.GET("/config", a.contentH.Config)
	pub.GET("/notices", a.contentH.Notices)
	pub.GET("/knowledges", a.contentH.Knowledges)
	pub.GET("/knowledges/:id", a.contentH.KnowledgeDetail)

	authed := g.Group("")
	authed.Use(middleware.Auth(d.JWT, d.Redis))
	authed.POST("/auth/logout", a.authH.Logout)
	authed.GET("/user/stat", a.userH.Stat)
	authed.GET("/user/profile", a.userH.Profile)
	authed.PUT("/user/profile", a.userH.UpdateProfile)
	authed.POST("/user/password/change", a.userH.ChangePassword)
	authed.GET("/user/sessions", a.userH.Sessions)              // F14 会话列表
	authed.DELETE("/user/sessions/:jti", a.userH.RevokeSession) // F14 踢下线
	authed.GET("/servers", a.serverH.List)
	// 交易
	authed.GET("/plans", a.orderH.Plans)
	authed.POST("/coupons/check", a.orderH.CouponCheck)
	authed.GET("/coupons/available", a.orderH.CouponsAvailable)
	authed.POST("/orders", middleware.Idempotency(d.Redis), a.orderH.Create)
	authed.GET("/orders", a.orderH.List)
	authed.GET("/orders/:order_no", a.orderH.Detail)
	authed.POST("/orders/:order_no/cancel", a.orderH.Cancel)
	authed.POST("/orders/:order_no/checkout", a.orderH.Checkout)
	// 订阅
	authed.GET("/user/subscribe", a.subH.UserSubscribe)
	authed.POST("/user/subscribe/reset", a.subH.Reset)
	authed.GET("/user/traffic-logs", a.subH.TrafficLogs)
	// 营销
	authed.GET("/invite/summary", a.inviteH.Summary)
	authed.GET("/invite/codes", a.inviteH.Codes)
	authed.POST("/invite/codes", a.inviteH.CreateCode)
	authed.DELETE("/invite/codes/:code", a.inviteH.DeleteCode)
	authed.GET("/invite/records", a.inviteH.Records)
	authed.POST("/invite/transfer", a.inviteH.Transfer)
	authed.POST("/invite/withdraw", a.inviteH.Withdraw)  // F02 佣金提现（仅代理商）
	authed.GET("/invite/withdraws", a.inviteH.Withdraws) // F02 提现记录
	authed.GET("/agent/status", a.inviteH.AgentStatus)
	authed.POST("/agent/apply", a.inviteH.ApplyAgent)
	// 工单
	authed.GET("/tickets", a.ticketH.List)
	authed.POST("/tickets", a.ticketH.Create)
	authed.GET("/tickets/:id", a.ticketH.Detail)
	authed.POST("/tickets/:id/reply", a.ticketH.Reply)
	authed.POST("/tickets/:id/close", a.ticketH.Close)
	authed.POST("/tickets/:id/reopen", a.ticketH.Reopen)
}

// registerAdmin 管理端 API（role=admin）。路径段由 security.admin_path 配置（F22，默认 admin）。
func registerAdmin(g *gin.RouterGroup, d Deps, a *app) {
	admin := g.Group("/" + d.Cfg.Security.AdminPathOrDefault())
	admin.Use(middleware.Auth(d.JWT, d.Redis), middleware.RequireRole(1))

	// 仪表盘
	admin.GET("/stat/overview", a.adminH.Overview)
	// 统计报表（F04 只读）
	admin.GET("/stat/orders", a.adminH.StatOrders)
	admin.GET("/stat/users", a.adminH.StatUsers)
	admin.GET("/stat/traffic", a.adminH.StatTraffic)
	// 用户
	admin.GET("/users", a.adminH.ListUsers)
	admin.GET("/users/export", a.adminH.ExportUsersCSV) // F05 CSV 导出（置于 :id 之前避免路由歧义）
	admin.PUT("/users/:id", a.adminH.UpdateUser)
	admin.POST("/users/:id/balance", a.adminH.AdjustBalance)
	admin.POST("/users/batch", a.adminH.BatchUsers)                      // F05 批量操作
	admin.POST("/users/mail", a.adminH.SendMail)                         // F05 发送邮件
	admin.POST("/users/:id/sub-token/reset", a.adminH.ResetUserSubToken) // F05 重置订阅密钥
	// 审计日志（F08 只读）
	admin.GET("/audit-logs", a.adminH.ListAuditLogs)
	// 套餐
	admin.GET("/plans", a.adminH.ListPlans)
	admin.POST("/plans", a.adminH.CreatePlan)
	admin.PUT("/plans/:id", a.adminH.UpdatePlan)
	admin.DELETE("/plans/:id", a.adminH.DeletePlan)
	// 节点
	admin.GET("/servers", a.adminH.ListServers)
	admin.POST("/servers", a.adminH.CreateServer)
	admin.POST("/servers/batch", a.adminH.BatchServers) // F09 批量操作（POST 无 :id 路由，无歧义）
	admin.POST("/servers/sort", a.adminH.SortServers)   // F09 排序
	admin.PUT("/servers/:id", a.adminH.UpdateServer)
	admin.DELETE("/servers/:id", a.adminH.DeleteServer)
	admin.POST("/servers/:id/node-key/reset", a.adminH.ResetNodeKey)
	admin.POST("/servers/:id/copy", a.adminH.CopyServer) // F09 复制节点
	admin.GET("/server-groups", a.adminH.ListServerGroups)
	admin.POST("/server-groups", a.adminH.CreateServerGroup)
	admin.PUT("/server-groups/:id", a.adminH.UpdateServerGroup)
	admin.DELETE("/server-groups/:id", a.adminH.DeleteServerGroup)
	// 订单
	admin.GET("/orders", a.adminH.ListOrders)
	admin.POST("/orders/:order_no/refund", a.adminH.Refund)
	admin.POST("/orders/:order_no/close", a.adminH.CloseOrder)
	// 优惠券
	admin.GET("/coupons", a.adminH.ListCoupons)
	admin.POST("/coupons", a.adminH.CreateCoupon)
	admin.PUT("/coupons/:id", a.adminH.UpdateCoupon)
	admin.DELETE("/coupons/:id", a.adminH.DeleteCoupon)
	// 内容
	admin.GET("/notices", a.adminH.ListNotices)
	admin.POST("/notices", a.adminH.CreateNotice)
	admin.POST("/notices/sort", a.adminH.SortNotices) // F15 公告排序（置于 :id 之前避免歧义）
	admin.PUT("/notices/:id", a.adminH.UpdateNotice)
	admin.DELETE("/notices/:id", a.adminH.DeleteNotice)
	admin.GET("/knowledges", a.adminH.ListKnowledges)
	admin.POST("/knowledges", a.adminH.CreateKnowledge)
	admin.POST("/knowledges/sort", a.adminH.SortKnowledges) // F15 知识库排序
	admin.PUT("/knowledges/:id", a.adminH.UpdateKnowledge)
	admin.DELETE("/knowledges/:id", a.adminH.DeleteKnowledge)
	admin.GET("/knowledge-categories", a.adminH.ListKnowledgeCategories) // F15 分类管理
	admin.POST("/knowledge-categories", a.adminH.CreateKnowledgeCategory)
	admin.PUT("/knowledge-categories/:id", a.adminH.UpdateKnowledgeCategory)
	admin.DELETE("/knowledge-categories/:id", a.adminH.DeleteKnowledgeCategory)
	// 工单
	admin.GET("/tickets", a.adminH.ListTickets)
	admin.GET("/tickets/:id", a.adminH.TicketDetail)
	admin.POST("/tickets/:id/reply", a.adminH.ReplyTicket)
	admin.POST("/tickets/:id/close", a.adminH.CloseTicket)
	admin.POST("/tickets/:id/withdraw/pay", a.adminH.WithdrawPay)       // F02 确认打款
	admin.POST("/tickets/:id/withdraw/reject", a.adminH.WithdrawReject) // F02 拒绝退回
	// 代理
	admin.GET("/agent/applies", a.adminH.ListAgentApplies)
	admin.POST("/agent/applies/:id/approve", a.adminH.ApproveAgent)
	admin.POST("/agent/applies/:id/reject", a.adminH.RejectAgent)
	// 佣金
	admin.GET("/commission-logs", a.adminH.ListCommissions)
	// 流量
	admin.POST("/traffic/import", a.adminH.ImportTraffic)
	admin.POST("/traffic/reset", a.adminH.ResetTraffic)      // F16 流量重置
	admin.GET("/traffic/resets", a.adminH.ListTrafficResets) // F16 重置记录
	// 设置
	admin.GET("/settings", a.adminH.ListSettings)
	admin.PUT("/settings", a.adminH.SaveSetting)
	// 邮件模板（F11）
	admin.GET("/mail-templates", a.adminH.ListMailTemplates)
	admin.PUT("/mail-templates/:name", a.adminH.SaveMailTemplate)
	admin.DELETE("/mail-templates/:name", a.adminH.ResetMailTemplate)
	admin.POST("/mail-templates/:name/test", a.adminH.TestMailTemplate)
	// 版本检查（F20）
	admin.GET("/version", a.adminH.Version)
}

// registerClient 订阅下发端点：代理客户端直连，免登录、任意来源。
// 路径段由 security.subscribe_path 配置（F22，默认 client），改路径需同步前端订阅链接缓存——
// 订阅 URL 均由服务端拼接下发，此处变更后旧 URL 立即 404，用户重新复制即可。
func registerClient(g *gin.RouterGroup, d Deps, a *app) {
	cli := g.Group("/" + d.Cfg.Security.SubscribePathOrDefault())
	cli.Use(middleware.CORSAny())
	cli.GET("/subscribe/:token", a.subH.ClientSubscribe)
}

// registerWebhook 支付异步通知：服务端间，免鉴权。
func registerWebhook(g *gin.RouterGroup, d Deps, a *app) {
	wh := g.Group("/payment")
	wh.POST("/notify/:method", a.orderH.Notify)
}

// registerNode 节点上报端点（模式 A）：X-Node-Key 密钥鉴权。
func registerNode(g *gin.RouterGroup, d Deps, a *app) {
	node := g.Group("/node")
	node.Use(middleware.NodeAuth(a.nodeSvc.ServerIDByKey, d.Redis))
	node.GET("/users", a.nodeH.Users)
	node.POST("/report", a.nodeH.Report)
}
