import { describe, expect, it } from 'vitest'
import { withLoading } from '../withLoading'

describe('withLoading', () => {
  it('成功路径：loading 置真后复位为假', async () => {
    const s = { loading: false }
    const seen: boolean[] = []
    await withLoading(s, async () => {
      seen.push(s.loading)
    })
    expect(seen).toEqual([true])
    expect(s.loading).toBe(false)
  })

  it('work 抛错：loading 仍在 finally 复位，异常向上传播', async () => {
    const s = { loading: false }
    await expect(
      withLoading(s, async () => {
        throw new Error('boom')
      }),
    ).rejects.toThrow('boom')
    expect(s.loading).toBe(false)
  })
})
