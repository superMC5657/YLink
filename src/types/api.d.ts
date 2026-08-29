/**
 * 接口契约类型 —— 与 docs/api/README.md 一一对应(前后端唯一事实来源)。
 * 契约变更流程:先改文档 → 评审 → 再改本文件。
 *
 * 单位约定:金额 number 元(两位小数);流量 integer 字节;时间 RFC3339。
 */

// ---------- 通用 ----------

export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data: T | null
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface ApiError {
  code: number
  message: string
  status: number
}

/** 通用分页请求参数 */
export interface PageQuery {
  page?: number
  page_size?: number
}

// ---------- 站点与验证码 ----------

export interface AppDownloadLinks {
  windows?: string
  macos?: string
  android?: string
}

export interface TelegramLinks {
  group_url?: string
  bot_url?: string
}

export interface AgentPolicy {
  required_valid_invites: number
  commission_rate: number
  benefits: string[]
  notes: string[]
}

export interface PaymentMethod {
  code: string
  name: string
  icon: string
  enabled: boolean
}

export interface SiteConfig {
  site_name: string
  site_logo: string
  site_description: string
  /** F19 品牌主色(Hex,空=默认) */
  primary_color: string
  /** F19 背景图 URL(空=默认) */
  background_url: string
  register_enabled: boolean
  invite_code_required: boolean
  app_downloads: AppDownloadLinks
  telegram: TelegramLinks
  customer_service_url: string
  free_traffic_tips: string
  agent_policy: AgentPolicy
  payment_methods: PaymentMethod[]
  languages: string[]
}

export interface CaptchaSendReq {
  email: string
  type: 'register' | 'forgot'
}

export interface CaptchaSendResp {
  expire_in: number
  resend_after: number
}

// ---------- 认证 ----------

export interface UserBrief {
  id: number
  email: string
  role: number
}

export interface AuthResp {
  access_token: string
  refresh_token: string
  token_type: 'Bearer'
  expires_in: number
  user: UserBrief
}

export interface LoginReq {
  email: string
  password: string
}

export interface RegisterReq {
  email: string
  password: string
  email_code: string
  invite_code?: string
}

export interface ForgotReq {
  email: string
  email_code: string
  password: string
}

export interface RefreshReq {
  refresh_token: string
}

// ---------- 用户 ----------

export interface UserStat {
  email: string
  balance: number
  commission_balance: number
  pending_order_count: number
  open_ticket_count: number
  invited_count: number
  is_agent: boolean
}

export interface ProfileUpdateReq {
  remind_expire: boolean
  remind_traffic: boolean
}

export interface ProfileResp {
  remind_expire: boolean
  remind_traffic: boolean
  telegram_bound?: boolean
}

// ---------- 用户端 · Telegram 绑定（F12） ----------

export interface TelegramBindCodeResp {
  code: string
  bot_username: string
  ttl_minutes: number
}

export interface ChangePasswordReq {
  old_password: string
  new_password: string
}

export interface SubscribePlanBrief {
  id: number
  name: string
}

export interface SubscribeInfo {
  has_subscription: boolean
  plan: SubscribePlanBrief | null
  expired_at: string | null
  is_expired: boolean
  expired_days: number
  transfer_enable: number
  u: number
  d: number
  remaining: number
  used_percent: number
  speed_limit: number | null
  device_limit: number
  subscribe_url: string
}

export interface TrafficLog {
  date: string
  u: number
  d: number
  total: number
}

// ---------- 公告 ----------

export interface Notice {
  id: number
  title: string
  content: string
  created_at: string
}

// ---------- 知识库 ----------

export interface KnowledgeItem {
  id: number
  title: string
  updated_at: string
}

export interface KnowledgeGroup {
  category: string
  items: KnowledgeItem[]
}

export interface KnowledgeListResp {
  groups: KnowledgeGroup[]
}

export interface KnowledgeDetail {
  id: number
  category: string
  title: string
  body: string
  language: string
  updated_at: string
}

// ---------- 套餐 ----------

export type PlanPeriod = 'month' | 'quarter' | 'half_year' | 'year' | 'onetime'

