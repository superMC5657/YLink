import { http } from '@/utils/http'
import type {
  CommissionRecord,
  InviteCode,
  InviteCodeListResp,
  InviteSummary,
  PageQuery,
  PageResult,
  TransferReq,
  TransferResp,
} from '@/types/api'

export const apiInvite = {
  summary: () => http.get<InviteSummary>('/invite/summary'),
  codes: () => http.get<InviteCodeListResp>('/invite/codes'),
  createCode: () => http.post<InviteCode>('/invite/codes'),
  deleteCode: (code: string) => http.delete<void>(`/invite/codes/${encodeURIComponent(code)}`),
  records: (q: PageQuery = {}) =>
    http.get<PageResult<CommissionRecord>>('/invite/records', {
      query: { page: q.page, page_size: q.page_size },
    }),
  transfer: (body: TransferReq) => http.post<TransferResp>('/invite/transfer', { body }),
}
