import { defineStore } from 'pinia'
import { apiAuth } from '@/api/auth'
import type { UserBrief } from '@/types/api'
import { clearTokens, readTokens, writeTokens } from '@/utils/http'

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
    isLoggedIn: (s) => !!s.accessToken,
  },
  actions: {
    applyAuth(data: {
      access_token: string
      refresh_token: string
      user?: UserBrief
    }) {
      this.accessToken = data.access_token
      this.refreshToken = data.refresh_token
      if (data.user) this.user = data.user
      writeTokens({ accessToken: data.access_token, refreshToken: data.refresh_token })
    },
    async login(email: string, password: string) {
      const data = await apiAuth.login({ email, password })
      this.applyAuth(data)
    },
    async register(body: { email: string; password: string; email_code: string; invite_code?: string }) {
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
    },
    /** 会话恢复:启动时从持久化读取并同步到 store */
    restore() {
      const tokens = readTokens()
      if (tokens) {
        this.accessToken = tokens.accessToken
        this.refreshToken = tokens.refreshToken
      }
    },
    reset() {
      this.accessToken = ''
      this.refreshToken = ''
      this.user = null
      clearTokens()
    },
  },
  persist: {
    key: 'auth',
    pick: ['accessToken', 'refreshToken', 'user'],
    storage: {
      getItem: (k) => window.localStorage.getItem(`app:${k}`),
      setItem: (k, v) => window.localStorage.setItem(`app:${k}`, v),
    },
  },
})
