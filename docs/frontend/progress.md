# 前端开发 · 进度追踪(已完成 / 未完成 / 前置条件)

> 本文档记录 `src/` 目录 Vue 3 用户端应用的开发状态,是 docs/frontend 与 docs/api 的实现对照表。
> 更新规则:每完成一个里程碑/修复一个缺陷,同步更新本文档「已完成」;新增缺口写入「未完成」并标注依赖。
> 最后更新:2026-08-11(M1–M8 已实现;2026-08-11 全量核对:typecheck/lint/vitest 31 用例通过;全量审查 33 项修复完成,见「全量审查修复」;管理后台 6 模块已完成,二期剩余 7 模块见第 2 节)

---

## 1. 已完成项

### M1 脚手架(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| 工程初始化 | Vue 3.5 + TS + Vite 6 + pnpm,`<script setup>`;Node ≥ 20 | `package.json`、`vite.config.ts`、`tsconfig.json` |
| 原子化 CSS | UnoCSS(presetUno / presetAttributify / presetIcons / transformerDirectives),自定义 shortcuts(`btn-primary`/`btn-olive`/`card-base` 等) | `uno.config.ts` |
| 路由 | Vue Router 4 hash 模式,16 条路由(guest/登录双守卫 + 页面标题) | `src/router/index.ts`、`guards.ts`、`nav.ts` |
| 状态管理 | Pinia + persistedstate,12 个业务 store | `src/stores/*` |
| 国际化 | vue-i18n@11,zh-CN / en-US 按模块命名空间,`Accept-Language` 联动 | `src/locales/*` |
| 环境变量 | `VITE_API_BASE_URL` / `VITE_USE_MOCK` / `VITE_APP_NAME`,运行时后端地址可改并持久化 | `.env.development`、`.env.production`、`utils/storage.ts` |
| 防闪烁 | `index.html` 内联首帧脚本,渲染前读持久化主题写入 `data-theme` | `index.html` |
| Mock | vite-plugin-mock,6 个模块(auth/business/config/order/server/user)严格按契约(含 401/错误码/支付自动完成模拟) | `mock/*.ts` |

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
| 仪表板 | Banner 统计卡(余额绿/佣金粉)、订阅卡(到期徽章/进度条 80%/95% 变色/五宫格)、公告手风琴(markdown-it + DOMPurify)、9 宫格快捷操作(一键导入/免费流量/APP 下载弹窗);窗口聚焦静默刷新 | `GET /user/stat`、`/user/subscribe`、`/notices` |
| 使用文档 | 300ms 防抖搜索(前端过滤 + 传 keyword)、语言切换、分类分组、详情 markdown 渲染(代码块/强标记营销红) | `GET /knowledges`、`/knowledges/{id}` |
| 我的订单 | 表格/卡片双视图切换、状态筛选、分页、详情弹窗(屏幕中央,全字段 + 支付入口 + 取消)、待支付 5s 轮询 | `GET /orders`、`/orders/{no}`、`cancel` |

### M4 交易闭环(✅ 完成)

| 能力 | 说明 | 数据来源 |
|---|---|---|
| 套餐卡 | 3/2/1 列响应式网格、周期胶囊切换联动价格与「省 N%」、markdown 营销红字 | `GET /plans` |
| 下单弹窗 | 周期选择 → 优惠券试算(成功显示减免/失败红字)→ 支付方式(余额不足置灰显差额)→ 费用明细 → 提交;`Idempotency-Key` 幂等建单 | `POST /coupons/check`、`POST /orders` |
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

### M6 桌面化(Tauri 2 骨架)(✅ 骨架完成,发布能力见第 2 节)

| 项 | 说明 | 位置 |
|---|---|---|
| Rust 工程 | Cargo.toml(10 个插件 crate)/ build.rs / main.rs / lib.rs(插件注册 + 托盘 + 单实例) | `src-tauri/` |
| 配置 | tauri.conf.json(1280×800 主窗、CSP、bundle targets all) | `src-tauri/tauri.conf.json` |
| 权限 | capabilities/default.json 最小授权(核心 + 10 插件,http scope 限 https/localhost) | `src-tauri/capabilities/default.json` |
| 图标 | `pnpm tauri icon` 从脚本生成源图产出全套(ico/icns/png/Android) | `src-tauri/icons/`、`scripts/gen-icon.py` |
| 前端适配 | platform.ts(clipboard/opener/deep-link 动态 import)、http.ts(plugin-http 原生 fetch)、app.ts(applyTheme 同步窗口标题栏)、main.ts(深链接路由跳转) | `src/utils/*`、`src/stores/app.ts`、`src/main.ts` |
| 验证 | `cargo check` 通过(首次编译约 5 分钟);Web 构建不受动态 import 影响 | — |

