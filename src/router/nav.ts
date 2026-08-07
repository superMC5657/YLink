/**
 * 导航结构定义:桌面侧边栏分组 + 移动端底栏 Tab。
 * 规范见 docs/frontend/pages.md §1-2。
 */

export interface NavItem {
  name: string
  path: string
  icon: string
}

export interface NavGroup {
  label: string
  items: NavItem[]
}

export const NAV_GROUPS: NavGroup[] = [
  {
    label: '基础',
    items: [
      { name: '仪表板', path: '/dashboard', icon: 'home' },
      { name: '使用文档', path: '/docs', icon: 'book' },
    ],
  },
  {
    label: '财务',
    items: [
      { name: '我的订单', path: '/orders', icon: 'order' },
      { name: '邀请赚钱', path: '/invite', icon: 'gift' },
      { name: '申请代理', path: '/agent', icon: 'agent' },
    ],
  },
  {
    label: '订阅',
    items: [
      { name: '购买订阅', path: '/plans', icon: 'zap' },
      { name: '节点状态', path: '/nodes', icon: 'server' },
    ],
  },
  {
    label: '用户',
    items: [
      { name: '个人信息', path: '/profile', icon: 'user' },
      { name: '我的工单', path: '/tickets', icon: 'ticket' },
      { name: '流量明细', path: '/traffic', icon: 'traffic' },
    ],
  },
]

export interface TabItem {
  name: string
  path: string
  icon: string
}

export const MOBILE_TABS: TabItem[] = [
  { name: '仪表板', path: '/dashboard', icon: 'home' },
  { name: '购买订阅', path: '/plans', icon: 'zap' },
  { name: '我的工单', path: '/tickets', icon: 'ticket' },
  { name: '我的', path: '/profile', icon: 'user' },
]

/** 展开导航所有路径,供路由激活匹配 */
export const ALL_NAV_PATHS = NAV_GROUPS.flatMap((g) => g.items.map((i) => i.path))
