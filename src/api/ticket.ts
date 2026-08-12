import { http } from '@/utils/http'
import type {
  CreateTicketReq,
  PageQuery,
  PageResult,
  Ticket,
  TicketDetail,
  TicketReplyReq,
} from '@/types/api'

export const apiTicket = {
  fetch: (q: PageQuery = {}) =>
    http.get<PageResult<Ticket>>('/tickets', {
      query: { page: q.page, page_size: q.page_size },
    }),
  create: (body: CreateTicketReq) => http.post<Ticket>('/tickets', { body }),
  detail: (id: number) => http.get<TicketDetail>(`/tickets/${id}`),
  reply: (id: number, body: TicketReplyReq) =>
    http.post<TicketDetail>(`/tickets/${id}/reply`, { body }),
  close: (id: number) => http.post<Ticket>(`/tickets/${id}/close`),
  reopen: (id: number) => http.post<Ticket>(`/tickets/${id}/reopen`),
}
