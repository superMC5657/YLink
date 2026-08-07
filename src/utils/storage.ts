/**
 * 持久化适配层:浏览器 → localStorage;Tauri → @tauri-apps/plugin-store。
 * 一期以浏览器实现为主,Tauri 分支按文档预留(见 docs/frontend/data-layer.md §3)。
 */
const PREFIX = 'app:'

export interface StorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

function detectBackend(): StorageLike {
  // Tauri 环境使用插件 store(JSON 文件);Web 使用 localStorage
  if (typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window) {
    // 动态加载避免 Web 打包引入 Rust 插件
    // 一期 Web 优先,统一走 localStorage;接入 Tauri 时替换为 plugin-store 实现
  }
  return window.localStorage
}

const backend: StorageLike = detectBackend()

export function getItem<T = unknown>(key: string): T | null {
  const raw = backend.getItem(PREFIX + key)
  if (raw === null || raw === undefined) return null
  try {
    return JSON.parse(raw) as T
  } catch {
    return raw as unknown as T
  }
}

export function setItem(key: string, value: unknown): void {
  backend.setItem(PREFIX + key, JSON.stringify(value))
}

export function removeItem(key: string): void {
  backend.removeItem(PREFIX + key)
}

/** 主题模式:light / dark / system */
export type ThemeMode = 'light' | 'dark' | 'system'

export function getThemeMode(): ThemeMode {
  return getItem<ThemeMode>('theme') ?? 'system'
}

export function setThemeMode(mode: ThemeMode): void {
  setItem('theme', mode)
}

/** 后端 API 地址(设置页可改) */
export function getApiBase(): string | null {
  return getItem<string>('apiBase')
}

export function setApiBase(url: string): void {
  setItem('apiBase', url)
}
