import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import { withLoading } from '@/composables/withLoading'
import type { Plan, PlanListResp } from '@/types/api'

/**
 * plan module 的传输核：本域 URL 与响应形态的唯一归属。
 * store 动作与 `@/api/plan` 适配缝共用同一核（一处修改，N 处调用）。
 */
export const planTransport = {
  fetch: () => http.get<PlanListResp>('/plans'),
}

interface PlanState {
  list: Plan[]
  loaded: boolean
  loading: boolean
}

export const usePlanStore = defineStore('plan', {
  state: (): PlanState => ({
    list: [],
    loaded: false,
    loading: false,
  }),
  actions: {
    async fetch(force = false) {
      if (this.loaded && !force) return
      // ADR-0001：service 层 make() 保证 list 非空，直接赋值
      await withLoading(this, async () => {
        const data = await planTransport.fetch()
        this.list = data.list
        this.loaded = true
      })
    },
    findById(id: number): Plan | undefined {
      return this.list.find((p) => p.id === id)
    },
  },
})
