# 前端开发 · 当前状态

> 本文档描述 `src/` 目录 Vue 3 用户端应用的**当前能力与状态**,是 docs/frontend 与 docs/api 的实现对照表。
> 维护规则:只记录当前态(能力清单、未完成项、前置条件),不堆叠历史流水账。端点与错误码以 [docs/api/README.md](../api/README.md) 为准,路由与逐页拆解以 [pages.md](pages.md) 为准,数据层以 [data-layer.md](data-layer.md) 为准,视觉规范以 [design-system.md](design-system.md) 为准,桌面端以 [desktop-tauri.md](desktop-tauri.md) 为准;历史修复明细见 [docs/reviews/](../reviews/) 与 git log。

## 1. 状态总览

| 项 | 状态 |
|---|---|
| 质量门禁 | `pnpm lint`(ESLint 0 warning)/ `pnpm lint:css`(Stylelint)/ `pnpm typecheck`(vue-tsc)/ `pnpm format:check` 全部通过 |
| 单测 | `pnpm test`(Vitest + jsdom)**63 用例**全绿 |
| E2E | `pnpm e2e`(Playwright,桌面 1280 + 移动 390 双 project)全量通过 |
| 构建 | `pnpm build` 成功(含 PWA 产物);`cargo check`(src-tauri)通过 |

## 2. 已完成能力

### 2.1 工程骨架

Vue 3.5 + TS + Vite 6 + pnpm(`<script setup>`,Node ≥ 20);UnoCSS(presetUno/Attributify/Icons + 自定义 shortcuts);Vue Router 4 hash 模式(用户端 + `/admin` 管理端双布局 + guest/登录/admin 三类守卫 + 页面标题);Pinia + persistedstate(12 个业务 store);vue-i18n@11 zh-CN/en-US 按模块命名空间、语言包懒加载;vite-plugin-mock 按契约造数(含 401/错误码/支付自动完成);`public/theme.js` 首帧防闪烁;运行时可改后端地址并持久化。

### 2.2 布局与设计系统

设计令牌 tokens.css(亮/暗两套 CSS 变量)+ Naive UI theme.ts 真实色值 overrides(**双源必须同步**);桌面 240px 侧边栏(折叠 72px)+ 毛玻璃吸顶顶栏;平板/手机 `<768px` 底栏 4 Tab + 抽屉菜单、768-1024 迷你侧边栏;AuthLayout 全局壳、客服浮球、toast 桥接;基础 UI 组件全局注册(AppIcon/UiCard/StatNumber/StatusBadge/PriceText/EmptyState/PageHeader/CopyText 等);桌面内容区 max-w-1440 + 窄桌面自动折叠侧边栏 + 表格 overflow 适配(`scripts/diag-layout.mjs` 可测量 5 分辨率溢出)。

### 2.3 用户端页面

仪表板(统计卡/订阅卡/公告 markdown + 优惠码高亮复制/快捷操作)、使用文档(搜索/分类分组/markdown)、我的订单(表格/卡片双视图/详情弹窗/待支付轮询)、套餐购买(周期切换/优惠券试算与点选/可用券列表/收银台二维码/余额直付)、一键导入(10 款客户端 scheme + 复制兜底)、邀请赚钱(统计卡/划转/邀请码/注册链接/提现入口与记录,F02)、申请代理、节点状态(60s 静默轮询,不暴露 host/port)、个人信息(改密/通知开关/Telegram 绑定卡片 F12/会话管理 F14/重置订阅)、工单(列表/对话流/回复关闭)、流量明细(ECharts)、门户分流页 `/portal`(管理员双卡分流,admin-console-split)。

### 2.4 管理后台

独立 `AdminLayout` + `AdminSidebar`(唯一菜单源,移动端汉堡抽屉),登录落点按角色分流(`/portal`);模块全量:总览(统计卡 + 按运营频率分组的 8 快捷入口)、用户管理(多选批量/CSV 导出/发邮件/重置订阅)、套餐、节点管理(多选批量/复制/排序/node_key)、订单(本地化枚举文案/退款)、工单(提现工单审核 F02)、优惠券(含一键公告)、公告/知识库(排序弹窗/分类管理 F15)、代理审批、佣金日志(提现流水徽章)、流量管理(导入/重置/重置记录三标签 F16)、统计报表(ECharts 五图 + 余额四系列 F04)、审计日志(可读化 target)、站点设置(含 telegram 键)、邮件模板(F11)、订阅模板(F10)、版本检查(F20)。

### 2.5 桌面端(Tauri 2)

Rust 工程 + 10 插件最小授权(http scope 限 https/localhost);平台适配层 `utils/platform.ts`(Web 自动降级,动态 import 保护);http 走 plugin-http 原生栈(不受 CORS 限制);深链接 `ylink://`(单实例转发已有实例)+ Android intent-filter;自动更新(updater 插件 + pubkey + Release 流水线产出 latest.json + 更新浮动卡片);本地通知(支付成功/工单回复/订阅到期,Tauri 与 Web Notification 降级);托盘 + 单实例;存储适配 plugin-store(localStorage 一次性迁移);NSIS 仅 Windows 打包(installer-hooks.nsh 必须保留,否则打包失败)。

