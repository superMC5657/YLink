import { http } from '@/utils/http'
import type {
  AdjustBalanceReq,
  AdminAgentApplyItem,
  AdminApproveReq,
  AdminCommissionItem,
  AdminCouponItem,
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
  AdminServerReq,
  AdminSettingsItem,
  AdminSettingsReq,
  AdminTicketDetail,
  AdminTicketItem,
  AdminUpdateUserReq,
  AdminUserItem,
  PageResult,
  RefundReq,
  TrafficImportReq,
} from '@/types/api'

/** 管理端 API(role=admin,契约见 docs/api/README.md 第 16 节) */
export const apiAdmin = {
  // 总览
  overview: () => http.get<AdminOverviewResp>('/admin/stat/overview'),

  // 用户
  users: (query: { page?: number; page_size?: number; keyword?: string }) =>
    http.get<PageResult<AdminUserItem>>('/admin/users', { query }),
  updateUser: (id: number, body: AdminUpdateUserReq) =>
    http.put<null>(`/admin/users/${id}`, { body }),
  adjustBalance: (id: number, body: AdjustBalanceReq) =>
    http.post<null>(`/admin/users/${id}/balance`, { body }),

  // 套餐
  plans: () => http.get<{ list: AdminPlanItem[] }>('/admin/plans'),
  createPlan: (body: AdminPlanReq) => http.post<AdminPlanItem>('/admin/plans', { body }),
  updatePlan: (id: number, body: AdminPlanReq) => http.put<null>(`/admin/plans/${id}`, { body }),
  deletePlan: (id: number) => http.delete<null>(`/admin/plans/${id}`),

  // 节点
  servers: () => http.get<{ list: AdminServerItem[] }>('/admin/servers'),
  createServer: (body: AdminServerReq) => http.post<AdminServerItem>('/admin/servers', { body }),
  updateServer: (id: number, body: AdminServerReq) =>
    http.put<null>(`/admin/servers/${id}`, { body }),
  deleteServer: (id: number) => http.delete<null>(`/admin/servers/${id}`),
  resetServerNodeKey: (id: number) =>
    http.post<{ node_key: string }>(`/admin/servers/${id}/node-key/reset`),
  serverGroups: () => http.get<{ list: AdminServerGroupItem[] }>('/admin/server-groups'),
  createServerGroup: (body: AdminServerGroupReq) =>
    http.post<AdminServerGroupItem>('/admin/server-groups', { body }),
  updateServerGroup: (id: number, body: AdminServerGroupReq) =>
    http.put<null>(`/admin/server-groups/${id}`, { body }),
  deleteServerGroup: (id: number) => http.delete<null>(`/admin/server-groups/${id}`),

  // 订单
  orders: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminOrderItem>>('/admin/orders', { query }),
  refund: (orderNo: string, body: RefundReq) =>
    http.post<null>(`/admin/orders/${orderNo}/refund`, { body }),
  closeOrder: (orderNo: string, body: RefundReq) =>
    http.post<null>(`/admin/orders/${orderNo}/close`, { body }),

  // 工单
  tickets: (query: { page?: number; page_size?: number }) =>
    http.get<PageResult<AdminTicketItem>>('/admin/tickets', { query }),
  ticketDetail: (id: number) => http.get<AdminTicketDetail>(`/admin/tickets/${id}`),
  replyTicket: (id: number, body: AdminReplyReq) =>
    http.post<null>(`/admin/tickets/${id}/reply`, { body }),
  closeTicket: (id: number) => http.post<null>(`/admin/tickets/${id}/close`),

  // 优惠券
  coupons: () => http.get<{ list: AdminCouponItem[] }>('/admin/coupons'),
  createCoupon: (body: AdminCouponReq) => http.post<AdminCouponItem>('/admin/coupons', { body }),
  updateCoupon: (id: number, body: AdminCouponReq) =>
    http.put<null>(`/admin/coupons/${id}`, { body }),
  deleteCoupon: (id: number) => http.delete<null>(`/admin/coupons/${id}`),

  // 公告
  notices: () => http.get<{ list: AdminNoticeItem[] }>('/admin/notices'),
  createNotice: (body: AdminNoticeReq) => http.post<AdminNoticeItem>('/admin/notices', { body }),
  updateNotice: (id: number, body: AdminNoticeReq) =>
    http.put<null>(`/admin/notices/${id}`, { body }),
  deleteNotice: (id: number) => http.delete<null>(`/admin/notices/${id}`),

  // 知识库
  knowledges: () => http.get<{ list: AdminKnowledgeItem[] }>('/admin/knowledges'),
  createKnowledge: (body: AdminKnowledgeReq) =>
    http.post<AdminKnowledgeItem>('/admin/knowledges', { body }),
  updateKnowledge: (id: number, body: AdminKnowledgeReq) =>
    http.put<null>(`/admin/knowledges/${id}`, { body }),
  deleteKnowledge: (id: number) => http.delete<null>(`/admin/knowledges/${id}`),

  // 代理审批
  agentApplies: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminAgentApplyItem>>('/admin/agent/applies', { query }),
  approveAgent: (id: number, body: AdminApproveReq) =>
    http.post<null>(`/admin/agent/applies/${id}/approve`, { body }),
  rejectAgent: (id: number, body: AdminApproveReq) =>
    http.post<null>(`/admin/agent/applies/${id}/reject`, { body }),

  // 佣金日志
  commissionLogs: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminCommissionItem>>('/admin/commission-logs', { query }),

  // 流量导入(模式 B)
  importTraffic: (body: TrafficImportReq) => http.post<null>('/admin/traffic/import', { body }),

  // 站点设置
  settings: () => http.get<{ list: AdminSettingsItem[] }>('/admin/settings'),
  saveSetting: (body: AdminSettingsReq) => http.put<null>('/admin/settings', { body }),
}
