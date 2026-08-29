package model

import "time"

// ---- 管理端 · 仪表盘 ----

type AdminOverviewResp struct {
	UserCount       int64   `json:"user_count"`
	AgentCount      int64   `json:"agent_count"`
	OrderCount      int64   `json:"order_count"`
	CompletedOrders int64   `json:"completed_orders"`
	TotalRevenue    float64 `json:"total_revenue"` // 已完成订单实收合计（元）
	TodayRevenue    float64 `json:"today_revenue"` // 今日实收（元）
	TotalBalance    float64 `json:"total_balance"` // 全体用户余额合计（元）
	PlanCount       int64   `json:"plan_count"`
}

// AdminPlanView 管理端套餐视图:价格统一为元,展开 group_ids/is_show(用户端 Plan 隐藏)。
type AdminPlanView struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Content       string   `json:"content"`
	MonthPrice    *float64 `json:"month_price"`
	QuarterPrice  *float64 `json:"quarter_price"`
	HalfYearPrice *float64 `json:"half_year_price"`
	YearPrice     *float64 `json:"year_price"`
	OnetimePrice  *float64 `json:"onetime_price"`
	TrafficGB     int      `json:"traffic_gb"`
	SpeedLimit    *int     `json:"speed_limit"`
	DeviceLimit   *int     `json:"device_limit"`
	GroupIDs      []int64  `json:"group_ids"`
	IsShow        bool     `json:"is_show"`
	Sort          int      `json:"sort"`
}

