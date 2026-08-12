# 前端开发 · 进度追踪(已完成 / 未完成 / 前置条件)

> 本文档记录 `src/` 目录 Vue 3 用户端应用的开发状态,是 docs/frontend 与 docs/api 的实现对照表。
> 更新规则:每完成一个里程碑/修复一个缺陷,同步更新本文档「已完成」;新增缺口写入「未完成」并标注依赖。
> 最后更新:2026-08-12(前置条件实测核对:Node 24.14.0 / pnpm 10.33.0 / Rust 1.97.1 均满足;路由 29 条、vitest 43 用例等与代码实况对齐;管理后台 13 模块全部完成,见 M8+M9)

---

## 1. 已完成项

### M1 脚手架(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| 工程初始化 | Vue 3.5 + TS + Vite 6 + pnpm,`<script setup>`;Node ≥ 20 | `package.json`、`vite.config.ts`、`tsconfig.json` |
| 原子化 CSS | UnoCSS(presetUno / presetAttributify / presetIcons / transformerDirectives),自定义 shortcuts(`btn-primary`/`btn-olive`/`card-base` 等) | `uno.config.ts` |
| 路由 | Vue Router 4 hash 模式,29 条路由(16 用户端 + 13 管理端:login/register/forgot/dashboard/docs/docs-detail/orders/invite/agent/plans/nodes/profile/tickets/tickets-detail/traffic/404 + admin 13 页,guest/登录双守卫 + 页面标题) | `src/router/index.ts`、`guards.ts`、`nav.ts` |
| 状态管理 | Pinia + persistedstate,12 个业务 store | `src/stores/*` |
| 国际化 | vue-i18n@11,zh-CN / en-US 按模块命名空间,`Accept-Language` 联动 | `src/locales/*` |
| 环境变量 | `VITE_API_BASE_URL` / `VITE_USE_MOCK` / `VITE_APP_NAME`,运行时后端地址可改并持久化 | `.env.development`、`.env.production`、`utils/storage.ts` |
| 防闪烁 | `public/theme.js` 独立首帧脚本(外链满足 CSP script-src 'self'),渲染前读持久化主题写入 `data-theme` | `public/theme.js`、`src/index.html` |
| Mock | vite-plugin-mock,8 个模块文件(auth/business/config/order/server/user/notices/admin)严格按契约(含 401/错误码/支付自动完成模拟);**2026-08-11 公告数据源合并至 `mock/notices.ts`(用户端 `GET /notices` 与管理端 `/admin/notices` CRUD 读写同一数组,修复管理后台发布的公告用户端不可见);管理端 Mock 在 `mock/admin.ts`(13 模块端点)** | `mock/*.ts` |

### M2 布局与设计系统(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| 设计令牌 | 亮/暗两套 CSS 变量(色彩/形状/阴影/动效/字体),业务代码不写死颜色 | `src/styles/tokens.css` |
| Naive UI 覆盖 | 亮/暗两套**真实色值** overrides(seemly 不支持 CSS 变量字符串,已规避) | `src/styles/theme.ts` |
| 桌面布局 | 240px 侧边栏(折叠 72px)+ 毛玻璃吸顶顶栏(折叠钮/站点名/主题/语言/用户 chip) | `components/app/AppSidebar.vue`、`AppHeader.vue` |
| 平板/手机 | `<768px` 底栏 4 Tab + 抽屉菜单(分组一致);`768-1024` 迷你侧边栏;useMediaQuery 断点驱动 | `MobileTabBar.vue`、`DrawerMenu.vue`、`MainLayout.vue` |
| 全局壳 | AuthLayout(居中卡片 + 氛围背景 + 语言/主题)、客服浮球、naive message/dialog provider + toast 桥接 | `layouts/*`、`CustomerServiceFab.vue`、`ToastBridge.vue` |
| 基础组件 | AppIcon(离线线性图标 60+)/ UiCard / StatNumber / StatusBadge / PriceText / EmptyState / PageHeader / CopyText / LanguageToggle / ThemeToggle(三态循环),全部全局注册 | `components/ui/*` |

### M3 核心页(✅ 完成)

| 页面 | 组件与交互 | 数据来源 |
|---|---|---|
| 仪表板 | Banner 统计卡(余额绿/佣金粉)、订阅卡(到期徽章/进度条 80%/95% 变色/五宫格)、公告手风琴(markdown-it + DOMPurify;**正文反引号包裹的优惠码渲染为高亮 chip,点击一键复制**,显示最新 10 条)、9 宫格快捷操作(一键导入/免费流量/APP 下载弹窗);窗口聚焦静默刷新 | `GET /user/stat`、`/user/subscribe`、`/notices` |
| 使用文档 | 300ms 防抖搜索(前端过滤 + 传 keyword)、语言切换、分类分组、详情 markdown 渲染(代码块/强标记营销红) | `GET /knowledges`、`/knowledges/{id}` |
| 我的订单 | 表格/卡片双视图切换、状态筛选、分页、详情弹窗(屏幕中央,全字段 + 支付入口 + 取消)、待支付 5s 轮询 | `GET /orders`、`/orders/{no}`、`cancel` |

