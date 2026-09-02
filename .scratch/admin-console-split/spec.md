# ✅ 管理端与用户端分拆 · 门户分流（Spec）

> Status: done（2026-08-28 实现完成，e2e 全绿，见 docs/frontend/progress.md）
> 日期: 2026-08-28
> 来源: 管理员登录后页面混乱的体验反馈
> 范围: 前端路由/布局/导航重构；后端权限模型（role=1）与 API 不变

---

## 1. 背景与痛点

当前管理员登录后与普通用户共用一个 `MainLayout`/`/dashboard`，管理端 13 个页面只是挂在同一布局下的 `/admin/*` 路由：

- `src/components/app/AppSidebar.vue` 与 `DrawerMenu.vue` 用 `[...NAV_GROUPS, ...ADMIN_NAV_GROUPS]` 把**用户菜单 10 项**（基础/财务/订阅/用户 4 组）与**管理菜单 13 项**直接拼在一个侧边栏，管理员侧边栏共 23 项，一屏装不下、两类功能混排；
- 管理页面与用户页面共用顶栏/客服浮球/下拉刷新等用户端设施，语境错位；
- 顶栏下拉只有单向「管理后台」入口，进入管理端后没有对称的「返回用户中心」入口；
- 移动端抽屉同样混排，管理功能未纳入任何独立呈现；
- `docs/frontend/pages.md` 中所有 `/admin/*` 页面标注的布局均为 MainLayout，需同步。

## 2. 目标

1. 管理端与用户端**彻底分拆布局与导航**，任一端侧边栏只含本端菜单；
2. 管理员登录后进入**门户分流页**（/portal），两张卡片「用户中心 / 管理后台」二选一；
3. 两端**对称切换**：侧边栏底部按钮 + 顶栏用户下拉同时提供互切入口；
4. 移动端：用户端抽屉不再含管理菜单；管理端提供**独立抽屉菜单**；
5. 普通用户完全不可见管理侧一切入口，访问 /portal 与 /admin/* 均重定向 /dashboard。

## 3. 方案设计

### 3.1 路由调整（src/router/index.ts）

- **新增 `/portal`**：独立组件 `PortalView`（居中双卡片，不带任何侧边菜单），`meta: { admin: true, title: 'portal.title' }`，仅管理员可访问；
- **`/admin/*` 整体迁移**：从 `MainLayout` children 移出，挂到新建的 `AdminLayout` 下；`meta.admin: true` 上移到 AdminLayout 路由记录（vue-router 父子 meta 自动合并），子路由只保留 `title`；新增 `/admin` → redirect `/admin/overview`；
- 用户端 `MainLayout` children 中不再存在任何 admin 路由；`/dashboard` 等 10 个用户页面不动。

### 3.2 布局与导航组件

**新增 `src/layouts/AdminLayout.vue`**
- 桌面：独立侧边栏 `AdminSidebar`（只渲染 `ADMIN_NAV_GROUPS`，13 项）+ 顶栏（站点名、「管理后台」徽标、主题/语言、用户下拉含「返回用户中心」）+ 底部「返回用户中心」按钮；
- 移动：顶栏汉堡 + 独立抽屉菜单（内容与桌面侧边栏一致），**不渲染 MobileTabBar**、不渲染客服浮球；
- 不接 `usePullToRefresh` 等用户端设施（可保留窗口聚焦静默刷新）。

**改造用户端**
- `AppSidebar.vue`：`groups` 只取 `NAV_GROUPS`；侧边栏底部新增「管理后台」按钮（`v-if="auth.isAdmin"`，跳 /admin/overview）；
- `DrawerMenu.vue`：移除管理菜单拼接，只保留用户端菜单；
- `AppHeader.vue`：用户下拉保留「管理后台」入口（文案明确：进入管理后台）。

**新增 `src/views/portal/PortalView.vue`**
- 居中卡片布局（视觉复用 AuthLayout 风格）；两张主卡片：「用户中心 → /dashboard」「管理后台 → /admin/overview」；
- 展示当前登录邮箱与退出登录；仅管理员可进入（meta.admin 兜底）。

### 3.3 导航定义（src/router/nav.ts）

- `NAV_GROUPS` / `MOBILE_TABS` 不变（纯用户端）；
- `ADMIN_NAV_GROUPS` 保留，作为 AdminLayout 唯一菜单源（顺带可按「经营总览/用户运营/业务管理/系统设置」二级分组，非必须）。

### 3.4 守卫与登录跳转（guards.ts / LoginView.vue）

| 场景 | 行为 |
|---|---|
| guest 页（/login 等）已登录 | 管理员 → `/portal`；普通用户 → `/dashboard` |
| 未登录访问任意页 | `/login?redirect=原路径`（不变） |
| 普通用户访问 `/admin/*` 或 `/portal` | 重定向 `/dashboard`（meta.admin，不变） |
| 登录提交成功 | `redirect` 参数优先；无 redirect 时管理员 → `/portal`、普通用户 → `/dashboard` |

### 3.5 i18n（src/locales/zh-CN.ts / en-US.ts）

新增 key：`portal.title`、`portal.welcome`、`portal.userCenter`、`portal.adminConsole`、`nav.backToUser`（返回用户中心）、`nav.enterAdmin`（管理后台入口）等。

### 3.6 测试调整（tests/e2e/admin.spec.ts）

- `loginAs` 期望 URL 由 `#/dashboard` 改为管理员落点 `#/portal`；
- 用例 1/2/3：登录后先点击门户卡「管理后台」再断言侧边栏/抽屉/顶栏内容；
- 新增断言：普通用户访问 `/portal` 重定向 /dashboard；管理员从用户端底部按钮/顶栏进入管理端；管理端「返回用户中心」回用户端；
- 普通用户"看不到管理菜单、访问 /admin/* 被重定向"的现有断言不变。

### 3.7 变更文件清单

| 动作 | 文件 |
|---|---|
| 新增 | `src/layouts/AdminLayout.vue`、`src/components/app/AdminSidebar.vue`、`src/views/portal/PortalView.vue` |
| 修改 | `src/router/index.ts`、`src/router/guards.ts`、`src/router/nav.ts`、`src/views/auth/LoginView.vue`、`src/components/app/AppSidebar.vue`、`src/components/app/AppHeader.vue`、`src/components/app/DrawerMenu.vue`、`src/locales/zh-CN.ts`、`src/locales/en-US.ts`、`tests/e2e/admin.spec.ts`、`docs/frontend/pages.md` |
| 后端 | 无（role=1 判定不变） |

### 3.8 非目标

- 不做独立管理端登录页（/admin/login），登录态与 token 体系不变；
- 不做管理端独立主题色（本次只拆结构，视觉强化可后续迭代）；
- 不改后端权限模型，不引入新角色。

## 4. 验收标准

- [x] A1 管理员登录 → `#/portal`，门户两卡可分别进入 `/dashboard` 与 `/admin/overview`
- [x] A2 用户端侧边栏/抽屉不再出现任何管理菜单项
- [x] A3 管理端侧边栏只含 13 项管理菜单，底部有「返回用户中心」可回用户端
- [x] A4 顶栏用户下拉在两端都含对称互切入口
- [x] A5 移动端：用户端抽屉无管理菜单；管理端独立抽屉只含管理菜单且可正常操作
- [x] A6 普通用户访问 `/admin/*`、`/portal` 均重定向 `/dashboard`，任何入口不可见
- [x] A7 `tests/e2e/admin.spec.ts` 全绿；`docs/frontend/pages.md` 路由表/布局章节同步更新
