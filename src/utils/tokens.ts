/**
 * 会话模块(单源会话接口):内存令牌 + 最小持久化 + 代际 + 单飞刷新 + 集中刷盘。
 * - access 仅内存,持久化只保留最小 refresh/user/savedAt(见 PersistedAuth);
 * - 单一写入 seam:登录/注册/静默刷新/登出都经本模块接口,http 传输适配与 auth store
 *   只读或调用本模块接口,不直写存储键;
 * - 代际:建立/终结会话时递增,role 核实结果凭代际丢弃,不复用 stale 结果;
 * - 单飞刷新:传输实现由 http 侧注入,状态机收口在本模块,并发调用共用一次换新;
 * - 集中刷盘:建立/终结/刷新成功后经由存储适配器落盘,Tauri 防抖窗口内被杀也不丢会话。
 * 白名单 refresh:{uid}:{jti} 与 SV auth:ver:{uid} 由后端持有,本模块只透传 refresh_token,不解析不断言。
 */
import { flushStorage, getItem, removeItem, setItem } from './storage'
import type { UserBrief } from '@/types/api'

/** refresh 最大保留期:超过后启动恢复时丢弃(后端 refresh 本身有轮换与有效期,此处为客户端兜底) */
const REFRESH_MAX_AGE = 30 * 24 * 60 * 60 * 1000

interface PersistedAuth {
  refreshToken: string
  user: UserBrief | null
  /** 最近一次令牌写入时间,当前态必填:缺失视为过期脏数据,恢复时丢弃并强制重新登录(类型保留可选仅为容忍历史 JSON 解析) */
  savedAt?: number
}

// 模块级内存令牌:不落盘、页面刷新即失(恢复流程从持久化仅取回 refresh)
let accessToken = ''
let refreshToken = ''

/** 会话代际:建立/终结会话时递增;role 核实结果凭它判定是否过期 */
let sessionEpoch = 0

/** 单飞刷新 flight:并发调用共用一次换新 */
let refreshFlight: Promise<boolean> | null = null

/** 当前会话代际(只读) */
export function getSessionEpoch(): number {
  return sessionEpoch
}

/** 仅写内存(恢复流程用,不动持久化,不重置 savedAt) */
export function setMemoryTokens(access: string, refresh: string): void {
  accessToken = access
  refreshToken = refresh
}

/** 写内存 + 最小化持久化 refresh(同步底层 seam,不刷盘;需落盘走 establish/refresh 接口) */
export function applyTokens(access: string, refresh: string, user?: UserBrief | null): void {
  setMemoryTokens(access, refresh)
  const prev = getItem<PersistedAuth>('auth')
  const next: PersistedAuth = {
    refreshToken: refresh,
    user: user ?? prev?.user ?? null,
    savedAt: Date.now(),
  }
  setItem('auth', next)
}

export function getAccessToken(): string {
  return accessToken
}

export function getRefreshToken(): string {
  return refreshToken
}

/** 启动恢复:从持久化读回 refresh(超期丢弃)与 user;access 不恢复,首个请求经 401 静默刷新取回 */
export function restorePersisted(): { refreshToken: string; user: UserBrief | null } {
  const auth = getItem<PersistedAuth>('auth')
  if (!auth?.refreshToken) return { refreshToken: '', user: null }
  if (!auth.savedAt || Date.now() - auth.savedAt > REFRESH_MAX_AGE) {
    removeItem('auth')
    return { refreshToken: '', user: null }
  }
  refreshToken = auth.refreshToken
  return { refreshToken: auth.refreshToken, user: auth.user ?? null }
}

/** 清空内存与持久化令牌(同步底层 seam,不刷盘;需落盘走 terminate 接口) */
export function clearTokens(): void {
  accessToken = ''
  refreshToken = ''
  removeItem('auth')
}

/** 会话令牌对(http 传输适配注入刷新结果用) */
export interface SessionPair {
  accessToken: string
  refreshToken: string
}

/**
 * 建立会话(登录/注册唯一入口):写内存+持久化,代际+1,并集中刷盘。
 * 返回新代际,调用方凭它标记核实结果归属。
 */
export async function establishSession(
  access: string,
  refresh: string,
  user?: UserBrief | null,
): Promise<number> {
  applyTokens(access, refresh, user)
  sessionEpoch += 1
  await flushStorage()
  return sessionEpoch
}

/**
 * 终结会话(登出/会话失效唯一出口):清内存与持久化,代际+1,并集中刷盘。
 * 返回新代际;跳转类调用方可 void 不等待(清内存与代际+1 在首个 await 前已同步生效)。
 */
export async function terminateSession(): Promise<number> {
  clearTokens()
  sessionEpoch += 1
  await flushStorage()
  return sessionEpoch
}

/**
 * 单飞刷新:传输实现由 http 侧注入,状态机收口在本模块。
 * 成功(同一身份续期,代际不变):更新令牌 + 集中刷盘 + 派发 auth:refreshed;
 * 失败返回 false,是否终结由调用方定(request 终结并跳登录,download 只抛错)。
 */
export function refreshSessionVia(refresher: () => Promise<SessionPair | null>): Promise<boolean> {
  refreshFlight ??= (async () => {
    let next: SessionPair | null
    try {
      next = await refresher()
    } catch {
      next = null
    }
    if (!next?.accessToken || !next?.refreshToken) return false
    try {
      applyTokens(next.accessToken, next.refreshToken)
      await flushStorage()
    } catch {
      // 落盘失败不否决已生效的内存续期:失败批次已回待写队列,随下次写重试
    }
    window.dispatchEvent(new CustomEvent('auth:refreshed'))
    return true
  })().finally(() => {
    refreshFlight = null
  })
  return refreshFlight
}
