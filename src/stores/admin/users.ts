import { download, http } from '@/utils/http'
import { withPage } from '@/utils/adminList'
import { adminPath as ap } from './path'
import type {
  AdjustBalanceReq,
  AdminBatchUserReq,
  AdminBatchUserResp,
  AdminSendMailReq,
  AdminSendMailResp,
  AdminUpdateUserReq,
  AdminUserItem,
  PageResult,
} from '@/types/api'

/**
 * 用户管理域传输核:本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const usersTransport = {
  users: (query: { page?: number; page_size?: number; keyword?: string }) =>
    http.get<PageResult<AdminUserItem>>(ap('users'), { query: withPage(query) }),
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
}
