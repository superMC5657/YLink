import { defineStore } from 'pinia'
import { apiUser } from '@/api/user'
import type { SubscribeInfo, TrafficLog, UserStat } from '@/types/api'

interface UserState {
  stat: UserStat | null
  subscribe: SubscribeInfo | null
  trafficLogs: TrafficLog[]
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    stat: null,
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
    async fetchSubscribe() {
      this.subscribe = await apiUser.subscribe()
    },
    async fetchTrafficLogs(from: string, to: string) {
      this.trafficLogs = await apiUser.trafficLogs(from, to)
    },
    /** 窗口聚焦时静默刷新仪表板数据 */
    async refreshDashboard() {
      await Promise.allSettled([this.fetchStat(), this.fetchSubscribe()])
    },
  },
})
