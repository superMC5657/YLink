import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

/**
 * layouts 懒加载:三个布局壳不进首屏 entry,按路由匹配并行加载各自 chunk
 * (AdminLayout 仅管理员进入后台时拉取,普通用户首屏不再为其买单)。
 * chunk 命名:Vite/Rollup 默认按模块名产出 AdminLayout-<hash>.js 等,无需 webpackChunkName 注释。
 */
const AuthLayout = () => import('@/layouts/AuthLayout.vue')
const MainLayout = () => import('@/layouts/MainLayout.vue')
const AdminLayout = () => import('@/layouts/AdminLayout.vue')

declare module 'vue-router' {
  interface RouteMeta {
    /** 游客页面:已登录访问重定向 /dashboard */
    guest?: boolean
    /** 管理端页面:非管理员访问重定向 /dashboard */
    admin?: boolean
    /** 页面标题(i18n key) */
    title?: string
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    component: AuthLayout,
    meta: { guest: true, title: 'auth.login' },
    children: [{ path: '', name: 'login', component: () => import('@/views/auth/LoginView.vue') }],
  },
  {
    path: '/register',
    component: AuthLayout,
    meta: { guest: true, title: 'auth.register' },
    children: [
      { path: '', name: 'register', component: () => import('@/views/auth/RegisterView.vue') },
    ],
  },
  {
    path: '/forgot',
    component: AuthLayout,
    meta: { guest: true, title: 'auth.forgot' },
    children: [
      { path: '', name: 'forgot', component: () => import('@/views/auth/ForgotView.vue') },
    ],
  },
  {
    path: '/',
    component: MainLayout,
    children: [
      { path: '', redirect: '/dashboard' },
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/dashboard/DashboardView.vue'),
        meta: { title: 'dashboard.title' },
      },
      {
        path: 'docs',
        name: 'docs',
        component: () => import('@/views/docs/DocsView.vue'),
        meta: { title: 'docs.title' },
      },
      {
        path: 'docs/:id',
        name: 'docs-detail',
        component: () => import('@/views/docs/DocsDetailView.vue'),
      },
      {
        path: 'orders',
        name: 'orders',
        component: () => import('@/views/order/OrdersView.vue'),
        meta: { title: 'order.title' },
      },
      {
        path: 'invite',
        name: 'invite',
        component: () => import('@/views/invite/InviteView.vue'),
        meta: { title: 'invite.title' },
      },
      {
        path: 'agent',
        name: 'agent',
        component: () => import('@/views/agent/AgentView.vue'),
        meta: { title: 'agent.title' },
      },
      {
        path: 'plans',
        name: 'plans',
        component: () => import('@/views/plan/PlansView.vue'),
        meta: { title: 'plan.title' },
      },
      {
        path: 'nodes',
        name: 'nodes',
        component: () => import('@/views/node/NodesView.vue'),
        meta: { title: 'node.title' },
      },
      {
        path: 'profile',
        name: 'profile',
        component: () => import('@/views/profile/ProfileView.vue'),
        meta: { title: 'profile.title' },
      },
      {
        path: 'tickets',
        name: 'tickets',
        component: () => import('@/views/ticket/TicketsView.vue'),
        meta: { title: 'ticket.title' },
      },
      {
        path: 'tickets/:id',
        name: 'tickets-detail',
        component: () => import('@/views/ticket/TicketDetailView.vue'),
      },
      {
        path: 'traffic',
        name: 'traffic',
        component: () => import('@/views/traffic/TrafficView.vue'),
        meta: { title: 'traffic.title' },
      },
    ],
  },
  {
    // 门户分流页:管理员登录后二选一(用户中心/管理后台),仅 role=1 可访问
    path: '/portal',
    name: 'portal',
    component: () => import('@/views/portal/PortalView.vue'),
    meta: { admin: true, title: 'portal.title' },
  },
  {
    // 管理后台独立布局(仅 role=1):meta.admin 挂父记录,vue-router 父子 meta 自动合并
    path: '/admin',
    component: AdminLayout,
    redirect: '/admin/overview',
    meta: { admin: true },
    children: [
      {
        path: 'overview',
        name: 'admin-overview',
        component: () => import('@/views/admin/AdminOverviewView.vue'),
        meta: { title: 'admin.overview' },
      },
      {
        path: 'reports',
        name: 'admin-reports',
        component: () => import('@/views/admin/AdminReportsView.vue'),
        meta: { title: 'admin.reports' },
      },
      {
        path: 'users',
        name: 'admin-users',
        component: () => import('@/views/admin/AdminUsersView.vue'),
        meta: { title: 'admin.users' },
      },
      {
        path: 'audit-logs',
        name: 'admin-audit-logs',
        component: () => import('@/views/admin/AdminAuditLogsView.vue'),
        meta: { title: 'admin.auditLogs' },
      },
      {
        path: 'plans',
        name: 'admin-plans',
        component: () => import('@/views/admin/AdminPlansView.vue'),
        meta: { title: 'admin.plans' },
      },
      {
        path: 'nodes',
        name: 'admin-nodes',
        component: () => import('@/views/admin/AdminNodesView.vue'),
        meta: { title: 'admin.nodes' },
      },
      {
        path: 'orders',
        name: 'admin-orders',
        component: () => import('@/views/admin/AdminOrdersView.vue'),
        meta: { title: 'admin.orders' },
      },
      {
        path: 'tickets',
        name: 'admin-tickets',
        component: () => import('@/views/admin/AdminTicketsView.vue'),
        meta: { title: 'admin.tickets' },
      },
      {
        path: 'coupons',
        name: 'admin-coupons',
        component: () => import('@/views/admin/AdminCouponsView.vue'),
        meta: { title: 'admin.coupons' },
      },
      {
        path: 'notices',
        name: 'admin-notices',
        component: () => import('@/views/admin/AdminNoticesView.vue'),
        meta: { title: 'admin.notices' },
      },
      {
        path: 'knowledges',
        name: 'admin-knowledges',
        component: () => import('@/views/admin/AdminKnowledgesView.vue'),
        meta: { title: 'admin.knowledges' },
      },
      {
        path: 'agent-applies',
        name: 'admin-agent-applies',
        component: () => import('@/views/admin/AdminAgentAppliesView.vue'),
        meta: { title: 'admin.agentApplies' },
      },
      {
        path: 'commission-logs',
        name: 'admin-commission-logs',
        component: () => import('@/views/admin/AdminCommissionLogsView.vue'),
        meta: { title: 'admin.commissionLogs' },
      },
      {
        path: 'traffic-import',
        name: 'admin-traffic-import',
        component: () => import('@/views/admin/AdminTrafficImportView.vue'),
        meta: { title: 'admin.trafficImport' },
      },
      {
        path: 'mail-templates',
        name: 'admin-mail-templates',
        component: () => import('@/views/admin/AdminMailTemplatesView.vue'),
        meta: { title: 'admin.mailTemplates' },
      },
      {
        path: 'subscription-templates',
        name: 'admin-subscription-templates',
        component: () => import('@/views/admin/AdminSubscriptionTemplatesView.vue'),
        meta: { title: 'admin.subscriptionTemplates' },
      },
      {
        path: 'version',
        name: 'admin-version',
        component: () => import('@/views/admin/AdminVersionView.vue'),
        meta: { title: 'admin.version' },
      },
      {
        path: 'settings',
        name: 'admin-settings',
        component: () => import('@/views/admin/AdminSettingsView.vue'),
        meta: { title: 'admin.settings' },
      },
    ],
  },
  {
    path: '/403',
    name: 'forbidden',
    component: () => import('@/views/ForbiddenView.vue'),
    meta: { title: 'forbidden.title' },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: 'notFound.title' },
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

/**
 * 空闲预取管理端重页面 chunk(仅管理员):reports(5 图表)/users(大表格)进入时
 * 零等待;普通用户不预取。在 router.isReady 后调用(需 pinia 已激活读 role)。
 */
export function prefetchAdminHeavyViews(isAdmin: boolean): void {
  if (!isAdmin) return
  const prefetch = (): void => {
    void import('@/views/admin/AdminReportsView.vue')
    void import('@/views/admin/AdminUsersView.vue')
  }
  if (typeof requestIdleCallback === 'function') requestIdleCallback(prefetch, { timeout: 3000 })
  else setTimeout(prefetch, 1500)
}

export default router
