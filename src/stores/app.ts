import { defineStore } from 'pinia'
import type { ThemeMode } from '@/utils/storage'
import { getThemeMode, setThemeMode, setApiBase, getApiBase } from '@/utils/storage'

interface AppState {
  sidebarCollapsed: boolean
  themeMode: ThemeMode
  language: string
  apiBase: string
}

function resolveTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return mode
}

export const useAppStore = defineStore('app', {
  state: (): AppState => ({
    sidebarCollapsed: false,
    themeMode: getThemeMode(),
    language: 'zh-CN',
    apiBase: getApiBase() ?? import.meta.env.VITE_API_BASE_URL ?? '/api/v1',
  }),
  getters: {
    isDark: (s) => resolveTheme(s.themeMode) === 'dark',
  },
  actions: {
    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed
    },
    setThemeMode(mode: ThemeMode) {
      this.themeMode = mode
      setThemeMode(mode)
      this.applyTheme()
    },
    /** 将当前模式写入 <html data-theme>,同步 Naive darkTheme */
    applyTheme() {
      const theme = resolveTheme(this.themeMode)
      document.documentElement.setAttribute('data-theme', theme)
      document.dispatchEvent(
        new CustomEvent('theme-changed', { detail: { theme, dark: theme === 'dark' } }),
      )
    },
    setLanguage(lang: string) {
      this.language = lang
    },
    setApiBase(url: string) {
      this.apiBase = url
      setApiBase(url)
      window.location.reload()
    },
    /** 系统主题变化时实时响应 */
    initSystemThemeListener() {
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (this.themeMode === 'system') this.applyTheme()
      })
    },
  },
  persist: {
    key: 'app',
    pick: ['sidebarCollapsed', 'themeMode', 'language', 'apiBase'],
    storage: {
      getItem: (k) => window.localStorage.getItem(`app:${k}`),
      setItem: (k, v) => window.localStorage.setItem(`app:${k}`, v),
    },
  },
})
