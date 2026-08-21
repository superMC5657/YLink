import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useInviteStore } from '../invite'
import { apiInvite } from '@/api/invite'
import { isTauri } from '@/utils/platform'

vi.mock('@/api/invite', () => ({
  apiInvite: {
    summary: vi.fn(),
    codes: vi.fn(),
    createCode: vi.fn(),
    records: vi.fn(),
    transfer: vi.fn(),
  },
}))
vi.mock('@/utils/platform', () => ({ isTauri: vi.fn(() => false) }))

describe('useInviteStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(isTauri).mockReturnValue(false)
  })

  it('fetchSummary 写入统计', async () => {
    vi.mocked(apiInvite.summary).mockResolvedValue({
      commission_balance: 88.6,
      commission_rate: 40,
      registered_count: 12,
      total_commission: 126.4,
      pending_commission: 6.0,
    })
    const store = useInviteStore()
    await store.fetchSummary()
    expect(store.summary?.commission_balance).toBe(88.6)
    expect(store.summary?.registered_count).toBe(12)
  })

  it('transfer 后同步 commission_balance', async () => {
    vi.mocked(apiInvite.summary).mockResolvedValue({
      commission_balance: 88.6,
      commission_rate: 40,
      registered_count: 0,
      total_commission: 0,
      pending_commission: 0,
    })
    vi.mocked(apiInvite.transfer).mockResolvedValue({
      balance: 188.6,
      commission_balance: 0,
    })
    const store = useInviteStore()
    await store.fetchSummary()
    await store.transfer(88.6)
    expect(store.summary?.commission_balance).toBe(0)
  })

  it('createCode 插入列表头部', async () => {
    vi.mocked(apiInvite.createCode).mockResolvedValue({
      code: 'NEWCODE1',
      used_count: 0,
      created_at: '2026-08-07T00:00:00+08:00',
    })
    vi.mocked(apiInvite.codes).mockResolvedValue({
      list: [{ code: 'OLD', used_count: 1, created_at: '2026-07-01T00:00:00+08:00' }],
      limit: 5,
      register_url_prefix: '/#/register?code=',
    })
    const store = useInviteStore()
    await store.fetchCodes()
    await store.createCode()
    expect(store.codes[0].code).toBe('NEWCODE1')
    expect(store.codes).toHaveLength(2)
  })

  it('effectiveRegisterUrlPrefix 用当前页面 origin 拼接(区分本地 5174 / Caddy 80 / 生产 443),带 hash 路由 #/', () => {
    const store = useInviteStore()
    // getter 不读后端返回的 registerUrlPrefix(契约占位,仅返回路径后缀 /#/register?code=),
    // 而是直接用 window.location.origin —— jsdom 默认 origin 为 http://localhost:3000
    expect(store.effectiveRegisterUrlPrefix).toBe('http://localhost:3000/#/register?code=')
  })

  it('Tauri WebView 下兜底相对路径,不用 origin 拼接(Windows 的 http://tauri.localhost 也不可分享)', () => {
    vi.mocked(isTauri).mockReturnValue(true)
    const store = useInviteStore()
    expect(store.effectiveRegisterUrlPrefix).toBe('/#/register?code=')
  })
})
