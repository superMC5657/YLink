# 前端开发文档 · 总览与架构

> 用户端应用：一套 Vue 3 SPA 代码，同时产出 **响应式 Web**（桌面/平板/手机浏览器）与 **Tauri 2 桌面应用**（Windows/macOS/Linux）。本文档覆盖技术选型、应用架构、目录结构、工程化与里程碑。

## 1. 目标与非目标

### 目标
- 1:1 还原并美化截图中的面板 UI（浅色毛玻璃卡片风格），完整支持 **暗色模式**。
- 桌面、平板、手机三端体验完整：桌面为「侧边栏 + 顶栏」，手机为「顶栏 + 底部标签栏 + 抽屉菜单」（详见 [design-system.md](design-system.md) 第 6 节）。
- 桌面端由 Tauri 2 承载，提供托盘、自启动、深链接一键导入等原生能力（自动更新待接入，见 desktop-tauri.md §5）。
- 与 Go/Gin 后端通过 [../api/README.md](../api/README.md) 契约对接。

### 非目标（一期不做）
- 内置代理内核（本应用是「面板客户端」，代理连接由本机 Clash/sing-box 等客户端完成，App 只负责一键导入）。
- Tauri 移动端：Android APK 打包已启用（见 [desktop-tauri.md](desktop-tauri.md) 第 9 节）；iOS 不打包（需 Apple 开发者账号与审核，代理类应用上架风险高）。

> 管理后台前端已随主 SPA 实现全部 13 模块（M8 核心 6 模块 + M9 二期 7 模块，2026-08-11），不再是「非目标」；详见 [progress.md](progress.md) §1 M8/M9。

## 2. 技术选型

| 层次 | 选型 | 说明 |
|---|---|---|
| 桌面壳 | Tauri 2（Rust） | 安装包 <10MB，v2 capabilities 权限模型 |
| 框架 | Vue 3.5 + TypeScript + `<script setup>` | 面板类页面开发效率高，生态契合 |
| 构建 | Vite 6 | `pnpm dev` 开发 / `tauri build` 打包 |
| UI 组件库 | Naive UI | themeOverrides + 内置 darkTheme，覆盖表格/分页/弹窗/表单等复杂组件 |
| 原子化 CSS | UnoCSS | 布局、间距、响应式断点、自定义 shortcuts |
| 路由 | Vue Router 4 | 嵌套布局、登录守卫 |
| 状态管理 | Pinia + pinia-plugin-persistedstate | 按业务域拆分 store |
| 国际化 | vue-i18n@11 | zh-CN / en 起步，语言包懒加载 |
| 图表 | ECharts 5（按需引入） | 流量明细曲线 |
| 图标 | @iconify/vue（Solar / MingCute 图标集） | 线性风格统一 |
| Markdown 渲染 | markdown-it + DOMPurify | 知识库/公告正文，防 XSS |
| 工具库 | dayjs、@vueuse/core | 时间与组合式工具 |
| 测试 | Vitest + Vue Test Utils、Playwright | 单测 + E2E |
| 包管理 | pnpm | workspace 预留 |

选型说明：截图 UI 为高度自定义的卡片/胶囊风格，Naive UI 仅承担复杂交互组件（表格、分页、下拉、弹窗、表单校验、开关、消息提示），视觉层全部由设计令牌 + UnoCSS 自定义实现，避免与组件库默认皮肤「打架」。

## 3. 应用架构

```
views/ 页面 ──► components/ 业务组件 ──► composables/ 逻辑复用
     │                │                       │
     └────────────────┴─────────► stores/（Pinia，业务域状态）
                                      │
                                api/（接口模块，按域划分，纯函数）
                                      │
                            utils/http.ts（统一封装）
                              │               │
                    isTauri() ? tauri-plugin-http : window.fetch
                                      │
                              后端 /api/v1
```

分层约束：
1. **页面不直接发请求**：views 只读写 store；store 调 api 模块；api 模块只依赖 http 封装与契约类型。
2. **接口类型集中**：`types/api.d.ts` 与契约文档一一对应，后端字段变更先改契约再改类型。
3. **平台能力适配层**：`utils/platform.ts` 暴露 `copyText / openExternal / importToClient` 等能力，内部按 Tauri/浏览器自动降级，业务代码不感知平台差异。

## 4. 目录结构

```
proxy-seller-web/
├── src/
│   ├── api/                  # 接口模块：auth.ts user.ts order.ts plan.ts
│   │                         #   invite.ts knowledge.ts ticket.ts server.ts config.ts
│   ├── assets/               # banner 图、logo、插画（暗色模式专用图单独命名）
│   ├── components/           # 通用组件
│   │   ├── app/              #   AppSidebar AppHeader MobileTabBar DrawerMenu
│   │   ├── business/         #   BannerStatCard NoticePanel QuickActionGrid
│   │   │                     #   SubscribeCard PlanCard OrderTable StatCard
│   │   └── ui/               #   Card StatusBadge StatNumber EmptyState PriceText ...
│   ├── composables/          # useTrafficFormat useMediaQuery useCountdown usePolling ...
│   ├── layouts/              # MainLayout.vue AuthLayout.vue
│   ├── locales/              # zh-CN.ts en-US.ts（按模块拆命名空间）
│   ├── router/               # index.ts + guards.ts
│   ├── stores/               # auth user subscribe notice plan order invite ...
│   ├── styles/               # tokens.css(亮/暗变量) theme.ts(Naive 覆盖) shortcuts.ts
│   ├── types/                # api.d.ts（契约类型）models.d.ts
│   ├── utils/                # http.ts platform.ts storage.ts format.ts deeplink.ts
│   ├── views/                # 页面（见 pages.md）
│   ├── index.html            # SPA 入口（root: src，构建输出 dist/index.html）
│   ├── App.vue / main.ts
├── src-tauri/                # Tauri 2 Rust 工程（见 desktop-tauri.md）
├── mock/                     # vite-plugin-mock 数据（严格按契约）
├── docs/                     # 本文档目录
├── .env / .env.development / .env.production
├── vite.config.ts / uno.config.ts / package.json
└── .github/workflows/        # CI：前端 quality+e2e、Rust check；CD：Tauri Release → 公开产物仓库（gh-proxy 加速更新）
```

