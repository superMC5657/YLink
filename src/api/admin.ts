import { http } from '@/utils/http'
import type {
  AdjustBalanceReq,
  AdminOrderItem,
  AdminOverviewResp,
  AdminPlanItem,
  AdminPlanReq,
  AdminReplyReq,
  AdminRole,
  AdminServerGroupItem,
  AdminServerGroupReq,
  AdminServerItem,
  AdminServerReq,
  AdminTicketDetail,
  AdminTicketItem,
  AdminUpdateUserReq,
  AdminUserItem,
  PageResult,
  RefundReq,
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
}

/** 角色文案:0=普通用户 1=管理员 2=代理商(管理端表格展示) */
export const ADMIN_ROLE_LABEL: Record<AdminRole, string> = {
  0: '普通用户',
  1: '管理员',
  2: '代理商',
}
