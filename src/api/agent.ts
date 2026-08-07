import { http } from '@/utils/http'
import type { AgentStatus } from '@/types/api'

export const apiAgent = {
  status: () => http.get<AgentStatus>('/agent/status'),
  apply: () => http.post<{ apply_status: string }>('/agent/apply'),
}