// AdminServerView 管理端节点视图:展开 group_id/host/port/config(用户端隐藏)。
type AdminServerView struct {
	ID      int64    `json:"id"`
	GroupID int64    `json:"group_id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Host    string   `json:"host"`
	Port    int      `json:"port"`
	Config  string   `json:"config"`
	Rate    float64  `json:"rate"`
	Tags    []string `json:"tags"`
	Status  int      `json:"status"`
	IsShow  bool     `json:"is_show"`
	Sort    int      `json:"sort"`
	NodeKey string   `json:"node_key"` // 节点上报密钥(X-Node-Key)
}

// ---- 管理端 · 用户 ----

type AdminUserItem struct {
	ID                int64      `json:"id"`
	Email             string     `json:"email"`
	Role              int        `json:"role"`
	Balance           float64    `json:"balance"`
	CommissionBalance float64    `json:"commission_balance"`
	IsBanned          bool       `json:"is_banned"`
	InviteByID        *int64     `json:"invite_by_id"`
	PlanID            *int64     `json:"plan_id"`
	ExpiredAt         *time.Time `json:"expired_at"`
	TransferEnable    int64      `json:"transfer_enable"`
	U                 int64      `json:"u"`
	D                 int64      `json:"d"`
	CreatedAt         time.Time  `json:"created_at"`
}

type AdminUpdateUserReq struct {
	Role   *int  `json:"role" binding:"omitempty,oneof=0 1 2"`
	Banned *bool `json:"banned"`
}

type AdjustBalanceReq struct {
	Amount float64 `json:"amount" binding:"required"` // 元，可正可负
	Remark string  `json:"remark"`
}

// ---- 管理端 · 套餐 ----

type AdminPlanReq struct {
	Name          string   `json:"name" binding:"required"`
	Content       string   `json:"content"`
	MonthPrice    *float64 `json:"month_price"`
	QuarterPrice  *float64 `json:"quarter_price"`
	HalfYearPrice *float64 `json:"half_year_price"`
	YearPrice     *float64 `json:"year_price"`
	OnetimePrice  *float64 `json:"onetime_price"`
	TrafficGB     int      `json:"traffic_gb" binding:"required"`
	SpeedLimit    *int     `json:"speed_limit"`
	DeviceLimit   *int     `json:"device_limit"`
	GroupIDs      *[]int64 `json:"group_ids"` // nil=不更新;空数组=清空(该套餐不关联任何分组)
	IsShow        *bool    `json:"is_show"`
	Sort          int      `json:"sort"`
}

// ---- 管理端 · 节点 ----

type AdminServerGroupReq struct {
	Name string `json:"name" binding:"required"`
	Sort int    `json:"sort"`
}

type AdminServerReq struct {
	GroupID int64     `json:"group_id" binding:"required"`
	Name    string    `json:"name" binding:"required"`
	Type    string    `json:"type" binding:"required,oneof=shadowsocks vmess vless trojan hysteria2 tuic"`
	Host    string    `json:"host" binding:"required"`
	Port    int       `json:"port" binding:"required"`
	Config  string    `json:"config" binding:"required"` // 协议私有参数 JSON
	Rate    float64   `json:"rate"`
	Tags    *[]string `json:"tags"` // nil=不更新;空数组=清空
	Status  int       `json:"status" binding:"omitempty,oneof=1 2 3"`
	IsShow  *bool     `json:"is_show"`
	Sort    int       `json:"sort"`
}

// ---- 管理端 · 订单 ----

type AdminOrderItem struct {
	OrderNo          string     `json:"order_no"`
	UserID           int64      `json:"user_id"`
	UserEmail        string     `json:"user_email"`
	PlanName         string     `json:"plan_name"`
	Period           string     `json:"period"`
	Amount           float64    `json:"amount"`
	DiscountAmount   float64    `json:"discount_amount"`
	BalanceUsed      float64    `json:"balance_used"`
	PayAmount        float64    `json:"pay_amount"`
	CommissionAmount *float64   `json:"commission_amount"` // 该订单产生的佣金(元);nil=无佣金记录
	Status           int        `json:"status"`
	PayMethod        *string    `json:"pay_method"`
	PaidAt           *time.Time `json:"paid_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

type RefundReq struct {
	Remark string `json:"remark"`
}

// ---- 管理端 · 优惠券 ----

type AdminCouponReq struct {
	Code         string     `json:"code" binding:"required"`
	Type         int        `json:"type" binding:"required,oneof=1 2"`
	Value        float64    `json:"value" binding:"required"`
	MinSpend     float64    `json:"min_spend"`
	LimitPerUser int        `json:"limit_per_user"`
	TotalLimit   int        `json:"total_limit"`
	ValidPeriods []string   `json:"valid_periods"`
	PlanIDs      []int64    `json:"plan_ids"`
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	IsEnable     *bool      `json:"is_enable"`
}

// AdminCouponView 管理端优惠券视图:展开 type/value/min_spend(元)/used_count/valid_periods/plan_ids。
type AdminCouponView struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	Type         int        `json:"type"`
	Value        float64    `json:"value"`
	MinSpend     float64    `json:"min_spend"`
	LimitPerUser int        `json:"limit_per_user"`
	TotalLimit   int        `json:"total_limit"`
	UsedCount    int        `json:"used_count"`
	ValidPeriods []string   `json:"valid_periods"`
	PlanIDs      []int64    `json:"plan_ids"`
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	IsEnable     bool       `json:"is_enable"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ---- 管理端 · 内容 ----

type AdminNoticeItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsShow    bool      `json:"is_show"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminKnowledgeItem struct {
	ID        int64     `json:"id"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Language  string    `json:"language"`
	IsShow    bool      `json:"is_show"`
	Sort      int       `json:"sort"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminNoticeReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	IsShow  *bool  `json:"is_show"`
	Sort    int    `json:"sort"`
}

type AdminKnowledgeReq struct {
	Category   string `json:"category" binding:"required"`
	CategoryID *int64 `json:"category_id"` // F15 显式归类（空则按 category 名称归并/自建）
	Title      string `json:"title" binding:"required"`
	Body       string `json:"body" binding:"required"`
	Language   string `json:"language" binding:"omitempty,oneof=zh-CN en-US"`
	IsShow     *bool  `json:"is_show"`
	Sort       int    `json:"sort"`
}

// ---- 管理端 · 工单 ----

type AdminTicketItem struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	UserEmail   string     `json:"user_email"`
	Subject     string     `json:"subject"`
	Level       int        `json:"level"`
	Type        int        `json:"type"` // 0=普通 1=佣金提现(F02)
	Status      int        `json:"status"`
	ReopenCount int        `json:"reopen_count"`
	LastReplyAt *time.Time `json:"last_reply_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AdminReplyReq struct {
	Message string `json:"message" binding:"required"`
}

// ---- 管理端 · 代理申请 ----

type AdminAgentApplyItem struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	UserEmail    string    `json:"user_email"`
	ValidInvites int64     `json:"valid_invites"`
	Status       int       `json:"status"` // 0=待审核 1=通过 2=拒绝
	CreatedAt    time.Time `json:"created_at"`
}

type AdminApproveReq struct {
	Remark string `json:"remark"`
}

// ---- 管理端 · 佣金日志 ----

type AdminCommissionItem struct {
	ID           int64      `json:"id"`
	InviteUserID int64      `json:"invite_user_id"`
	InviteEmail  string     `json:"invite_email"`
	FromUserID   int64      `json:"from_user_id"`
	FromEmail    string     `json:"from_email"`
	OrderNo      string     `json:"order_no"`
	OrderAmount  float64    `json:"order_amount"`
	Rate         int        `json:"rate"`
	Amount       float64    `json:"amount"`
	Type         int        `json:"type"` // 0=订单佣金 1=提现流水(F02)
	Status       int        `json:"status"`
	ConfirmedAt  *time.Time `json:"confirmed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ---- 管理端 · 流量导入 ----

type TrafficImportItem struct {
	UserID int64  `json:"user_id" binding:"required"`
	Date   string `json:"date" binding:"required"` // YYYY-MM-DD
	U      int64  `json:"u"`
	D      int64  `json:"d"`
}

type TrafficImportReq struct {
	Items []TrafficImportItem `json:"items" binding:"required,min=1"`
}

// ---- 管理端 · 设置 ----

type AdminSettingsReq struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"` // JSON 字符串
}

