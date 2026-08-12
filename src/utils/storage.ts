/**
 * 持久化适配层:浏览器 → localStorage;Tauri → @tauri-apps/plugin-store(app-settings.json)。
 * 见 docs/frontend/data-layer.md §3。
 *
 * persistedstate(同步 StorageLike 接口)与 plugin-store(异步 API)冲突,这里用
 * 「同步 facade + 异步落盘」桥接:
 *   - 启动时 `initStorage()` 预载 plugin-store 全部键到内存 Map(getItem/setItem 同步走内存);
 *   - setItem/removeItem 同步更新内存,再排队异步写回 plugin-store(防抖由插件 autoSave 处理);
 *   - 浏览器分支完全等价 localStorage,行为不变。
 */
import { isTauri } from './platform'

const PREFIX = 'app:'

/** 同步读写接口(与 persistedstate StorageLike 兼容) */
export interface StorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

type TauriStore = {
  get(key: string): Promise<unknown>
  set(key: string, value: unknown): Promise<void>
  delete(key: string): Promise<boolean>
  keys(): Promise<string[]>
}

let tauriStore: TauriStore | null = null
// 内存缓存:plugin-store 异步落盘 + 同步读取的桥
const cache = new Map<string, string>()
let saveQueue: Promise<void> = Promise.resolve()

/** 旧 WebView localStorage → plugin-store 一次性迁移完成标记(key 带 app: 前缀) */
const MIGRATED_KEY = PREFIX + '_legacy:migrated:v1'

/**
 * 一次性迁移:迁移前 Tauri 用户的 token/设置存在 WebView localStorage(app: 前缀)。
 * 预载 plugin-store 后,把 plugin-store 缺失的旧键导入并落盘;plugin-store 已有同键时
 * 保留新值不覆盖。迁移完成写标记,避免每次启动重复遍历(见 docs/reviews/review-0.6.0.md P2)。
 */
async function migrateLegacyLocalStorage(): Promise<void> {
  if (cache.has(MIGRATED_KEY)) return
  try {
    const legacyKeys: string[] = []
    for (let i = 0; i < window.localStorage.length; i++) {
      const k = window.localStorage.key(i)
      if (k && k.startsWith(PREFIX)) legacyKeys.push(k)
    }
    for (const k of legacyKeys) {
      if (cache.has(k)) continue // plugin-store 已有同键,不覆盖新值
      const v = window.localStorage.getItem(k)
      if (v === null) continue
      const ok = await persistToStore(k, v)
      if (!ok) return // 落盘失败:不写标记,下次启动重试,避免旧键永久不可达
      cache.set(k, v)
    }
    cache.set(MIGRATED_KEY, '1')
    await persistToStore(MIGRATED_KEY, '1')
  } catch {
    // 迁移失败静默:不写标记,下次启动重试
  }
}

/** 启动时调用一次:预载 plugin-store 到内存并迁移旧存储(仅 Tauri;Web 端 no-op)。 */
export async function initStorage(): Promise<void> {
  if (!isTauri()) return
  try {
    const { load } = await import('@tauri-apps/plugin-store')
    tauriStore = (await load('app-settings.json')) as unknown as TauriStore
    for (const k of await tauriStore.keys()) {
      const v = await tauriStore.get(k)
      if (v !== undefined && v !== null) cache.set(k, JSON.stringify(v))
    }
    await migrateLegacyLocalStorage()
  } catch {
    // 预载失败静默:后续 setItem 仍会尝试落盘
    tauriStore = null
  }
}

function rawGet(key: string): string | null {
  if (isTauri()) return cache.get(key) ?? null
  return window.localStorage.getItem(key)
}

function rawSet(key: string, value: string): void {
  if (isTauri()) {
    cache.set(key, value)
    void persistToStore(key, value)
    return
  }
  window.localStorage.setItem(key, value)
}

function rawRemove(key: string): void {
  if (isTauri()) {
    cache.delete(key)
    if (tauriStore) {
      saveQueue = saveQueue.then(() => tauriStore!.delete(key).then(() => {})).catch(() => {})
    }
    return
  }
  window.localStorage.removeItem(key)
}

/** plugin-store 落盘:JSON 字符串还原为结构化值写入(防抖由插件 autoSave)。返回是否成功。 */
async function persistToStore(key: string, value: string): Promise<boolean> {
  if (!tauriStore) return false
  let parsed: unknown
  try {
    parsed = JSON.parse(value)
  } catch {
    parsed = value
  }
  const op = saveQueue.then(async () => {
    await tauriStore!.set(key, parsed)
  })
  // 队列吞错避免断链(前一次失败不应阻塞后续写);本次结果单独取
  saveQueue = op.catch(() => {})
  try {
    await op
    return true
  } catch {
    return false
  }
}

/** 供 persistedstate 使用的同步适配(storage.ts 统一前缀 'app:') */
export const storageLike: StorageLike = {
  getItem: (k) => rawGet(PREFIX + k),
  setItem: (k, v) => rawSet(PREFIX + k, v),
  removeItem: (k) => rawRemove(PREFIX + k),
}

/** 业务层通用读写(自动 JSON 序列化) */
export function getItem<T = unknown>(key: string): T | null {
  const raw = rawGet(PREFIX + key)
  if (raw === null || raw === undefined) return null
  try {
    return JSON.parse(raw) as T
  } catch {
    return raw as unknown as T
  }
}

export function setItem(key: string, value: unknown): void {
  rawSet(PREFIX + key, JSON.stringify(value))
}

export function removeItem(key: string): void {
  rawRemove(PREFIX + key)
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

/**
 * 等待所有排队落盘完成(Tauri;Web 端 no-op)。
 * reload/退出前调用,避免 plugin-store 异步写尚未落地就销毁页面导致数据丢失。
 */
export async function flushStorage(): Promise<void> {
  if (!isTauri()) return
  await saveQueue
}
