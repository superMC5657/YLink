/**
 * HTTP 传输适配 —— 统一通道 / 鉴权注入 / envelope 解包 / 401 静默刷新。
 * 在会话模块 seam 内只承担传输适配:令牌读写与刷新状态机收口于会话模块,
 * 本模块注入刷新传输实现,不直写存储键。
 * 规范见 docs/frontend/data-layer.md §1 与 docs/api/README.md §1。
 */
import type { ApiEnvelope, ApiError } from '@/types/api'
import { getItem } from '@/utils/storage'
import {
  applyTokens,
  clearTokens,
  getAccessToken,
  getRefreshToken,
  refreshSessionVia,
  terminateSession,
} from '@/utils/tokens'
import { clientTag, isTauri } from '@/utils/platform'
import { i18n } from '@/i18n'

/** 全局 i18n 翻译(http 在组件外使用,直接读全局实例) */
function tt(key: string, params?: Record<string, unknown>): string {
  return i18n.global.t(key, params ?? {})
}

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

/**
 * 惰性解析 API 基址:每次请求时读取,而不是模块导入时。
 * 否则 Tauri 下 http.ts 被 import 时 initStorage() 尚未执行,持久化的自定义 apiBase 读不到,
 * 整个会话会错误地使用环境变量/默认地址(见 docs/reviews/review-0.6.0.md P1)。
 */
function resolveApiBase(): string {
  return getItem<string>('apiBase') ?? import.meta.env.VITE_API_BASE_URL ?? '/api/v1'
}

/** 令牌读取:access 仅内存、refresh 经 utils/tokens.ts 收口(内存+最小持久化) */
export function readTokens(): RefreshTokens | null {
  if (!getAccessToken() && !getRefreshToken()) return null
  return { accessToken: getAccessToken(), refreshToken: getRefreshToken() }
}

export function writeTokens(t: RefreshTokens): void {
  applyTokens(t.accessToken, t.refreshToken)
}

export { clearTokens }

let language = 'zh-CN'
export function setHttpLanguage(lang: string): void {
  language = lang
}

// ---------- 401 静默刷新(状态机收口于会话模块,本模块只提供传输适配) ----------

/**
 * 刷新传输适配:经 Tauri/浏览器通道 POST /auth/refresh。
 * 线路语义不变:body 键仍为 refresh_token(白名单 refresh:{uid}:{jti} 与 SV 语义由后端持有,前端只透传)。
 * 成功解析出令牌对即返回;失败(无 refresh/网络错/业务码/空 data)返回 null 不抛错,
 * 调用方按 false 走失效分支。
 */