export interface Plan {
  id: number
  name: string
  prices: Partial<Record<PlanPeriod, number>>
  traffic_gb: number
  speed_limit: number | null
  device_limit: number
  content: string
  sort: number
}

export interface PlanListResp {
  list: Plan[]
}

// ---------- 优惠券 ----------

export interface CouponCheckReq {
  code: string
  plan_id: number
  period: PlanPeriod
}

export interface CouponCheckResp {
  valid: boolean
  discount_amount: number
  pay_amount: number
}

/** 用户可见的可用优惠券（GET /coupons/available，契约 §9） */
export interface CouponItem {
  code: string
  /** 1=固定金额 2=百分比 */
  type: 1 | 2
  /** type=1 为元；type=2 为百分比数值（如 10 表示 10%） */
  value: number
  /** 元；0 表示无门槛 */
  min_spend: number
  /** 空=不限周期 */
  valid_periods: PlanPeriod[]
  /** 空=全部套餐 */
  plan_ids: number[]
  started_at: string | null
  ended_at: string | null
}

export interface CouponAvailableResp {
  list: CouponItem[]
}

// ---------- 订单与支付 ----------

export type OrderStatus = 0 | 1 | 2 | 3

export interface CreateOrderReq {
  plan_id: number
  period: PlanPeriod
  coupon_code?: string | null
}

export interface Order {
  order_no: string
  plan_name: string
  period: PlanPeriod
  amount: number
  discount_amount: number
  balance_used: number
  pay_amount: number
  coupon_code: string | null
  status: OrderStatus
  pay_method: string | null
  paid_at: string | null
  created_at: string
}

export interface CheckoutReq {
  method: string
}

export interface CheckoutResp {
  type: 'url' | 'qrcode' | 'paid'
  content: string | null
  expire_in: number
}

// ---------- 邀请与佣金 ----------

export interface InviteSummary {
  commission_balance: number
  commission_rate: number
  registered_count: number
  total_commission: number
  pending_commission: number
}

export interface InviteCode {
  code: string
  used_count: number
  created_at: string
}

export interface InviteCodeListResp {
  list: InviteCode[]
  limit: number
  register_url_prefix: string
}

export interface CommissionRecord {
  order_no: string
  amount: number
  rate: number
  status: number
  confirmed_at: string | null
  created_at: string
}

export interface TransferReq {
  amount: number
}

export interface TransferResp {
  balance: number
  commission_balance: number
}

// ---------- 代理商 ----------

export type ApplyStatus = 'none' | 'pending' | 'approved' | 'rejected'

export interface AgentCondition {
  met: boolean
  text: string
}

export interface AgentStatus {
  is_agent: boolean
  apply_status: ApplyStatus
  qualified: boolean
  valid_invites: number
  required_valid_invites: number
  conditions: AgentCondition[]
}

// ---------- 工单 ----------

export type TicketStatus = 0 | 1 | 2
export type TicketLevel = 0 | 1 | 2

export interface Ticket {
  id: number
  subject: string
  level: TicketLevel
  /** 0=普通 1=佣金提现(F02) */
  type: number
  status: TicketStatus
  reopen_count: number
  last_reply_at: string | null
  created_at: string
}

export interface CreateTicketReq {
  subject: string
  level: TicketLevel
  message: string
}

export interface TicketMessage {
  id: number
  sender_type: 0 | 1
  message: string
  created_at: string
}

export interface TicketWithdrawInfo {
  id: number
  user_id: number
  amount: number
  method: string
  account: string
  /** 0=处理中 1=已发放 2=已退回 */
  status: number
  review_remark: string | null
  reviewed_at: string | null
  created_at: string
}

export interface TicketDetail {
  id: number
  subject: string
  level: TicketLevel
  /** 0=普通 1=佣金提现(F02) */
  type: number
  status: TicketStatus
  reopen_count: number
  created_at: string
  messages: TicketMessage[]
  withdraw?: TicketWithdrawInfo | null
}

// ---------- 佣金提现（F02，仅代理商） ----------

export interface WithdrawCreateReq {
  amount: number
  method: string
  account: string
}

