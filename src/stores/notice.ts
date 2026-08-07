import { defineStore } from 'pinia'
import { apiNotice } from '@/api/notice'
import type { Notice } from '@/types/api'

interface NoticeState {
  list: Notice[]
  total: number
  loading: boolean
}

export const useNoticeStore = defineStore('notice', {
  state: (): NoticeState => ({
    list: [],
    total: 0,
    loading: false,
  }),
  actions: {
    async fetch(pageSize = 5) {
      this.loading = true
      try {
        const data = await apiNotice.fetch({ page: 1, page_size: pageSize })
        this.list = data?.list ?? []
        this.total = data.total
      } finally {
        this.loading = false
      }
    },
  },
})
