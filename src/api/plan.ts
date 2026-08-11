import { http } from '@/utils/http'
import type { PlanListResp } from '@/types/api'

export const apiPlan = {
  fetch: () => http.get<PlanListResp>('/plans'),
}
