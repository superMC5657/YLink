import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import type {
  CreateTicketReq,
  PageQuery,
  PageResult,
  Ticket,
  TicketDetail,
  TicketLevel,
  TicketReplyReq,
} from '@/types/api'

/**
 * ticket module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/ticket` 适配缝共用同一核（一处修改，N 处调用）；
 * 适配缝只走传输核、不写 store 异步态，后台轮询不掀翻列表视图。
 */
export const ticketTransport = {
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

interface TicketState {
  list: Ticket[]
  total: number
  detail: TicketDetail | null
  loading: boolean
}

export const useTicketStore = defineStore('ticket', {
  state: (): TicketState => ({
    list: [],
    total: 0,
    detail: null,
    loading: false,
  }),
  actions: {
    async fetch(page = 1, pageSize = 10) {
      this.loading = true
      try {
        const data = await ticketTransport.fetch({ page, page_size: pageSize })
        this.list = data?.list ?? []
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async create(body: CreateTicketReq) {
      const ticket = await ticketTransport.create(body)
      this.list.unshift(ticket)
      this.total += 1
      return ticket
    },
    async fetchDetail(id: number) {
      this.detail = await ticketTransport.detail(id)
      return this.detail
    },
    async reply(message: string) {
      if (!this.detail) return
      this.detail = await ticketTransport.reply(this.detail.id, { message })
      return this.detail
    },
    async close() {
      if (!this.detail) return
      const updated = await ticketTransport.close(this.detail.id)
      const idx = this.list.findIndex((t) => t.id === updated.id)
      if (idx >= 0) this.list[idx] = updated
      if (this.detail) this.detail.status = updated.status
      return updated
    },
    async reopen() {
      if (!this.detail) return
      const updated = await ticketTransport.reopen(this.detail.id)
      const idx = this.list.findIndex((t) => t.id === updated.id)
      if (idx >= 0) this.list[idx] = updated
      if (this.detail) {
        this.detail.status = updated.status
        this.detail.reopen_count = updated.reopen_count
      }
      return updated
    },
    levelOf(t: Ticket | TicketDetail): TicketLevel {
      return t.level
    },
  },
})