export interface WithdrawItem {
  id: number
  amount: number
  method: string
  account: string
  /** 0=处理中 1=已发放 2=已退回 */
  status: number
  review_remark: string | null
  ticket_id: number
  reviewed_at: string | null
  created_at: string
}

// ---------- 会话管理（F14） ----------

export interface UserSessionItem {
  jti: string
  current: boolean
  ip: string
  user_agent: string
  /** 历史会话可能无元数据（升级前白名单值）→ null，前端显示 -- */
  created_at: string | null
}

export interface TicketReplyReq {
  message: string
}

// ---------- 节点 ----------

export type ServerStatus = 1 | 2 | 3

export interface ServerNode {
  id: number
  name: string
  type: string
  rate: number
  status: ServerStatus
  tags: string[]
}

export interface ServerGroup {
  group: string
  servers: ServerNode[]
}

export interface ServerListResp {
  groups: ServerGroup[]
}

// ---------- 管理端 ----------

export type AdminRole = 0 | 1 | 2

export interface AdminOverviewResp {
  user_count: number
  agent_count: number
  order_count: number
  completed_orders: number
  total_revenue: number
  today_revenue: number
  total_balance: number
  plan_count: number
}

export interface AdminUserItem {
  id: number
  email: string
  role: AdminRole
  balance: number
  commission_balance: number
  is_banned: boolean
  invite_by_id: number | null
  plan_id: number | null
  expired_at: string | null
  transfer_enable: number
  u: number
  d: number
  created_at: string
}

export interface AdminUpdateUserReq {
  role?: AdminRole
  banned?: boolean
}

export interface AdjustBalanceReq {
  amount: number
  remark?: string
}

/** 管理端套餐(与用户端 Plan 字段不同:直接暴露各周期价格) */
export interface AdminPlanItem {
  id: number
  name: string
  content: string
  month_price: number | null
  quarter_price: number | null
  half_year_price: number | null
  year_price: number | null
  onetime_price: number | null
  traffic_gb: number
  speed_limit: number | null
  device_limit: number | null
  group_ids: number[] | null
  is_show: boolean
  sort: number
}

export interface AdminPlanReq {
  name: string
  content?: string
  month_price?: number | null
  quarter_price?: number | null
  half_year_price?: number | null
  year_price?: number | null
  onetime_price?: number | null
  traffic_gb: number
  speed_limit?: number | null
  device_limit?: number | null
  group_ids?: number[] | null
  is_show?: boolean
  sort?: number
}

export interface AdminServerGroupItem {
  id: number
  name: string
  sort: number
}

export interface AdminServerGroupReq {
  name: string
  sort?: number
}

export interface AdminServerItem {
  id: number
  group_id: number
  name: string
  type: string
  host: string
  port: number
  config: string
  rate: number
  tags: string[] | null
  status: ServerStatus
  is_show: boolean
  sort: number
  node_key: string
}

export interface AdminServerReq {
  group_id: number
  name: string
  type: string
  host: string
  port: number
  config: string
  rate?: number
  tags?: string[] | null
  status?: ServerStatus
  is_show?: boolean
  sort?: number
}

export interface AdminOrderItem {
  order_no: string
  user_id: number
  user_email: string
  plan_name: string
  period: string
  amount: number
  discount_amount: number
  balance_used: number
  pay_amount: number
  commission_amount: number | null
  status: OrderStatus
  pay_method: string | null
  paid_at: string | null
  created_at: string
}

export interface RefundReq {
  remark?: string
}

export interface AdminTicketItem {
  id: number
  user_id: number
  user_email: string
  subject: string
  level: TicketLevel
  /** 0=普通 1=佣金提现(F02) */
  type: number
  status: TicketStatus
  reopen_count: number
  last_reply_at: string | null
  created_at: string
}

export interface AdminTicketDetail {
  id: number
  subject: string
  level: TicketLevel
  /** 0=普通 1=佣金提现(F02) */
  type: number
  status: TicketStatus
  created_at: string
  messages: TicketMessage[]
  withdraw?: TicketWithdrawInfo | null
}

export interface AdminReplyReq {
  message: string
}

