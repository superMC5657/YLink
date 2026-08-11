import type { MockMethod } from 'vite-plugin-mock'
import { verifyAccess } from './auth'
import type { Order, PlanPeriod } from '../src/types/api'
import dayjs from 'dayjs'

function ok(data: unknown) {
  return { code: 0, message: 'ok', data }
}

function unauthorized() {
  return { code: 40100, message: '未登录或 token 失效', data: null }
}

let orderSeq = 0

function makeOrderNo() {
  return `2026${String(++orderSeq).padStart(19, '0')}`
}

/** 订单种子数据 */
const orders: Order[] = [
  {
    order_no: '20260701000000000000001',
    plan_name: '猎户座',
    period: 'month',
    amount: 20.0,
    discount_amount: 0,
    balance_used: 0,
    pay_amount: 20.0,
    coupon_code: null,
    status: 1,
    pay_method: 'epay_alipay',
    paid_at: '2026-07-01T10:00:00+08:00',
    created_at: '2026-07-01T09:58:00+08:00',
  },
  {
    order_no: '20260705000000000000002',
    plan_name: '白羊座',
    period: 'year',
    amount: 96.0,
    discount_amount: 19.2,
    balance_used: 0,
    pay_amount: 76.8,
    coupon_code: '618SALE',
    status: 1,
    pay_method: 'epay_wxpay',
    paid_at: '2026-07-05T15:30:00+08:00',
    created_at: '2026-07-05T15:28:00+08:00',
  },
  {
    order_no: '20260710000000000000003',
    plan_name: '射手座',
    period: 'quarter',
    amount: 54.0,
    discount_amount: 0,
    balance_used: 54.0,
    pay_amount: 0,
    coupon_code: null,
    status: 1,
    pay_method: 'balance',
    paid_at: '2026-07-10T20:00:00+08:00',
    created_at: '2026-07-10T19:59:00+08:00',
  },
  {
    order_no: '20260715000000000000004',
    plan_name: '金牛座',
    period: 'month',
    amount: 15.0,
    discount_amount: 0,
    balance_used: 0,
    pay_amount: 15.0,
    coupon_code: null,
    status: 2,
    pay_method: null,
    paid_at: null,
    created_at: '2026-07-15T11:20:00+08:00',
  },
  {
    order_no: '20260718000000000000005',
    plan_name: '白羊座',
    period: 'quarter',
    amount: 27.0,
    discount_amount: 2.0,
    balance_used: 0,
    pay_amount: 25.0,
    coupon_code: 'NEW10',
    status: 0,
    pay_method: null,
    paid_at: null,
    created_at: '2026-07-18T09:05:00+08:00',
  },
]

/** 支付状态模拟:倒计时后自动支付成功 */
function scheduleAutoPay(orderNo: string) {
  setTimeout(() => {
    const o = orders.find((x) => x.order_no === orderNo)
    if (o && o.status === 0) {
      o.status = 1
      o.paid_at = dayjs().toISOString()
      o.pay_method = 'epay_alipay'
    }
  }, 8000)
}

