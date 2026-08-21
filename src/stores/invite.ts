import { defineStore } from 'pinia'
import { apiInvite } from '@/api/invite'
import { isTauri } from '@/utils/platform'
import type { CommissionRecord, InviteCode, InviteSummary } from '@/types/api'

/** 注册链接路径后缀(前端为 hash 路由;后端 register_url_prefix 契约占位返回同值) */
const REGISTER_URL_SUFFIX = '/#/register?code='

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
     * 3. 兜底相对路径 /#/register?code=(仅 Tauri WebView 且未注入时走到:其 origin 为 tauri://(macOS/Linux)
     *    或 http(s)://tauri.localhost(Windows),均非可分享的 Web 地址,故用 isTauri() 判定而非匹配 origin 前缀;
     *    打包版必须配置 VITE_WEB_BASE_URL,见 .env.production.example,避免生成不可访问的链接)。
     * 最终形如:http://localhost:5174/#/register?code=
     */
    effectiveRegisterUrlPrefix(): string {
      const webBase = import.meta.env.VITE_WEB_BASE_URL as string | undefined
      if (webBase) return `${webBase.replace(/\/+$/, '')}${REGISTER_URL_SUFFIX}`
      const origin = typeof window !== 'undefined' ? window.location?.origin : ''
      if (origin && !isTauri()) return `${origin}${REGISTER_URL_SUFFIX}`
      return REGISTER_URL_SUFFIX
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
