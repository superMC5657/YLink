import { defineStore } from 'pinia'
import { apiAgent } from '@/api/agent'
import type { AgentStatus } from '@/types/api'

interface AgentState {
  status: AgentStatus | null
}

export const useAgentStore = defineStore('agent', {
  state: (): AgentState => ({
    status: null,
  }),
  actions: {
    async fetchStatus() {
      this.status = await apiAgent.status()
    },
    async apply() {
      const data = await apiAgent.apply()
      if (this.status) this.status.apply_status = data.apply_status as AgentStatus['apply_status']
      return data
    },
  },
})