export default [
  {
    url: '/api/v1/coupons/available',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      // 与 /coupons/check 的 discountMap 保持一致（Mock 内存态，契约字段对齐 AdminCouponView 展开）
      return ok({
        list: [
          {
            code: '618SALE',
            type: 1,
            value: 2.0,
            min_spend: 0,
            valid_periods: ['month', 'quarter', 'half_year', 'year'],
            plan_ids: [],
            started_at: null,
            ended_at: null,
          },
          {
            code: 'NEW10',
            type: 2,
            value: 10,
            min_spend: 0,
            valid_periods: ['month', 'quarter', 'half_year', 'year'],
            plan_ids: [],
            started_at: null,
            ended_at: null,
          },
          {
            code: 'WELCOME',
            type: 1,
            value: 5.0,
            min_spend: 20,
            valid_periods: ['month', 'quarter', 'half_year', 'year'],
            plan_ids: [],
            started_at: null,
            ended_at: null,
          },
        ],
      })
    },
  },
  {
    url: '/api/v1/coupons/check',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { code?: string; plan_id?: number; period?: PlanPeriod }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const code = body?.code ?? ''
      if (!code) return { code: 12001, message: '请输入优惠码', data: null }
      const discountMap: Record<string, number> = {
        '618SALE': 2.0,
        NEW10: 1.5,
        WELCOME: 5.0,
      }
      if (!discountMap[code.toUpperCase()]) {
        return { code: 12001, message: '优惠券无效或已过期', data: null }
      }
      return ok({ valid: true, discount_amount: discountMap[code.toUpperCase()], pay_amount: 0 })
    },
  },
  {
    url: '/api/v1/orders',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { plan_id?: number; period?: PlanPeriod; coupon_code?: string | null }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const planNameMap: Record<number, string> = { 1: '白羊座', 2: '金牛座', 3: '射手座' }
      const priceMap: Record<number, Partial<Record<PlanPeriod, number>>> = {
        1: { month: 10, quarter: 27, year: 96 },
        2: { month: 15, quarter: 40.5, year: 144 },
        3: { month: 20, quarter: 54, year: 192 },
      }
      const planName = planNameMap[body?.plan_id ?? -1] ?? '白羊座'
      const price = priceMap[body?.plan_id ?? 1]?.[body?.period ?? 'month'] ?? 10
      const discount = body?.coupon_code ? (body.coupon_code === '618SALE' ? 2 : 1.5) : 0
      const order: Order = {
        order_no: makeOrderNo(),
        plan_name: planName,
        period: body?.period ?? 'month',
        amount: price,
        discount_amount: discount,
        balance_used: 0,
        pay_amount: Math.max(0, +(price - discount).toFixed(2)),
        coupon_code: body?.coupon_code ?? null,
        status: 0,
        pay_method: null,
        paid_at: null,
        created_at: dayjs().toISOString(),
      }
      orders.unshift(order)
      return ok(order)
    },
  },
  {
    url: '/api/v1/orders',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { status?: string; page?: string; page_size?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const size = Number(query.page_size ?? 10)
      let list = orders
      if (query.status !== undefined && query.status !== '') {
        list = orders.filter((o) => o.status === Number(query.status))
      }
      const start = (page - 1) * size
      return ok({
        total: list.length,
        page,
        page_size: size,
        list: list.slice(start, start + size),
      })
    },
  },
  {
    url: '/api/v1/orders/:order_no',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { order_no?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const order = orders.find((o) => o.order_no === query.order_no)
      if (!order) return { code: 40400, message: '订单不存在', data: null }
      return ok(order)
    },
  },
  {
    url: '/api/v1/orders/:order_no/cancel',
    method: 'post',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { order_no?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const order = orders.find((o) => o.order_no === query.order_no)
      if (!order) return { code: 40400, message: '订单不存在', data: null }
      if (order.status !== 0) return { code: 11003, message: '当前状态不允许取消', data: null }
      order.status = 2
      return ok(order)
    },
  },
  {
    url: '/api/v1/orders/:order_no/checkout',
    method: 'post',
    response: ({
      headers,
      query,
      body,
    }: {
      headers: Record<string, string>
      query: { order_no?: string }
      body: { method?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const order = orders.find((o) => o.order_no === query.order_no)
      if (!order) return { code: 40400, message: '订单不存在', data: null }
      if (order.status !== 0) return { code: 11003, message: '订单状态不允许支付', data: null }

      const method = body?.method ?? 'epay_alipay'
      if (method === 'balance') {
        // 演示:余额充足,直接支付成功(真实逻辑由服务端校验余额)
        order.status = 1
        order.pay_method = 'balance'
        order.paid_at = dayjs().toISOString()
        return ok({ type: 'paid', content: null, expire_in: 0 })
      }
      if (method === 'epay_wxpay') {
        // 微信:返回跳转 URL
        return ok({
          type: 'url',
          content: 'https://pay.example.com/submit/mock-wxpay',
          expire_in: 1800,
        })
      }
      // 支付宝:返回二维码内容
      scheduleAutoPay(order.order_no)
      return ok({
        type: 'qrcode',
        content: 'alipays://platformapi/startapp?appId=20000067&orderId=mock-alipay',
        expire_in: 1800,
      })
    },
  },
] as MockMethod[]
