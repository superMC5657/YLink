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
  status: TicketStatus
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

export interface TicketDetail {
  id: number
  subject: string
  level: TicketLevel
  status: TicketStatus
  created_at: string
  messages: TicketMessage[]
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
  status: TicketStatus
  last_reply_at: string | null
  created_at: string
}

export interface AdminTicketDetail {
  id: number
  subject: string
  level: TicketLevel
  status: TicketStatus
  created_at: string
  messages: TicketMessage[]
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
  title: string
  body: string
  language: string
  is_show: boolean
  sort: number
  updated_at: string
}

export interface AdminKnowledgeReq {
  category: string
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
  status: CommissionStatus
  confirmed_at: string | null
  created_at: string
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