async function fetchRefreshedPair(): Promise<RefreshTokens | null> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return null
  try {
    const fetcher = await resolveFetcher()
    const resp = await fetcher(`${resolveApiBase()}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    const json = (await resp.json()) as ApiEnvelope<{
      access_token: string
      refresh_token: string
    }>
    if (!resp.ok || json.code !== 0 || !json.data) return null
    return { accessToken: json.data.access_token, refreshToken: json.data.refresh_token }
  } catch {
    return null
  }
}

/** 经会话模块单飞刷新,并发请求共用一次换新;成功时模块内已更新令牌/刷盘并派发 auth:refreshed */
function refreshTokens(): Promise<boolean> {
  return refreshSessionVia(fetchRefreshedPair)
}

function redirectToLogin(): void {
  // 失效终结走会话模块同一 seam(清令牌+清持久化+代际+1+集中刷盘,不等待落盘即跳转)
  void terminateSession()
  // 应用使用 hash 路由(createWebHashHistory):跳转必须落在 hash 内,
  // 否则 Tauri 自定义协议下 /login 不是真实文件,会出现空白页。
  const current = window.location.hash.slice(1) || '/'
  window.location.hash = `#/login?redirect=${encodeURIComponent(current)}`
}

// ---------- 请求核心 ----------

/** 底层通道:Tauri 用 plugin-http(原生栈,无 CORS 限制),浏览器用 window.fetch */
async function resolveFetcher(): Promise<typeof fetch> {
  if (isTauri()) {
    const { fetch: tauriFetch } = await import('@tauri-apps/plugin-http')
    return tauriFetch as unknown as typeof fetch
  }
  return window.fetch
}

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

/** 由 main.ts 注入全局 toast(naive-ui message),避免传输适配依赖视图模块 */
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
  const fetcher = await resolveFetcher()
  try {
    resp = await fetcher(resolveApiBase() + url + queryStr, {
      method,
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      signal: opts.signal,
    })
  } catch (e) {
    if ((e as Error).name === 'AbortError') throw e
    if (!opts.silent) showErrorToast(tt('network.networkError'))
    throw new ApiErrorImpl(-1, tt('network.networkError'), 0)
  }

  let json: ApiEnvelope<T> | null = null
  try {
    json = (await resp.json()) as ApiEnvelope<T>
  } catch {
    // 非 JSON 响应
  }

  // 会话过期 = HTTP 401 或业务码 40100(真实后端为 HTTP 401;mock 以 200 包裹 40100,
  // 两种形态都走静默刷新重放);其余业务性 401(如登录失败 40101)直接透出后端 message
  const expiredEnvelope = json !== null && json.code === 40100
  if (resp.status === 401 || expiredEnvelope) {
    if (resp.status === 401 && json && typeof json.code === 'number' && json.code !== 40100) {
      const err = new ApiErrorImpl(
        json.code,
        json.message || tt('network.requestFailed'),
        resp.status,
      )
      if (!opts.silent) showErrorToast(err.message)
      throw err
    }
    const ok = await refreshTokens()
    if (ok) {
      const retryTokens = readTokens()
      if (retryTokens?.accessToken) headers.Authorization = `Bearer ${retryTokens.accessToken}`
      resp = await fetcher(resolveApiBase() + url + queryStr, {
        method,
        headers,
        body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
        signal: opts.signal,
      })
      try {
        json = (await resp.json()) as ApiEnvelope<T>
      } catch {
        json = null
      }
    } else {
      if (!opts.silent) showErrorToast(tt('network.sessionExpired'))
      redirectToLogin()
      throw new ApiErrorImpl(40100, tt('network.sessionExpiredShort'), 401)
    }
  }

  if (json && json.code !== 0) {
    const err = new ApiErrorImpl(
      json.code,
      json.message || tt('network.requestFailed'),
      resp.status,
    )
    if (!opts.silent) showErrorToast(err.message)
    throw err
  }
  if (!resp.ok && !json) {
    const err = new ApiErrorImpl(
      resp.status * 100,
      tt('network.requestFailedStatus', { status: resp.status }),
      resp.status,
    )
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

/** 拼接查询串(与 request 内部一致) */
function buildQueryStr(query?: HttpOptions['query']): string {
  if (!query) return ''
  const pairs = Object.entries(query).filter(([, v]) => v !== undefined && v !== null && v !== '')
  if (pairs.length === 0) return ''
  return (
    '?' +
    pairs.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`).join('&')
  )
}

/**
 * 下载二进制资源(F05 用户 CSV 导出):GET + Bearer,401 时静默刷新重试一次。
 * 与 request<T> 分离:CSV 不走 envelope,响应体由调用方消费(blob)。
 */
export async function download(url: string, query?: HttpOptions['query']): Promise<Blob> {
  const queryStr = buildQueryStr(query)
  const doFetch = async (): Promise<Response> => {
    const headers: Record<string, string> = {
      Accept: 'text/csv, application/json;q=0.8, */*;q=0.5',
      'Accept-Language': language,
      'X-Client': clientTag(),
    }
    const tokens = readTokens()
    if (tokens?.accessToken) headers.Authorization = `Bearer ${tokens.accessToken}`
    const fetcher = await resolveFetcher()
    return fetcher(resolveApiBase() + url + queryStr, { method: 'GET', headers })
  }

  let resp = await doFetch()
  if (resp.status === 401) {
    const ok = await refreshTokens()
    if (ok) resp = await doFetch()
  }
  if (!resp.ok) {
    const err = new ApiErrorImpl(
      resp.status * 100,
      tt('network.requestFailedStatus', { status: resp.status }),
      resp.status,
    )
    showErrorToast(err.message)
    throw err
  }
  return resp.blob()
}

/** 供外部读取当前解析后的 API 基址(与请求层一致) */
export function getApiBaseUrl(): string {
  return resolveApiBase()
}
