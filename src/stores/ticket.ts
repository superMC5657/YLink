import { defineStore } from 'pinia'
import { apiTicket } from '@/api/ticket'
import type { CreateTicketReq, Ticket, TicketDetail, TicketLevel } from '@/types/api'

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
        const data = await apiTicket.fetch({ page, page_size: pageSize })
        this.list = data?.list ?? []
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
    async create(body: CreateTicketReq) {
      const ticket = await apiTicket.create(body)
      this.list.unshift(ticket)
      this.total += 1
      return ticket
    },
    async fetchDetail(id: number) {
      this.detail = await apiTicket.detail(id)
      return this.detail
    },
    async reply(message: string) {
      if (!this.detail) return
      this.detail = await apiTicket.reply(this.detail.id, { message })
      return this.detail
    },
    async close() {
      if (!this.detail) return
      const updated = await apiTicket.close(this.detail.id)
      const idx = this.list.findIndex((t) => t.id === updated.id)
      if (idx >= 0) this.list[idx] = updated
      if (this.detail) this.detail.status = updated.status
      return updated
    },
    async reopen() {
      if (!this.detail) return
      const updated = await apiTicket.reopen(this.detail.id)
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
