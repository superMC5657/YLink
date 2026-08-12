import { defineStore } from 'pinia'
import type { ThemeMode } from '@/utils/storage'
import { getThemeMode, setThemeMode, setApiBase, getApiBase, storageLike } from '@/utils/storage'

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
    /** 将当前模式写入 <html data-theme>,同步 Naive darkTheme,并通知 Tauri 窗口标题栏 */
    applyTheme() {
      const theme = resolveTheme(this.themeMode)
      document.documentElement.setAttribute('data-theme', theme)
      document.dispatchEvent(
        new CustomEvent('theme-changed', { detail: { theme, dark: theme === 'dark' } }),
      )
      // Tauri 桌面端:标题栏亮暗跟随(Rust 侧 set_window_theme)
      if (typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window) {
        void import('@tauri-apps/api/core').then(({ invoke }) => {
          void invoke('set_window_theme', { dark: theme === 'dark' })
        })
      }
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
    storage: storageLike,
  },
})