### 工程化与质量门禁(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| ESLint 9 flat config | js + typescript-eslint + eslint-plugin-vue(flat/recommended)+ eslint-config-prettier,0 error 0 warning | `eslint.config.ts` |
| Prettier 3 | 全仓格式化 + format:check | `.prettierrc.json`、`.prettierignore` |
| Vitest 单测 | jsdom,29 用例(格式化/http 401 刷新重放/invite store/倒计时),v8 覆盖率 | `vitest.config.ts`、`src/**/__tests__/*` |
| Playwright E2E | 正式套件 30 用例(桌面 + 移动双 project、webServer 自动起 Mock),替代原冒烟脚本 | `playwright.config.ts`、`tests/e2e/*` |
| CI | PR/push 双 job:quality(lint→typecheck→format:check→test→build)+ e2e(失败上传报告) | `.github/workflows/ci.yml` |
| i18n 懒加载 | 语言包按需动态 import,useLocale 统一切换 | `src/locales/index.ts`、`src/composables/useLocale.ts` |
| 注册页强制邀请码 | 站点 `invite_code_required=true` 时校验必填 | `src/views/auth/RegisterView.vue` |
| 离线横幅 | 顶部红色常驻横幅 + 恢复 toast | `src/components/app/ToastBridge.vue` |

### M8 管理后台(✅ 核心模块完成,2026-08-10)

| 项 | 说明 | 位置 |
|---|---|---|
| 角色区分基建 | auth store `isAdmin` getter;路由 meta `admin` 标志 + 守卫(非管理员访问 `/admin/*` 重定向 `/dashboard`);侧边栏/移动端抽屉按角色追加「管理后台」分组;顶栏用户下拉管理员增加「管理后台」入口 | `src/stores/auth.ts`、`src/router/*`、`components/app/*` |
| 管理端契约 | `src/api/admin.ts` 封装管理端全量端点;类型对齐后端 DTO(价格统一为元) | `src/api/admin.ts`、`src/types/api.d.ts` |
| 总览 | 7 项运营统计卡 + 快捷入口 | `views/admin/AdminOverviewView.vue` |
| 用户管理 | 搜索/分页/封禁/角色调整/调余额(均写审计) | `views/admin/AdminUsersView.vue` |
| 套餐管理 | CRUD:周期定价(元)/流量/设备/限速/节点分组/上架/排序 | `views/admin/AdminPlansView.vue` |
| 节点管理 | 分组 CRUD + 节点 CRUD(6 协议/地址/配置 JSON/倍率/状态/标签) | `views/admin/AdminNodesView.vue` |
| 订单管理 | 状态筛选/分页/退款(余额退回+佣金回滚) | `views/admin/AdminOrdersView.vue` |
| 工单管理 | 列表/详情/客服回复/关闭 | `views/admin/AdminTicketsView.vue` |
| Mock + E2E | `mock/admin.ts` 管理端 Mock;`tests/e2e/admin.spec.ts` 角色区分 5 用例 × 双 project(管理员可见入口/普通用户不可见且访问被重定向) | `mock/*`、`tests/e2e/*` |

### 验证状态(✅,2026-08-10 全量实测)

- `pnpm typecheck`(vue-tsc --noEmit)零错误
- `pnpm build`(vue-tsc + vite build)成功,主包 gzip ≈ 253.3KB
- `pnpm lint` 0 error 0 warning;`pnpm test` **31/31 通过**(此前文档记 29,系后续新增用例未同步)
- `pnpm e2e`(Playwright)**40 例通过**:登录 → 仪表板 → 套餐 → 下单(余额支付)→ 订单 → 8 页面可达性 → 暗色切换 → 移动端底栏导航 + **管理后台角色区分 10 例**(调试残留 `zz-errfail.spec.ts` 已移除)
- `cargo check`(src-tauri)通过

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

---

## 2. 未完成项