### M4 交易闭环(✅ 完成)

| 能力 | 说明 | 数据来源 |
|---|---|---|
| 套餐卡 | 3/2/1 列响应式网格、周期胶囊切换联动价格与「省 N%」、markdown 营销红字 | `GET /plans` |
| 下单弹窗 | 周期选择 → 优惠券试算(成功显示减免/失败红字)→ **可用优惠券列表展示(点选自动填入试算, `GET /coupons/available`)** → 支付方式(余额不足置灰显差额)→ 费用明细 → 提交;`Idempotency-Key` 幂等建单 | `POST /coupons/check`、`POST /orders` |
| 收银台 | 二维码(qrcode 库渲染)/ 跳转 URL 两种形态、1800s 倒计时、3s 轮询订单、支付成功结果卡、余额直付即完成 | `POST /orders/{no}/checkout`、`GET /orders/{no}` |
| 一键导入 | 客户端选择弹窗(Clash/sing-box/Shadowrocket/v2rayNG 等 10 款),scheme 唤起 + 复制订阅链接兜底 | `utils/deeplink.ts` |

### M5 营销与用户页(✅ 完成)

| 页面 | 说明 |
|---|---|
| 邀请赚钱 | 5 张统计卡(佣金粉/比例/注册数/累计/确认中)+ 划转弹窗(≤ 可划转余额)、邀请码增删复制、注册链接复制、佣金记录(仅已发放) |
| 申请代理 | 左侧申请卡(四态按钮:未达标灰/可申请主色/审核中 warning/已代理 success)+ 条件 ✓/✗ + 特权/注意事项(取站点配置) |
| 节点状态 | 分组卡片、状态点(正常 ping 动画/拥挤黄/维护灰)、类型/倍率/标签徽章、60s 静默轮询;**不展示** host/port |
| 个人信息 | Banner 复用、通知开关即改即存、下划线式改密(清空/保存)、Telegram 外链、重置订阅(密码二次确认 → 新链接展示 + 提示重导) |
| 我的工单 | 列表卡片、新建弹窗(主题/优先级/内容)、详情对话气泡流(用户右/客服左)、回复/关闭(已关闭置灰) |
| 流量明细 | 近 7/30 天/自定义范围、ECharts 堆叠柱状图(亮暗主题联动)、汇总卡、明细表 |
| 404 | 渐变数字 + 返回首页 |

### M7 桌面分辨率适配(✅ 完成)

| 项 | 说明 |
|---|---|
| 内容区宽度 | `max-w-[1200px]` → `max-w-[1440px]`:1920 屏利用率 62%→75%,2560 屏留白减半;窄屏(<1200)自动全宽 |
| 窄桌面侧边栏 | 1024-1279px 视口进入主布局时自动折叠为 72px(内容区 +160px),用户可手动展开 |
| 表格适配 | 订单表/邀请页两张表包 `overflow-x-auto` 容器:窄屏表格内部滚动,页面不再被撑破;订单表紧凑化(单元格 padding/字号)+ 订单号截断 12 字符,1440 屏免滚动 |
| docs 搜索修复 | 搜索行 flex 组合 `flex-1 md:w-64 md:flex-none` 冲突导致溢出,改 `min-w-0 flex-1 md:max-w-72` |
| 溢出兜底 | `<main>` 增加 `overflow-x-hidden`,任何横向泄漏不外溢 |
| 诊断工具 | `scripts/diag-layout.mjs`:5 分辨率 × 12 路由自动测量(页面级溢出/内容区宽度/表格滚动),当前 0 异常 |
| 验证 | E2E 32/32(登录用例适配登录页不再预填账号)+ typecheck + lint 全绿 |

### 审查修复(✅ 已修复,阻断项清零)

| 问题 | 修复 |
|---|---|
| **全站字号放大 4 倍(字体比例不协调根因)** | presetUno 的 `text-<number>` 规则按 4px 基准转 rem(`text-13` → `3.25rem` = 52px),导致 292 处字号类全部放大 4 倍(11→44px、16→64px、28→112px),文字溢出按钮/撑爆卡片。在 `uno.config.ts` 增加自定义规则 `[/^text-(\d+)$/ → font-size: Npx]` 覆盖(UnoCSS 用户规则优先),全站字号回归规范阶梯 11/12/13/14/16/18/20/28/32;`scripts/diag-font.mjs` 测量验证 |
| persistedstate 插件用 `app.use()` 注册导致 `context.pinia` 未定义,首帧崩溃 | 改为 `pinia.use(piniaPluginPersistedstate)` |
| naive-ui `create()` 默认注册空组件,`n-config-provider` 等全部无法解析 | 显式列出 18 个用到的组件 |
| themeOverrides 传 CSS 变量(`var(--c-primary)`)被 seemly 解析报错 | 拆亮/暗两套真实色值 overrides,与 tokens.css 双源同步(见前置条件 3.5) |
| 业务组件直接使用 `<AppIcon>`/`<StatusBadge>` 等而未局部 import | 基础 UI 组件统一全局注册 |
| vite-plugin-mock 按文件隔离编译,跨文件 sessions Map 不共享导致 401 | token 校验改无状态格式匹配(`Bearer mock-access-*`) |
| 冒烟访问 `/plans` 被重定向到 dashboard | hash 路由下脚本改用 `/#/` 前缀 |

