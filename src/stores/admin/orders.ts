import { http } from '@/utils/http'
import { withPage } from '@/utils/adminList'
import { adminPath as ap } from './path'
import type { AdminOrderItem, PageResult, RefundReq } from '@/types/api'

/**
 * 订单域传输核:本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const ordersTransport = {
  orders: (query: { page?: number; page_size?: number; status?: number | '' }) =>
    http.get<PageResult<AdminOrderItem>>(ap('orders'), { query: withPage(query) }),
  refund: (orderNo: string, body: RefundReq) =>
    http.post<null>(ap(`orders/${orderNo}/refund`), { body }),
  closeOrder: (orderNo: string, body: RefundReq) =>
    http.post<null>(ap(`orders/${orderNo}/close`), { body }),
}
