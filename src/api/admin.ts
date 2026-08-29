import { download, http } from '@/utils/http'
import type {
  AdjustBalanceReq,
  AdminAgentApplyItem,
  AdminApproveReq,
  AdminAuditLogsResp,
  AdminBatchServerReq,
  AdminBatchServerResp,
  AdminBatchUserReq,
  AdminBatchUserResp,
  AdminSendMailReq,
  AdminSendMailResp,
  AdminCommissionItem,
  AdminCouponItem,
  AdminKnowledgeCategoryItem,
  AdminKnowledgeCategoryReq,
  AdminKnowledgeCategoryUpdateReq,
  AdminCouponReq,
  AdminKnowledgeItem,
  AdminKnowledgeReq,
  AdminNoticeItem,
  AdminNoticeReq,
  AdminOrderItem,
  AdminOverviewResp,
  AdminPlanItem,
  AdminPlanReq,
  AdminReplyReq,
  AdminServerGroupItem,
  AdminServerGroupReq,
  AdminServerItem,
  AdminMailTemplateItem,
  AdminMailTemplateReq,
  AdminMailTemplateTestReq,
  AdminSubscriptionTemplateItem,
  AdminSubscriptionTemplatePreviewResp,
  AdminSubscriptionTemplateReq,
  AdminTelegramWebhookSetupResp,
  AdminServerReq,
  AdminSettingsItem,
  AdminSettingsReq,
  AdminSortReq,
  AdminSortServerReq,
  AdminStatOrdersResp,
  AdminStatTrafficResp,
  AdminStatUsersResp,
  AdminTicketDetail,
  AdminTicketItem,
  AdminTrafficResetLogItem,
  AdminTrafficResetReq,
  AdminTrafficResetResp,
  AdminUpdateUserReq,
  AdminUserItem,
  AdminVersionResp,
  PageResult,
  RefundReq,
  TrafficImportReq,
} from '@/types/api'

/**
 * 管理端 API 路径段(F22 security.admin_path)。
 * 后端经 config.yaml security.admin_path 或 APP_SECURITY_ADMIN_PATH 定制;
 * 前端部署同步设置 VITE_ADMIN_PATH 重新构建,默认 admin 保持两端一致。
 */
const ADMIN_PATH = (import.meta.env.VITE_ADMIN_PATH as string | undefined) || 'admin'
const ap = (path: string) => `/${ADMIN_PATH}/${path}`

