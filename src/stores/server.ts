import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import { withLoading } from '@/composables/withLoading'
import type { ServerGroup, ServerListResp } from '@/types/api'

/**
 * server module 的传输核：本域 URL 与响应形态的唯一归属。
 */
export const serverTransport = {
  fetch: () => http.get<ServerListResp>('/servers'),
}

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
      await withLoading(this, async () => {
        const data = await serverTransport.fetch()
        // 兜底 null:后端空数据时避免模板 groups.length 崩溃
        this.groups = data?.groups ?? []
        this.lastUpdated = Date.now()
      })
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
