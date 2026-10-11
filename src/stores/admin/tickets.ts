import { http } from '@/utils/http'
import { withPage } from '@/utils/adminList'
import { adminPath as ap } from './path'
import type {
  AdminApproveReq,
  AdminReplyReq,
  AdminTicketDetail,
  AdminTicketItem,
  PageResult,
} from '@/types/api'

/**
 * 工单域传输核:本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const adminTicketsTransport = {
  tickets: (query: { page?: number; page_size?: number }) =>
    http.get<PageResult<AdminTicketItem>>(ap('tickets'), { query: withPage(query) }),
  ticketDetail: (id: number) => http.get<AdminTicketDetail>(ap(`tickets/${id}`)),
  replyTicket: (id: number, body: AdminReplyReq) =>
    http.post<null>(ap(`tickets/${id}/reply`), { body }),
  closeTicket: (id: number) => http.post<null>(ap(`tickets/${id}/close`)),
  // 佣金提现审核（F02）
  withdrawPay: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`tickets/${id}/withdraw/pay`), { body }),
  withdrawReject: (id: number, body: AdminApproveReq) =>
    http.post<null>(ap(`tickets/${id}/withdraw/reject`), { body }),
}
