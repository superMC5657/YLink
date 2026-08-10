import { defineStore } from 'pinia'
import { apiInvite } from '@/api/invite'
import type { CommissionRecord, InviteCode, InviteSummary } from '@/types/api'

interface InviteState {
  summary: InviteSummary | null
  codes: InviteCode[]
  codeLimit: number
  registerUrlPrefix: string
  records: CommissionRecord[]
  recordsTotal: number
}

export const useInviteStore = defineStore('invite', {
  state: (): InviteState => ({
    summary: null,
    codes: [],
    codeLimit: 0,
    registerUrlPrefix: '',
    records: [],
    recordsTotal: 0,
  }),
  actions: {
    async fetchSummary() {
      this.summary = await apiInvite.summary()
    },
    async fetchCodes() {
      const data = await apiInvite.codes()
      this.codes = data?.list ?? []
      this.codeLimit = data.limit
      this.registerUrlPrefix = data.register_url_prefix
    },
    async createCode() {
      const code = await apiInvite.createCode()
      this.codes.unshift(code)
      return code
    },
    async deleteCode(code: string) {
      await apiInvite.deleteCode(code)
      this.codes = this.codes.filter((c) => c.code !== code)
    },
    async fetchRecords(page = 1, pageSize = 10) {
      const data = await apiInvite.records({ page, page_size: pageSize })
      this.records = data?.list ?? []
      this.recordsTotal = data.total
    },
    async transfer(amount: number) {
      const data = await apiInvite.transfer({ amount })
      if (this.summary) {
        this.summary.commission_balance = data.commission_balance
      }
      return data
    },
    async refreshAll() {
      await Promise.allSettled([this.fetchSummary(), this.fetchCodes(), this.fetchRecords()])
    },
  },
})
