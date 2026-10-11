import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import type { AgentStatus } from '@/types/api'

/**
 * agent module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/agent` 适配缝共用同一核（一处修改，N 处调用）。
 */
export const agentTransport = {
  status: () => http.get<AgentStatus>('/agent/status'),
  apply: () => http.post<{ apply_status: string }>('/agent/apply'),
}

interface AgentState {
  status: AgentStatus | null
}

export const useAgentStore = defineStore('agent', {
  state: (): AgentState => ({
    status: null,
  }),
  actions: {
    async fetchStatus() {
      this.status = await agentTransport.status()
    },
    async apply() {
      const data = await agentTransport.apply()
      if (this.status) this.status.apply_status = data.apply_status as AgentStatus['apply_status']
      return data
    },
  },
})
