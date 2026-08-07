/**
 * 格式化工具:金额 / 流量 / 时间 / 百分比。
 * 规范见 docs/api/README.md §1.4 与 docs/frontend/data-layer.md §4。
 */
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import type { PlanPeriod } from '@/types/api'

dayjs.extend(relativeTime)

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
  if (mbps === null || mbps === undefined) return '无限制'
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

/** 相对时间:中文友好(N 天前 / 刚刚 / N 小时前) */
export function fromNow(iso: string | null | undefined): string {
  if (!iso) return '-'
  return dayjs(iso).locale('zh-cn').fromNow()
}

/** 订阅到期:返回人类可读文案,如「已过期 15 天」「剩余 7 天」「今日到期」 */
export function formatExpiry(expiredAt: string | null | undefined, isExpired: boolean): string {
  if (!expiredAt) return '暂无订阅'
  const now = dayjs()
  const end = dayjs(expiredAt)
  const days = end.diff(now, 'day')
  if (isExpired || days < 0) {
    return `已过期 ${Math.abs(days)} 天`
  }
  if (days === 0) return '今日到期'
  return `剩余 ${days} 天`
}

/** 周期 key → 中文标签 */
export function periodLabel(period: string): string {
  const map: Record<string, string> = {
    month: '月付',
    quarter: '季付',
    half_year: '半年付',
    year: '年付',
    onetime: '一次性',
  }
  return map[period] ?? period
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

/** 订单状态 → 文案 */
export function orderStatusLabel(status: number): string {
  const map: Record<number, string> = { 0: '待支付', 1: '已完成', 2: '已取消', 3: '已退款' }
  return map[status] ?? '未知'
}

/** 工单状态 → 文案 */
export function ticketStatusLabel(status: number): string {
  const map: Record<number, string> = { 0: '待回复', 1: '已回复', 2: '已关闭' }
  return map[status] ?? '未知'
}

/** 工单级别 → 文案 */
export function ticketLevelLabel(level: number): string {
  const map: Record<number, string> = { 0: '低', 1: '中', 2: '高' }
  return map[level] ?? '低'
}

/** 节点状态 → 文案与色 */
export function serverStatusMeta(status: number): { label: string; color: string } {
  const map: Record<number, { label: string; color: string }> = {
    1: { label: '正常', color: 'var(--c-success)' },
    2: { label: '拥挤', color: 'var(--c-warning)' },
    3: { label: '维护', color: 'var(--c-text-sub)' },
  }
  return map[status] ?? { label: '未知', color: 'var(--c-text-sub)' }
}
