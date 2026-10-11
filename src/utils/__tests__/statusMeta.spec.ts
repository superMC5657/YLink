import { describe, expect, it } from 'vitest'
import {
  statusBadge,
  statusLabel,
  statusMeta,
  statusTable,
  type StatusBadge,
  type StatusDomain,
} from '../statusMeta'
import {
  adminRoleLabel,
  adminTicketLevelBadge,
  adminTicketLevelLabel,
  adminTicketStatusBadge,
  adminTicketStatusLabel,
  agentApplyStatusBadge,
  agentApplyStatusLabel,
  commissionStatusBadge,
  commissionStatusLabel,
  orderStatusBadge,
  orderStatusLabel,
  serverStatusMeta,
  ticketLevelLabel,
  ticketStatusBadge,
  ticketStatusLabel,
} from '../format'

// claim:状态元表是全部标签/徽标映射的唯一归属;新增状态只加一行。
// 最怕的失败:用户/管理两端徽标分叉(已漂移过一次,aa7df0f)、未知状态回落不一致。
const EXPECTED_STATUSES: Record<StatusDomain, number[]> = {
  order: [0, 1, 2, 3],
  commission: [0, 1, 2],
  agentApply: [0, 1, 2],
  adminTicket: [0, 1, 2],
  adminTicketLevel: [0, 1, 2],
  ticket: [0, 1, 2],
  ticketLevel: [0, 1, 2],
  adminRole: [0, 1, 2],
  server: [1, 2, 3],
}

/** 旧导出(现为委托)与表的对照:deletion test 的另一半,删旧表不丢行为 */
const LEGACY: Record<
  StatusDomain,
  { label: (n: number) => string; badge?: (n: number) => StatusBadge }
> = {
  order: { label: orderStatusLabel, badge: orderStatusBadge },
  commission: { label: commissionStatusLabel, badge: commissionStatusBadge },
  agentApply: { label: agentApplyStatusLabel, badge: agentApplyStatusBadge },
  adminTicket: { label: adminTicketStatusLabel, badge: adminTicketStatusBadge },
  adminTicketLevel: { label: adminTicketLevelLabel, badge: adminTicketLevelBadge },
  ticket: { label: ticketStatusLabel, badge: ticketStatusBadge },
  ticketLevel: { label: ticketLevelLabel },
  adminRole: { label: adminRoleLabel },
  server: { label: (s) => serverStatusMeta(s).label },
}

describe('statusTable 完备性', () => {
  it.each(Object.entries(EXPECTED_STATUSES))('%s 覆盖全部状态(缺行即红)', (domain, statuses) => {
    expect(
      Object.keys(statusTable[domain as StatusDomain])
        .map(Number)
        .sort(),
    ).toEqual(statuses)
  })

  it('对称:用户/管理工单徽标逐状态相等(注释改断言)', () => {
    for (const s of [0, 1, 2]) {
      expect(statusBadge('ticket', s)).toBe(statusBadge('adminTicket', s))
      expect(statusBadge('ticketLevel', s)).toBe(statusBadge('adminTicketLevel', s))
    }
  })

  it('对称:用户/管理工单键分域(徽标同形,键不合并)', () => {
    expect(statusMeta('ticket', 0).labelKey).toBe('ticket.pending')
    expect(statusMeta('adminTicket', 0).labelKey).toBe('adminTickets.pending')
  })
})

describe('回落语义(与收敛前逐字一致)', () => {
  const cases: Array<[StatusDomain, number, string, StatusBadge]> = [
    ['order', 99, 'common.unknown', 'neutral'],
    ['commission', 99, 'common.unknown', 'danger'],
    ['agentApply', 99, 'common.unknown', 'danger'],
    ['adminTicket', 99, 'common.unknown', 'neutral'],
    ['adminTicketLevel', 99, 'adminTickets.low', 'danger'],
    ['ticket', 99, 'common.unknown', 'neutral'],
    ['ticketLevel', 99, 'ticket.low', 'neutral'],
    ['adminRole', 99, 'adminUsers.roleNormal', 'neutral'],
    ['server', 99, 'common.unknown', 'neutral'],
  ]
  it.each(cases)('%s 未知状态回落', (domain, status, labelKey, badge) => {
    const meta = statusMeta(domain, status)
    expect(meta.labelKey).toBe(labelKey)
    expect(meta.badge).toBe(badge)
  })

  it('server 行兼带着色,未知回落 text-sub', () => {
    expect(statusMeta('server', 1)).toEqual({
      labelKey: 'node.normal',
      badge: 'success',
      color: 'var(--c-success)',
    })
    expect(statusMeta('server', 2).color).toBe('var(--c-warning)')
    expect(statusMeta('server', 99).color).toBe('var(--c-text-sub)')
  })
})

describe('旧导出委托(删旧表不丢行为)', () => {
  it.each(Object.keys(EXPECTED_STATUSES) as StatusDomain[])(
    '%s 旧文案/徽标与表一致(含未知回落)',
    (domain) => {
      const legacy = LEGACY[domain]
      for (const s of [...EXPECTED_STATUSES[domain], 99]) {
        expect(legacy.label(s)).toBe(statusLabel(domain, s))
        if (legacy.badge) expect(legacy.badge(s)).toBe(statusBadge(domain, s))
      }
    },
  )

  it('serverStatusMeta 文案与色委托', () => {
    for (const s of [1, 2, 3, 99]) {
      const meta = statusMeta('server', s)
      expect(serverStatusMeta(s)).toEqual({
        label: statusLabel('server', s),
        color: meta.color ?? 'var(--c-text-sub)',
      })
    }
  })

  it('中文实文案抽查(setup 预载 zh-CN)', () => {
    expect(statusLabel('order', 0)).toBe('待支付')
    expect(statusLabel('adminRole', 1)).toBe('管理员')
    expect(statusLabel('adminRole', 2)).toBe('代理商')
    expect(statusLabel('adminRole', 0)).toBe('普通用户')
    expect(statusLabel('server', 1)).toBe('正常')
  })
})