### M6 桌面化(Tauri 2)(✅ 骨架 + 发布能力完成,2026-08-12)

| 项 | 说明 | 位置 |
|---|---|---|
| Rust 工程 | Cargo.toml(10 个插件 crate)/ build.rs / main.rs / lib.rs(插件注册 + 托盘 + 单实例) | `src-tauri/` |
| 配置 | tauri.conf.json(1280×800 主窗、CSP、bundle targets nsis 仅 Windows) | `src-tauri/tauri.conf.json` |
| 权限 | capabilities/default.json 最小授权(核心 + 10 插件,http scope 限 https/localhost) | `src-tauri/capabilities/default.json` |
| 图标 | `pnpm tauri icon` 从脚本生成源图产出全套(ico/icns/png/Android) | `src-tauri/icons/`、`scripts/gen-icon.py` |
| 前端适配 | platform.ts(clipboard/opener/deep-link 动态 import)、http.ts(plugin-http 原生 fetch)、app.ts(applyTheme 同步窗口标题栏)、main.ts(深链接路由跳转) | `src/utils/*`、`src/stores/app.ts`、`src/main.ts` |
| 验证 | `cargo check` 通过(首次编译约 5 分钟);Web 构建不受动态 import 影响 | — |

### M6 发布能力收尾(✅ 2026-08-12,更新卡片 / 单实例深链转发 / 本地通知)

| 项 | 说明 | 位置 |
|---|---|---|
| 更新卡片 UI | `utils/updater.ts`(checkForUpdate / downloadAndInstall,动态 import,Web 端自动降级)+ `components/app/UpdateCard.vue`(右下角浮动卡片:版本号 + 更新日志 + 下载进度 + 立即更新/稍后,监听 `app:check-update` 事件)+ App.vue 挂载启动静默检查 + 设置页「检查更新」入口(仅 Tauri 显示,含当前版本号);失败静默忽略 | `src/utils/updater.ts`、`src/components/app/UpdateCard.vue`、`src/App.vue`、`views/profile/ProfileView.vue` |
| 单实例深链接转发 | `lib.rs` 单实例回调从 argv 提取 `ylink://` URL,用 deep-link 插件同名事件 `deep-link://new-url`(payload URL 数组)emit 给已有实例,前端 `onOpenUrl` 直接路由跳转;端到端链路打通 | `src-tauri/src/lib.rs`(Emitter) |
| 本地通知触发点 | `utils/notify.ts` 统一封装(Tauri plugin-notification / Web Notification API 降级);触发点:支付成功(PaymentModal)、工单已回复(MainLayout 60s 轮询 + 状态快照去重)、订阅到期 ≤3 天(窗口聚焦刷新检测 + 按到期日去重) | `src/utils/notify.ts`、`src/composables/useLocalNotifications.ts`、`layouts/MainLayout.vue`、`components/business/PaymentModal.vue` |

### 工程化与质量门禁(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| ESLint 9 flat config | js + typescript-eslint + eslint-plugin-vue(flat/recommended)+ eslint-config-prettier,0 error 0 warning | `eslint.config.ts` |
| Prettier 3 | 全仓格式化 + format:check | `.prettierrc.json`、`.prettierignore` |
| Vitest 单测 | jsdom,**43 用例**(格式化/http 401 刷新重放/invite store/倒计时 + PlanCard/OrderTable/BannerStatCard 组件测试 12 例),v8 覆盖率 | `vitest.config.ts`、`src/**/__tests__/*` |
| Playwright E2E | 正式套件 42 例(21 用例 × 桌面/移动双 project、webServer 自动起 Mock),替代原冒烟脚本 | `playwright.config.ts`、`tests/e2e/*` |
| CI | 仅发布 tag(`v*`)触发 3 job(2026-08-12 调整:此前 push main/PR 每次提交都跑,改后日常检查走本地 `pnpm lint/typecheck/test/format:check`):`frontend-quality`(lint→typecheck→format:check→test→build:web)+ `frontend-e2e`(失败上传报告)+ `rust`(windows-latest:build:web → cargo check);Go 后端不走 Actions(项目决策) | `.github/workflows/ci.yml` |
| i18n 懒加载 | 语言包按需动态 import,useLocale 统一切换 | `src/locales/index.ts`、`src/composables/useLocale.ts` |
| 注册页强制邀请码 | 站点 `invite_code_required=true` 时校验必填 | `src/views/auth/RegisterView.vue` |
| 离线横幅 | 顶部红色常驻横幅 + 恢复 toast | `src/components/app/ToastBridge.vue` |

