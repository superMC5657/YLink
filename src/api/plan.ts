import { http } from '@/utils/http'
import type { Plan, PlanListResp } from '@/types/api'

export const apiPlan = {
  fetch: () => http.get<PlanListResp>('/plans'),
  list: () => http.get<Plan[]>('/plans', { silent: true }),
}
