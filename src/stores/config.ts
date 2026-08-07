import { defineStore } from 'pinia'
import { apiConfig } from '@/api/user'
import type { SiteConfig } from '@/types/api'
import { getItem, setItem } from '@/utils/storage'

interface ConfigState {
  config: SiteConfig | null
  loadedAt: number
}

const CACHE_TTL = 24 * 60 * 60 * 1000 // 24h 缓存

export const useConfigStore = defineStore('config', {
  state: (): ConfigState => ({
    config: null,
    loadedAt: 0,
  }),
  getters: {
    siteName: (s) => s.config?.site_name ?? import.meta.env.VITE_APP_NAME ?? 'NanoCloud',
    paymentMethods: (s) => s.config?.payment_methods ?? [],
  },
  actions: {
    /** 启动拉取,带 24h 本地缓存;force=true 强制刷新 */
    async fetchConfig(force = false) {
      const cached = getItem<ConfigState>('configCache')
      if (!force && cached?.config && Date.now() - cached.loadedAt < CACHE_TTL) {
        this.config = cached.config
        this.loadedAt = cached.loadedAt
        return
      }
      const config = await apiConfig.fetch()
      this.config = config
      this.loadedAt = Date.now()
      setItem('configCache', { config, loadedAt: this.loadedAt })
    },
  },
})