### M8 管理后台(✅ 核心模块完成,2026-08-10)

| 项 | 说明 | 位置 |
|---|---|---|
| 角色区分基建 | auth store `isAdmin` getter;路由 meta `admin` 标志 + 守卫(非管理员访问 `/admin/*` 重定向 `/dashboard`);侧边栏/移动端抽屉按角色追加「管理后台」分组;顶栏用户下拉管理员增加「管理后台」入口 | `src/stores/auth.ts`、`src/router/*`、`components/app/*` |
| 管理端契约 | `src/api/admin.ts` 封装已实现 6 模块端点(总览/用户/套餐/节点/订单/工单);类型对齐后端 DTO(价格统一为元);剩余 7 组端点(优惠券/公告/知识库/代理审批/佣金/流量导入/站点设置)当时未封装(**2026-08-11 M9 已全部补齐**,见下) | `src/api/admin.ts`、`src/types/api.d.ts` |
| 总览 | 7 项运营统计卡 + 快捷入口 | `views/admin/AdminOverviewView.vue` |
| 用户管理 | 搜索/分页/封禁/角色调整/调余额(均写审计) | `views/admin/AdminUsersView.vue` |
| 套餐管理 | CRUD:周期定价(元)/流量/设备/限速/节点分组/上架/排序 | `views/admin/AdminPlansView.vue` |
| 节点管理 | 分组 CRUD + 节点 CRUD(6 协议/地址/配置 JSON/倍率/状态/标签) | `views/admin/AdminNodesView.vue` |
| 订单管理 | 状态筛选/分页/退款(余额退回+佣金回滚) | `views/admin/AdminOrdersView.vue` |
| 工单管理 | 列表/详情/客服回复/关闭 | `views/admin/AdminTicketsView.vue` |
| Mock + E2E | `mock/admin.ts` 管理端 Mock;`tests/e2e/admin.spec.ts` 角色区分 6 用例 × 双 project(管理员可见入口/普通用户不可见且访问被重定向) | `mock/*`、`tests/e2e/*` |

### M9 管理后台二期剩余 7 模块(✅ 完成,2026-08-11)

| 模块 | 说明 | 位置 |
|---|---|---|
| 优惠券管理 | 列表/新建/编辑/删除:类型(固定金额/百分比)/面值/最低消费/每人限用/总量/适用周期/适用套餐/生效失效时间/启停;**一键公告(按优惠券生成公告草稿,标题/正文预填且可编辑,优惠码反引号包裹发布后用户端高亮可复制,调 `POST /admin/notices`)** | `views/admin/AdminCouponsView.vue` |
| 公告管理 | 列表/发布/编辑/删除:标题/内容(Markdown)/展示开关/排序 | `views/admin/AdminNoticesView.vue` |
| 知识库管理 | 列表(语言筛选+关键词搜索)/新建/编辑/删除:分类/标题/正文(Markdown)/语言/展示/排序 | `views/admin/AdminKnowledgesView.vue` |
| 代理审批 | 状态筛选/分页/通过/拒绝(通过后升级代理商) | `views/admin/AdminAgentAppliesView.vue` |
| 佣金日志 | 状态筛选(确认中/已发放/已撤销)/分页,展示邀请人/被邀请人/订单/比例/佣金 | `views/admin/AdminCommissionLogsView.vue` |
| 流量导入 | 模式 B 手工导入:多行 user_id/date/u/d(字节),批量提交写审计 | `views/admin/AdminTrafficImportView.vue` |
| 站点设置 | 按 key(site/payment/invite/agent/order/templates)编辑配置 JSON,保存后缓存失效 | `views/admin/AdminSettingsView.vue` |
| API 封装 | `apiAdmin` 新增 7 组端点(优惠券/公告/知识库/代理审批/佣金日志/流量导入/站点设置),类型对齐契约 §16.1 | `src/api/admin.ts`、`src/types/api.d.ts` |
| Mock | `mock/admin.ts` 补齐 7 组端点(含 CRUD 状态变更);移除无契约引用的遗留「佣金管理」幽灵端点 | `mock/admin.ts` |
| 后端配套 | 新增 `GET /admin/notices`、`GET /admin/knowledges`(含隐藏);优惠券列表改返回 `AdminCouponView`(展开 type/value/used_count 等,价格转元) | `server/internal/**`(见 backend progress) |
| i18n | 修复既存缺陷:admin 页面标题 key 从未在语言包定义(标签页显示原始 key),补 `admin.*` 13 项(zh/en) + nav 7 项 | `src/locales/*.ts` |
| config 缓存修复 | 站点配置 localStorage 缓存 24h→60s(对齐后端 Redis 60s),申请代理页进页 `fetchConfig(true)` 强制刷新:管理后台改代理政策/注册开关/支付方式后用户端 ≤60s 生效 | `src/stores/config.ts`、`views/agent/AgentView.vue`(见 data-layer.md §8) |

### 验证状态(✅,2026-08-10 全量实测)

