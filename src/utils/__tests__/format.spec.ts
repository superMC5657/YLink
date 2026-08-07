import { describe, expect, it } from 'vitest'
import {
  formatBytes,
  formatDate,
  formatExpiry,
  formatMoney,
  formatPercent,
  formatSpeed,
  formatTime,
  fromNow,
  orderStatusLabel,
  periodLabel,
  serverStatusMeta,
  ticketLevelLabel,
  ticketStatusLabel,
} from '../format'
import dayjs from 'dayjs'

describe('formatMoney', () => {
  it('格式化为 ¥ 千分位两位小数', () => {
    expect(formatMoney(10)).toBe('¥10.00')
    expect(formatMoney(1234.5)).toBe('¥1,234.50')
    expect(formatMoney(0.1)).toBe('¥0.10')
  })
  it('空值返回 ¥0.00', () => {
    expect(formatMoney(null)).toBe('¥0.00')
    expect(formatMoney(undefined)).toBe('¥0.00')
    expect(formatMoney(Number.NaN)).toBe('¥0.00')
  })
})

describe('formatBytes', () => {
  it('按 1024 进制转换', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1024)).toBe('1.00 K')
    expect(formatBytes(100 * 1024 * 1024)).toBe('100 M')
    expect(formatBytes(107374182400)).toBe('100 G')
  })
  it('空值返回 0 B', () => {
    expect(formatBytes(null)).toBe('0 B')
  })
})

describe('formatSpeed', () => {
  it('null 表示无限制', () => {
    expect(formatSpeed(null)).toBe('无限制')
    expect(formatSpeed(100)).toBe('100 Mbps')
  })
})

describe('formatPercent', () => {
  it('截断到 0-100 并保留 1 位小数', () => {
    expect(formatPercent(70)).toBe('70.0%')
    expect(formatPercent(150)).toBe('100.0%')
    expect(formatPercent(-5)).toBe('0.0%')
    expect(formatPercent(null)).toBe('0%')
  })
})

describe('formatTime / formatDate / fromNow', () => {
  const iso = '2026-06-24T00:53:35+08:00'
  it('RFC3339 → YYYY/M/D HH:mm:ss', () => {
    expect(formatTime(iso)).toBe('2026/6/24 00:53:35')
  })
  it('date 形式', () => {
    expect(formatDate('2026-07-01T10:00:00+08:00')).toBe('2026-07-01')
  })
  it('非法/空输入返回 -', () => {
    expect(formatTime(null)).toBe('-')
    expect(formatTime('not-a-date')).toBe('-')
  })
  it('fromNow 中文相对时间', () => {
    const past = dayjs().subtract(3, 'day').toISOString()
    expect(fromNow(past)).toContain('3 天前')
  })
})

describe('formatExpiry', () => {
  const future = dayjs().add(15, 'day').toISOString()
  const past = dayjs().subtract(2, 'day').toISOString()
  it('未过期返回剩余天数(±1 天内)', () => {
    const text = formatExpiry(future, false)
    expect(text).toMatch(/剩余 1[45] 天/)
  })
  it('已过期返回已过期天数', () => {
    const pastText = formatExpiry(past, true)
    expect(pastText).toMatch(/已过期 [12] 天/)
    expect(formatExpiry(past, false)).toContain('已过期')
  })
  it('无到期时间返回暂无订阅', () => {
    expect(formatExpiry(null, false)).toBe('暂无订阅')
  })
})

describe('标签映射', () => {
  it('periodLabel', () => {
    expect(periodLabel('month')).toBe('月付')
    expect(periodLabel('year')).toBe('年付')
    expect(periodLabel('unknown')).toBe('unknown')
  })
  it('orderStatusLabel', () => {
    expect(orderStatusLabel(0)).toBe('待支付')
    expect(orderStatusLabel(1)).toBe('已完成')
    expect(orderStatusLabel(2)).toBe('已取消')
    expect(orderStatusLabel(3)).toBe('已退款')
  })
  it('ticketStatusLabel / ticketLevelLabel', () => {
    expect(ticketStatusLabel(0)).toBe('待回复')
    expect(ticketStatusLabel(2)).toBe('已关闭')
    expect(ticketLevelLabel(2)).toBe('高')
  })
  it('serverStatusMeta', () => {
    expect(serverStatusMeta(1).label).toBe('正常')
    expect(serverStatusMeta(3).label).toBe('维护')
  })
})
