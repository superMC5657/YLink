import { describe, expect, it, vi } from 'vitest'
import { useCountdown } from '../useCountdown'

describe('useCountdown', () => {
  it('start 后倒计时并归零停止', async () => {
    vi.useFakeTimers()
    const { remaining, running, start } = useCountdown(60)
    start(3)
    expect(running.value).toBe(true)
    expect(remaining.value).toBe(3)
    vi.advanceTimersByTime(1000)
    expect(remaining.value).toBe(2)
    vi.advanceTimersByTime(3000)
    expect(remaining.value).toBe(0)
    expect(running.value).toBe(false)
    vi.useRealTimers()
  })
})
