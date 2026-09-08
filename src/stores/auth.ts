import { defineStore } from 'pinia'
import { apiAuth } from '@/api/auth'
import type { UserBrief } from '@/types/api'
import { clearTokens, applyTokens, restorePersisted, setMemoryTokens } from '@/utils/tokens'
import { copyText } from '@/utils/platform'
import { flushStorage } from '@/utils/storage'

interface AuthState {
  accessToken: string
  refreshToken: string
  user: UserBrief | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    accessToken: '',
    refreshToken: '',
    user: null,
  }),
  getters: {
    // access 或 refresh 任一在即视为已登录:刷新后 access 不恢复(仅内存),
    // 靠持久化的 refresh 经首个请求 401 静默刷新补回,守卫据此放行
    isLoggedIn: (s) => !!(s.accessToken || s.refreshToken),
    /** 管理员(role=1):控制管理后台入口与路由守卫 */
    isAdmin: (s) => s.user?.role === 1,
  },
  actions: {
    applyAuth(data: { access_token: string; refresh_token: string; user?: UserBrief }) {
      this.accessToken = data.access_token
      this.refreshToken = data.refresh_token
      if (data.user) this.user = data.user
      // access 仅内存,refresh 最小化持久化(见 utils/tokens.ts)
      applyTokens(data.access_token, data.refresh_token, this.user)
    },
    async login(email: string, password: string) {
      const data = await apiAuth.login({ email, password })
      this.applyAuth(data)
    },
    async register(body: {
      email: string
      password: string
      email_code: string
      invite_code?: string
    }) {
      const data = await apiAuth.register(body)
      this.applyAuth(data)
    },
    async logout() {
      try {
        await apiAuth.logout()
      } catch {
        // 忽略登出失败,本地照常清会话
      }
      this.reset()
      // 登出收尾:等待挂起的持久化写落盘(防 token 残留),并尽力清掉剪贴板中可能复制的订阅链接
      await flushStorage()
      await copyText('')
    },
    /** 会话恢复:仅取回 refresh + user;access 不恢复,首个请求 401 后经静默刷新取回 */
    restore() {
      const persisted = restorePersisted()
      this.refreshToken = persisted.refreshToken
      this.user = persisted.user ?? null
      setMemoryTokens('', persisted.refreshToken)
    },
    reset() {
      this.accessToken = ''
      this.refreshToken = ''
      this.user = null
      clearTokens()
    },
  },
})
