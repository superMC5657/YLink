import { http } from '@/utils/http'
import { withPage } from '@/utils/adminList'
import { adminPath as ap } from './path'
import type {
  AdminAgentApplyItem,
  AdminApproveReq,
  AdminCommissionItem,
  PageResult,
} from '@/types/api'

/**
 * 代理与佣金域传输核:本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const agentAdminTransport = {
  // 代理审批
  agentApplies: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminAgentApplyItem>>(ap('agent/applies'), { query: withPage(query) }),
  approveAgent: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`agent/applies/${id}/approve`), { body }),
  rejectAgent: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`agent/applies/${id}/reject`), { body }),

  // 佣金日志
  commissionLogs: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminCommissionItem>>(ap('commission-logs'), { query: withPage(query) }),
}