type AdminSettingsResp struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ---- 管理端 · 审计日志（F08） ----

// AdminAuditLogItem 管理端审计日志条目（含操作人邮箱；detail 为 jsonb 原始字符串）。
// TargetKind/TargetDisplay 为展示增强：按 action 把 target 翻译成实体类型与可读名称
// （用户邮箱 / 节点名 / 分类名等），解析失败为 null，前端回退显示原始 target。
type AdminAuditLogItem struct {
	ID            int64     `json:"id"`
	AdminID       int64     `json:"admin_id"`
	AdminEmail    string    `json:"admin_email"`
	Action        string    `json:"action"`
	Target        *string   `json:"target"`
	TargetKind    *string   `json:"target_kind"`
	TargetDisplay *string   `json:"target_display"`
	Detail        *string   `json:"detail"`
	IP            *string   `json:"ip"`
	CreatedAt     time.Time `json:"created_at"`
}

// ---- 管理端 · 节点批量操作 / 复制 / 排序（F09） ----

// AdminBatchServerReq 节点批量操作：delete=删除；update=批量更新公共字段（至少一项，status/is_show/group_id/rate）。
type AdminBatchServerReq struct {
	Action  string   `json:"action" binding:"required,oneof=delete update"`
	IDs     []int64  `json:"ids" binding:"required,min=1,max=500"`
	Status  *int     `json:"status" binding:"omitempty,oneof=1 2 3"`
	IsShow  *bool    `json:"is_show"`
	GroupID *int64   `json:"group_id"`
	Rate    *float64 `json:"rate"`
}

// AdminBatchServerResp 批量节点操作结果汇总（复用 F05 的失败明细结构）。
type AdminBatchServerResp struct {
	Success int64                  `json:"success"`
	Failed  []AdminBatchFailedItem `json:"failed"`
}

// AdminSortServerReq 节点排序：items 按 sort 值更新（前端按展示顺序传 0..n）。
type AdminSortServerReq struct {
	Items []AdminSortItem `json:"items" binding:"required,min=1,max=500,dive"`
}

type AdminSortItem struct {
	ID   int64 `json:"id" binding:"required"`
	Sort int   `json:"sort"`
}

// ---- 管理端 · 流量重置（F16） ----

// AdminTrafficResetReq 按用户批量重置流量：clear_usage=清零用量；reset_quota=另按当前套餐额度重新给量。
type AdminTrafficResetReq struct {
	UserIDs []int64 `json:"user_ids" binding:"required,min=1,max=500"`
	Mode    string  `json:"mode" binding:"required,oneof=clear_usage reset_quota"`
}

