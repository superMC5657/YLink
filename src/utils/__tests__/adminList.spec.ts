import { describe, expect, it } from 'vitest'
import { batchAuditTarget, clampPage, clampPageSize, withPage } from '@/utils/adminList'

// claim ica2-7:行为变更=Admin 列表机械收敛(分页钳制/批量/audit-shape 同源),
// 视觉与交互意图不动;必须不变=正常分页透传、后端钳制语义一致、摘要 batch:<count>;
// 最怕的失败=钳制改错致列表取数漂移、摘要形状回退裸 ID 列表。
describe('adminList', () => {
  it('clampPage: 非法回落 1,正常下取整透传', () => {
    expect(clampPage(0)).toBe(1)
    expect(clampPage(-3)).toBe(1)
    expect(clampPage(Number.NaN)).toBe(1)
    expect(clampPage(1)).toBe(1)
    expect(clampPage(2.7)).toBe(2)
  })

  it('clampPageSize: 越界回落 10(与后端逐 List 规则同形)', () => {
    expect(clampPageSize(0)).toBe(10)
    expect(clampPageSize(-1)).toBe(10)
    expect(clampPageSize(51)).toBe(10)
    expect(clampPageSize(Number.NaN)).toBe(10)
    expect(clampPageSize(10)).toBe(10)
    expect(clampPageSize(50)).toBe(50)
  })

  it('withPage: 只钳制分页键,其余筛选原样透传', () => {
    expect(withPage({ page: 0, page_size: 99, keyword: 'a' })).toEqual({
      page: 1,
      page_size: 10,
      keyword: 'a',
    })
    expect(withPage({ page: 2, page_size: 20 })).toEqual({ page: 2, page_size: 20 })
    expect(withPage({ keyword: 'a' })).toEqual({ keyword: 'a' })
  })

  it('batchAuditTarget: 批量摘要 batch:<count>,与后端同形', () => {
    expect(batchAuditTarget(2)).toBe('batch:2')
    expect(batchAuditTarget(0)).toBe('batch:0')
  })
})