// ---------- 管理端 · 二期模块(契约 docs/api/README.md §16.1) ----------

/** 管理端优惠券视图:展开 type/value(元)/min_spend(元)/used_count/valid_periods/plan_ids */
export interface AdminCouponItem {
  id: number
  code: string
  type: 1 | 2 // 1=固定金额 2=百分比
  value: number
  min_spend: number
  limit_per_user: number
  total_limit: number
  used_count: number
  valid_periods: string[]
  plan_ids: number[]
  started_at: string | null
  ended_at: string | null
  is_enable: boolean
  created_at: string
}

export interface AdminCouponReq {
  code: string
  type: 1 | 2
  value: number
  min_spend?: number
  limit_per_user?: number
  total_limit?: number
  valid_periods?: string[]
  plan_ids?: number[]
  started_at?: string | null
  ended_at?: string | null
  is_enable?: boolean
}

export interface AdminNoticeItem {
  id: number
  title: string
  content: string
  is_show: boolean
  sort: number
  created_at: string
}

export interface AdminNoticeReq {
  title: string
  content: string
  is_show?: boolean
  sort?: number
}

export interface AdminKnowledgeItem {
  id: number
  category: string
  /** F15 归属分类 ID(未归类为空) */
  category_id?: number | null
  title: string
  body: string
  language: string
  is_show: boolean
  sort: number
  updated_at: string
}

export interface AdminKnowledgeReq {
  category: string
  /** F15 显式归类(空则按 category 名称归并/自建) */
  category_id?: number | null
  title: string
  body: string
  language?: string
  is_show?: boolean
  sort?: number
}

export type AdminApplyStatus = 0 | 1 | 2 // 0=待审核 1=通过 2=拒绝

export interface AdminAgentApplyItem {
  id: number
  user_id: number
  user_email: string
  valid_invites: number
  status: AdminApplyStatus
  created_at: string
}

export interface AdminApproveReq {
  remark?: string
}

export type CommissionStatus = 0 | 1 | 2 // 0=确认中 1=已发放 2=已撤销

export interface AdminCommissionItem {
  id: number
  invite_user_id: number
  invite_email: string
  from_user_id: number
  from_email: string
  order_no: string
  order_amount: number
  rate: number
  amount: number
  /** 0=订单佣金 1=提现流水(F02) */
  type: number
  status: CommissionStatus
  confirmed_at: string | null
  created_at: string
}

/** 管理端审计日志条目(F08,detail 为 jsonb 原始字符串) */
export interface AdminAuditLogItem {
  id: number
  admin_id: number
  admin_email: string
  action: string
  target: string | null
  /** 目标实体类型(user/users/server/knowledge_category/order/mail_template),未收录动作/空 target 为 null */
  target_kind: string | null
  /** 目标可读名称(用户邮箱/节点名/分类名等),解析失败为 null,展示时回退 target */
  target_display: string | null
  detail: string | null
  ip: string | null
  created_at: string
}

/** 审计日志查询响应(含可选动作聚合,供筛选下拉) */
export interface AdminAuditLogsResp {
  list: AdminAuditLogItem[]
  total: number
  page: number
  page_size: number
  actions?: string[]
}

/** F05 批量用户操作:ban/unban/adjust_balance */
export type AdminBatchAction = 'ban' | 'unban' | 'adjust_balance'

export interface AdminBatchUserReq {
  action: AdminBatchAction
  ids: number[]
  /** adjust_balance 必填(元,可正可负) */
  amount?: number
  remark?: string
}

export interface AdminBatchFailedItem {
  id: number
  reason: string
}

export interface AdminBatchUserResp {
  success: number
  failed: AdminBatchFailedItem[]
}

export interface AdminSendMailReq {
  ids: number[]
  subject: string
  body: string
}

export interface AdminSendMailResp {
  sent: number
  failed: AdminBatchFailedItem[]
}

// ---------- 管理端 · 节点批量 / 复制 / 排序（F09） ----------

export type AdminBatchServerAction = 'delete' | 'update'

export interface AdminBatchServerReq {
  action: AdminBatchServerAction
  ids: number[]
  /** update 可选公共字段 */
  status?: ServerStatus
  is_show?: boolean
  group_id?: number
  rate?: number
}

