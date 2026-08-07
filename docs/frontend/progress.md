# 前端开发 · 进度追踪(已完成 / 未完成 / 前置条件)

> 本文档记录 `src/` 目录 Vue 3 用户端应用的开发状态,是 docs/frontend 与 docs/api 的实现对照表。
> 更新规则:每完成一个里程碑/修复一个缺陷,同步更新本文档「已完成」;新增缺口写入「未完成」并标注依赖。
> 最后更新:2026-08-07(M1–M5 全部实现完成,构建 + Playwright 冒烟 24/24 通过)

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
| Mock | vite-plugin-mock,5 个模块严格按契约(含 401/错误码/支付自动完成模拟) | `mock/*.ts` |

### M2 布局与设计系统(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| 设计令牌 | 亮/暗两套 CSS 变量(色彩/形状/阴影/动效/字体),业务代码不写死颜色 | `src/styles/tokens.css` |
| Naive UI 覆盖 | 亮/暗两套**真实色值** overrides(seemly 不支持 CSS 变量字符串,已规避) | `src/styles/theme.ts` |
| 桌面布局 | 240px 侧边栏(折叠 72px)+ 毛玻璃吸顶顶栏(折叠钮/站点名/主题/语言/用户 chip) | `components/app/AppSidebar.vue`、`AppHeader.vue` |
| 平板/手机 | `<768px` 底栏 4 Tab + 抽屉菜单(分组一致);`768-1024` 迷你侧边栏;useMediaQuery 断点驱动 | `MobileTabBar.vue`、`DrawerMenu.vue`、`MainLayout.vue` |
| 全局壳 | AuthLayout(居中卡片 + 氛围背景 + 语言/主题)、客服浮球、naive message/dialog provider + toast 桥接 | `layouts/*`、`CustomerServiceFab.vue`、`ToastBridge.vue` |
| 基础组件 | AppIcon(离线线性图标 60+)/ UiCard / StatNumber / StatusBadge / PriceText / EmptyState / PageHeader / CopyText / ThemeToggle(三态循环),全部全局注册 | `components/ui/*` |

### M3 核心页(✅ 完成)

| 页面 | 组件与交互 | 数据来源 |
|---|---|---|
| 仪表板 | Banner 统计卡(余额绿/佣金粉)、订阅卡(到期徽章/进度条 80%/95% 变色/五宫格)、公告手风琴(markdown-it + DOMPurify)、9 宫格快捷操作(一键导入/免费流量/APP 下载弹窗);窗口聚焦静默刷新 | `GET /user/stat`、`/user/subscribe`、`/notices` |
| 使用文档 | 300ms 防抖搜索(前端过滤 + 传 keyword)、语言切换、分类分组、详情 markdown 渲染(代码块/强标记营销红) | `GET /knowledges`、`/knowledges/{id}` |
| 我的订单 | 表格/卡片双视图切换、状态筛选、分页、详情抽屉(全字段 + 支付入口 + 取消)、待支付 5s 轮询 | `GET /orders`、`/orders/{no}`、`cancel` |

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

### 审查修复(✅ 已修复,阻断项清零)

| 问题 | 修复 |
|---|---|
| persistedstate 插件用 `app.use()` 注册导致 `context.pinia` 未定义,首帧崩溃 | 改为 `pinia.use(piniaPluginPersistedstate)` |
| naive-ui `create()` 默认注册空组件,`n-config-provider` 等全部无法解析 | 显式列出 18 个用到的组件 |
| themeOverrides 传 CSS 变量(`var(--c-primary)`)被 seemly 解析报错 | 拆亮/暗两套真实色值 overrides,与 tokens.css 双源同步(见前置条件 3.5) |
| 业务组件直接使用 `<AppIcon>`/`<StatusBadge>` 等而未局部 import | 基础 UI 组件统一全局注册 |
| vite-plugin-mock 按文件隔离编译,跨文件 sessions Map 不共享导致 401 | token 校验改无状态格式匹配(`Bearer mock-access-*`) |
| 冒烟访问 `/plans` 被重定向到 dashboard | hash 路由下脚本改用 `/#/` 前缀 |

