import { http } from '@/utils/http'
import { adminPath as ap } from './path'
import type {
  AdminOverviewResp,
  AdminStatOrdersResp,
  AdminStatTrafficResp,
  AdminStatUsersResp,
} from '@/types/api'

/**
 * 仪表盘与统计报表域传输核:本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const overviewTransport = {
  overview: () => http.get<AdminOverviewResp>(ap('stat/overview')),

  // 统计报表(F04)
  statOrders: (query: { days?: number }) =>
    http.get<AdminStatOrdersResp>(ap('stat/orders'), { query }),
  statUsers: (query: { days?: number }) =>
    http.get<AdminStatUsersResp>(ap('stat/users'), { query }),
  statTraffic: (query: { days?: number }) =>
    http.get<AdminStatTrafficResp>(ap('stat/traffic'), { query }),
}