## 5. 环境变量

| 变量 | 说明 | 示例 |
|---|---|---|
| `VITE_API_BASE_URL` | 后端 API 根地址 | `https://api.example.com/api/v1` |
| `VITE_USE_MOCK` | 是否启用本地 Mock | `true` / `false` |
| `VITE_APP_NAME` | 站点名兜底（运行时以后端配置为准） | `YLink` |

运行时可以修改后端地址并持久化（设置页入口），便于同一客户端连接不同站点；默认读取打包时的 `VITE_API_BASE_URL`。

## 6. 双端运行策略

| 能力 | Tauri 桌面 | 浏览器 |
|---|---|---|
| HTTP 请求 | `@tauri-apps/plugin-http`（原生栈，无 CORS/CSP 限制） | `fetch`（需后端开启 CORS） |
| 复制订阅链接 | plugin-clipboard-manager | `navigator.clipboard` |
| 打开外部链接 | plugin-opener | `window.open` |
| 一键导入客户端 | 直接打开 `clash://` 等 scheme | 同左（浏览器同样支持唤起） |
| 持久化 | plugin-store（JSON 文件） | localStorage |
| 系统能力 | 托盘（Rust 已注册）；自启/通知/自动更新（插件已装，前端入口未接入） | 无（功能入口自动隐藏或降级） |

`utils/platform.ts` 提供 `isTauri()` 与统一能力接口，所有降级逻辑收敛在此。

## 7. Mock 与联调

1. 使用 `vite-plugin-mock`，`mock/` 目录按契约逐接口造数，覆盖空态/多页/异常（401、限流）场景。
2. 开发流程：契约评审通过 → 前端 Mock 并行开发 → 后端 Swagger 就绪后切换 `VITE_USE_MOCK=false` 联调。
3. E2E（Playwright）固定跑在 Mock 环境，保证 CI 稳定。
4. 本地联调日志（`.dev/vite.log` 与 `.dev/api.log` 等）已禁用 ANSI 颜色码，避免 ESC 转义序列造成的乱码（`scripts/dev-up.sh` 为 vite 设置 `NO_COLOR=1`，GORM 日志 `Colorful: false`）。

## 8. 工程化与代码规范

- **包管理**：pnpm；Node >= 20，Rust >= 1.77.2（`src-tauri/Cargo.toml` rust-version）。
- **质量门禁**：ESLint 9（flat config）+ Prettier + `vue-tsc --noEmit`；husky/lint-staged（pre-commit 仅处理暂存文件）+ commitlint（commit-msg 校验 Conventional Commits）已接入（CI 另在 GitHub Actions 强制 lint/typecheck/format/test/build）；Stylelint 未引入。
- **命名**：组件 PascalCase；composable `useXxx`；store `useXxxStore`；api 模块按业务域小写命名；CSS 变量 `--c-*` 颜色、`--r-*` 圆角、`--s-*` 阴影、`--t-*` 动效。
- **样式**：优先 UnoCSS 原子类；组件私有样式用 `<style scoped>`；主题相关一律使用 CSS 变量，禁止写死颜色（保证暗色模式正确）。
- **注释**：业务组件顶部注明对应截图页面与契约接口；复杂逻辑写「为什么」而非「做什么」。

## 9. 测试策略

| 层 | 工具 | 范围 |
|---|---|---|
| 单测 | Vitest + Vue Test Utils | http 封装（拦截/刷新/解包）、格式化工具、下单金额计算、表单校验、store 逻辑 |
| 组件测试（未写，可后补） | Vitest + jsdom | StatCard、PlanCard、OrderTable 等关键组件渲染与交互（见 progress.md §2 一期缺口） |
| E2E | Playwright | 登录 → 仪表板 → 购买套餐（Mock 支付）→ 订单详情；移动端视口（390×844）冒烟 |

## 10. 里程碑

| 阶段 | 内容 | 验收 |
|---|---|---|
| M1 脚手架 | 工程初始化、路由/守卫、Pinia、i18n、设计令牌、UnoCSS、Naive UI 集成、Mock | 登录页可跑通 Mock 登录 |
| M2 布局 | MainLayout（桌面侧边栏/移动端底栏+抽屉）、顶栏、暗色模式切换 | 三端视口布局正确，亮暗切换无闪烁 |
| M3 核心页 | 仪表板、使用文档、我的订单、个人信息 | 对应截图页面还原 |
| M4 交易闭环 | 套餐、下单弹窗、优惠券、收银台、支付轮询 | Mock 环境完成购买全链路 |
| M5 营销页 | 邀请赚钱、申请代理、工单、节点状态、流量明细 | 全部页面上线 |
| M6 桌面化 | Tauri 插件接入、托盘、深链接、Android 打包（骨架完成） | 自动更新与三平台发布未接入（见 desktop-tauri.md §5/§7 与 progress.md §2） |
