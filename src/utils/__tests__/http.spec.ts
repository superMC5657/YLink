import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiErrorImpl, http, setToastProvider, writeTokens } from '../http'
import type { ApiEnvelope } from '@/types/api'

/** 构造 envelope 响应 */
function jsonResponse<T>(payload: ApiEnvelope<T>, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('http 封装', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('成功响应解包 data', async () => {
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({ code: 0, message: 'ok', data: { id: 1, name: '猎户座' } }),
    )
    const data = await http.get<{ id: number; name: string }>('/plans')
    expect(data).toEqual({ id: 1, name: '猎户座' })
    // 断言请求头
    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('/api/v1/plans')
    expect((init?.headers as Record<string, string>).Accept).toBe('application/json')
  })

  it('业务错误抛 ApiError 并携带 code/message', async () => {
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({ code: 12001, message: '优惠券已过期', data: null }, 400),
    )
    await expect(http.get('/coupons/check')).rejects.toMatchObject({
      name: 'ApiError',
      code: 12001,
      message: '优惠券已过期',
    })
  })

  it('silent=true 时不触发 toast', async () => {
    const toast = vi.fn()
    setToastProvider(toast)
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({ code: 10003, message: '发送过于频繁', data: null }, 429),
    )
    await expect(http.post('/captcha/email', { body: {}, silent: true })).rejects.toBeInstanceOf(
      ApiErrorImpl,
    )
    expect(toast).not.toHaveBeenCalled()
  })

  it('业务错误默认触发 toast', async () => {
    const toast = vi.fn()
    setToastProvider(toast)
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({ code: 50000, message: '服务器内部错误', data: null }, 500),
    )
    await expect(http.get('/x')).rejects.toBeInstanceOf(ApiErrorImpl)
    expect(toast).toHaveBeenCalledWith('服务器内部错误', 'error')
  })

  it('携带 Authorization: Bearer token', async () => {
    writeTokens({ accessToken: 'abc', refreshToken: 'r' })
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ code: 0, message: 'ok', data: null }))
    await http.get('/user/stat')
    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer abc')
  })

  it('业务性 401(登录失败 40101)直接透出 message,不触发刷新/跳转', async () => {
    const toast = vi.fn()
    setToastProvider(toast)
    localStorage.removeItem('app:auth') // 未登录
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({ code: 40101, message: '邮箱或密码错误', data: null }, 401),
    )
    await expect(http.post('/auth/login', { body: {} })).rejects.toMatchObject({
      code: 40101,
      message: '邮箱或密码错误',
    })
    // 只调用一次:不尝试 refresh,不重放
    expect(vi.mocked(fetch)).toHaveBeenCalledTimes(1)
    expect(toast).toHaveBeenCalledWith('邮箱或密码错误', 'error')
  })

  it('业务性 401(40101)即使已有 token 也直接透出,不静默刷新', async () => {
    writeTokens({ accessToken: 'abc', refreshToken: 'r' })
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({ code: 40101, message: '邮箱或密码错误', data: null }, 401),
    )
    await expect(http.post('/auth/login', { body: {} })).rejects.toMatchObject({ code: 40101 })
    expect(vi.mocked(fetch)).toHaveBeenCalledTimes(1)
  })

  it('401 时用 refresh_token 静默换新并重放原请求', async () => {
    writeTokens({ accessToken: 'expired', refreshToken: 'valid-refresh' })
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        jsonResponse({ code: 40100, message: 'unauthorized', data: null }, 401),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          code: 0,
          message: 'ok',
          data: { access_token: 'new-access', refresh_token: 'new-refresh' },
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data: { ok: true } }))

    const data = await http.get<{ ok: boolean }>('/user/stat')
    expect(data).toEqual({ ok: true })
    // 三次调用:原请求 → refresh → 重放
    expect(vi.mocked(fetch)).toHaveBeenCalledTimes(3)
    const refreshCall = vi.mocked(fetch).mock.calls[1]
    expect(refreshCall[0]).toBe('/api/v1/auth/refresh')
    const replayCall = vi.mocked(fetch).mock.calls[2]
    expect((replayCall[1]?.headers as Record<string, string>).Authorization).toBe(
      'Bearer new-access',
    )
  })

  it('query 参数拼接并过滤空值', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ code: 0, message: 'ok', data: null }))
    await http.get('/orders', { query: { page: 2, page_size: 10, status: '', keyword: undefined } })
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('/api/v1/orders?page=2&page_size=10')
  })

  it('网络异常抛网络错误 ApiError', async () => {
    vi.mocked(fetch).mockRejectedValue(new TypeError('Failed to fetch'))
    await expect(http.get('/x', { silent: true })).rejects.toMatchObject({
      code: -1,
      message: '网络异常,请稍后再试',
    })
  })
})
