import { i18n } from '@/i18n'

/**
 * 状态元深模块(C4):(domain, status) → { labelKey, badge } 的唯一数据表。
 *
 * 收敛前 format.ts 以 ~14 个近似函数各持小表,用户/管理两端同形只靠注释维系
 * (已漂移过一次,见 aa7df0f)。本表是该映射的唯一归属:
 * 语义真不同的域各占一行(i18n 键不强行合并),形态相同的域由对称断言锁定。
 *
 * 约定:
 * - 未知 status 走同域 FALLBACK(各域回落语义与收敛前逐字一致,含三元函数的 danger 偏向);
 * - server 行兼带 color(圆点/文字着色沿用),徽标只读 badge,不再从色值反推;
 * - 新增状态只加一行 + 单测期望表一行,零边际成本。
 */
export type StatusBadge = 'warning' | 'success' | 'neutral' | 'danger'

export type StatusDomain =
  | 'order'
  | 'commission'
  | 'agentApply'
  | 'adminTicket'
  | 'adminTicketLevel'
  | 'ticket'
  | 'ticketLevel'
  | 'adminRole'
  | 'server'

export interface StatusRow {
  labelKey: string
  badge?: StatusBadge
  color?: string
}

export interface StatusMeta {
  labelKey: string
  badge: StatusBadge
  color?: string
}

/** 域状态表:语义真不同的域各占一行,不强行合并 i18n 键 */
export const statusTable: Record<StatusDomain, Record<number, StatusRow>> = {
  // 订单(用户端与管理端同域同形,同一行)
  order: {
    0: { labelKey: 'common.pending', badge: 'warning' },
    1: { labelKey: 'common.completed', badge: 'success' },
    2: { labelKey: 'common.cancelled', badge: 'neutral' },
    3: { labelKey: 'common.refunded', badge: 'danger' },
  },
  // 佣金(0 确认中/1 已发放/2 已撤销;徽标三元形态保留)
  commission: {
    0: { labelKey: 'adminCommissionLogs.confirming', badge: 'warning' },
    1: { labelKey: 'adminCommissionLogs.paid', badge: 'success' },
    2: { labelKey: 'adminCommissionLogs.revoked', badge: 'danger' },
  },
  // 代理申请(0 待审核/1 通过/2 拒绝;徽标三元形态保留)
  agentApply: {
    0: { labelKey: 'adminAgentApplies.pending', badge: 'warning' },
    1: { labelKey: 'adminAgentApplies.approved', badge: 'success' },
    2: { labelKey: 'adminAgentApplies.rejected', badge: 'danger' },
  },
  // 管理端工单状态(用户端 ticket.* 键不同域,徽标形态由对称断言锁定)
  adminTicket: {
    0: { labelKey: 'adminTickets.pending', badge: 'warning' },
    1: { labelKey: 'adminTickets.replied', badge: 'success' },
    2: { labelKey: 'adminTickets.closed', badge: 'neutral' },
  },
  // 管理端工单级别
  adminTicketLevel: {
    0: { labelKey: 'adminTickets.low', badge: 'neutral' },
    1: { labelKey: 'adminTickets.medium', badge: 'warning' },
    2: { labelKey: 'adminTickets.high', badge: 'danger' },
  },
  // 用户端工单状态(徽标形态与 adminTicket 一致,见对称测试)
  ticket: {
    0: { labelKey: 'ticket.pending', badge: 'warning' },
    1: { labelKey: 'ticket.replied', badge: 'success' },
    2: { labelKey: 'ticket.closed', badge: 'neutral' },
  },
  // 用户端工单级别(徽标形态与 adminTicketLevel 一致;此前无 badge 函数,属对称补齐)
  ticketLevel: {
    0: { labelKey: 'ticket.low', badge: 'neutral' },
    1: { labelKey: 'ticket.medium', badge: 'warning' },
    2: { labelKey: 'ticket.high', badge: 'danger' },
  },
  // 管理端角色(纯文案三态,无徽标)
  adminRole: {
    0: { labelKey: 'adminUsers.roleNormal' },
    1: { labelKey: 'adminUsers.roleAdmin' },
    2: { labelKey: 'adminUsers.roleAgent' },
  },
  // 节点状态(1/2/3;label + 徽标 + 着色三元同行,不再从色值反推徽标)
  server: {
    1: { labelKey: 'node.normal', badge: 'success', color: 'var(--c-success)' },
    2: { labelKey: 'node.busy', badge: 'warning', color: 'var(--c-warning)' },
    3: { labelKey: 'node.maintenance', badge: 'neutral', color: 'var(--c-text-sub)' },
  },
}

/** 未知 status 回落(与收敛前各函数回落逐字一致,含三元函数的 danger 偏向) */
const statusFallback: Record<
  StatusDomain,
  { labelKey: string; badge: StatusBadge; color?: string }
> = {
  order: { labelKey: 'common.unknown', badge: 'neutral' },
  commission: { labelKey: 'common.unknown', badge: 'danger' },
  agentApply: { labelKey: 'common.unknown', badge: 'danger' },
  adminTicket: { labelKey: 'common.unknown', badge: 'neutral' },
  adminTicketLevel: { labelKey: 'adminTickets.low', badge: 'danger' },
  ticket: { labelKey: 'common.unknown', badge: 'neutral' },
  ticketLevel: { labelKey: 'ticket.low', badge: 'neutral' },
  adminRole: { labelKey: 'adminUsers.roleNormal', badge: 'neutral' },
  server: { labelKey: 'common.unknown', badge: 'neutral', color: 'var(--c-text-sub)' },
}

/** 查表(含回落):本模块唯一出口 */
export function statusMeta(domain: StatusDomain, status: number): StatusMeta {
  const row = statusTable[domain]?.[status]
  const fb = statusFallback[domain]
  return {
    labelKey: row?.labelKey ?? fb.labelKey,
    badge: row?.badge ?? fb.badge,
    color: row?.color ?? fb.color,
  }
}

/** 全局 i18n 翻译(工具函数在组件外使用,直接读全局实例) */
function tt(key: string): string {
  return i18n.global.t(key, {})
}

/** 文案 */
export function statusLabel(domain: StatusDomain, status: number): string {
  return tt(statusMeta(domain, status).labelKey)
}

/** 徽标 */
export function statusBadge(domain: StatusDomain, status: number): StatusBadge {
  return statusMeta(domain, status).badge
}
