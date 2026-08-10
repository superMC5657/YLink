import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

/**
 * 登录守卫:
 * - guest 页:已登录 → /dashboard
 * - 其余:未登录 → /login?redirect=原路径
 */
export function setupGuards(router: Router): void {
  router.beforeEach((to) => {
    const auth = useAuthStore()
    const loggedIn = auth.isLoggedIn

    if (to.meta.guest) {
      if (loggedIn) return { name: 'dashboard' }
      return true
    }

    if (!loggedIn && to.name !== 'not-found') {
      return {
        name: 'login',
        query: { redirect: to.fullPath },
      }
    }

    // 管理端页面:非管理员(role=1)重定向仪表板
    if (to.meta.admin && !auth.isAdmin) {
      return { name: 'dashboard' }
    }
    return true
  })

  router.afterEach((to) => {
    const title = to.meta.title
    document.title = title
      ? `${import.meta.env.VITE_APP_NAME ?? 'YLink'} · ${title}`
      : (import.meta.env.VITE_APP_NAME ?? 'YLink')
  })
}
