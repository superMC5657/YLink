import { describe, expect, it, vi } from 'vitest'
import { reportBatchResult, useAdminList } from '../useAdminList'
import type { PageResult } from '@/types/api'

interface Row {
  id: number
  name: string
}

function pageOf(list: Row[], total: number): PageResult<Row> {
  return { list, total, page: 1, page_size: 10 }
}

const r1: Row = { id: 1, name: 'a' }
const r2: Row = { id: 2, name: 'b' }

// claim:列表深模块是取数/分页/多选/汇总的唯一归属;行为=钳制+空页回退+失败保旧表+文案逐字。
// 最怕的失败:改一处各域行为分叉(删空页白屏/失败清表/汇总文案漂移)。
describe('useAdminList', () => {
  it('load 写入 list/total,loading 复位,分页键随行', async () => {
    const fetch = vi.fn(async () => pageOf([r1, r2], 2))
    const { loading, list, total, load } = useAdminList<Row, Record<string, unknown>>({ fetch })
    expect(loading.value).toBe(false)
    const p = load()
    expect(loading.value).toBe(true)
    await p
    expect(loading.value).toBe(false)
    expect(list.value).toEqual([r1, r2])
    expect(total.value).toBe(2)
    expect(fetch).toHaveBeenCalledWith({ page: 1, page_size: 10 })
  })

  it('applyFilters 换筛选并回首页', async () => {
    const fetch = vi.fn(async () => pageOf([], 0))
    const { page, applyFilters, onPageChange } = useAdminList<Row, { keyword?: string }>({
      fetch,
    })
    await onPageChange(3)
    expect(page.value).toBe(3)
    await applyFilters({ keyword: 'x' })
    expect(page.value).toBe(1)
    expect(fetch).toHaveBeenLastCalledWith({ keyword: 'x', page: 1, page_size: 10 })
  })

  it('onPageChange 钳制非法页', async () => {
    const fetch = vi.fn(async () => pageOf([], 0))
    const { page, onPageChange } = useAdminList<Row, Record<string, unknown>>({ fetch })
    await onPageChange(0)
    expect(page.value).toBe(1)
    await onPageChange(2.7)
    expect(page.value).toBe(2)
    expect(fetch).toHaveBeenLastCalledWith({ page: 2, page_size: 10 })
  })

  it('setPageSize 钳制并回首页', async () => {
    const fetch = vi.fn(async () => pageOf([], 0))
    const { page, pageSize, setPageSize, onPageChange } = useAdminList<
      Row,
      Record<string, unknown>
    >({
      fetch,
      pageSize: 20,
    })
    expect(pageSize.value).toBe(20)
    await onPageChange(2)
    await setPageSize(99)
    expect(pageSize.value).toBe(10)
    expect(page.value).toBe(1)
  })

  it('失败保留旧表,loading 复位,错误重抛', async () => {
    const fetch = vi
      .fn<() => Promise<PageResult<Row>>>()
      .mockResolvedValueOnce(pageOf([r1], 1))
      .mockRejectedValueOnce(new Error('boom'))
    const { list, loading, load } = useAdminList<Row, Record<string, unknown>>({ fetch })
    await load()
    await expect(load()).rejects.toThrow('boom')
    expect(loading.value).toBe(false)
    expect(list.value).toEqual([r1])
  })

  it('翻到删空页自动退一页重取(单步有界)', async () => {
    const fetch = vi
      .fn<() => Promise<PageResult<Row>>>()
      .mockResolvedValueOnce(pageOf([], 5))
      .mockResolvedValueOnce(pageOf([r1], 5))
      .mockResolvedValueOnce(pageOf([r1], 5))
    const { list, page, load, onPageChange } = useAdminList<Row, Record<string, unknown>>({
      fetch,
    })
    await onPageChange(2)
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(page.value).toBe(1)
    expect(list.value).toEqual([r1])
    await load()
    expect(fetch).toHaveBeenCalledTimes(3)
  })

  it('首页空即接受,不重取;total=0 不回退', async () => {
    const fetch = vi.fn(async () => pageOf([], 0))
    const { list, load } = useAdminList<Row, Record<string, unknown>>({ fetch })
    await load()
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(list.value).toEqual([])
  })

  it('非分页 {list} 响应 total 取长度', async () => {
    const fetch = vi.fn(async () => ({ list: [r1, r2] }))
    const { list, total, load } = useAdminList<Row, Record<string, unknown>>({ fetch })
    await load()
    expect(list.value).toEqual([r1, r2])
    expect(total.value).toBe(2)
  })

  it('onResponse 透出每次成功响应', async () => {
    const seen: unknown[] = []
    const fetch = vi.fn(async () => pageOf([r1], 1))
    const { load } = useAdminList<Row, Record<string, unknown>>({
      fetch,
      onResponse: (res) => seen.push(res),
    })
    await load()
    expect(seen).toHaveLength(1)
  })

  it('多选:行选/全选/清空/allChecked', async () => {
    const fetch = vi.fn(async () => pageOf([r1, r2], 2))
    const { load, selectedIds, isSelected, toggleRow, toggleAll, clearSelection, allChecked } =
      useAdminList<Row, Record<string, unknown>>({ fetch })
    await load()
    expect(allChecked.value).toBe(false)
    toggleRow(1, true)
    expect(isSelected(1)).toBe(true)
    expect(isSelected(2)).toBe(false)
    toggleRow(1, true)
    expect(selectedIds.value).toEqual([1])
    toggleAll(true)
    expect(allChecked.value).toBe(true)
    expect(selectedIds.value).toEqual([1, 2])
    toggleAll(false)
    expect(selectedIds.value).toEqual([])
    toggleRow(2, true)
    toggleRow(2, false)
    expect(selectedIds.value).toEqual([])
    clearSelection()
    toggleRow(1, true)
    clearSelection()
    expect(selectedIds.value).toEqual([])
  })

  it('多选缺数字 id 抛错(防静默错选)', async () => {
    const fetch = vi.fn(async () => ({ list: [{ n: 1 }] }) as never)
    const { load, toggleAll } = useAdminList<{ n: number }, Record<string, unknown>>({ fetch })
    await load()
    expect(() => toggleAll(true)).toThrow('useAdminList')
  })

  it('getId 自定义键', async () => {
    const fetch = vi.fn(async () => ({ list: [{ no: 'A1' }] }) as never)
    const { load, toggleAll, selectedIds } = useAdminList<{ no: string }, Record<string, unknown>>({
      fetch,
      getId: (item) => Number(item.no.slice(1)),
    })
    await load()
    toggleAll(true)
    expect(selectedIds.value).toEqual([1])
  })

  it('reportBatch 文案逐字:全成功 / 部分失败', () => {
    const notify = { success: vi.fn(), warning: vi.fn() }
    const { reportBatch } = useAdminList<Row, Record<string, unknown>>({
      fetch: vi.fn(),
      notify,
      failedWord: '失败',
    })
    reportBatch('批量操作', { success: 3, failed: [] })
    expect(notify.success).toHaveBeenCalledWith('批量操作: 3')
    reportBatch('批量操作', { success: 2, failed: [{ id: 7, reason: '无权限' }] })
    expect(notify.warning).toHaveBeenCalledWith('批量操作: 2 / 失败 1 — #7 无权限')
  })

  it('reportBatch 无 notify 为空操作', () => {
    const { reportBatch } = useAdminList<Row, Record<string, unknown>>({ fetch: vi.fn() })
    expect(() => reportBatch('t', { success: 1, failed: [] })).not.toThrow()
  })

  it('reportBatchResult 纯函数同形', () => {
    const notify = { success: vi.fn(), warning: vi.fn() }
    reportBatchResult(notify, 't', { success: 1, failed: [] }, '败')
    expect(notify.success).toHaveBeenCalledWith('t: 1')
    reportBatchResult(undefined, 't', { success: 1, failed: [] }, '败')
  })
})