- `pnpm typecheck`(vue-tsc --noEmit)零错误
- `pnpm build`(vue-tsc + vite build)成功,主包 gzip ≈ 254KB;PWA 产物:dist/sw.js(precache 62 项)+ manifest.webmanifest + registerSW.js
- `pnpm lint` 0 error 0 warning;`pnpm test` **43/43 通过**(2026-08-11 一期小缺口收尾后:新增 PlanCard/OrderTable/BannerStatCard 组件测试 12 例)
- `pnpm e2e`(Playwright)**42 例通过**(2026-08-11 `--list` 复核:21 用例 × 双 project):登录 → 仪表板 → 套餐 → 下单(余额支付)→ 订单 → 7 页面可达性 → 暗色切换 → 移动端底栏导航 + **管理后台角色区分 12 例**(6 用例 × 双 project;调试残留 `zz-errfail.spec.ts` 已移除)
- `cargo check`(src-tauri)通过
- **M9 二期 7 模块(2026-08-11)**:`pnpm typecheck` / `pnpm lint`(0 warning)/ `pnpm test`(31/31)/ `pnpm build` 全绿;后端 `go build`/`go vet`/`gofmt -l`(0 输出)/`go test`(47 函数)全绿;7 个新页面已注册路由 + 管理端菜单,交互验证待手动(Mock 管理员 `admin@example.com` / `Admin@123456`)
- **一期小缺口收尾(2026-08-11)**:`pnpm typecheck`/`pnpm lint`/`pnpm test`(43/43)/`pnpm build`(含 PWA)全绿;commit-msg 钩子实测规范消息通过、无 type 消息被拒
- **M6 发布能力收尾(2026-08-12)**:更新卡片 UI / 单实例深链转发 / 本地通知三项完成(见第 1 节);`pnpm typecheck`/`pnpm lint`(0 warning)/`pnpm test`(43/43)/`pnpm build`(含 PWA)/`pnpm format:check` 全绿;`cargo check` 通过;后端 `go build`/`go vet`/`gofmt -l`(0 输出)/`go test`(**64 函数**)全绿(含工单重开 4 例);桌面端交互(更新卡片/深链唤起/本地通知)待手动实测

### 全量审查修复(✅ 2026-08-11)

第三轮全仓库审查(见 docs/reviews/review-0.2.0.md)的前端修复:

| 问题 | 修复 |
|---|---|
| 流量明细响应形状与契约不符(裸数组 vs `{list}`) | `src/api/user.ts` 类型改为 `{ list }`,`user store` 解析 `.list`,`mock/user.ts` 对齐契约 |
| 会话过期跳转丢失 hash 路由 / Tauri 下 404 | `utils/http.ts` 改为 `location.hash = '#/login?redirect=…'` |
| 标签页标题显示原始 i18n key | `router/guards.ts` 用 `i18n.global.t(title)` 翻译 |
| FOUC 脚本读取错误形状 + 内联脚本被 CSP 拦截 | 内联脚本移至 `public/theme.js`(JSON.parse 直接取模式),满足 `script-src 'self'` |
| 通知开关未回填 / 视图直调 api | `ProfileView` 挂载时 `fetchProfile` 回填,改用 `user store` actions;`OrderConfirmModal` 优惠券试算改走 `orderStore.checkCoupon`;`InviteView` 不再直改 store 状态 |
| deep-link 插件未注册 | `Cargo.toml` 加 `tauri-plugin-deep-link`,`lib.rs` 注册,capabilities 加 `deep-link:default`;删除 `tauri.conf.json` updater 占位 |
| 二维码 / ECharts 硬编码颜色 | `PaymentModal`、`TrafficView` 运行时读取 CSS 变量(设计规范 tokens.css) |
| 省 N% 计算与周期文案重复 3 份 | 抽取共享工具 `planSavePercent` / `periodLabel`(`utils/format.ts`) |
| `apiPlan.list` 死代码/类型矛盾 | 删除 |
| CopyText 展示文本不响应 prop 变化 | 改用 `computed` |
| AdminUsersView 重复实现 formatBytes | 删除本地实现,统一用 `formatBytes` |
| E2E 调试残留 | 删除 `zz-errfail.spec.ts`;`mobile.spec.ts` 未断言调用改为 `expect(...).toBeVisible()` |
| Mock 优惠券可无限次使用(2026-08-12) | `mock/order.ts` 的 `POST /orders` 原先无任何限用校验,任意用户可无限次用同一张券下单;补齐「每人限用」:`couponLimitPerUser`/`couponUsage`(下单即占用,种子订单计入) + `/coupons/available` 过滤已用满 + `/coupons/check` 与 `/orders` 拒绝 12001;同时统一三处折扣口径(couponDiscount 共享),修掉 WELCOME 在 check=5.0 与 orders=1.5 不一致;真实后端校验本就完整,另把 `validateCoupon`/`AvailableCoupons` 中 `err == nil &&` 的宽松判断改为查询失败保守拒绝/过滤 |

---

## 2. 未完成项

