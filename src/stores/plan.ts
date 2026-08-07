import { defineStore } from 'pinia'
import { apiPlan } from '@/api/plan'
import type { Plan } from '@/types/api'

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
      this.loading = true
      try {
        const data = await apiPlan.fetch()
        this.list = data.list
        this.loaded = true
      } finally {
        this.loading = false
      }
    },
    findById(id: number): Plan | undefined {
      return this.list.find((p) => p.id === id)
    },
  },
})
