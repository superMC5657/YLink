import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import type { ServerGroup, ServerListResp } from '@/types/api'

/**
 * server module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/server` 适配缝共用同一核（一处修改，N 处调用）。
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
      this.loading = true
      try {
        const data = await serverTransport.fetch()
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
