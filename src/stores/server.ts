import { defineStore } from 'pinia'
import { apiServer } from '@/api/server'
import type { ServerGroup } from '@/types/api'

interface ServerState {
  groups: ServerGroup[]
  loading: boolean
  lastUpdated: number
  timer: ReturnType<typeof setInterval> | null
}

export const useServerStore = defineStore('server', {
  state: (): ServerState => ({
    groups: [],
    loading: false,
    lastUpdated: 0,
    timer: null,
  }),
  actions: {
    async fetch() {
      this.loading = true
      try {
        const data = await apiServer.fetch()
        // 兜底 null:后端空数据时避免模板 groups.length 崩溃
        this.groups = data?.groups ?? []
        this.lastUpdated = Date.now()
      } finally {
        this.loading = false
      }
    },
    /** 60s 静默轮询 */
    startPolling() {
      this.stopPolling()
      this.timer = setInterval(() => {
        void this.fetch()
      }, 60_000)
    },
    stopPolling() {
      if (this.timer) {
        clearInterval(this.timer)
        this.timer = null
      }
    },
  },
})
