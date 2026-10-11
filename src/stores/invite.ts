import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import { isTauri } from '@/utils/platform'
import type {
  CommissionRecord,
  InviteCode,
  InviteCodeListResp,
  InviteSummary,
  PageQuery,
  PageResult,
  TransferReq,
  TransferResp,
  WithdrawCreateReq,
  WithdrawItem,
} from '@/types/api'

/**
 * invite module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/invite` 适配缝共用同一核（一处修改，N 处调用）。
 */
export const inviteTransport = {
  summary: () => http.get<InviteSummary>('/invite/summary'),
  codes: () => http.get<InviteCodeListResp>('/invite/codes'),
  createCode: () => http.post<InviteCode>('/invite/codes'),
  deleteCode: (code: string) => http.delete<void>(`/invite/codes/${encodeURIComponent(code)}`),
  records: (q: PageQuery = {}) =>
    http.get<PageResult<CommissionRecord>>('/invite/records', {
      query: { page: q.page, page_size: q.page_size },
    }),
  transfer: (body: TransferReq) => http.post<TransferResp>('/invite/transfer', { body }),
  // 佣金提现（F02，仅代理商）
  withdraw: (body: WithdrawCreateReq) => http.post<WithdrawItem>('/invite/withdraw', { body }),
  withdraws: (q: PageQuery = {}) =>
    http.get<PageResult<WithdrawItem>>('/invite/withdraws', {
      query: { page: q.page, page_size: q.page_size },
    }),
}

/** 注册链接路径后缀(前端为 hash 路由;后端 register_url_prefix 契约占位返回同值) */
const REGISTER_URL_SUFFIX = '/#/register?code='

interface InviteState {
  summary: InviteSummary | null
  codes: InviteCode[]
  codeLimit: number
  registerUrlPrefix: string
  records: CommissionRecord[]
  recordsTotal: number
  withdraws: WithdrawItem[]
  withdrawsTotal: number
}

export const useInviteStore = defineStore('invite', {
  state: (): InviteState => ({
    summary: null,
    codes: [],
    codeLimit: 0,
    registerUrlPrefix: '',
    records: [],
    recordsTotal: 0,
    withdraws: [],
    withdrawsTotal: 0,
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
      this.summary = await inviteTransport.summary()
    },
    async fetchCodes() {
      const data = await inviteTransport.codes()
      this.codes = data?.list ?? []
      this.codeLimit = data.limit
      this.registerUrlPrefix = data.register_url_prefix
    },
    async createCode() {
      const code = await inviteTransport.createCode()
      this.codes.unshift(code)
      return code
    },
    async deleteCode(code: string) {
      await inviteTransport.deleteCode(code)
      this.codes = this.codes.filter((c) => c.code !== code)
    },
    async fetchRecords(page = 1, pageSize = 10) {
      const data = await inviteTransport.records({ page, page_size: pageSize })
      this.records = data?.list ?? []
      this.recordsTotal = data.total
    },
    async transfer(amount: number) {
      const data = await inviteTransport.transfer({ amount })
      if (this.summary) {
        this.summary.commission_balance = data.commission_balance
      }
      return data
    },
    // 佣金提现（F02，仅代理商）
    async fetchWithdraws(page = 1, pageSize = 10) {
      const data = await inviteTransport.withdraws({ page, page_size: pageSize })
      this.withdraws = data?.list ?? []
      this.withdrawsTotal = data.total
    },
    async withdraw(body: { amount: number; method: string; account: string }) {
      const item = await inviteTransport.withdraw(body)
      this.withdraws.unshift(item)
      // 提交即扣减佣金余额，同步刷新总览
      await this.fetchSummary()
      return item
    },
    async refreshAll() {
      await Promise.allSettled([this.fetchSummary(), this.fetchCodes(), this.fetchRecords()])
    },
  },
})
