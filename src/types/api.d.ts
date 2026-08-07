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
