/**
 * 令牌管理(安全持久面收敛):access token 仅存内存,持久化只保留最小 refresh
 * (XSS 即使读到存储也拿不到长效 access;refresh 过期/登出即失效)。
 * - 持久化键 'auth':{ refreshToken, user, savedAt },savedAt 用于过期清理;
 * - 单一写入方:本模块负责 'auth' 键的读写,auth store 与 http 层都经此收口,
 *   避免 pinia persist 与 http 层双写互相覆盖;
 * - 过期清理:恢复时 savedAt 超过最大保留期(30d)即丢弃,需重新登录。
 */
import { getItem, removeItem, setItem } from './storage'
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

/** 仅写内存(恢复流程用,不动持久化,不重置 savedAt) */
export function setMemoryTokens(access: string, refresh: string): void {
  accessToken = access
  refreshToken = refresh
}

/** 写内存 + 最小化持久化 refresh(登录/注册/静默刷新) */
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

/** 清空内存与持久化令牌(登出/会话失效) */
export function clearTokens(): void {
  accessToken = ''
  refreshToken = ''
  removeItem('auth')
}
