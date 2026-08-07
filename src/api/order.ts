import { http } from '@/utils/http'
import type {
  CheckoutReq,
  CheckoutResp,
  CouponCheckReq,
  CouponCheckResp,
  CreateOrderReq,
  Order,
  OrderStatus,
  PageQuery,
  PageResult,
} from '@/types/api'

export const apiCoupon = {
  check: (body: CouponCheckReq) => http.post<CouponCheckResp>('/coupons/check', { body }),
}

export const apiOrder = {
  create: (body: CreateOrderReq, idempotencyKey: string) =>
    http.post<Order>('/orders', { body, headers: { 'Idempotency-Key': idempotencyKey } }),
  fetch: (q: PageQuery & { status?: OrderStatus | '' } = {}) =>
    http.get<PageResult<Order>>('/orders', {
      query: { status: q.status, page: q.page, page_size: q.page_size },
    }),
  detail: (orderNo: string) => http.get<Order>(`/orders/${orderNo}`),
  cancel: (orderNo: string) => http.post<Order>(`/orders/${orderNo}/cancel`),
  checkout: (orderNo: string, body: CheckoutReq) =>
    http.post<CheckoutResp>(`/orders/${orderNo}/checkout`, { body }),
}
