import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { i18n } from '@/i18n'

/**
 * 登录守卫:
 * - guest 页:已登录 → 管理员 /portal / 普通用户 /dashboard
 * - 其余:未登录 → /login?redirect=原路径
 * - meta.admin:先向服务端核实 role(每会话一次,防持久化 user 过期),
 *   非管理员访问 /admin/* 进 /403;门户分流页保持仪表板兜底(A6)。
 */
export function setupGuards(router: Router): void {
  router.beforeEach(async (to) => {
    const auth = useAuthStore()
    const loggedIn = auth.isLoggedIn

    if (to.meta.guest) {
      if (loggedIn) return auth.isAdmin ? { path: '/portal' } : { name: 'dashboard' }
      return true
    }

    if (!loggedIn && to.name !== 'not-found') {
      return {
        name: 'login',
        query: { redirect: to.fullPath },
      }
    }

    // 管理端页面:放行前核实 role;核实过程中会话失效(静默刷新失败)则回登录页
    if (to.meta.admin) {
      await auth.ensureRole()
      if (!auth.isLoggedIn) {
        return { name: 'login', query: { redirect: to.fullPath } }
      }
      if (!auth.isAdmin) {
        return to.name === 'portal' ? { name: 'dashboard' } : { name: 'forbidden' }
      }
    }
    return true
  })

  router.afterEach((to) => {
    // meta.title 为 i18n key,翻译后写入标签页标题
    const title = to.meta.title ? i18n.global.t(to.meta.title) : ''
    document.title = title
      ? `${import.meta.env.VITE_APP_NAME ?? 'YLink'} · ${title}`
      : (import.meta.env.VITE_APP_NAME ?? 'YLink')
  })
}