### M6 桌面化(Tauri 2)—— 发布能力已收尾(✅ 2026-08-12)

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| `src-tauri/` 工程 | ✅ `Cargo.toml`/`build.rs`/`main.rs`/`lib.rs`/`tauri.conf.json`/`capabilities/default.json`/全套图标,`cargo check` 通过 | 契约见 docs/frontend/desktop-tauri.md;Rust 1.97 已满足 |
| 插件接入(Rust) | ✅ http / store / clipboard / opener / single-instance / autostart / notification / process / window-state / os | capabilities 最小授权;http scope 限 `https://**` + localhost |
| 平台适配层(前端) | ✅ `utils/platform.ts` 启用 Tauri 分支(clipboard/opener/deep-link 动态 import);`utils/http.ts` 走 plugin-http 原生 fetch;`stores/app.ts` applyTheme 同步窗口标题栏;`main.ts` 深链接路由跳转 | Web 端自动降级不受影响 |
| 托盘/单实例 | ✅ 托盘菜单(显示主窗口/退出)+ single-instance 聚焦已有窗口 | lib.rs setup |
| deep-link 注册 | ✅ 插件已注册(Rust `#[cfg(desktop)]` init)+ capabilities `deep-link:default`;前端 `onDeepLink` 路由跳转已就绪(main.ts);**2026-08-12 单实例已转发 argv/深链 URL**(见第 1 节 M6 发布能力收尾) | Release 域名与 `ylink://` 注册见 desktop-tauri.md §3/§4/§5 |
| 自动更新 | ✅ 已全部就绪(2026-08-12):Rust 已注册 updater、`tauri.conf.json` 已配 pubkey+endpoints、Release 流水线产出 `latest.json`,前端更新卡片已实现(见第 1 节 M6 发布能力收尾) | desktop-tauri.md §5 |
| 通知触发点 | ✅ 已实现(2026-08-12):到期/工单回复/支付成功本地通知(见第 1 节 M6 发布能力收尾) | — |
| Windows 打包与 updater 签名 | ✅ 已配置(2026-08-12):Release 流水线 + 签名密钥已生成;2026-08-12 收窄为仅 Windows 打包 | `.github/workflows/release-tauri.yml` + `TAURI_SIGNING_PRIVATE_KEY`(见 desktop-tauri.md §5/§7) |
| 存储适配 | ⚠️ 一期统一 localStorage(WebView 持久化);plugin-store(JSON 文件)异步 API 与 persistedstate 同步接口冲突,迁移需异步化改造 | 见 storage.ts 注释 |

### 一期小缺口收尾(✅ 2026-08-11)

| 项 | 说明 | 位置 |
|---|---|---|
| 组件渲染测试 | 补关键业务组件测试:PlanCard(价格/周期切换/购买 emit/Markdown 净化)、OrderTable(行渲染/状态徽章/去支付条件/事件)、BannerStatCard(邮箱/金额/空态兜底);共享 i18n helper(语言包懒加载后 setLocaleMessage) | `src/components/business/__tests__/*` |
| husky + lint-staged + commitlint | Conventional Commits 约定工具化:pre-commit 跑 lint-staged(eslint --fix + prettier 仅处理暂存文件),commit-msg 跑 commitlint(type-enum 白名单);配置为 ESM(项目 type: module) | `.husky/pre-commit`、`.husky/commit-msg`、`commitlint.config.js`、`package.json` |

