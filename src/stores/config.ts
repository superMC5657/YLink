import { defineStore } from 'pinia'
import { apiConfig } from '@/api/user'
import type { SiteConfig } from '@/types/api'
import { getItem, setItem } from '@/utils/storage'

interface ConfigState {
  config: SiteConfig | null
  loadedAt: number
}

const CACHE_TTL = 60 * 1000 // 60s:站点配置含运营可调项(代理政策/注册开关/支付方式),需及时反映管理后台改动;与后端 Redis 60s 缓存对齐
/** 缓存版本:站点品牌/配置结构变更时 +1,旧缓存立即失效(如品牌更名后用户仍读到旧 site_name) */
const CACHE_VERSION = 2

interface ConfigCache {
  config: SiteConfig | null
  loadedAt: number
  version: number
}

export const useConfigStore = defineStore('config', {
  state: (): ConfigState => ({
    config: null,
    loadedAt: 0,
  }),
  getters: {
    siteName: (s) => s.config?.site_name ?? import.meta.env.VITE_APP_NAME ?? 'YLink',
    paymentMethods: (s) => s.config?.payment_methods ?? [],
    registerEnabled: (s) => s.config?.register_enabled ?? true,
    inviteCodeRequired: (s) => s.config?.invite_code_required ?? false,
  },
  actions: {
    /** 启动拉取,带 60s 本地缓存;force=true 强制刷新(政策/开关类展示页用) */
    async fetchConfig(force = false) {
      const cached = getItem<ConfigCache>('configCache')
      if (
        !force &&
        cached?.config &&
        cached.version === CACHE_VERSION &&
        Date.now() - cached.loadedAt < CACHE_TTL
      ) {
        this.config = cached.config
        this.loadedAt = cached.loadedAt
        return
      }
      const config = await apiConfig.fetch()
      this.config = config
      this.loadedAt = Date.now()
      setItem('configCache', { config, loadedAt: this.loadedAt, version: CACHE_VERSION })
    },
  },
})