### 验证状态(✅)

- `pnpm typecheck`(vue-tsc --noEmit)零错误
- `pnpm build`(vue-tsc + vite build)成功,主包 gzip ≈ 257KB
- `node scripts/smoke.mjs`(Playwright + 系统 Chrome)**24/24 通过**:登录 → 仪表板 → 套餐 → 下单(余额支付)→ 订单 → 9 页面可达性 → 暗色切换 → 移动端 390×844 底栏导航

---

## 2. 未完成项

### M6 桌面化(Tauri 2)—— 一期未做

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| `src-tauri/` 工程 | 未创建(Rust 侧完全空白) | 契约见 docs/frontend/desktop-tauri.md;需 Rust ≥ 1.78 |
| 平台插件接入 | 未接(http/store/clipboard/opener/deep-link/single-instance/autostart/notification/updater/window-state/os) | `utils/platform.ts`、`utils/storage.ts`、`utils/http.ts` 已留 Tauri 分支,接入即用 |
| 托盘/自启/深链接/自动更新 | 未实现 | 依赖 `src-tauri/` 与 capabilities 最小授权 |
| 三平台打包与 updater 签名 | 未配置 | 需 GitHub Actions + `TAURI_SIGNING_PRIVATE_KEY` |

### 一期内已知缺口(可后补)

| 项 | 说明 |
|---|---|
| 单元测试 | Vitest 未引入;文档(data-layer §9)规划的 http 封装/格式化/下单金额/store 单测均未写 |
| 组件测试与 E2E 套件 | 仅 `scripts/smoke.mjs` 冒烟脚本(非正式 Playwright 项目);按测试策略需补 `tests/e2e/*.spec.ts` 与 CI 固定 Mock 环境 |
| ESLint / Prettier / Stylelint | `package.json` 有 `lint` 脚本但 **flat config 未配置**;`vue-tsc` 已承担类型门禁 |
| husky + lint-staged + commitlint | 未配置(Conventional Commits 约定见 docs/README §5,尚未工具化) |
| CI 流水线 | `.github/workflows/` 未建;PR CI(lint→typecheck→test→build)与 Release 矩阵均缺 |
| 公告/知识库语言懒加载 | i18n 语言包为整包引入,未按模块拆分懒加载 |
| 移动端下拉刷新 / 加载更多 | pages.md §6.3 建议项,未实现(分页器在平板以上已可用) |
| PWA | 手机端一期仅响应式 Web,未加 manifest/离线壳(desktop-tauri.md §9 标注可后补) |
| 注册页邀请码必填联动 | `invite_code_required=true` 时表单未强制必填(契约 §4.1),Mock 环境未开该开关 |
| 网络离线顶部横幅 | 当前为 toast 提示(ToastBridge),未做 pages.md §3.13 的顶部横幅 + 自动重试 |

### 二期 / 明确标注待办

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| 管理后台前端 | 未开始(独立应用,复用 admin API) | 契约范围见 api/README.md 第 16 节 |
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

### 3.2 冒烟测试

| 前置 | 说明 |
|---|---|
| 浏览器 | `node scripts/smoke.mjs` 默认用系统 Chrome(`channel:'chrome'`);或先 `pnpm exec playwright install chromium` 后改回默认 |
| 依赖 | `pnpm add -D playwright`(已装) |
| dev server | 冒烟前需 `pnpm dev` 已在 5173 端口运行 |
| 注意 | 冒烟会创建订单/触发支付,反复运行会产生累积数据(Mock 内存态,重启 dev 即清空) |

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

### 3.5 上线(生产)

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
- 桌面化:以 [docs/frontend/desktop-tauri.md](desktop-tauri.md) 为准(M6 未开始,见第 2 节)
