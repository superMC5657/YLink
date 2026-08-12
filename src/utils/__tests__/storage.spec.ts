import { beforeEach, describe, expect, it, vi } from 'vitest'

/** 内存版 plugin-store mock:记录 set 调用,便于断言迁移落盘;failSetOnce 注入一次落盘失败 */
function makeStore(initial: Record<string, unknown> = {}, opts: { failSetOnce?: boolean } = {}) {
  const map = new Map<string, unknown>(Object.entries(initial))
  const sets: Array<{ key: string; value: unknown }> = []
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
    delete: async (k: string) => map.delete(k),
    keys: async () => [...map.keys()],
  }))
  return { map, sets, load }
}

/** 模拟 Tauri 环境重新加载 storage 模块(cache 每次全新),返回模块与 store 句柄 */
async function loadTauriStorage(
  initial: Record<string, unknown> | ReturnType<typeof makeStore> = {},
) {
  const store = ('map' in initial ? (initial as ReturnType<typeof makeStore>) : makeStore(initial))
  vi.resetModules()
  // 注意:mock 路径须与 storage.ts 内部解析结果一致(相对 src/utils),否则 isTauri mock 不生效
  vi.doMock('@/utils/platform', () => ({ isTauri: () => true }))
  vi.doMock('@tauri-apps/plugin-store', () => ({ load: store.load }))
  const mod = await import('../storage')
  return { mod, store }
}

describe('storage Tauri 分支与旧 localStorage 迁移', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('预载 plugin-store 到内存,getItem 走缓存', async () => {
    const { mod } = await loadTauriStorage({ 'app:apiBase': 'https://new/api/v1' })
    await mod.initStorage()
    expect(mod.getItem('apiBase')).toBe('https://new/api/v1')
  })

  it('迁移旧 localStorage 键:plugin-store 缺失的键导入内存并落盘', async () => {
    localStorage.setItem('app:auth', JSON.stringify({ accessToken: 'legacy-token' }))
    localStorage.setItem('app:theme', JSON.stringify('dark'))
    const { mod, store } = await loadTauriStorage()
    await mod.initStorage()
    // 内存可读
    expect(mod.getItem('auth')).toEqual({ accessToken: 'legacy-token' })
    expect(mod.getItem('theme')).toBe('dark')
    // 已落盘到 plugin-store
    expect(store.map.get('app:auth')).toEqual({ accessToken: 'legacy-token' })
    expect(store.map.get('app:theme')).toBe('dark')
    // 写入了迁移完成标记(一次性)
    expect(store.map.has('app:_legacy:migrated:v1')).toBe(true)
  })

  it('plugin-store 已有同键时保留新值,不覆盖旧 localStorage 值', async () => {
    localStorage.setItem('app:apiBase', JSON.stringify('https://old/api/v1'))
    const { mod, store } = await loadTauriStorage({ 'app:apiBase': 'https://new/api/v1' })
    await mod.initStorage()
    expect(mod.getItem('apiBase')).toBe('https://new/api/v1')
    expect(store.map.get('app:apiBase')).toBe('https://new/api/v1')
  })

  it('迁移一次性:已迁移后再次 initStorage 不再导入新增旧键', async () => {
    localStorage.setItem('app:auth', JSON.stringify({ accessToken: 'legacy-token' }))
    const { mod } = await loadTauriStorage()
    await mod.initStorage()
    // 第一次迁移后 localStorage 新增的旧键不再导入
    localStorage.setItem('app:theme', JSON.stringify('dark'))
    await mod.initStorage()
    expect(mod.getItem('theme')).toBeNull()
    // 第一次迁移的键保留
    expect(mod.getItem('auth')).toEqual({ accessToken: 'legacy-token' })
  })

  it('预载失败(store load 抛错)静默,不迁移', async () => {
    localStorage.setItem('app:auth', JSON.stringify({ accessToken: 'legacy-token' }))
    const store = makeStore()
    store.load.mockRejectedValueOnce(new Error('store unavailable'))
    const { mod } = await loadTauriStorage(store)
    await mod.initStorage() // 不抛错
    expect(mod.getItem('auth')).toBeNull() // 未迁移成功
  })

  it('flushStorage 等待排队落盘完成(Tauri)', async () => {
    const { mod, store } = await loadTauriStorage()
    await mod.initStorage()
    mod.setItem('apiBase', 'https://x/api/v1')
    await mod.flushStorage()
    expect(store.map.get('app:apiBase')).toBe('https://x/api/v1')
  })

  it('迁移落盘失败时不写标记,下次启动可重试', async () => {
    localStorage.setItem('app:auth', JSON.stringify({ accessToken: 'legacy-token' }))
    const { mod, store } = await loadTauriStorage(makeStore({}, { failSetOnce: true }))
    await mod.initStorage() // 不抛错
    expect(mod.getItem('auth')).toBeNull() // 落盘失败,键未导入内存
    expect(store.map.has('app:auth')).toBe(false)
    expect(store.map.has('app:_legacy:migrated:v1')).toBe(false) // 不写标记 → 下次重试
  })
})
