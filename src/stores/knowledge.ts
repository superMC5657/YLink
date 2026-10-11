import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import type { KnowledgeDetail, KnowledgeGroup, KnowledgeListResp } from '@/types/api'

/**
 * knowledge module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/knowledge` 适配缝共用同一核（一处修改，N 处调用）。
 */
export const knowledgeTransport = {
  fetch: (params: { language?: string; keyword?: string } = {}) =>
    http.get<KnowledgeListResp>('/knowledges', {
      query: { language: params.language, keyword: params.keyword },
    }),
  detail: (id: number) => http.get<KnowledgeDetail>(`/knowledges/${id}`),
}

interface KnowledgeState {
  groups: KnowledgeGroup[]
  keyword: string
  language: string
  detail: KnowledgeDetail | null
  loading: boolean
}

export const useKnowledgeStore = defineStore('knowledge', {
  state: (): KnowledgeState => ({
    groups: [],
    keyword: '',
    language: 'zh-CN',
    detail: null,
    loading: false,
  }),
  actions: {
    async fetch(params: { language?: string; keyword?: string } = {}) {
      this.loading = true
      try {
        if (params.language !== undefined) this.language = params.language
        if (params.keyword !== undefined) this.keyword = params.keyword
        const data = await knowledgeTransport.fetch({
          language: this.language,
          keyword: this.keyword,
        })
        // 后端可能返回 groups:null(数据库无文档),必须兜底为数组,
        // 否则模板里 groups.length 对 null 抛 TypeError → 渲染崩溃 → 转圈卡死
        this.groups = data?.groups ?? []
      } finally {
        this.loading = false
      }
    },
    async fetchDetail(id: number) {
      this.detail = await knowledgeTransport.detail(id)
      return this.detail
    },
  },
})
