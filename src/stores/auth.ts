import { defineStore } from 'pinia'
import { apiAuth } from '@/api/auth'
import { apiUser } from '@/api/user'
import type { UserBrief } from '@/types/api'
import {
  establishSession,
  getAccessToken,
  getRefreshToken,
  getSessionEpoch,
  restorePersisted,
  setMemoryTokens,
  terminateSession,
} from '@/utils/tokens'
import { copyText } from '@/utils/platform'

interface AuthState {
  user: UserBrief | null
  /** 会话模块核实过的 role(每代会话首次进管理路由时拉取):null=未核实,守卫回退持久化 user.role */
  verifiedRole: number | null
  /** 核实结果所属会话代际:与会话模块代际不一致视为过期,守卫只读接口自动回退 user.role */
  verifiedEpoch: number
}

/** role 核实单飞 flight:并发守卫共用一次拉取,竞态收口在模块内 */
let roleFlight: Promise<void> | null = null

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    verifiedRole: null,
    verifiedEpoch: getSessionEpoch(),
  }),
  getters: {
    // 登录态只读会话模块单源:access 或 refresh 任一在即视为已登录。
    // 刷新后 access 不恢复(仅内存),靠持久化的 refresh 经首个请求 401 静默刷新补回,
    // 守卫据此放行;令牌不再存 pinia 双源,传输适配侧失效清令牌后此处即时可见。
    isLoggedIn: () => !!(getAccessToken() || getRefreshToken()),
    /** 管理员(role=1):控制管理后台入口与路由守卫;同代已核实时以核实结果优先 */
    isAdmin: (s) =>
      (s.verifiedEpoch === getSessionEpoch() ? (s.verifiedRole ?? s.user?.role) : s.user?.role) ===
      1,
  },
  actions: {
    /** 新身份建立:写用户 + 丢弃上一代核实结果,令牌与刷盘走会话模块同一 seam */
    async applyAuth(data: { access_token: string; refresh_token: string; user?: UserBrief }) {
      if (data.user) this.user = data.user
      this.verifiedRole = null
      this.verifiedEpoch = await establishSession(data.access_token, data.refresh_token, this.user)
    },
    async login(email: string, password: string) {
      const data = await apiAuth.login({ email, password })
      await this.applyAuth(data)
    },
    async register(body: {
      email: string
      password: string
      email_code: string
      invite_code?: string
    }) {
      const data = await apiAuth.register(body)
      await this.applyAuth(data)
    },
    async logout() {
      try {
        await apiAuth.logout()
      } catch {
        // 忽略登出失败,本地照常清会话
      }
      this.user = null
      this.verifiedRole = null
      // 会话终结走会话模块同一 seam:清令牌+清持久化+代际+1+集中刷盘
      this.verifiedEpoch = await terminateSession()
      // 登出收尾:尽力清掉剪贴板中可能复制的订阅链接
      await copyText('')
    },
    /** 会话恢复:仅取回 refresh + user;access 不恢复,首个请求 401 后经静默刷新取回 */
    restore() {
      const persisted = restorePersisted()
      this.user = persisted.user ?? null
      this.verifiedRole = null
      this.verifiedEpoch = getSessionEpoch()
      setMemoryTokens('', persisted.refreshToken)
    },
    /**
     * 向会话模块核实 role(GET /user/profile):每代会话首次进入管理路由时拉取一次;
     * 防持久化 user 过期(如被降权的管理员仍进后台)。
     * 拉取失败(网络/会话失效)不阻塞守卫,回退持久化 role,后端接口仍会 403 兜底。
     */
    async ensureRole(): Promise<void> {
      // 跨代先丢弃上一代核实结果(重登/登出后 stale 结果不复用)
      if (this.verifiedEpoch !== getSessionEpoch()) {
        this.verifiedRole = null
        this.verifiedEpoch = getSessionEpoch()
      }
      if (this.verifiedRole !== null || !this.isLoggedIn) return
      roleFlight ??= this.verifyRoleOnce().finally(() => {
        roleFlight = null
      })
      await roleFlight
    },
    /** 单次 role 拉取:拉取期间会话换代则丢弃结果(后完成的旧代请求不覆盖新代) */
    async verifyRoleOnce(): Promise<void> {
      try {
        const profile = await apiUser.profile()
        if (this.verifiedEpoch !== getSessionEpoch()) return
        this.verifiedRole = profile.role
        this.verifiedEpoch = getSessionEpoch()
        // 同步回 user,保持门户页/菜单等处 role 判定一致(不覆盖 id/email)
        if (this.user && this.user.role !== profile.role) {
          this.user = { ...this.user, role: profile.role }
        }
      } catch {
        // 静默失败:401 会话失效时传输适配已清会话并跳登录,守卫再按 isLoggedIn 分流
      }
    },
  },
})