> **修复(2026-08-12)**:`.husky/` 钩子文件此前缺失(仅剩 `git config core.hooksPath=.husky/_` 指向,提交时钩子静默不生效)。已重建 `.husky/pre-commit`(`npx lint-staged`)、`.husky/commit-msg`(`npx --no -- commitlint --edit "$1"`)并入库;`pnpm prepare` 重建 `.husky/_/` stub。验证:合法 commit 消息通过、非法消息被拦截(exit 1),pre-commit 桥接 lint-staged 正常。
| 移动端下拉刷新 | `usePullToRefresh` composable:原生 touch 监听(passive:false,仅 scrollTop=0 且下拉时 preventDefault),MainLayout 集成指示器(下拉刷新/释放立即刷新/刷新中),触发与窗口聚焦一致的静默刷新仪表板 | `src/composables/usePullToRefresh.ts`、`layouts/MainLayout.vue` |
| 订单加载更多 | 移动端卡片视图用「加载更多」翻页追加(store fetch 支持 append),桌面表格视图保留分页器 | `src/stores/order.ts`、`views/order/OrdersView.vue` |
| PWA | vite-plugin-pwa@1.3:manifest(name/short_name/theme_color #6558F5/128-192-512 图标含 maskable)+ Workbox 离线壳(globPatterns 预缓存 + navigateFallback index.html);registerType autoUpdate + injectRegister script(非 inline,兼容 Tauri CSP script-src 'self');dev 不启用;Tauri 端不受影响 | `vite.config.ts`、`src/index.html`、`public/pwa-*.png` |

### 一期内已知缺口(可后补)

| 项 | 说明 |
|---|---|
| Stylelint | 未引入(可选,ESLint + Prettier 已覆盖) |
| ~~既有 format:check 告警~~ | **已解决(2026-08-12)**：`0ac021c` 全量格式化修复 23 个不合规文件，review-0.4.0 补格式化 `scripts/build-latest-json.mjs` 后，`pnpm format:check` 全仓通过(见 docs/reviews/review-0.4.0.md) |

### 二期 / 明确标注待办

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| 移动端深链/分享面板 | **深链 ✅ 已接入(2026-08-13)**,分享面板未做 | deep-link 注册移出 cfg(desktop) + tauri.conf.json `plugins.deep-link.mobile`(scheme ylink) + Android manifest intent-filter(gen/ 不入库,本地构建生效) |
| ~~更新卡片 UI~~ | **已实现(2026-08-12)**,见第 1 节 M6 发布能力收尾 | — |
| ~~开机自启~~ | **需求已移除(2026-08-13)**:不再提供开机自启开关;`utils/platform.ts` 自启封装与设置页入口已还原(autostart 插件仍注册于 Rust 侧,前端不暴露) | — |
| ~~存储适配 plugin-store~~ | **已实现(2026-08-13)**：`utils/storage.ts` 同步 facade(内存 Map + 异步落盘 `app-settings.json`),`initStorage()` 启动预载,persistedstate 与全部 localStorage 调用点统一走 storage 层 | — |
| ~~存储适配评审修复(review-0.6.0 P1/P2)~~ | **已修复(2026-08-13)**：① `http.ts` 的 `API_BASE` 改为请求时惰性解析(`resolveApiBase()`),持久化自定义 apiBase 在 `initStorage()` 后生效;② `initStorage()` 增加一次性迁移,将旧 WebView `localStorage`(`app:` 前缀)导入 plugin-store,已有键不覆盖、`app:_legacy:migrated:v1` 标记保证一次性。单测:http.spec.ts + 新增 storage.spec.ts | 见 review-0.6.0.md |

### 工程化与待办(2026-08-11 核对)

| 项 | 状态 | 说明 |
|---|---|---|
| 后端 CI / Rust CI | Rust ✅ 已接入(2026-08-12;2026-08-12 迁至 windows-latest 与打包平台一致);Go 后端 ❌ 不接入(项目决策:后端不走 Actions) | `.github/workflows/ci.yml` `rust` job(windows-latest:先 `pnpm build:web` 生成 dist → `cargo check`);`backend` job 已删除(2026-08-12),后端由本地 make/手动构建 |
| Release / updater(仅 Windows 打包) | ✅ 已接入(2026-08-12,公开产物仓库方案;2026-08-12 收窄为仅 Windows) | 代码仓库 private;`.github/workflows/release-tauri.yml`:tag `v*` / 手动触发 → guard(tag 校验)→ 单平台构建(windows-latest `pnpm tauri build --bundles nsis`,TAURI_SIGNING_PRIVATE_KEY 自动签名 .sig)→ `scripts/build-latest-json.mjs` 合并 latest.json(url 加 `gh-proxy.com` 前缀)→ `gh release` 推送到**公开产物仓库** `superMC5657/ylink-releases`(`RELEASES_PAT` secret);`tauri-plugin-updater` 已注册 + pubkey 写入 + capabilities 补 `updater:default`;updater endpoints = gh-proxy 优先 + 直连兑底;前端更新卡片入口仍待做(见下行) |
| 自启 / 通知 / 更新卡片前端入口 | **自启 ❌ 需求已移除(2026-08-13)**(不再提供开关,autostart 插件仍注册于 Rust 侧但前端不暴露);通知/更新卡片 ✅ 已实现(2026-08-12),见第 1 节 M6 发布能力收尾 | deep-link 前端监听已就绪(`utils/platform.ts` `onDeepLink` + `main.ts` 路由跳转) |
| 单实例深链接转发 | ✅ 已实现(2026-08-12) | Rust `lib.rs` 单实例回调提取 argv 中 `ylink://` URL,emit `deep-link://new-url` 转发已有实例(见第 1 节 M6 发布能力收尾) |
| 移动端深链 `ylink://` | ✅ 已接入(2026-08-13) | deep-link 注册移出 cfg(desktop)+ `tauri.conf.json` `plugins.deep-link.mobile`(scheme ylink)+ Android manifest intent-filter(gen/ 不入库,本地构建生效;desktop-tauri.md §9.5 已更新) |

---

## 3. 前置条件(运行 / 联调 / 上线)

### 3.1 本地运行

| 前置 | 说明 |
|---|---|
| Node ≥ 20 + pnpm ≥ 10 | ✅ 已满足:本机 Node 24.14.0、pnpm 10.33.0(2026-08-12 实测) |
| 安装依赖 | `pnpm install`(pnpm 10 需批准 esbuild 构建脚本:`pnpm.onlyBuiltDependencies` 已在 package.json 配置) |
| 启动 | `pnpm dev` → http://localhost:5174(Mock 环境默认开启) |
| 演示账号 | `2734921923@qq.com` / `Passw0rd`(Mock 仅校验该口令;任意 `Bearer mock-access-*` 视为有效) |
| 构建 | `pnpm build`(vue-tsc 类型检查 + vite 构建,产物 `dist/` 可独立部署静态托管) |

### 3.2 测试与质量门禁

| 前置 | 说明 |
|---|---|
| 单元测试 | `pnpm test`(Vitest + jsdom,**43 用例**,2026-08-12 实测 43/43);`pnpm test:coverage` 看覆盖率 |
| E2E | `pnpm e2e`(Playwright):webServer 以 `pnpm dev --mode e2e` 启动,**固定使用 `.env.e2e`(Mock)**,不受 `.env.development.local` 联调覆盖影响;本地默认系统 Chrome(`channel:'chrome'`),CI 用 `playwright install chromium`;双 project(桌面 1280 / 移动 390×844) |
| 布局诊断 | `node scripts/diag-layout.mjs`:5 分辨率(1024-2560)× 12 路由测量页面级横向溢出与内容区宽度(需先起 Mock dev) |
| 质量门禁 | `pnpm lint`(ESLint 0 error 0 warning)、`pnpm typecheck`、`pnpm format:check`(Prettier) |
| 注意 | E2E 会创建订单/触发支付,反复运行产生累积数据(Mock 内存态,重启 dev 即清空);登录页**不预填账号**,E2E 手动填写 Mock 演示账号 |

### 3.3 对接真实后端

| 前置 | 说明 |
|---|---|
| 关闭 Mock | `.env.development` 设 `VITE_USE_MOCK=false` |
| API 地址 | `VITE_API_BASE_URL=https://{host}/api/v1`;运行时也可在设置入口改后端地址并持久化(`utils/storage.ts` 的 `app:apiBase`) |
| CORS | 后端需允许 Web 域名(浏览器 fetch);Tauri 版无此要求(http 插件原生栈) |
| 契约同步 | 后端字段变更先改 docs/api/README.md → 再改 `src/types/api.d.ts` → api 模块(见 docs/README §5 变更流程) |

### 3.4 设计令牌维护(重要)

| 前置 | 说明 |
|---|---|
| 双源同步 | CSS 变量在 `tokens.css`;Naive UI overrides 在 `theme.ts`(亮/暗两套真实色值)。**改色必须两处同步**,否则 naive 组件(按钮/开关/分页)与自绘组件视觉漂移 |
| 禁止写死颜色 | 业务代码一律用 `var(--c-*)`,保证暗色正确;naive 侧只能用 theme.ts 的 overrides 间接引用 |

### 3.5 Tauri 桌面端

| 前置 | 说明 |
|---|---|
| Rust 工具链 | Rust ≥ 1.77(`src-tauri/Cargo.toml` rust-version = 1.77.2);✅ 已满足:本机 Rust 1.97.1(2026-08-12 实测);WebView2(Win);`pnpm tauri:dev` 开发联动,`pnpm tauri:build` 打包 |
| 首次编译 | cargo 拉取 10+ 插件 crate,首次约 5 分钟;`cargo check` 可单独快速验证 |
| 图标 | 改品牌后运行 `python scripts/gen-icon.py && pnpm tauri icon app-icon.png` 重新生成全套 |
| 深链接/更新 | 发布前需注册 `ylink://` 协议(Rust 侧 deep-link 插件)与 `tauri signer generate` 密钥(见第 2 节) |
| 平台适配 | Web 与 Tauri 共用一套代码,`utils/platform.ts` 自动降级;勿在 Web 端调用 Tauri 专属 API(已全部动态 import 保护) |

### 3.6 上线(生产)

| 前置 | 说明 |
|---|---|
| 构建产物 | `dist/` 部署到 Nginx/Caddy/对象存储 + CDN;hash 路由无需服务端 rewrite |
| 后端地址 | 打包时 `VITE_API_BASE_URL` 写入产物;或首次打开后由用户在设置页配置并持久化 |
| 安全 | 见 desktop-tauri.md §6(Web 版按 CSP 收紧;Markdown 已 DOMPurify 二次过滤,后端仍应按契约做写入侧清洗) |

---

## 4. 与契约文档的对照基准

- 端点、错误码、信封格式、单位约定:以 [docs/api/README.md](../api/README.md) 为准(本实现已对齐;`types/api.d.ts` 与契约一一对应)
- 视觉令牌/响应式/组件规范:以 [docs/frontend/design-system.md](design-system.md) 为准
- 路由表与逐页拆解:以 [docs/frontend/pages.md](pages.md) 为准(29 条路由全部落地:16 用户端 + 13 管理端)
- 数据层(HTTP 封装/store/i18n/深链接):以 [docs/frontend/data-layer.md](data-layer.md) 为准
- 桌面化:以 [docs/frontend/desktop-tauri.md](desktop-tauri.md) 为准(M6 骨架已完成,deep-link/更新/三平台打包等发布能力见第 2 节)
