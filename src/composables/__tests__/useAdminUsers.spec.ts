import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAdminUsers } from '../useAdminUsers'
import { usersTransport } from '@/stores/admin/users'
import { useAuthStore } from '@/stores/auth'
import type { AdminUserItem } from '@/types/api'

vi.mock('@/stores/admin/users', () => ({
  usersTransport: {
    users: vi.fn(),
    updateUser: vi.fn(),
    adjustBalance: vi.fn(),
    exportUsersCSV: vi.fn(),
    batchUsers: vi.fn(),
    sendMail: vi.fn(),
    resetUserSubToken: vi.fn(),
  },
}))

const ui = vi.hoisted(() => ({
  success: vi.fn(),
  warning: vi.fn(),
  dialogWarning: vi.fn(),
  dialogInfo: vi.fn(),
}))

vi.mock('naive-ui', () => ({
  useMessage: () => ({ success: ui.success, warning: ui.warning }),
  useDialog: () => ({ warning: ui.dialogWarning, info: ui.dialogInfo }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (k: string) => k, te: () => false }),
}))

function user(over: Partial<AdminUserItem> = {}): AdminUserItem {
  return {
    id: 10086,
    email: '2734921923@qq.com',
    role: 0,
    balance: 198.5,
    commission_balance: 42.8,
    is_banned: false,
    invite_by_id: null,
    plan_id: 1,
    expired_at: null,
    transfer_enable: 107374182400,
    u: 0,
    d: 0,
    created_at: '2026-05-20T10:00:00+08:00',
    ...over,
  }
}

function pageOf(list: AdminUserItem[], total: number) {
  return { list, total, page: 1, page_size: 10 }
}

type DialogOpts = { onPositiveClick?: () => unknown }
let captured: DialogOpts[]

function confirmLastDialog() {
  const opts = captured[captured.length - 1]
  return opts.onPositiveClick?.()
}

