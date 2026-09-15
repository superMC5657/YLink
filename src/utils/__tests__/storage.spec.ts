import { beforeEach, describe, expect, it, vi } from 'vitest'

/** 内存版 plugin-store mock:记录 set 调用,便于断言落盘;failSetOnce 注入一次落盘失败 */
function makeStore(initial: Record<string, unknown> = {}, opts: { failSetOnce?: boolean } = {}) {
  const map = new Map<string, unknown>(Object.entries(initial))
  const sets: Array<{ key: string; value: unknown }> = []
  const deletes: string[] = []
  const load = vi.fn(async () => ({
    get: async (k: string) => (map.has(k) ? map.get(k) : undefined),
    set: async (k: string, v: unknown) => {
      if (opts.failSetOnce) {
        opts.failSetOnce = false
        throw new Error('persist failed')
      }
      sets.push({ key: k, value: v })
      map.set(k, v)
    },
    delete: async (k: string) => {
      deletes.push(k)
      map.delete(k)
    },
    keys: async () => [...map.keys()],
  }))
  return { map, sets, deletes, load }
}

/** 模拟 Tauri 环境重新加载 storage 模块(cache 每次全新),返回模块与 store 句柄 */
async function loadTauriStorage(
  initial: Record<string, unknown> | ReturnType<typeof makeStore> = {},
) {
  const store = 'map' in initial ? (initial as ReturnType<typeof makeStore>) : makeStore(initial)
  vi.resetModules()
  // 注意:mock 路径须与 storage.ts 内部解析结果一致(相对 src/utils),否则 isTauri mock 不生效
  vi.doMock('@/utils/platform', () => ({ isTauri: () => true }))
  vi.doMock('@tauri-apps/plugin-store', () => ({ load: store.load }))
  const mod = await import('../storage')
  return { mod, store }
}

describe('storage Tauri 分支', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('预载 plugin-store 到内存,getItem 走缓存', async () => {
    const { mod } = await loadTauriStorage({ 'app:apiBase': 'https://new/api/v1' })
    await mod.initStorage()
    expect(mod.getItem('apiBase')).toBe('https://new/api/v1')
  })

  it('setItem 写入内存并经 flushStorage 落盘到 store', async () => {
    const { mod, store } = await loadTauriStorage()
    await mod.initStorage()
    mod.setItem('theme', 'dark')
    // 内存立即生效
    expect(mod.getItem('theme')).toBe('dark')
    // 刷写后 store 落盘
    await mod.flushStorage()
    expect(store.map.get('app:theme')).toBe('dark')
  })

  it('removeItem 从内存和 store 中移除键', async () => {
    const { mod, store } = await loadTauriStorage({ 'app:theme': 'dark' })
    await mod.initStorage()
    expect(mod.getItem('theme')).toBe('dark')
    mod.removeItem('theme')
    expect(mod.getItem('theme')).toBeNull()
    await mod.flushStorage()
    expect(store.map.has('app:theme')).toBe(false)
  })

  it('预载失败(store load 抛错)静默,不影响后续操作', async () => {
    const store = makeStore()
    store.load.mockRejectedValueOnce(new Error('store unavailable'))
    const { mod } = await loadTauriStorage(store)
    await mod.initStorage() // 不抛错
    expect(mod.getItem('auth')).toBeNull()
  })

  it('flushStorage 等待排队落盘完成(Tauri)', async () => {
    const { mod, store } = await loadTauriStorage()
    await mod.initStorage()
    mod.setItem('apiBase', 'https://x/api/v1')
    await mod.flushStorage()
    expect(store.map.get('app:apiBase')).toBe('https://x/api/v1')
  })
})
