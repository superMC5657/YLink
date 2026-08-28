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
  WithdrawCreateReq,
  WithdrawItem,
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
  // 佣金提现（F02，仅代理商）
  withdraw: (body: WithdrawCreateReq) => http.post<WithdrawItem>('/invite/withdraw', { body }),
  withdraws: (q: PageQuery = {}) =>
    http.get<PageResult<WithdrawItem>>('/invite/withdraws', {
      query: { page: q.page, page_size: q.page_size },
    }),
}
