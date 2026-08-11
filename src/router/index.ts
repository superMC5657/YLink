import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'
import AuthLayout from '@/layouts/AuthLayout.vue'

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
      // ---------- 管理后台(仅 role=1,守卫见 guards.ts) ----------
      {
        path: 'admin/overview',
        name: 'admin-overview',
        component: () => import('@/views/admin/AdminOverviewView.vue'),
        meta: { admin: true, title: 'admin.overview' },
      },
      {
        path: 'admin/users',
        name: 'admin-users',
        component: () => import('@/views/admin/AdminUsersView.vue'),
        meta: { admin: true, title: 'admin.users' },
      },
      {
        path: 'admin/plans',
        name: 'admin-plans',
        component: () => import('@/views/admin/AdminPlansView.vue'),
        meta: { admin: true, title: 'admin.plans' },
      },
      {
        path: 'admin/nodes',
        name: 'admin-nodes',
        component: () => import('@/views/admin/AdminNodesView.vue'),
        meta: { admin: true, title: 'admin.nodes' },
      },
      {
        path: 'admin/orders',
        name: 'admin-orders',
        component: () => import('@/views/admin/AdminOrdersView.vue'),
        meta: { admin: true, title: 'admin.orders' },
      },
      {
        path: 'admin/tickets',
        name: 'admin-tickets',
        component: () => import('@/views/admin/AdminTicketsView.vue'),
        meta: { admin: true, title: 'admin.tickets' },
      },
      {
        path: 'admin/coupons',
        name: 'admin-coupons',
        component: () => import('@/views/admin/AdminCouponsView.vue'),
        meta: { admin: true, title: 'admin.coupons' },
      },
      {
        path: 'admin/notices',
        name: 'admin-notices',
        component: () => import('@/views/admin/AdminNoticesView.vue'),
        meta: { admin: true, title: 'admin.notices' },
      },
      {
        path: 'admin/knowledges',
        name: 'admin-knowledges',
        component: () => import('@/views/admin/AdminKnowledgesView.vue'),
        meta: { admin: true, title: 'admin.knowledges' },
      },
      {
        path: 'admin/agent-applies',
        name: 'admin-agent-applies',
        component: () => import('@/views/admin/AdminAgentAppliesView.vue'),
        meta: { admin: true, title: 'admin.agentApplies' },
      },
      {
        path: 'admin/commission-logs',
        name: 'admin-commission-logs',
        component: () => import('@/views/admin/AdminCommissionLogsView.vue'),
        meta: { admin: true, title: 'admin.commissionLogs' },
      },
      {
        path: 'admin/traffic-import',
        name: 'admin-traffic-import',
        component: () => import('@/views/admin/AdminTrafficImportView.vue'),
        meta: { admin: true, title: 'admin.trafficImport' },
      },
      {
        path: 'admin/settings',
        name: 'admin-settings',
        component: () => import('@/views/admin/AdminSettingsView.vue'),
        meta: { admin: true, title: 'admin.settings' },
      },
    ],
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

export default router
