/**
 * 格式化工具:金额 / 流量 / 时间 / 百分比。
 * 规范见 docs/api/README.md §1.4 与 docs/frontend/data-layer.md §4。
 */
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import type { PlanPeriod } from '@/types/api'
import { i18n } from '@/i18n'
import { statusBadge, statusLabel, statusMeta } from './statusMeta'

dayjs.extend(relativeTime)

/** 全局 i18n 翻译(工具函数在组件外使用,直接读全局实例) */
function tt(key: string, params?: Record<string, unknown>): string {
  return i18n.global.t(key, params ?? {})
}

/** 金额:元 number → `¥10.00`(千分位、两位小数) */
export function formatMoney(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '¥0.00'
  const n = Number(value)
  const fixed = n.toFixed(2)
  const [int, dec] = fixed.split('.')
  const grouped = int.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return `¥${grouped}.${dec}`
}

/** 字节 → `100.00 G` / `512.3 M` / `800 K` */
export function formatBytes(bytes: number | null | undefined): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'K', 'M', 'G', 'T']
  let v = Number(bytes)
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  const digits = v >= 100 ? 0 : v >= 10 ? 1 : 2
  return `${v.toFixed(digits)} ${units[i]}`
}

/** 速率:Mbps(空值显示「无限制」) */
export function formatSpeed(mbps: number | null | undefined): string {
  if (mbps === null || mbps === undefined) return tt('common.noLimit')
  return `${mbps} Mbps`
}

/** 百分比 0-100,最多 1 位小数 */
export function formatPercent(value: number | null | undefined): string {
  if (value === null || value === undefined) return '0%'
  const v = Math.min(100, Math.max(0, Number(value)))
  return `${v.toFixed(1)}%`
}

/** RFC3339 → `YYYY/M/D HH:mm:ss`(截图风格) */
export function formatTime(iso: string | null | undefined, withSeconds = true): string {
  if (!iso) return '-'
  const d = dayjs(iso)
  if (!d.isValid()) return '-'
  return withSeconds ? d.format('YYYY/M/D HH:mm:ss') : d.format('YYYY/M/D HH:mm')
}

/** 纯日期 YYYY-MM-DD */
export function formatDate(iso: string | null | undefined): string {
  if (!iso) return '-'
  const d = dayjs(iso)
  return d.isValid() ? d.format('YYYY-MM-DD') : '-'
}

/**
 * 本地日期键 YYYY-MM-DD:按用户时区取日期(日期选择器值 → 后端 YYYY-MM-DD 筛选参数)。
 * 勿用 toISOString().slice(0, 10)——那是 UTC 日期,东八区 0:00–7:59 会前移一天。
 */
export function localDateKey(value: Date | number | string): string {
  return dayjs(value).format('YYYY-MM-DD')
}

/** 相对时间:中文友好(N 天前 / 刚刚 / N 小时前) */
export function fromNow(iso: string | null | undefined): string {
  if (!iso) return '-'
  return dayjs(iso).locale('zh-cn').fromNow()
}

/** 订阅到期:返回人类可读文案,如「已过期 15 天」「剩余 7 天」「今日到期」 */
export function formatExpiry(expiredAt: string | null | undefined, isExpired: boolean): string {
  if (!expiredAt) return tt('profile.noExpiry')
  const now = dayjs()
  const end = dayjs(expiredAt)
  const days = end.diff(now, 'day')
  if (isExpired || days < 0) {
    return tt('profile.expiredDays', { days: Math.abs(days) })
  }
  if (days === 0) return tt('profile.expireToday')
  return tt('profile.remainingDays', { days })
}

/** 周期 key → 中文标签 */
export function periodLabel(period: string): string {
  const map: Record<string, string> = {
    month: 'plan.periodMonth',
    quarter: 'plan.periodQuarter',
    half_year: 'plan.periodHalfYear',
    year: 'plan.periodYear',
    onetime: 'plan.periodOnetime',
  }
  return map[period] ? tt(map[period]) : period
}

/**
 * 订阅链接掩码展示:保留协议+域名,其余以 •••• + 末 4 位示意(令牌主体不进 DOM)。
 * 非法/非标准 URL 原样返回(调用方兜底为复制按钮,原文仅在显式「显示」时进 DOM)。
 */
export function maskUrl(url: string, keepTail = 4): string {
  const schemeEnd = url.indexOf('://')
  if (schemeEnd < 0) return url
  const pathStart = url.indexOf('/', schemeEnd + 3)
  if (pathStart < 0) return url
  const origin = url.slice(0, pathStart)
  const rest = url.slice(pathStart)
  if (rest.length <= keepTail + 1) return url
  return `${origin}/••••${rest.slice(-keepTail)}`
}

