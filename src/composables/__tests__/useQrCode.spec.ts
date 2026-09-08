import { describe, expect, it, vi } from 'vitest'
import { useQrCode } from '../useQrCode'

// useQrCode 动态 import 且解构命名导出,mock 需同时暴露 default 与命名导出
const qrMock = vi.hoisted(() => ({ toDataURL: vi.fn() }))
vi.mock('qrcode', () => ({ default: qrMock, ...qrMock }))

describe('useQrCode', () => {
  it('成功时返回 dataURL 且 failed=false', async () => {
    qrMock.toDataURL.mockResolvedValueOnce('data:image/png;base64,AAAA')
    const { loading, failed, toDataURL } = useQrCode()
    const url = await toDataURL('https://ylink.example.com')
    expect(url).toBe('data:image/png;base64,AAAA')
    expect(failed.value).toBe(false)
    expect(loading.value).toBe(false)
    expect(qrMock.toDataURL).toHaveBeenCalledWith('https://ylink.example.com', {
      width: 512,
      margin: 1,
      color: { dark: '#1F2430', light: '#FFFFFF' },
    })
  })

  it('生成失败返回空串且 failed=true,loading 复位', async () => {
    qrMock.toDataURL.mockRejectedValueOnce(new Error('too long'))
    const { loading, failed, toDataURL } = useQrCode()
    const url = await toDataURL('x'.repeat(3000), { width: 220 })
    expect(url).toBe('')
    expect(failed.value).toBe(true)
    expect(loading.value).toBe(false)
    expect(qrMock.toDataURL).toHaveBeenCalledWith(expect.any(String), {
      width: 220,
      margin: 1,
      color: { dark: '#1F2430', light: '#FFFFFF' },
    })
  })

  it('失败后再次调用恢复 failed=false', async () => {
    qrMock.toDataURL.mockRejectedValueOnce(new Error('fail'))
    qrMock.toDataURL.mockResolvedValueOnce('data:image/png;base64,BBBB')
    const { failed, toDataURL } = useQrCode()
    await toDataURL('a')
    expect(failed.value).toBe(true)
    await toDataURL('b')
    expect(failed.value).toBe(false)
  })
})
