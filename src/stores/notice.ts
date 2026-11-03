import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import { withLoading } from '@/composables/withLoading'
import type { Notice, PageQuery, PageResult } from '@/types/api'

/**
 * notice module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/notice` 适配缝共用同一核（一处修改，N 处调用）。
 */
export const noticeTransport = {
  fetch: (q: PageQuery = {}) =>
    http.get<PageResult<Notice>>('/notices', { query: { page: q.page, page_size: q.page_size } }),
}

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
      // ADR-0001：/notices 走 PageOK，list 非空保证在后端，直接赋值
      await withLoading(this, async () => {
        const data = await noticeTransport.fetch({ page: 1, page_size: pageSize })
        this.list = data.list
        this.total = data.total
      })
    },
  },
})
