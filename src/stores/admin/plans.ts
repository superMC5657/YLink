import { http } from '@/utils/http'
import { adminPath as ap } from './path'
import type { AdminPlanItem, AdminPlanReq } from '@/types/api'

/**
 * 套餐域传输核:本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const plansTransport = {
  plans: () => http.get<{ list: AdminPlanItem[] }>(ap('plans')),
  createPlan: (body: AdminPlanReq) => http.post<AdminPlanItem>(ap('plans'), { body }),
  updatePlan: (id: number, body: AdminPlanReq) => http.put<null>(ap(`plans/${id}`), { body }),
  deletePlan: (id: number) => http.delete<null>(ap(`plans/${id}`)),
}
