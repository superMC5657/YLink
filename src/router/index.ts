import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'
import AdminLayout from '@/layouts/AdminLayout.vue'
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
        path: 'users',
        name: 'admin-users',
        component: () => import('@/views/admin/AdminUsersView.vue'),
        meta: { title: 'admin.users' },
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
        path: 'settings',
        name: 'admin-settings',
        component: () => import('@/views/admin/AdminSettingsView.vue'),
        meta: { title: 'admin.settings' },
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
