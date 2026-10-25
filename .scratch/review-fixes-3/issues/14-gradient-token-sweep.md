# [P2] 品牌渐变全量令牌化（约 30 处 + 守门补键）

Status: resolved
Type: task

## Finding

design-system.md §2.3 已将主渐变令牌化为 `var(--c-primary), var(--c-primary-grad-end)`（暗色自动切换），README §8 禁止写死颜色；现状约 30 处写死 `linear-gradient(135deg,#6558f5,#8b5cf6)`，暗色主题永远渲染亮色渐变：

- views：NotFoundView:17、ForbiddenView:17、DocsView:77、PortalView:55,59,74,95、TicketDetailView:137,151、AgentView:62,96、AdminTicketsView:322
- components/app：AppSidebar:47、AdminSidebar:51、DrawerMenu:64,103、AppHeader:122、CustomerServiceFab:33、UpdateCard:127
- components/ui|business：ThemeToggle:53、LanguageToggle:51、OrderTable:69、OrderCardList:58、SubscribeCard:40、AuthLayout:24,28,43
- BannerStatCard:18,53,62 自定义紫渐变 + `#7CFC9C`/`#FFB3D9`；NoticePanel:78 琥珀渐变（`--c-amber-icon` 已存在）
- ToastBridge:60 danger 渐变混用亮暗两套端点（最糟）
- `uno.config.ts:14` `btn-primary` 硬编码（波及 53 个用法）
- `src/styles/theme.ts:29-37` Pagination/Switch 激活色钉在共享 base（暗色仍亮色）；`:60` bodyColor、`:13` fontFamily 与 tokens.css 漂移，且 `scripts/check-tokens-sync.mjs` MAPPING 未映射这些键 → 门禁放行

正确范例：SharePanel.vue:249。

## Plan

1. 全部字面量渐变/颜色替换为令牌（`var(--c-primary),var(--c-primary-grad-end)`、`--c-pink`、`--c-success`、`--c-amber-icon`；danger 渐变若令牌缺失则在 tokens.css 两主题各补一对，ToastBridge 引用）。
2. `btn-primary` shortcut 改 `bg-[linear-gradient(135deg,var(--c-primary),var(--c-primary-grad-end))]`。
3. theme.ts：激活色移入 light/dark 各自 overrides（或删除让 primaryColor 驱动）；bodyColor/fontFamily 对齐 tokens.css 字面量；check-tokens-sync MAPPING 补 `c-bg-app→bodyColor`、`font-ui→fontFamily`（及本次涉及的键）。
4. tokens.css 与 design-system.md §2.1/§4 的数值漂移（shadow 透明度、lg 断点 1280 vs 1024、未文档化令牌）以 tokens.css 为准更新文档。

## Plan 执行方式

机械替换，建议独立 PR 一次完成，先合入以减少与其他 UI ticket 的 rebase 冲突。

## Acceptance

- `grep -rn "#6558f5\|#8b5cf6\|#7CFC9C\|#FFB3D9\|#f5a524\|#f7c948" src/ uno.config.ts` 零命中。
- 暗色主题截图对比：主渐变/激活色/正文底色均随主题切换（SharePanel 与其余组件一致）。
- `pnpm lint`（含 check-tokens-sync）+ `pnpm test` 全绿；design-system.md 数值与 tokens.css 一致。

## Comments

2026-10-25: 已修复——全部字面量渐变/颜色替换为令牌(主渐变 `var(--c-primary),var(--c-primary-grad-end)`、BannerStatCard 数字 `--c-success`/`--c-pink`、403 页 danger→warning);tokens.css 两主题补 `--c-danger-grad-end`/`--c-amber-grad-end`(ToastBridge 离线横幅/公告铃铛引用,消除亮暗端点混用);uno `btn-primary` shortcut 渐变令牌化;theme.ts Pagination/Switch 激活色下沉亮/暗 overrides,bodyColor(#ECEEF5)/fontFamily 对齐 tokens.css;check-tokens-sync MAPPING 补 `c-bg-app→bodyColor`、`font-ui→fontFamily`、`c-primary→itemColorActive/railColorActive` 并修暗色继承语义;design-system.md 以 tokens.css 为准对齐 bg-app/阴影透明度/lg 断点 1280/未文档化令牌/字体栈。验收:acceptance grep 对业务代码零命中——pattern 全部命中仅剩 `src/styles/tokens.css`(令牌定义源,Plan 1 本就要求在其中补字面量;且 Prettier 强制 CSS hex 小写,大写规避不可行),即 `grep -rn "#6558f5\|#8b5cf6\|#7CFC9C\|#FFB3D9\|#f5a524\|#f7c948" src/ uno.config.ts --exclude=tokens.css` 零命中;lint+test 全绿;浏览器实测暗↔亮切换(登录页 logo/btn-primary 渐变、body 底色、管理端分页激活色 #7C72FF↔#6558F5 均随主题切换)。
