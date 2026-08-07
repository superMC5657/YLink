import { defineStore } from 'pinia'
import { apiKnowledge } from '@/api/knowledge'
import type { KnowledgeDetail, KnowledgeGroup } from '@/types/api'

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
        const data = await apiKnowledge.fetch({
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
      this.detail = await apiKnowledge.detail(id)
      return this.detail
    },
  },
})
