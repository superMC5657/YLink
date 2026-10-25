# [P2] UI 规范批（mediaQuery / 侧栏宽 / ForgotView / palette / statusMeta-withdraw / 小项）

Status: resolved
Type: task

## Finding

1. `components/app/AppHeader.vue:27`：`computed(() => window.innerWidth < 768)` 无响应源，挂载即冻结；design-system §6.1 要求 `useMediaQuery`。
2. `AppSidebar.vue:3,38`：宽 200px vs pages.md §2.1 记载 240px/折叠 72px（二选一对齐）；`:3` 注释「ADMIN_NAV_GROUPS(13 项)」实为 18。
3. `views/auth/ForgotView.vue:100`：缺 `:show-feedback="false"` 与 4px 表单间距，pages.md §3.2「与登录页一致」。
4. `views/admin/AdminNodesView.vue:175,178,231`：在线/离线点用 `bg-emerald-400/500`、`bg-gray-400` 调色板类（README §8）→ `var(--c-success)`/`var(--c-text-sub)`。
5. **提现状态绕开 statusMeta**：`views/invite/InviteView.vue:42-49` 与 `views/admin/AdminTicketsView.vue:124-131,281-287` 各持本地 0/1/2 映射表，违反 `utils/statusMeta.ts` 唯一归属（progress.md 明言各域同源）。
6. 小项：`UserMailDialog.vue:15-20,38,47` 手写 `:value/@update:value`（兄弟均 defineModel）；`UserBalanceDialog.vue:53` 缺 batch 版有的 `amount===null` 守卫；`AdminTicketsView.vue:65` `openDetail(t)` 参数遮蔽 i18n `t`。

## Plan

1. AppHeader 改 `useMediaQuery('(max-width: 767px)')`。
2. **已拍板：代码为准，文档随改**：保留代码现状 200px 展开 / 72px 折叠（AppSidebar.vue:38、AdminSidebar.vue:24 实测；DrawerMenu 移动抽屉 280px 不变），pages.md §2.1「宽 240px」改为「宽 200px」（折叠 72px 与代码一致，无需改）；修 AppSidebar.vue:3 陈旧注释「13 项」→ 18 项。
3. ForgotView 补 show-feedback 与间距（对齐 Login/Register 写法）。
4. AdminNodesView 状态点改 CSS 变量。
5. statusMeta 增 `withdraw` 域（0/1/2 → 文案/徽标），两个视图改 `statusLabel/statusBadge('withdraw', …)`，删本地表。
6. 三个小组件项逐一修正。

## Acceptance

- 缩放窗口后 AppHeader 移动端分支随宽度切换（e2e 或组件测试）。
- `grep -rn "bg-emerald-\|bg-gray-400" src/views/admin` 语义色零命中（结构性灰阶除外）。
- 提现状态三处展示同源；`pnpm test` + e2e withdraw 全绿。
- pages.md §2.1 宽度描述与代码一致（200px/72px），无残留 240px 表述。

## Comments

2026-10-25: 用户拍板——侧栏宽度按「代码为准、文档随改」执行：保留 200px/72px，改 pages.md §2.1（原默认建议是改代码至 240px，已推翻）。

2026-10-25: 已修复 AppHeader 走 useMediaQuery（mobile.spec 新增视口切换 e2e 锁定双向响应）、AdminSidebar 陈旧注释 13→18 项、pages.md/progress.md 侧栏宽度 240→200px、ForgotView 补 show-feedback+4px 间距+按钮 mt-9、AdminNodesView 在线/离线点改 var(--c-success)/var(--c-text-sub)、statusMeta 新增 withdraw 域（InviteView×2 处与 AdminTicketsView 详情删本地 0/1/2 映射表同源，invite.withdrawStatus_0/1/2 死键已删，statusMeta.spec 补覆盖与 zh 文案守卫）、UserMailDialog 改 defineModel、UserBalanceDialog 补 amount===null 守卫、AdminTicketsView openDetail 参数改名不再遮蔽 i18n t。验证：pnpm lint/typecheck/format:check/test(244)/e2e withdraw×3 次全绿；Acceptance grep（bg-emerald-、bg-gray-400、240px、withdrawStatus_）零命中。

2026-10-25: 勘误——Finding/Plan 中「AppSidebar.vue:3 注释 ADMIN_NAV_GROUPS(13 项)」实际位于 AdminSidebar.vue:3（AppSidebar 用的是 NAV_GROUPS），已按实际位置修复。

2026-10-25: 范围外发现（未顺手修）——① tests/e2e/admin.spec.ts:7 文件头注释仍写「13 项管理菜单」（实为 18 项），与本次修的 AdminSidebar 注释同源陈旧；② tests/e2e/withdraw.spec.ts:47 `getByText('佣金提现申请')` 存在 strict-mode 歧义（子串同时命中工单标题 h1 与消息正文，高负载下偶发失败；干净 HEAD 复跑全量 e2e 同样失败，确属既有抖动，建议改 getByRole('heading', …)）。