// AdminTrafficResetResp 重置结果汇总。
type AdminTrafficResetResp struct {
	Success int64                  `json:"success"`
	Failed  []AdminBatchFailedItem `json:"failed"`
}

// AdminTrafficResetLogItem 重置记录条目（含用户邮箱联表）。
type AdminTrafficResetLogItem struct {
	ID                   int64     `json:"id"`
	UserID               int64     `json:"user_id"`
	UserEmail            string    `json:"user_email"`
	Mode                 string    `json:"mode"`
	BeforeU              int64     `json:"before_u"`
	BeforeD              int64     `json:"before_d"`
	BeforeTransferEnable int64     `json:"before_transfer_enable"`
	AfterTransferEnable  int64     `json:"after_transfer_enable"`
	CreatedAt            time.Time `json:"created_at"`
}

// ---- 管理端 · 统计报表（F04） ----

// AdminStatOrderPoint 订单日趋势点（date=YYYY-MM-DD）。
type AdminStatOrderPoint struct {
	Date            string  `json:"date"`
	OrderCount      int64   `json:"order_count"`      // 当日创建订单数
	CompletedCount  int64   `json:"completed_count"`  // 当日完成订单数
	Revenue         float64 `json:"revenue"`          // 当日实收现金部分（元，按 paid_at）
	Refunded        float64 `json:"refunded"`         // 当日退款现金部分（元，按 updated_at 近似）
	BalanceUsed     float64 `json:"balance_used"`     // 当日完成订单余额支付部分（元，按 paid_at）
	BalanceRefunded float64 `json:"balance_refunded"` // 当日退款订单余额部分（元，按 updated_at 近似）
}

type AdminStatOrdersResp struct {
	Days  int                   `json:"days"`
	Items []AdminStatOrderPoint `json:"items"` // 含无数据日补零，便于折线图
}

type AdminStatUserPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// AdminStatPlanSlice 套餐分布切片（当前生效订阅按套餐聚合）。
type AdminStatPlanSlice struct {
	PlanID   int64  `json:"plan_id"`
	PlanName string `json:"plan_name"`
	Users    int64  `json:"users"`
}

type AdminStatUsersResp struct {
	Days             int                  `json:"days"`
	RegisterTrend    []AdminStatUserPoint `json:"register_trend"` // 含无数据日补零
	PlanDistribution []AdminStatPlanSlice `json:"plan_distribution"`
}

type AdminStatUserTraffic struct {
	UserID     int64  `json:"user_id"`
	Email      string `json:"email"`
	TotalBytes int64  `json:"total_bytes"` // 时间范围内 u+d 合计
}

type AdminStatNodeTraffic struct {
	ServerID int64  `json:"server_id"`
	Name     string `json:"name"`
	Bytes    int64  `json:"bytes"` // 上报累计值合计（未乘倍率，节点全周期）
}

type AdminStatTrafficResp struct {
	Days    int                    `json:"days"`
	UserTop []AdminStatUserTraffic `json:"user_top"` // 流量消耗 TopN
	NodeTop []AdminStatNodeTraffic `json:"node_top"` // 节点流量分布 TopN
}

// ---- 管理端 · 用户批量操作 / 邮件（F05） ----

// AdminBatchUserReq 批量用户操作：ban/unban/adjust_balance。
type AdminBatchUserReq struct {
	Action string   `json:"action" binding:"required,oneof=ban unban adjust_balance"`
	IDs    []int64  `json:"ids" binding:"required,min=1,max=500"`
	Amount *float64 `json:"amount"` // adjust_balance 必填（元，可正可负）
	Remark string   `json:"remark"`
}

// AdminBatchFailedItem 批量操作失败明细。
type AdminBatchFailedItem struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}

// AdminBatchUserResp 批量操作结果汇总。
type AdminBatchUserResp struct {
	Success int64                  `json:"success"`
	Failed  []AdminBatchFailedItem `json:"failed"`
}

// AdminSendMailReq 管理端向用户发送邮件。
type AdminSendMailReq struct {
	IDs     []int64 `json:"ids" binding:"required,min=1,max=100"`
	Subject string  `json:"subject" binding:"required,max=200"`
	Body    string  `json:"body" binding:"required,max=10000"`
}

