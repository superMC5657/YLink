import { http } from '@/utils/http'
import { adminPath as ap } from './path'
import type {
  AdminCouponItem,
  AdminCouponReq,
  AdminKnowledgeCategoryItem,
  AdminKnowledgeCategoryReq,
  AdminKnowledgeCategoryUpdateReq,
  AdminKnowledgeItem,
  AdminKnowledgeReq,
  AdminNoticeItem,
  AdminNoticeReq,
  AdminSortReq,
} from '@/types/api'

/**
 * 内容域传输核(优惠券/公告/知识库/分类):本域 URL 与响应形态的唯一归属
 * (C3 transport-core 收敛,决策见 ADR-0003)。
 */
export const contentTransport = {
  // 优惠券
  coupons: () => http.get<{ list: AdminCouponItem[] }>(ap('coupons')),
  createCoupon: (body: AdminCouponReq) => http.post<AdminCouponItem>(ap('coupons'), { body }),
  updateCoupon: (id: number, body: AdminCouponReq) => http.put<null>(ap(`coupons/${id}`), { body }),
  deleteCoupon: (id: number) => http.delete<null>(ap(`coupons/${id}`)),

  // 公告
  notices: () => http.get<{ list: AdminNoticeItem[] }>(ap('notices')),
  createNotice: (body: AdminNoticeReq) => http.post<AdminNoticeItem>(ap('notices'), { body }),
  sortNotices: (body: AdminSortReq) => http.post<null>(ap('notices/sort'), { body }),
  updateNotice: (id: number, body: AdminNoticeReq) => http.put<null>(ap(`notices/${id}`), { body }),
  deleteNotice: (id: number) => http.delete<null>(ap(`notices/${id}`)),

  // 知识库
  knowledges: () => http.get<{ list: AdminKnowledgeItem[] }>(ap('knowledges')),
  createKnowledge: (body: AdminKnowledgeReq) =>
    http.post<AdminKnowledgeItem>(ap('knowledges'), { body }),
  sortKnowledges: (body: AdminSortReq) => http.post<null>(ap('knowledges/sort'), { body }),
  updateKnowledge: (id: number, body: AdminKnowledgeReq) =>
    http.put<null>(ap(`knowledges/${id}`), { body }),
  deleteKnowledge: (id: number) => http.delete<null>(ap(`knowledges/${id}`)),

  // 知识库分类（F15）
  knowledgeCategories: (query: { language?: string }) =>
    http.get<{ list: AdminKnowledgeCategoryItem[] }>(ap('knowledge-categories'), { query }),
  createKnowledgeCategory: (body: AdminKnowledgeCategoryReq) =>
    http.post<AdminKnowledgeCategoryItem>(ap('knowledge-categories'), { body }),
  updateKnowledgeCategory: (id: number, body: AdminKnowledgeCategoryUpdateReq) =>
    http.put<null>(ap(`knowledge-categories/${id}`), { body }),
  deleteKnowledgeCategory: (id: number) => http.delete<null>(ap(`knowledge-categories/${id}`)),
}