### M6 桌面化(Tauri 2)—— 骨架完成,发布能力待接入

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| `src-tauri/` 工程 | ✅ `Cargo.toml`/`build.rs`/`main.rs`/`lib.rs`/`tauri.conf.json`/`capabilities/default.json`/全套图标,`cargo check` 通过 | 契约见 docs/frontend/desktop-tauri.md;Rust 1.97 已满足 |
| 插件接入(Rust) | ✅ http / store / clipboard / opener / single-instance / autostart / notification / process / window-state / os | capabilities 最小授权;http scope 限 `https://**` + localhost |
| 平台适配层(前端) | ✅ `utils/platform.ts` 启用 Tauri 分支(clipboard/opener/deep-link 动态 import);`utils/http.ts` 走 plugin-http 原生 fetch;`stores/app.ts` applyTheme 同步窗口标题栏;`main.ts` 深链接路由跳转 | Web 端自动降级不受影响 |
| 托盘/单实例 | ✅ 托盘菜单(显示主窗口/退出)+ single-instance 聚焦已有窗口 | lib.rs setup |
| deep-link 注册 | ✅ `tauri-plugin-deep-link` 已注册(Rust `#[cfg(desktop)]` init)+ capabilities `deep-link:default`;前端 onDeepLink 路由跳转 | Release 域名与 `ylink://` 协议注册见 desktop-tauri.md §3/§4 |
| 自动更新 | ⚠️ tauri.conf.json 有 updater 配置(空 pubkey/示例端点),Rust 与前端未接入 | 需 `tauri signer generate` + Release 流水线(desktop-tauri.md §5) |
| 通知触发点 | ⚠️ 插件已注册,前端未实现到期/工单回复/支付成功本地通知 | 依赖轮询钩子(可后补) |
| 三平台打包与 updater 签名 | ❌ 未配置 | 需 GitHub Actions Release + `TAURI_SIGNING_PRIVATE_KEY` |
| 存储适配 | ⚠️ 一期统一 localStorage(WebView 持久化);plugin-store(JSON 文件)异步 API 与 persistedstate 同步接口冲突,迁移需异步化改造 | 见 storage.ts 注释 |

### 一期内已知缺口(可后补)

| 项 | 说明 |
|---|---|
| 组件测试 | 已建 Vitest 但仅覆盖 utils/store/composables;关键组件(StatCard/PlanCard/OrderTable)渲染测试未写(测试策略 frontend/README §9) |
| husky + lint-staged + commitlint | 未配置(Conventional Commits 约定见 docs/README §5,尚未工具化) |
| 移动端下拉刷新 / 加载更多 | pages.md §6.3 建议项,未实现(分页器在平板以上已可用) |
| PWA | 手机端一期仅响应式 Web,未加 manifest/离线壳(desktop-tauri.md §9 标注可后补) |
| Stylelint | 未引入(可选,ESLint + Prettier 已覆盖) |

### 二期 / 明确标注待办

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| 管理后台(二期剩余模块) | **核心 6 模块已实现**(总览/用户/套餐/节点/订单/工单,见第 1 节 M8);剩余:优惠券/公告/知识库/代理审批/佣金日志/流量导入/站点设置 | 后端 13 组 admin API 已全量就绪;契约见 [docs/api/README.md](../api/README.md) 第 16 节 |
| 移动端深链/分享面板 | 未做 | 需 Tauri Mobile 评估(desktop-tauri.md §9) |
| 更新卡片 UI | 未做 | 依赖 M6 updater 插件 |

---

## 3. 前置条件(运行 / 联调 / 上线)

### 3.1 本地运行

| 前置 | 说明 |
|---|---|
| Node ≥ 20 + pnpm ≥ 10 | 本机 pnpm 10.33.0 已满足 |
| 安装依赖 | `pnpm install`(pnpm 10 需批准 esbuild 构建脚本:`pnpm.onlyBuiltDependencies` 已在 package.json 配置) |
| 启动 | `pnpm dev` → http://localhost:5173(Mock 环境默认开启) |
| 演示账号 | `2734921923@qq.com` / `Passw0rd`(Mock 仅校验该口令;任意 `Bearer mock-access-*` 视为有效) |
| 构建 | `pnpm build`(vue-tsc 类型检查 + vite 构建,产物 `dist/` 可独立部署静态托管) |

### 3.2 测试与质量门禁

| 前置 | 说明 |
|---|---|
| 单元测试 | `pnpm test`(Vitest + jsdom,31 用例);`pnpm test:coverage` 看覆盖率 |
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
| Rust 工具链 | Rust ≥ 1.77 + WebView2(Win);`pnpm tauri:dev` 开发联动,`pnpm tauri:build` 打包 |
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
- 路由表与逐页拆解:以 [docs/frontend/pages.md](pages.md) 为准(16 条路由全部落地)
- 数据层(HTTP 封装/store/i18n/深链接):以 [docs/frontend/data-layer.md](data-layer.md) 为准
- 桌面化:以 [docs/frontend/desktop-tauri.md](desktop-tauri.md) 为准(M6 骨架已完成,deep-link/更新/三平台打包等发布能力见第 2 节)
