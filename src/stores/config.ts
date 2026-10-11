import { defineStore } from 'pinia'
import { http } from '@/utils/http'
import type { SiteConfig } from '@/types/api'
import { getItem, setItem } from '@/utils/storage'

/**
 * config module 的传输核：站点配置 URL 与响应形态的唯一归属。
 * store 的 SWR 动作与 `@/api/user` 内 `apiConfig` 适配缝共用同一核（一处修改，N 处调用）。
 */
export const configTransport = {
  fetch: () => http.get<SiteConfig>('/config', { silent: true }),
}

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
    /**
     * SWR:缓存命中且未过 TTL → 直接用;缓存存在但过期 → 立即返回旧值并后台再验证,
     * 避免首屏等待;无缓存/force → 同步等待拉取。
     * TTL 60s:站点配置含运营可调项(代理政策/注册开关/支付方式),与后端 Redis 60s 缓存对齐。
     */
    async fetchConfig(force = false) {
      const cached = getItem<ConfigCache>('configCache')
      const fresh =
        !force &&
        cached?.config &&
        cached.version === CACHE_VERSION &&
        Date.now() - cached.loadedAt < CACHE_TTL
      if (cached?.config && cached.version === CACHE_VERSION) {
        // 有缓存:先落地展示(即使已过期)
        this.config = cached.config
        this.loadedAt = cached.loadedAt
      }
      if (fresh) return
      if (cached?.config) {
        // SWR 后台再验证:不阻塞调用方,失败保留旧值
        void configTransport
          .fetch()
          .then((config) => {
            this.config = config
            this.loadedAt = Date.now()
            setItem('configCache', { config, loadedAt: this.loadedAt, version: CACHE_VERSION })
          })
          .catch(() => {})
        return
      }
      const config = await configTransport.fetch()
      this.config = config
      this.loadedAt = Date.now()
      setItem('configCache', { config, loadedAt: this.loadedAt, version: CACHE_VERSION })
    },
  },
})
