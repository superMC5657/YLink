/**
 * 管理端契约生成片段 —— DO NOT EDIT,改 Go DTO 后跑 pnpm contract:gen。
 * Owner:server/internal/model/dto_admin.go(形状) + docs/api/README.md §16(语义)。
 * 消费:新管理端类型直接引此文件;src/types/api.d.ts 存量接口由
 * scripts/check-admin-contract.mjs 逐字段审计(决策见 ADR-0002)。
 * source sha256:70b280957148
 */

export interface AdminOverviewResp {
  user_count: number
  agent_count: number
  order_count: number
  completed_orders: number
  total_revenue: number // 已完成订单实收合计（元）
  today_revenue: number // 今日实收（元）
  total_balance: number // 全体用户余额合计（元）
  plan_count: number
}

// AdminPlanView 管理端套餐视图:价格统一为元,展开 group_ids/is_show(用户端 Plan 隐藏)。
export interface AdminPlanView {
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

// NodeSystemStats 节点机器探针与负载数据(模式 A / F17)。
export interface NodeSystemStats {
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  net_speed_in: number // 瞬时下行带宽(B/s)
  net_speed_out: number // 瞬时上行带宽(B/s)
  connections: number // 主机级 TCP 表项近似值，含系统噪声，不代表代理在线用户，仅供负载参考
  uptime: number // 开机时长(秒)
}

// AdminServerView 管理端节点视图:展开 group_id/host/port/config(用户端隐藏)。
export interface AdminServerView {
  id: number
  group_id: number
  name: string
  type: string
  host: string
  port: number
  config: string
  rate: number
  tags: string[] | null
  status: number
  is_show: boolean
  sort: number
  node_key: string // 节点上报密钥(X-Node-Key)
  is_online: boolean
  last_heartbeat_at: number | null
  system?: NodeSystemStats | null
}

export interface AdminUserItem {
  id: number
  email: string
  role: number
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
  role: number | null
  banned: boolean | null
}

export interface AdjustBalanceReq {
  amount: number // 元，可正可负
  remark: string
}

export interface AdminPlanReq {
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
  group_ids: number[] | null // nil=不更新;空数组=清空(该套餐不关联任何分组)
  is_show: boolean | null
  sort: number
}

export interface AdminServerGroupReq {
  name: string
  sort: number
}

export interface AdminServerReq {
  group_id: number
  name: string
  type: string
  host: string
  port: number
  config: string // 协议私有参数 JSON
  rate: number
  tags: string[] | null // nil=不更新;空数组=清空
  status: number
  is_show: boolean | null
  sort: number
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
  commission_amount: number | null // 该订单产生的佣金(元);nil=无佣金记录
  status: number
  pay_method: string | null
  paid_at: string | null
  created_at: string
}

export interface RefundReq {
  remark: string
  gateway_refund_id?: string // 外部网关退款单号:现金支付需线下在网关侧退款后回填,仅审计留痕,系统不做网关调用
}

export interface AdminCouponReq {
  code: string
  type: number
  value: number
  min_spend: number
  limit_per_user: number
  total_limit: number
  valid_periods: string[] | null
  plan_ids: number[] | null
  started_at: string | null
  ended_at: string | null
  is_enable: boolean | null
}

// AdminCouponView 管理端优惠券视图:展开 type/value/min_spend(元)/used_count/valid_periods/plan_ids。
export interface AdminCouponView {
  id: number
  code: string
  type: number
  value: number
  min_spend: number
  limit_per_user: number
  total_limit: number
  used_count: number
  valid_periods: string[] | null
  plan_ids: number[] | null
  started_at: string | null
  ended_at: string | null
  is_enable: boolean
  created_at: string
}

export interface AdminNoticeItem {
  id: number
  title: string
  content: string
  is_show: boolean
  sort: number
  created_at: string
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

export interface AdminNoticeReq {
  title: string
  content: string
  is_show: boolean | null
  sort: number
}

export interface AdminKnowledgeReq {
  category: string
  category_id: number | null // F15 显式归类（空则按 category 名称归并/自建）
  title: string
  body: string
  language: string
  is_show: boolean | null
  sort: number
}

export interface AdminTicketItem {
  id: number
  user_id: number
  user_email: string
  subject: string
  level: number
  type: number // 0=普通 1=佣金提现(F02)
  status: number
  reopen_count: number
  last_reply_at: string | null
  created_at: string
}

export interface AdminReplyReq {
  message: string
}

export interface AdminAgentApplyItem {
  id: number
  user_id: number
  user_email: string
  valid_invites: number
  status: number // 0=待审核 1=通过 2=拒绝
  created_at: string
}

export interface AdminApproveReq {
  remark: string
}

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
  type: number // 0=订单佣金 1=提现流水(F02)
  status: number
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
  items: TrafficImportItem[] | null
}

export interface AdminSettingsReq {
  key: string
  value: string // JSON 字符串
}

export interface AdminSettingsResp {
  key: string
  value: string
}

// AdminAuditLogItem 管理端审计日志条目（含操作人邮箱；detail 为 jsonb 原始字符串）。
// TargetKind/TargetDisplay 为展示增强：按 action 把 target 翻译成实体类型与可读名称
// （用户邮箱 / 节点名 / 分类名等），解析失败为 null，前端回退显示原始 target。
export interface AdminAuditLogItem {
  id: number
  admin_id: number
  admin_email: string
  action: string
  target: string | null
  target_kind: string | null
  target_display: string | null
  detail: string | null
  ip: string | null
  created_at: string
}

// AdminBatchServerReq 节点批量操作：delete=删除；update=批量更新公共字段（至少一项，status/is_show/group_id/rate）。
export interface AdminBatchServerReq {
  action: string
  ids: number[] | null
  status: number | null
  is_show: boolean | null
  group_id: number | null
  rate: number | null
}

// AdminBatchServerResp 批量节点操作结果汇总（复用 F05 的失败明细结构）。
export interface AdminBatchServerResp {
  success: number
  failed: AdminBatchFailedItem[] | null
}

// AdminSortServerReq 节点排序：items 按 sort 值更新（前端按展示顺序传 0..n）。
export interface AdminSortServerReq {
  items: AdminSortItem[] | null
}

export interface AdminSortItem {
  id: number
  sort: number
}

// AdminTrafficResetReq 按用户批量重置流量：clear_usage=清零用量；reset_quota=另按当前套餐额度重新给量。
export interface AdminTrafficResetReq {
  user_ids: number[] | null
  mode: string
}

// AdminTrafficResetResp 重置结果汇总。
export interface AdminTrafficResetResp {
  success: number
  failed: AdminBatchFailedItem[] | null
}

// AdminTrafficResetLogItem 重置记录条目（含用户邮箱联表）。
export interface AdminTrafficResetLogItem {
  id: number
  user_id: number
  user_email: string
  mode: string
  before_u: number
  before_d: number
  before_transfer_enable: number
  after_transfer_enable: number
  created_at: string
}

// AdminStatOrderPoint 订单日趋势点（date=YYYY-MM-DD）。
export interface AdminStatOrderPoint {
  date: string
  order_count: number // 当日创建订单数
  completed_count: number // 当日完成订单数
  revenue: number // 当日实收现金部分（元，按 paid_at）
  refunded: number // 当日退款现金部分（元，按 updated_at 近似）
  balance_used: number // 当日完成订单余额支付部分（元，按 paid_at）
  balance_refunded: number // 当日退款订单余额部分（元，按 updated_at 近似）
}

export interface AdminStatOrdersResp {
  days: number
  items: AdminStatOrderPoint[] | null // 含无数据日补零，便于折线图
}

export interface AdminStatUserPoint {
  date: string
  count: number
}

// AdminStatPlanSlice 套餐分布切片（当前生效订阅按套餐聚合）。
export interface AdminStatPlanSlice {
  plan_id: number
  plan_name: string
  users: number
}

export interface AdminStatUsersResp {
  days: number
  register_trend: AdminStatUserPoint[] | null // 含无数据日补零
  plan_distribution: AdminStatPlanSlice[] | null
}

export interface AdminStatUserTraffic {
  user_id: number
  email: string
  total_bytes: number // 时间范围内 u+d 合计
}

export interface AdminStatNodeTraffic {
  server_id: number
  name: string
  bytes: number // 上报累计值合计（未乘倍率，节点全周期）
}

export interface AdminStatTrafficResp {
  days: number
  user_top: AdminStatUserTraffic[] | null // 流量消耗 TopN
  node_top: AdminStatNodeTraffic[] | null // 节点流量分布 TopN
}

// AdminBatchUserReq 批量用户操作：ban/unban/adjust_balance。
export interface AdminBatchUserReq {
  action: string
  ids: number[] | null
  amount: number | null // adjust_balance 必填（元，可正可负）
  remark: string
}

// AdminBatchFailedItem 批量操作失败明细。
export interface AdminBatchFailedItem {
  id: number
  reason: string
}

// AdminBatchUserResp 批量操作结果汇总。
export interface AdminBatchUserResp {
  success: number
  failed: AdminBatchFailedItem[] | null
}

// AdminSendMailReq 管理端向用户发送邮件。
export interface AdminSendMailReq {
  ids: number[] | null
  subject: string
  body: string
}

// AdminSendMailResp 发送结果汇总。
export interface AdminSendMailResp {
  sent: number
  failed: AdminBatchFailedItem[] | null
}

// AdminSortReq 内容排序（公告/知识库共用）：items 按 sort 值更新（前端按展示顺序传 0..n）。
export interface AdminSortReq {
  items: AdminSortItem[] | null
}

// AdminKnowledgeCategoryItem 知识库分类（含文档计数）。
export interface AdminKnowledgeCategoryItem {
  id: number
  language: string
  name: string
  sort: number
  knowledge_count: number
}

// AdminKnowledgeCategoryReq 新建分类。
export interface AdminKnowledgeCategoryReq {
  language: string
  name: string
  sort: number | null
}

// AdminKnowledgeCategoryUpdateReq 更新分类（改名级联同步知识文档展示分类）。
export interface AdminKnowledgeCategoryUpdateReq {
  name: string
  sort: number | null
}

// AdminMailTemplateItem 邮件模板视图：custom=已自定义，否则为内置默认文案。
export interface AdminMailTemplateItem {
  name: string
  subject: string
  body: string
  is_custom: boolean
  placeholders: string[] | null
  remark: string
  updated_at: string | null
}

// AdminMailTemplateReq 保存邮件模板（Go template 语法，保存前校验可解析）。
export interface AdminMailTemplateReq {
  subject: string
  body: string
}

// AdminMailTemplateTestReq 测试发送（走真实 SMTP）。
export interface AdminMailTemplateTestReq {
  to_email: string
}

// AdminVersionResp 版本信息：latest 为空表示未配置更新源或拉取失败。
export interface AdminVersionResp {
  version: string
  latest: string | null
  has_update: boolean | null
  notes: string | null
}

// AdminSubscriptionTemplateItem 订阅模板视图：custom=已自定义，否则为内置生成器模板。
export interface AdminSubscriptionTemplateItem {
  name: string
  content: string
  is_custom: boolean
  variables: string[] | null
  remark: string
  updated_at: string | null
}

// AdminSubscriptionTemplateReq 保存订阅模板（Go template 语法，保存前用示例数据渲染校验）。
export interface AdminSubscriptionTemplateReq {
  content: string
}

// AdminSubscriptionTemplatePreviewResp 预览渲染结果（示例数据）。
export interface AdminSubscriptionTemplatePreviewResp {
  name: string
  content: string
}

// TelegramBindCodeResp 绑定验证码：用户在 10 分钟内向 bot 发送 /bind <code>。
export interface TelegramBindCodeResp {
  code: string
  bot_username: string
  ttl_minutes: number
}

// AdminTelegramWebhookSetupResp webhook 注册结果。
export interface AdminTelegramWebhookSetupResp {
  webhook_url: string
  message: string
}
