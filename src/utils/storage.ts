/**
 * 持久化适配层:浏览器 → localStorage;Tauri → @tauri-apps/plugin-store(app-settings.json)。
 * 见 docs/frontend/data-layer.md §3。
 *
 * 在会话模块 seam 内只承担存储适配:会话建立/终结/刷新成功后的刷盘由会话模块集中触发,
 * 调用方不自行拼 flush 时序;本模块只暴露同步读写 facade 与 flushStorage 落盘语义。
 *
 * persistedstate(同步 StorageLike 接口)与 plugin-store(异步接口)冲突,这里用
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

// ---------- 高频写防抖:100ms 内的多次写合并为一次批量落盘 ----------
const FLUSH_DELAY = 100
/** key → 待写值(null 表示删除) */
const pendingWrites = new Map<string, string | null>()
let flushTimer: ReturnType<typeof setTimeout> | null = null

function enqueueWrite(key: string, value: string | null): void {
  pendingWrites.set(key, value)
  flushTimer ??= setTimeout(() => {
    flushTimer = null
    void flushPending()
  }, FLUSH_DELAY)
}

/** 批量落盘挂起的写/删;失败批次回滚到待写队列,随下次写入重试 */
async function flushPending(): Promise<void> {
  if (!tauriStore || pendingWrites.size === 0) {
    pendingWrites.clear()
    return
  }
  const batch = new Map(pendingWrites)
  pendingWrites.clear()
  const op = saveQueue.then(async () => {
    for (const [k, v] of batch) {
      if (v === null) {
        await tauriStore!.delete(k)
        continue
      }
      let parsed: unknown
      try {
        parsed = JSON.parse(v)
      } catch {
        parsed = v
      }
      await tauriStore!.set(k, parsed)
    }
  })
  // 队列吞错避免断链(前一次失败不应阻塞后续写);本次结果单独取
  saveQueue = op.catch(() => {})
  try {
    await op
  } catch {
    // 失败的键放回待写队列(不覆盖期间的新值),随下次 flush 重试
    for (const [k, v] of batch) {
      if (!pendingWrites.has(k)) pendingWrites.set(k, v)
    }
  }
}

/** 启动时调用一次:预载 plugin-store 到内存(仅 Tauri;Web 端 no-op)。 */
export async function initStorage(): Promise<void> {
  if (!isTauri()) return
  try {
    const { load } = await import('@tauri-apps/plugin-store')
    tauriStore = (await load('app-settings.json')) as unknown as TauriStore
    for (const k of await tauriStore.keys()) {
      const v = await tauriStore.get(k)
      if (v !== undefined && v !== null) cache.set(k, JSON.stringify(v))
    }
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
    enqueueWrite(key, value)
    return
  }
  window.localStorage.setItem(key, value)
}

function rawRemove(key: string): void {
  if (isTauri()) {
    cache.delete(key)
    enqueueWrite(key, null)
    return
  }
  window.localStorage.removeItem(key)
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
 * 等待所有挂起写与队列落盘完成(Tauri;Web 端 no-op)。
 * reload/登出前调用,避免防抖窗口内的写尚未落地就销毁页面导致数据丢失。
 */
export async function flushStorage(): Promise<void> {
  if (!isTauri()) return
  if (flushTimer) {
    clearTimeout(flushTimer)
    flushTimer = null
  }
  await flushPending()
  await saveQueue
}
