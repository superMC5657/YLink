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
  getters: {
    /**
     * 前端实际使用的注册链接前缀(修复:后端 base_url 是 API 地址,不能直接拼注册链接;
     * 且前端为 hash 路由,链接需带 #/ 段;state 里的 registerUrlPrefix 仅为契约占位,本 getter 不消费其值):
     * 1. 构建时注入 VITE_WEB_BASE_URL(生产 / Tauri 打包显式指定前端站点域名);
     * 2. 否则取当前页面 origin —— Web 下自动区分本地 Vite dev(5174)、Caddy(80)、生产 HTTPS(443);
     * 3. 兜底相对路径 /#/register?code=(仅 Tauri 且未注入时走到;打包版必须配置 VITE_WEB_BASE_URL,
     *    见 .env.production 注释,避免回退到后端 API 地址产生 8081 错误链接)。
     * 最终形如:http://localhost:5174/#/register?code=
     */
    effectiveRegisterUrlPrefix(): string {
      const webBase = import.meta.env.VITE_WEB_BASE_URL as string | undefined
      if (webBase) return `${webBase.replace(/\/+$/, '')}/#/register?code=`
      const origin = typeof window !== 'undefined' ? window.location?.origin : ''
      if (origin && !origin.startsWith('tauri:')) return `${origin}/#/register?code=`
      return '/#/register?code='
    },
  },
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
