import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useInviteStore } from '../invite'
import { apiInvite } from '@/api/invite'

vi.mock('@/api/invite', () => ({
  apiInvite: {
    summary: vi.fn(),
    codes: vi.fn(),
    createCode: vi.fn(),
    records: vi.fn(),
    transfer: vi.fn(),
  },
}))

describe('useInviteStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
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
      register_url_prefix: 'https://x/register?code=',
    })
    const store = useInviteStore()
    await store.fetchCodes()
    await store.createCode()
    expect(store.codes[0].code).toBe('NEWCODE1')
    expect(store.codes).toHaveLength(2)
  })
})
