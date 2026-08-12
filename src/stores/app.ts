import { defineStore } from 'pinia'
import type { ThemeMode } from '@/utils/storage'
import { getThemeMode, setThemeMode, setApiBase, storageLike, flushStorage } from '@/utils/storage'
import { getApiBaseUrl } from '@/utils/http'

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
    apiBase: getApiBaseUrl(),
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
    async setApiBase(url: string) {
      this.apiBase = url
      setApiBase(url)
      // Tauri 下 plugin-store 异步落盘,先 flush 再 reload,避免新地址未落地就销毁页面
      await flushStorage()
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
