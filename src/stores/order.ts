import { defineStore } from 'pinia'
import { apiOrder } from '@/api/order'
import type { CheckoutReq, CreateOrderReq, Order, OrderStatus, PageQuery } from '@/types/api'

interface OrderState {
  list: Order[]
  total: number
  page: number
  pageSize: number
  loading: boolean
  detail: Order | null
  pollTimer: ReturnType<typeof setInterval> | null
}

export const useOrderStore = defineStore('order', {
  state: (): OrderState => ({
    list: [],
    total: 0,
    page: 1,
    pageSize: 10,
    loading: false,
    detail: null,
    pollTimer: null,
  }),
  actions: {
    async fetch(query: PageQuery & { status?: OrderStatus | '' } = {}) {
      this.loading = true
      try {
        const data = await apiOrder.fetch({
          page: query.page ?? this.page,
          page_size: query.page_size ?? this.pageSize,
          status: query.status,
        })
        this.list = data.list
        this.total = data.total
        this.page = data.page
      } finally {
        this.loading = false
      }
    },
    async fetchDetail(orderNo: string) {
      this.detail = await apiOrder.detail(orderNo)
      return this.detail
    },
    async create(body: CreateOrderReq, idempotencyKey: string) {
      const order = await apiOrder.create(body, idempotencyKey)
      this.list.unshift(order)
      return order
    },
    async checkout(orderNo: string, body: CheckoutReq) {
      return apiOrder.checkout(orderNo, body)
    },
    async cancel(orderNo: string) {
      const updated = await apiOrder.cancel(orderNo)
      const idx = this.list.findIndex((o) => o.order_no === orderNo)
      if (idx >= 0) this.list[idx] = updated
      if (this.detail?.order_no === orderNo) this.detail = updated
      return updated
    },
    /** 存在待支付订单时开启轮询(5s),离开页面调用 stopPolling */
    startPolling(cb?: (order: Order) => void) {
      this.stopPolling()
      this.pollTimer = setInterval(async () => {
        const pending = this.list.filter((o) => o.status === 0)
        for (const o of pending) {
          try {
            const fresh = await apiOrder.detail(o.order_no)
            const idx = this.list.findIndex((x) => x.order_no === o.order_no)
            if (idx >= 0) this.list[idx] = fresh
            if (fresh.status === 1) cb?.(fresh)
          } catch {
            // 单条轮询失败忽略
          }
        }
      }, 5000)
    },
    stopPolling() {
      if (this.pollTimer) {
        clearInterval(this.pollTimer)
        this.pollTimer = null
      }
    },
  },
})
