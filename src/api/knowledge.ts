import { http } from '@/utils/http'
import type { KnowledgeDetail, KnowledgeListResp } from '@/types/api'

export const apiKnowledge = {
  fetch: (params: { language?: string; keyword?: string } = {}) =>
    http.get<KnowledgeListResp>('/knowledges', {
      query: { language: params.language, keyword: params.keyword },
    }),
  detail: (id: number) => http.get<KnowledgeDetail>(`/knowledges/${id}`),
}
