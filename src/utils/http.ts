/**
 * HTTP 客户端封装 —— 统一通道 / 鉴权注入 / envelope 解包 / 401 静默刷新。
 * 规范见 docs/frontend/data-layer.md §1 与 docs/api/README.md §1。
 */
import type { ApiEnvelope, ApiError } from '@/types/api'
import { getItem, removeItem, setItem } from '@/utils/storage'
import { clientTag } from '@/utils/platform'

export interface HttpOptions {
  query?: Record<string, string | number | boolean | null | undefined>
  body?: unknown
  signal?: AbortSignal
  /** 关闭默认错误 toast */
  silent?: boolean
  headers?: Record<string, string>
}

export interface RefreshTokens {
  accessToken: string
  refreshToken: string
}

const API_BASE: string =
  getItem<string>('apiBase') ?? import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

/** 令牌读写(与 useAuthStore 持久化共用 localStorage 'app:auth') */
export function readTokens(): RefreshTokens | null {
  const auth = getItem<{ accessToken: string; refreshToken: string }>('auth')
  if (!auth?.accessToken) return null
  return { accessToken: auth.accessToken, refreshToken: auth.refreshToken }
}

export function writeTokens(t: RefreshTokens): void {
  const prev = getItem<Record<string, unknown>>('auth') ?? {}
  setItem('auth', { ...prev, accessToken: t.accessToken, refreshToken: t.refreshToken })
}

export function clearTokens(): void {
  removeItem('auth')
}

let language = 'zh-CN'
export function setHttpLanguage(lang: string): void {
  language = lang
}

// ---------- 401 静默刷新(single-flight) ----------

let refreshing: Promise<boolean> | null = null

async function doRefresh(): Promise<boolean> {
  try {
    const tokens = readTokens()
    if (!tokens?.refreshToken) return false
    const resp = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: tokens.refreshToken }),
    })
    const json = (await resp.json()) as ApiEnvelope<{
      access_token: string
      refresh_token: string
    }>
    if (!resp.ok || json.code !== 0 || !json.data) return false
    writeTokens({
      accessToken: json.data.access_token,
      refreshToken: json.data.refresh_token,
    })
    window.dispatchEvent(new CustomEvent('auth:refreshed'))
    return true
  } catch {
    return false
  }
}

function refreshTokens(): Promise<boolean> {
  if (!refreshing) {
    refreshing = doRefresh().finally(() => {
      refreshing = null
    })
  }
  return refreshing
}

function redirectToLogin(): void {
  clearTokens()
  const current = window.location.pathname + window.location.search
  window.location.href = `/login?redirect=${encodeURIComponent(current)}`
}

// ---------- 请求核心 ----------

export class ApiErrorImpl extends Error implements ApiError {
  code: number
  status: number
  constructor(code: number, message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

let toastFn: ((message: string, type?: 'error' | 'warning' | 'success') => void) | null = null

/** 由 main.ts 注入全局 toast(naive-ui message),避免 http 层依赖组件库 */
export function setToastProvider(fn: typeof toastFn): void {
  toastFn = fn
}

function showErrorToast(message: string): void {
  toastFn?.(message, 'error')
}

async function request<T>(method: string, url: string, opts: HttpOptions = {}): Promise<T> {
  const tokens = readTokens()
  const headers: Record<string, string> = {
    Accept: 'application/json',
    'Accept-Language': language,
    'X-Client': clientTag(),
    ...(opts.body !== undefined ? { 'Content-Type': 'application/json' } : {}),
    ...opts.headers,
  }
  if (tokens?.accessToken) headers.Authorization = `Bearer ${tokens.accessToken}`

  const queryStr = opts.query
    ? '?' +
      Object.entries(opts.query)
        .filter(([, v]) => v !== undefined && v !== null && v !== '')
        .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
        .join('&')
    : ''

  let resp: Response
  try {
    resp = await fetch(API_BASE + url + queryStr, {
      method,
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      signal: opts.signal,
    })
  } catch (e) {
    if ((e as Error).name === 'AbortError') throw e
    if (!opts.silent) showErrorToast('网络异常,请稍后再试')
    throw new ApiErrorImpl(-1, '网络异常,请稍后再试', 0)
  }

  // 401:尝试静默刷新后重放一次
  if (resp.status === 401) {
    const ok = await refreshTokens()
    if (ok) {
      const retryTokens = readTokens()
      if (retryTokens?.accessToken) headers.Authorization = `Bearer ${retryTokens.accessToken}`
      resp = await fetch(API_BASE + url + queryStr, {
        method,
        headers,
        body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
        signal: opts.signal,
      })
    } else {
      if (!opts.silent) showErrorToast('登录已过期,请重新登录')
      redirectToLogin()
      throw new ApiErrorImpl(40100, '登录已过期', 401)
    }
  }

  let json: ApiEnvelope<T> | null = null
  try {
    json = (await resp.json()) as ApiEnvelope<T>
  } catch {
    // 非 JSON 响应
  }

  if (json && json.code !== 0) {
    const err = new ApiErrorImpl(json.code, json.message || '请求失败', resp.status)
    if (!opts.silent) showErrorToast(err.message)
    throw err
  }
  if (!resp.ok && !json) {
    const err = new ApiErrorImpl(resp.status * 100, `请求失败(${resp.status})`, resp.status)
    if (!opts.silent) showErrorToast(err.message)
    throw err
  }
  return (json?.data ?? null) as T
}

export const http = {
  get: <T>(url: string, opts?: HttpOptions) => request<T>('GET', url, opts),
  post: <T>(url: string, opts?: HttpOptions) => request<T>('POST', url, opts),
  put: <T>(url: string, opts?: HttpOptions) => request<T>('PUT', url, opts),
  delete: <T>(url: string, opts?: HttpOptions) => request<T>('DELETE', url, opts),
}

export { API_BASE }
