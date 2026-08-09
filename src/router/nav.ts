/**
 * 导航结构定义:桌面侧边栏分组 + 移动端底栏 Tab。
 * 规范见 docs/frontend/pages.md §1-2。
 */

export interface NavItem {
  /** i18n key(渲染时用 t(name) 取当前语言文案) */
  name: string
  path: string
  icon: string
}

export interface NavGroup {
  /** i18n key(分组标题) */
  label: string
  items: NavItem[]
}

export const NAV_GROUPS: NavGroup[] = [
  {
    label: 'nav.groupBasic',
    items: [
      { name: 'nav.dashboard', path: '/dashboard', icon: 'home' },
      { name: 'nav.docs', path: '/docs', icon: 'book' },
    ],
  },
  {
    label: 'nav.groupFinance',
    items: [
      { name: 'nav.orders', path: '/orders', icon: 'order' },
      { name: 'nav.invite', path: '/invite', icon: 'gift' },
      { name: 'nav.agent', path: '/agent', icon: 'agent' },
    ],
  },
  {
    label: 'nav.groupPlan',
    items: [
      { name: 'nav.plans', path: '/plans', icon: 'zap' },
      { name: 'nav.nodes', path: '/nodes', icon: 'server' },
    ],
  },
  {
    label: 'nav.groupUser',
    items: [
      { name: 'nav.profile', path: '/profile', icon: 'user' },
      { name: 'nav.tickets', path: '/tickets', icon: 'ticket' },
      { name: 'nav.traffic', path: '/traffic', icon: 'traffic' },
    ],
  },
]

export interface TabItem {
  /** i18n key(渲染时用 t(name) 取当前语言文案) */
  name: string
  path: string
  icon: string
}

export const MOBILE_TABS: TabItem[] = [
  { name: 'nav.dashboard', path: '/dashboard', icon: 'home' },
  { name: 'nav.plans', path: '/plans', icon: 'zap' },
  { name: 'nav.tickets', path: '/tickets', icon: 'ticket' },
  { name: 'nav.mine', path: '/profile', icon: 'user' },
]

/** 展开导航所有路径,供路由激活匹配 */
export const ALL_NAV_PATHS = NAV_GROUPS.flatMap((g) => g.items.map((i) => i.path))
