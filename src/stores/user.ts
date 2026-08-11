import { defineStore } from 'pinia'
import { apiUser } from '@/api/user'
import type {
  ChangePasswordReq,
  ProfileResp,
  ProfileUpdateReq,
  SubscribeInfo,
  TrafficLog,
  UserStat,
} from '@/types/api'

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
      this.stat = await apiUser.stat()
    },
    async fetchProfile() {
      this.profile = await apiUser.profile()
      return this.profile
    },
    async updateProfile(body: ProfileUpdateReq) {
      this.profile = await apiUser.updateProfile(body)
      return this.profile
    },
    async changePassword(body: ChangePasswordReq) {
      return apiUser.changePassword(body)
    },
    async resetSubscribe(body: { password: string }) {
      return apiUser.resetSubscribe(body)
    },
    async fetchSubscribe() {
      this.subscribe = await apiUser.subscribe()
    },
    async fetchTrafficLogs(from: string, to: string) {
      const data = await apiUser.trafficLogs(from, to)
      this.trafficLogs = data?.list ?? []
    },
    /** 窗口聚焦时静默刷新仪表板数据 */
    async refreshDashboard() {
      await Promise.allSettled([this.fetchStat(), this.fetchSubscribe()])
    },
  },
})