/** 支付方式 key → 本地化标签(余额 / 支付宝 / 微信;未知渠道回退原始值) */
export function payMethodLabel(method: string | null | undefined): string {
  if (!method) return '-'
  const map: Record<string, string> = {
    balance: 'order.payMethodBalance',
    epay_alipay: 'order.payMethodAlipay',
    epay_wxpay: 'order.payMethodWxpay',
  }
  return map[method] ? tt(map[method]) : method
}

/** 套餐周期折扣:相对月付省 N%(无折扣返回 null) */
export function planSavePercent(
  plan: { prices: Partial<Record<PlanPeriod, number>> },
  period: PlanPeriod,
): number | null {
  const month = plan.prices.month
  if (!month || period === 'month') return null
  const months: Record<PlanPeriod, number> = {
    month: 1,
    quarter: 3,
    half_year: 6,
    year: 12,
    onetime: 12,
  }
  const m = months[period]
  if (!m) return null
  const price = plan.prices[period] ?? 0
  const pct = Math.round((1 - price / (month * m)) * 100)
  return pct > 0 ? pct : null
}

/** 订单状态 → 文案(表归 statusMeta,本函数只剩签名兼容) */
export function orderStatusLabel(status: number): string {
  return statusLabel('order', status)
}

/** 订单状态 → 徽标色（用户端与管理端同域同形，由对称测试锁定） */
export function orderStatusBadge(status: number): 'warning' | 'success' | 'neutral' | 'danger' {
  return statusBadge('order', status)
}

/** 佣金状态 → 文案（表归 statusMeta,本函数只剩签名兼容） */
export function commissionStatusLabel(status: number): string {
  return statusLabel('commission', status)
}

/** 佣金状态 → 徽标色（确认中警告/已发放成功/已撤销危险） */
export function commissionStatusBadge(status: number): 'warning' | 'success' | 'danger' {
  return statusBadge('commission', status) as 'warning' | 'success' | 'danger'
}

/** 代理申请状态 → 文案（表归 statusMeta,本函数只剩签名兼容） */
export function agentApplyStatusLabel(status: number): string {
  return statusLabel('agentApply', status)
}

/** 代理申请状态 → 徽标色（待审核警告/通过成功/拒绝危险） */
export function agentApplyStatusBadge(status: number): 'warning' | 'success' | 'danger' {
  return statusBadge('agentApply', status) as 'warning' | 'success' | 'danger'
}

/** 管理端工单状态 → 文案（用户端 ticket.* 键不同域，不混用） */
export function adminTicketStatusLabel(status: number): string {
  return statusLabel('adminTicket', status)
}

/** 管理端工单状态 → 徽标色（待回复警告/已回复成功/已关闭中性，由对称测试锁定） */
export function adminTicketStatusBadge(status: number): 'warning' | 'success' | 'neutral' {
  return statusBadge('adminTicket', status) as 'warning' | 'success' | 'neutral'
}

/** 管理端工单级别 → 文案 */
export function adminTicketLevelLabel(level: number): string {
  return statusLabel('adminTicketLevel', level)
}

/** 管理端工单级别 → 徽标色 */
export function adminTicketLevelBadge(level: number): 'neutral' | 'warning' | 'danger' {
  return statusBadge('adminTicketLevel', level) as 'neutral' | 'warning' | 'danger'
}

/** 工单状态 → 文案 */
export function ticketStatusLabel(status: number): string {
  return statusLabel('ticket', status)
}

/** 工单状态 → 徽标色（待回复警告/已回复成功/已关闭中性，由对称测试锁定） */
export function ticketStatusBadge(status: number): 'warning' | 'success' | 'neutral' {
  return statusBadge('ticket', status) as 'warning' | 'success' | 'neutral'
}

/** 工单级别 → 文案 */
export function ticketLevelLabel(level: number): string {
  return statusLabel('ticketLevel', level)
}

/** 管理端角色 → 文案(用户/代理/管理员三态,与 useAdminUsers.roleLabel 同源) */
export function adminRoleLabel(role: number): string {
  return statusLabel('adminRole', role)
}

/** 节点状态 → 文案与色 */
export function serverStatusMeta(status: number): { label: string; color: string } {
  const meta = statusMeta('server', status)
  return { label: tt(meta.labelKey), color: meta.color ?? 'var(--c-text-sub)' }
}
