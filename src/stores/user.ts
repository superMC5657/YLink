import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import type {
  CaptchaSendReq,
  CaptchaSendResp,
  ChangePasswordReq,
  ProfileResp,
  ProfileUpdateReq,
  SubscribeInfo,
  TelegramBindCodeResp,
  TrafficLog,
  UserSessionItem,
  UserStat,
} from '@/types/api'

/**
 * user module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/user` 适配缝共用同一核（一处修改，N 处调用）；
 * 适配缝只走传输核、不写 store 异步态，直接调用方保持原有行为。
 */
export const captchaTransport = {
  sendEmail: (body: CaptchaSendReq) => http.post<CaptchaSendResp>('/captcha/email', { body }),
}

export const userTransport = {
  stat: () => http.get<UserStat>('/user/stat'),
  profile: () => http.get<ProfileResp>('/user/profile'),
  updateProfile: (body: ProfileUpdateReq) => http.put<ProfileResp>('/user/profile', { body }),
  changePassword: (body: ChangePasswordReq) => http.post<null>('/user/password/change', { body }),
  subscribe: () => http.get<SubscribeInfo>('/user/subscribe'),
  resetSubscribe: (body: { password: string }) =>
    http.post<{ subscribe_url: string }>('/user/subscribe/reset', { body }),
  trafficLogs: (from: string, to: string) =>
    http.get<{ list: TrafficLog[] }>('/user/traffic-logs', { query: { from, to } }),
  // 会话管理（F14）
  sessions: () => http.get<{ list: UserSessionItem[] }>('/user/sessions'),
  revokeSession: (jti: string) => http.delete<null>(`/user/sessions/${jti}`),
  // Telegram 绑定（F12）
  telegramBindCode: () => http.post<TelegramBindCodeResp>('/user/telegram/bind-code'),
  telegramUnbind: () => http.post<null>('/user/telegram/unbind'),
}

interface UserState {
  stat: UserStat | null
  profile: ProfileResp | null
  subscribe: SubscribeInfo | null
  trafficLogs: TrafficLog[]
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    stat: null,
    profile: null,
    subscribe: null,
    trafficLogs: [],
  }),
  getters: {
    balance: (s) => s.stat?.balance ?? 0,
    commissionBalance: (s) => s.stat?.commission_balance ?? 0,
    isAgent: (s) => s.stat?.is_agent ?? false,
  },
  actions: {
    async fetchStat() {
      this.stat = await userTransport.stat()
    },
    async fetchProfile() {
      this.profile = await userTransport.profile()
      return this.profile
    },
    async updateProfile(body: ProfileUpdateReq) {
      this.profile = await userTransport.updateProfile(body)
      return this.profile
    },
    async changePassword(body: ChangePasswordReq) {
      return userTransport.changePassword(body)
    },
    async resetSubscribe(body: { password: string }) {
      return userTransport.resetSubscribe(body)
    },
    async fetchSubscribe() {
      this.subscribe = await userTransport.subscribe()
    },
    async fetchTrafficLogs(from: string, to: string) {
      const data = await userTransport.trafficLogs(from, to)
      this.trafficLogs = data?.list ?? []
    },
    /** 窗口聚焦时静默刷新仪表板数据 */
    async refreshDashboard() {
      await Promise.allSettled([this.fetchStat(), this.fetchSubscribe()])
    },
  },
})