// AdminSendMailResp 发送结果汇总。
type AdminSendMailResp struct {
	Sent   int64                  `json:"sent"`
	Failed []AdminBatchFailedItem `json:"failed"`
}

// ---- 管理端 · 内容排序与知识库分类（F15） ----

// AdminSortReq 内容排序（公告/知识库共用）：items 按 sort 值更新（前端按展示顺序传 0..n）。
type AdminSortReq struct {
	Items []AdminSortItem `json:"items" binding:"required,min=1,max=500,dive"`
}

// AdminKnowledgeCategoryItem 知识库分类（含文档计数）。
type AdminKnowledgeCategoryItem struct {
	ID             int64  `json:"id"`
	Language       string `json:"language"`
	Name           string `json:"name"`
	Sort           int    `json:"sort"`
	KnowledgeCount int64  `json:"knowledge_count"`
}

// AdminKnowledgeCategoryReq 新建分类。
type AdminKnowledgeCategoryReq struct {
	Language string `json:"language" binding:"required,max=10"`
	Name     string `json:"name" binding:"required,max=64"`
	Sort     *int   `json:"sort"`
}

// AdminKnowledgeCategoryUpdateReq 更新分类（改名级联同步知识文档展示分类）。
type AdminKnowledgeCategoryUpdateReq struct {
	Name string `json:"name" binding:"required,max=64"`
	Sort *int   `json:"sort"`
}

// ---- 管理端 · 邮件模板（F11） ----

// AdminMailTemplateItem 邮件模板视图：custom=已自定义，否则为内置默认文案。
type AdminMailTemplateItem struct {
	Name         string     `json:"name"`
	Subject      string     `json:"subject"`
	Body         string     `json:"body"`
	IsCustom     bool       `json:"is_custom"`
	Placeholders []string   `json:"placeholders"`
	Remark       string     `json:"remark"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// AdminMailTemplateReq 保存邮件模板（Go template 语法，保存前校验可解析）。
type AdminMailTemplateReq struct {
	Subject string `json:"subject" binding:"required,max=255"`
	Body    string `json:"body" binding:"required"`
}

// AdminMailTemplateTestReq 测试发送（走真实 SMTP）。
type AdminMailTemplateTestReq struct {
	ToEmail string `json:"to_email" binding:"required,email"`
}

// ---- 管理端 · 版本检查（F20） ----

// AdminVersionResp 版本信息：latest 为空表示未配置更新源或拉取失败。
type AdminVersionResp struct {
	Version   string  `json:"version"`
	Latest    *string `json:"latest"`
	HasUpdate *bool   `json:"has_update"`
	Notes     *string `json:"notes"`
}

// ---- 管理端 · 订阅模板（F10） ----

// AdminSubscriptionTemplateItem 订阅模板视图：custom=已自定义，否则为内置生成器模板。
type AdminSubscriptionTemplateItem struct {
	Name      string     `json:"name"`
	Content   string     `json:"content"`
	IsCustom  bool       `json:"is_custom"`
	Variables []string   `json:"variables"`
	Remark    string     `json:"remark"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// AdminSubscriptionTemplateReq 保存订阅模板（Go template 语法，保存前用示例数据渲染校验）。
type AdminSubscriptionTemplateReq struct {
	Content string `json:"content" binding:"required"`
}

// AdminSubscriptionTemplatePreviewResp 预览渲染结果（示例数据）。
type AdminSubscriptionTemplatePreviewResp struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ---- 用户端 · Telegram 绑定（F12） ----

// TelegramBindCodeResp 绑定验证码：用户在 10 分钟内向 bot 发送 /bind <code>。
type TelegramBindCodeResp struct {
	Code        string `json:"code"`
	BotUsername string `json:"bot_username"`
	TTLMinutes  int    `json:"ttl_minutes"`
}

// ---- 管理端 · Telegram（F12） ----

// AdminTelegramWebhookSetupResp webhook 注册结果。
type AdminTelegramWebhookSetupResp struct {
	WebhookURL string `json:"webhook_url"`
	Message    string `json:"message"`
}

// ---- 管理端 · 用户会话（F14，挂 dto_admin 便于聚合引用） ----