### 2.6 工程化与质量门禁

ESLint 9 flat config + Prettier 3 + Stylelint 17(接入 lint-staged);husky pre-commit/commit-msg(Conventional Commits);Vitest 63 用例(含 i18n 消息编译回归锁);Playwright E2E 双 project(webServer 固定 `.env.e2e` Mock);CI 仅 tag 触发 3 job(frontend-quality / frontend-e2e / rust);Release 流水线(Windows NSIS + 签名 + 公开产物仓库);PWA(manifest + Workbox 离线壳,dev 不启用);**错误 toast 单一出口约定**(http 封装层是唯一 toast 出口,组件 catch 禁止转发错误 message;`scripts/check-error-toast.mjs` 串入 `pnpm lint` 守护,详见 data-layer.md §1.1/§7)。

## 3. 未完成与已决策

| 项 | 状态 |
|---|---|
| 开机自启开关 | ❌ 需求已移除:autostart 插件仍注册于 Rust 侧,前端不暴露入口 |
| Go 后端 CI | ❌ 不接入(项目决策);前端 CI 仅发布 tag 触发,日常检查走本地门禁 |
| 总览快捷操作「待办计数」 | 暂缓:接口无待办数据,需后端先行 |
| 移动端 App 打包 | 未启动(Tauri 移动端策略见 desktop-tauri.md;Android 构建为本地行为,gen/ 不入库) |

## 4. 前置条件(运行 / 联调 / 上线)

### 4.1 本地运行

| 前置 | 说明 |
|---|---|
| Node ≥ 20 + pnpm ≥ 10 | `pnpm install`(esbuild 构建脚本已在 `pnpm.onlyBuiltDependencies` 批准) |
| 启动 | `pnpm dev` → http://localhost:5174(Mock 默认开启) |
| Mock 演示账号 | `2734921923@qq.com` / `Passw0rd`(Mock 仅校验该口令;任意 `Bearer mock-access-*` 有效) |
| 构建 | `pnpm build`(vue-tsc + vite,产物 `dist/` 可独立静态托管) |

### 4.2 测试与质量门禁

| 项 | 说明 |
|---|---|
| 单测 | `pnpm test`;`pnpm test:coverage` 看覆盖率(v8) |
| E2E | `pnpm e2e`:webServer 以 `pnpm dev --mode e2e` 启动,固定 `.env.e2e`(Mock),不受 `.env.development.local` 影响;本地用系统 Chrome,CI 用 `playwright install chromium`;双 project(桌面 1280 / 移动 390×844);E2E 会创建订单/触发支付(Mock 内存态,重启即清空);登录页不预填账号 |
| 布局诊断 | `node scripts/diag-layout.mjs`(需先起 Mock dev):5 分辨率 × 路由测横向溢出 |
| 门禁 | `pnpm lint` / `pnpm lint:css` / `pnpm typecheck` / `pnpm format:check` |

### 4.3 对接真实后端

`.env.development` 设 `VITE_USE_MOCK=false`;`VITE_API_BASE_URL=https://{host}/api/v1`(运行时也可在设置入口改并持久化);浏览器端需后端 CORS 放行 Web 域名(Tauri 版原生栈不受限);全容器联调直接用 `scripts/dev-docker.sh`(Caddy 同域反代,无需 CORS);浏览器若持久化过旧 `app:apiBase`,在登录页点「重置后端接口地址」。契约变更流程:先改 docs/api/README.md → 再改 `src/types/api.d.ts` → api 模块。

### 4.4 设计令牌维护(重要)

CSS 变量在 `tokens.css`,Naive UI overrides 在 `theme.ts`(亮/暗两套真实色值)——**改色必须两处同步**,否则 naive 组件与自绘组件视觉漂移;业务代码禁止写死颜色,一律 `var(--c-*)`。

### 4.5 Tauri 桌面端

Rust ≥ 1.77(本机已验证)+ WebView2(Win);`pnpm tauri:dev` 联动开发,`pnpm tauri:build` 打包;首次 cargo 编译约 5 分钟(`cargo check` 可快速验证);改品牌后 `python scripts/gen-icon.py && pnpm tauri icon app-icon.png` 重新生成图标;发布前需注册 `ylink://` 协议与 updater 签名密钥(见 desktop-tauri.md §5/§7)。

### 4.6 上线(生产)

`dist/` 部署到 Nginx/Caddy/对象存储 + CDN(hash 路由无需服务端 rewrite);打包时写入 `VITE_API_BASE_URL`(或用户首次打开后自配);安全基线见 desktop-tauri.md §6(CSP 收紧;Markdown 已 DOMPurify 过滤,后端仍做写入侧清洗)。

## 5. 历史记录指引

- 评审批次与修复明细:[docs/reviews/](../reviews/)(review-round-01 起,冻结快照)
- 需求立项与决策:[.scratch/](../../.scratch/)(各 feature 的 spec.md)
- 逐次变更:git log(Conventional Commits)