/** 管理端 API(role=admin,契约见 docs/api/README.md 第 16 节) */
export const apiAdmin = {
  // 总览
  overview: () => http.get<AdminOverviewResp>(ap('stat/overview')),

  // 用户
  users: (query: { page?: number; page_size?: number; keyword?: string }) =>
    http.get<PageResult<AdminUserItem>>(ap('users'), { query }),
  updateUser: (id: number, body: AdminUpdateUserReq) => http.put<null>(ap(`users/${id}`), { body }),
  adjustBalance: (id: number, body: AdjustBalanceReq) =>
    http.post<null>(ap(`users/${id}/balance`), { body }),

  // 用户管理增强(F05)
  exportUsersCSV: (query: { keyword?: string }) => download(ap('users/export'), query), // CSV 直下(blob)
  batchUsers: (body: AdminBatchUserReq) =>
    http.post<AdminBatchUserResp>(ap('users/batch'), { body }),
  sendMail: (body: AdminSendMailReq) => http.post<AdminSendMailResp>(ap('users/mail'), { body }),
  resetUserSubToken: (id: number) =>
    http.post<{ subscribe_url: string }>(ap(`users/${id}/sub-token/reset`)),

  // 套餐
  plans: () => http.get<{ list: AdminPlanItem[] }>(ap('plans')),
  createPlan: (body: AdminPlanReq) => http.post<AdminPlanItem>(ap('plans'), { body }),
  updatePlan: (id: number, body: AdminPlanReq) => http.put<null>(ap(`plans/${id}`), { body }),
  deletePlan: (id: number) => http.delete<null>(ap(`plans/${id}`)),

  // 节点
  servers: () => http.get<{ list: AdminServerItem[] }>(ap('servers')),
  createServer: (body: AdminServerReq) => http.post<AdminServerItem>(ap('servers'), { body }),
  updateServer: (id: number, body: AdminServerReq) => http.put<null>(ap(`servers/${id}`), { body }),
  deleteServer: (id: number) => http.delete<null>(ap(`servers/${id}`)),
  resetServerNodeKey: (id: number) =>
    http.post<{ node_key: string }>(ap(`servers/${id}/node-key/reset`)),

  // 节点批量 / 复制 / 排序（F09）
  batchServers: (body: AdminBatchServerReq) =>
    http.post<AdminBatchServerResp>(ap('servers/batch'), { body }),
  copyServer: (id: number) => http.post<AdminServerItem>(ap(`servers/${id}/copy`)),
  sortServers: (body: AdminSortServerReq) => http.post<null>(ap('servers/sort'), { body }),
  serverGroups: () => http.get<{ list: AdminServerGroupItem[] }>(ap('server-groups')),
  createServerGroup: (body: AdminServerGroupReq) =>
    http.post<AdminServerGroupItem>(ap('server-groups'), { body }),
  updateServerGroup: (id: number, body: AdminServerGroupReq) =>
    http.put<null>(ap(`server-groups/${id}`), { body }),
  deleteServerGroup: (id: number) => http.delete<null>(ap(`server-groups/${id}`)),

  // 订单
  orders: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminOrderItem>>(ap('orders'), { query }),
  refund: (orderNo: string, body: RefundReq) =>
    http.post<null>(ap(`orders/${orderNo}/refund`), { body }),
  closeOrder: (orderNo: string, body: RefundReq) =>
    http.post<null>(ap(`orders/${orderNo}/close`), { body }),

  // 工单
  tickets: (query: { page?: number; page_size?: number }) =>
    http.get<PageResult<AdminTicketItem>>(ap('tickets'), { query }),
  ticketDetail: (id: number) => http.get<AdminTicketDetail>(ap(`tickets/${id}`)),
  replyTicket: (id: number, body: AdminReplyReq) =>
    http.post<null>(ap(`tickets/${id}/reply`), { body }),
  closeTicket: (id: number) => http.post<null>(ap(`tickets/${id}/close`)),
  // 佣金提现审核（F02）
  withdrawPay: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`tickets/${id}/withdraw/pay`), { body }),
  withdrawReject: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`tickets/${id}/withdraw/reject`), { body }),

  // 优惠券
  coupons: () => http.get<{ list: AdminCouponItem[] }>(ap('coupons')),
  createCoupon: (body: AdminCouponReq) => http.post<AdminCouponItem>(ap('coupons'), { body }),
  updateCoupon: (id: number, body: AdminCouponReq) => http.put<null>(ap(`coupons/${id}`), { body }),
  deleteCoupon: (id: number) => http.delete<null>(ap(`coupons/${id}`)),

  // 公告
  notices: () => http.get<{ list: AdminNoticeItem[] }>(ap('notices')),
  createNotice: (body: AdminNoticeReq) => http.post<AdminNoticeItem>(ap('notices'), { body }),
  sortNotices: (body: AdminSortReq) => http.post<null>(ap('notices/sort'), { body }),
  updateNotice: (id: number, body: AdminNoticeReq) => http.put<null>(ap(`notices/${id}`), { body }),
  deleteNotice: (id: number) => http.delete<null>(ap(`notices/${id}`)),

  // 知识库
  knowledges: () => http.get<{ list: AdminKnowledgeItem[] }>(ap('knowledges')),
  createKnowledge: (body: AdminKnowledgeReq) =>
    http.post<AdminKnowledgeItem>(ap('knowledges'), { body }),
  sortKnowledges: (body: AdminSortReq) => http.post<null>(ap('knowledges/sort'), { body }),
  updateKnowledge: (id: number, body: AdminKnowledgeReq) =>
    http.put<null>(ap(`knowledges/${id}`), { body }),
  deleteKnowledge: (id: number) => http.delete<null>(ap(`knowledges/${id}`)),

  // 知识库分类（F15）
  knowledgeCategories: (query: { language?: string }) =>
    http.get<{ list: AdminKnowledgeCategoryItem[] }>(ap('knowledge-categories'), { query }),
  createKnowledgeCategory: (body: AdminKnowledgeCategoryReq) =>
    http.post<AdminKnowledgeCategoryItem>(ap('knowledge-categories'), { body }),
  updateKnowledgeCategory: (id: number, body: AdminKnowledgeCategoryUpdateReq) =>
    http.put<null>(ap(`knowledge-categories/${id}`), { body }),
  deleteKnowledgeCategory: (id: number) => http.delete<null>(ap(`knowledge-categories/${id}`)),

  // 代理审批
  agentApplies: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminAgentApplyItem>>(ap('agent/applies'), { query }),
  approveAgent: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`agent/applies/${id}/approve`), { body }),
  rejectAgent: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`agent/applies/${id}/reject`), { body }),

  // 佣金日志
  commissionLogs: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminCommissionItem>>(ap('commission-logs'), { query }),

  // 审计日志(F08 只读)
  auditLogs: (query: {
    page?: number
    page_size?: number
    admin_id?: number | ''
    action?: string
    target?: string
    from?: string
    to?: string
  }) => http.get<AdminAuditLogsResp>(ap('audit-logs'), { query }),

  // 流量导入(模式 B)
  importTraffic: (body: TrafficImportReq) => http.post<null>(ap('traffic/import'), { body }),

  // 流量重置(F16)
  resetTraffic: (body: AdminTrafficResetReq) =>
    http.post<AdminTrafficResetResp>(ap('traffic/reset'), { body }),
  trafficResets: (query: { page?: number; page_size?: number; user_id?: number | '' }) =>
    http.get<PageResult<AdminTrafficResetLogItem>>(ap('traffic/resets'), { query }),

  // 统计报表(F04)
  statOrders: (query: { days?: number }) =>
    http.get<AdminStatOrdersResp>(ap('stat/orders'), { query }),
  statUsers: (query: { days?: number }) =>
    http.get<AdminStatUsersResp>(ap('stat/users'), { query }),
  statTraffic: (query: { days?: number }) =>
    http.get<AdminStatTrafficResp>(ap('stat/traffic'), { query }),

  // 邮件模板（F11）
  mailTemplates: () => http.get<{ list: AdminMailTemplateItem[] }>(ap('mail-templates')),
  saveMailTemplate: (name: string, body: AdminMailTemplateReq) =>
    http.put<null>(ap(`mail-templates/${name}`), { body }),
  resetMailTemplate: (name: string) => http.delete<null>(ap(`mail-templates/${name}`)),
  testMailTemplate: (name: string, body: AdminMailTemplateTestReq) =>
    http.post<null>(ap(`mail-templates/${name}/test`), { body }),

  // 订阅模板（F10）
  subscriptionTemplates: () =>
    http.get<{ list: AdminSubscriptionTemplateItem[] }>(ap('subscription-templates')),
  saveSubscriptionTemplate: (name: string, body: AdminSubscriptionTemplateReq) =>
    http.put<null>(ap(`subscription-templates/${name}`), { body }),
  resetSubscriptionTemplate: (name: string) =>
    http.delete<null>(ap(`subscription-templates/${name}`)),
  previewSubscriptionTemplate: (name: string) =>
    http.post<AdminSubscriptionTemplatePreviewResp>(ap(`subscription-templates/${name}/preview`)),

  // Telegram（F12）
  setupTelegramWebhook: () =>
    http.post<AdminTelegramWebhookSetupResp>(ap('telegram/webhook/setup')),

  // 版本检查（F20）
  version: () => http.get<AdminVersionResp>(ap('version')),

  // 站点设置
  settings: () => http.get<{ list: AdminSettingsItem[] }>(ap('settings')),
  saveSetting: (body: AdminSettingsReq) => http.put<null>(ap('settings'), { body }),
}