export interface AdminBatchServerResp {
  success: number
  failed: AdminBatchFailedItem[]
}

export interface AdminSortItem {
  id: number
  sort: number
}

export interface AdminSortServerReq {
  items: AdminSortItem[]
}

// ---------- 管理端 · 流量重置（F16） ----------

export type TrafficResetMode = 'clear_usage' | 'reset_quota'

export interface AdminTrafficResetReq {
  user_ids: number[]
  mode: TrafficResetMode
}

export interface AdminTrafficResetResp {
  success: number
  failed: AdminBatchFailedItem[]
}

export interface AdminTrafficResetLogItem {
  id: number
  user_id: number
  user_email: string
  mode: TrafficResetMode
  before_u: number
  before_d: number
  before_transfer_enable: number
  after_transfer_enable: number
  created_at: string
}

// ---------- 管理端 · 统计报表（F04） ----------

export interface AdminStatOrderPoint {
  date: string
  order_count: number
  completed_count: number
  revenue: number
  refunded: number
  /** 当日完成订单的余额支付部分（元，与 revenue 按 paid_at 同口径） */
  balance_used: number
  /** 当日退款订单的余额部分（元，与 refunded 按 updated_at 同口径） */
  balance_refunded: number
}

export interface AdminStatOrdersResp {
  days: number
  items: AdminStatOrderPoint[]
}

export interface AdminStatUserPoint {
  date: string
  count: number
}

export interface AdminStatPlanSlice {
  plan_id: number
  plan_name: string
  users: number
}

export interface AdminStatUsersResp {
  days: number
  register_trend: AdminStatUserPoint[]
  plan_distribution: AdminStatPlanSlice[]
}

export interface AdminStatUserTraffic {
  user_id: number
  email: string
  total_bytes: number
}

export interface AdminStatNodeTraffic {
  server_id: number
  name: string
  bytes: number
}

export interface AdminStatTrafficResp {
  days: number
  user_top: AdminStatUserTraffic[]
  node_top: AdminStatNodeTraffic[]
}

export interface TrafficImportItem {
  user_id: number
  date: string // YYYY-MM-DD
  u: number
  d: number
}

export interface TrafficImportReq {
  items: TrafficImportItem[]
}

export interface AdminSettingsItem {
  key: string
  value: string // 配置项 JSON 字符串
}

export interface AdminSettingsReq {
  key: string
  value: string
}

// ---------- 管理端 · 内容排序与知识库分类（F15） ----------

/** 公告/知识库排序（items 按 sort 值更新） */
export interface AdminSortReq {
  items: AdminSortItem[]
}

export interface AdminKnowledgeCategoryItem {
  id: number
  language: string
  name: string
  sort: number
  knowledge_count: number
}

export interface AdminKnowledgeCategoryReq {
  language: string
  name: string
  sort?: number
}

export interface AdminKnowledgeCategoryUpdateReq {
  name: string
  sort?: number
}

// ---------- 管理端 · 邮件模板（F11） ----------

export interface AdminMailTemplateItem {
  name: string
  subject: string
  body: string
  is_custom: boolean
  placeholders: string[]
  remark: string
  updated_at: string | null
}

export interface AdminMailTemplateReq {
  subject: string
  body: string
}

export interface AdminMailTemplateTestReq {
  to_email: string
}

// ---------- 管理端 · 订阅模板（F10） ----------

export interface AdminSubscriptionTemplateItem {
  name: string
  content: string
  is_custom: boolean
  variables: string[]
  remark: string
  updated_at: string | null
}

export interface AdminSubscriptionTemplateReq {
  content: string
}

export interface AdminSubscriptionTemplatePreviewResp {
  name: string
  content: string
}

// ---------- 管理端 · Telegram（F12） ----------

export interface AdminTelegramWebhookSetupResp {
  webhook_url: string
  message: string
}

// ---------- 管理端 · 版本检查（F20） ----------

export interface AdminVersionResp {
  version: string
  latest: string | null
  has_update: boolean | null
  notes: string | null
}
