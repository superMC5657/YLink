import { describe, expect, it } from 'vitest'
import {
  batchAuditTarget,
  clampPage,
  clampPageSize,
  clampPageSizeAudit,
  withPage,
  withPageAudit,
} from '@/utils/adminList'

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

  // ADR-0004 双档:审计档缺省 20/上限 100(与 pkg/page.Audit 同值),
  // 标准档不受影响——最怕的失败=审计页选 100/页被标准档静默压回 10。
  it('clampPageSizeAudit: 审计档 20-100 透传,越界回落 20', () => {
    expect(clampPageSizeAudit(100)).toBe(100)
    expect(clampPageSizeAudit(50)).toBe(50)
    expect(clampPageSizeAudit(20)).toBe(20)
    expect(clampPageSizeAudit(101)).toBe(20)
    expect(clampPageSizeAudit(0)).toBe(20)
    expect(clampPageSizeAudit(Number.NaN)).toBe(20)
    // 标准档仍是 {10,50}
    expect(clampPageSize(100)).toBe(10)
    expect(clampPageSize(50)).toBe(50)
  })

  it('withPageAudit: 审计档查询归一,page_size=100 原样透传', () => {
    expect(withPageAudit({ page: 0, page_size: 100, action: 'ban_user' })).toEqual({
      page: 1,
      page_size: 100,
      action: 'ban_user',
    })
    expect(withPageAudit({ page: 2, page_size: 101 })).toEqual({ page: 2, page_size: 20 })
    expect(withPageAudit({ keyword: 'a' })).toEqual({ keyword: 'a' })
  })
})