// claim:用户域组合(取数/状态/变更/批量/错误恢复)走传输核,不直拼 URL;
// 自操作拦截与批量汇总是组合层语义,测组合不断言 URL。
// 最怕的失败:拦截漏判致自己封自己、批量后不清选中、失败清表。
describe('useAdminUsers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    captured = []
    ui.dialogWarning.mockImplementation((opts: DialogOpts) => {
      captured.push(opts)
    })
    const auth = useAuthStore()
    auth.user = { id: 1, email: 'admin@example.com', role: 1 }
  })

  it('load 写入 list/total,首参不带多余筛选', async () => {
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([user()], 1))
    const u = useAdminUsers()
    await u.load()
    expect(usersTransport.users).toHaveBeenCalledWith({ page: 1, page_size: 10 })
    expect(u.list.value).toHaveLength(1)
    expect(u.total.value).toBe(1)
    expect(u.loading.value).toBe(false)
  })

  it('onSearch 换 keyword 并回首页', async () => {
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([], 0))
    const u = useAdminUsers()
    await u.onPageChange(3)
    expect(u.page.value).toBe(3)
    u.keyword.value = 'x'
    u.onSearch()
    await vi.waitFor(() => {
      expect(usersTransport.users).toHaveBeenCalledTimes(2)
    })
    expect(u.page.value).toBe(1)
    expect(usersTransport.users).toHaveBeenLastCalledWith({
      keyword: 'x',
      page: 1,
      page_size: 10,
    })
  })

  it('toggleBan 自己:拦截,无弹窗无调用', () => {
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([], 0))
    const u = useAdminUsers()
    u.toggleBan(user({ id: 1, email: 'admin@example.com' }))
    expect(ui.warning).toHaveBeenCalledWith('adminUsers.cannotOperateSelf')
    expect(ui.dialogWarning).not.toHaveBeenCalled()
    expect(usersTransport.updateUser).not.toHaveBeenCalled()
  })

  it('toggleBan 他人:确认后取反封禁并重取', async () => {
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([user()], 1))
    vi.mocked(usersTransport.updateUser).mockResolvedValue(null)
    const u = useAdminUsers()
    await u.load()
    u.toggleBan(user({ is_banned: false }))
    expect(ui.dialogWarning).toHaveBeenCalledTimes(1)
    await confirmLastDialog()
    expect(usersTransport.updateUser).toHaveBeenCalledWith(10086, { banned: true })
    await vi.waitFor(() => {
      expect(usersTransport.users).toHaveBeenCalledTimes(2)
    })
  })

  it('saveBalance:调账参数透传,关弹窗并重取', async () => {
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([user()], 1))
    vi.mocked(usersTransport.adjustBalance).mockResolvedValue(null)
    const u = useAdminUsers()
    await u.load()
    u.openBalance(user())
    u.balanceAmount.value = 10
    u.balanceRemark.value = 'r'
    await u.saveBalance()
    expect(usersTransport.adjustBalance).toHaveBeenCalledWith(10086, { amount: 10, remark: 'r' })
    expect(u.balanceModal.value).toBe(false)
    await vi.waitFor(() => {
      expect(usersTransport.users).toHaveBeenCalledTimes(2)
    })
  })

  it('batchBan 选中含自己:拦截,无批量调用', async () => {
    vi.mocked(usersTransport.users).mockResolvedValue(
      pageOf([user({ id: 1, email: 'admin@example.com' })], 1),
    )
    const u = useAdminUsers()
    await u.load()
    u.toggleRow(1, true)
    u.batchBan(true)
    expect(ui.warning).toHaveBeenCalledWith('adminUsers.cannotOperateSelf')
    expect(usersTransport.batchUsers).not.toHaveBeenCalled()
  })

  it('batchBan 全成功:汇总文案精确,清选中并重取', async () => {
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([user()], 1))
    vi.mocked(usersTransport.batchUsers).mockResolvedValue({ success: 2, failed: [] })
    const u = useAdminUsers()
    await u.load()
    u.toggleRow(10086, true)
    u.batchBan(true)
    await confirmLastDialog()
    expect(usersTransport.batchUsers).toHaveBeenCalledWith({ action: 'ban', ids: [10086] })
    expect(ui.success).toHaveBeenCalledWith('adminUsers.batchDone: 2')
    expect(u.selectedIds.value).toEqual([])
    await vi.waitFor(() => {
      expect(usersTransport.users).toHaveBeenCalledTimes(2)
    })
  })

  it('batchBan 部分失败:汇总列明细,失败词现取', async () => {
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([user()], 1))
    vi.mocked(usersTransport.batchUsers).mockResolvedValue({
      success: 2,
      failed: [{ id: 9, reason: '不在' }],
    })
    const u = useAdminUsers()
    await u.load()
    u.toggleRow(10086, true)
    u.batchBan(false)
    await confirmLastDialog()
    expect(ui.warning).toHaveBeenCalledWith(
      'adminUsers.batchDone: 2 / adminUsers.failed 1 — #9 不在',
    )
  })

  it('取数失败:保留旧表,loading 复位,错误重抛', async () => {
    vi.mocked(usersTransport.users)
      .mockResolvedValueOnce(pageOf([user()], 1))
      .mockRejectedValueOnce(new Error('boom'))
    const u = useAdminUsers()
    await u.load()
    await expect(u.load()).rejects.toThrow('boom')
    expect(u.loading.value).toBe(false)
    expect(u.list.value).toHaveLength(1)
  })

  it('exportCSV 走下载链', async () => {
    const blob = new Blob(['a'], { type: 'text/csv' })
    vi.mocked(usersTransport.users).mockResolvedValue(pageOf([], 0))
    vi.mocked(usersTransport.exportUsersCSV).mockResolvedValue(blob)
    const createObjectURL = vi.fn(() => 'blob:x')
    const revokeObjectURL = vi.fn()
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const origCreate = URL.createObjectURL
    const origRevoke = URL.revokeObjectURL
    URL.createObjectURL = createObjectURL as typeof URL.createObjectURL
    URL.revokeObjectURL = revokeObjectURL as typeof URL.revokeObjectURL
    try {
      const u = useAdminUsers()
      u.keyword.value = 'q'
      await u.exportCSV()
      expect(usersTransport.exportUsersCSV).toHaveBeenCalledWith({ keyword: 'q' })
      expect(createObjectURL).toHaveBeenCalledWith(blob)
      expect(clickSpy).toHaveBeenCalledTimes(1)
      expect(u.exporting.value).toBe(false)
    } finally {
      URL.createObjectURL = origCreate
      URL.revokeObjectURL = origRevoke
      clickSpy.mockRestore()
    }
  })
})
