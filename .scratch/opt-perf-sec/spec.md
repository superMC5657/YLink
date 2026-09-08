# 性能·安全·可维护性优化批次

> 状态: ✅ 已完成(2026-09-08)
> 日期: 2026-09-07
> 范围: 前端首屏体积(P0)/敏感信息与支付一致性(P0)/桌面CSP(P0)/工程债务(P1)/体验细节(P2)
> 基线: main @ ae87f3b，门禁 `pnpm lint/typecheck/test/format:check` + `go test ./...`
> 约定: 一 ticket 一 commit；先改契约/类型再改实现；每个 ticket 必须过对应门禁才可提交

## Requirements

### ✅ [P0] 首屏体积：重依赖懒加载与分包(01-05)
- ~~`qrcode` 同步打入首屏(PaymentModal/SharePanel)~~ → useQrCode 懒加载
- ~~`markdown-it` 三处各 `new` 且两处在首屏链路~~ → renderMarkdown 单例懒加载 + XSS 策略集中
- ~~`echarts` 两路由重复同步引入~~ → 共享 useEcharts + 动态 import + Admin 5 图视口懒初始化
- ~~`vite.config` 无 manualChunks/压缩、warningLimit 掩盖告警~~ → vendor 分包 + gzip/brotli 预压缩 + visualizer 取数

### ✅ [P1] 布局与体验(06)
- ~~layouts 同步进首包~~ → 三布局懒加载独立 chunk,管理员空闲预取重页面

### ✅ [P0] 敏感信息：Token 与订阅链接(07)
- ~~`auth.ts persist` 明文存 access+refresh~~ → access 仅内存 + refresh 最小持久化 + 过期清理
- ~~`CopyText title=text` 全文进 DOM/剪贴板~~ → 订阅默认掩码展示

### ✅ [P0] 交易一致性：回调与退款(08)
- ~~Notify 对业务终态回 fail 致重试风暴~~ → 验签失败 fail / 业务终态 success+告警
- ~~金额 `float*100` 差1分失败~~ → 字符串分位解析;`notify_payload` 落库
- ~~退款仅本地、无网关调用~~ → UI 明示 + `gateway_refund_id` 审计字段

### ✅ [P0] 桌面安全：CSP 与能力收敛(09)
- ~~CSP 缺 connect-src/object-src/base-uri~~ → 已补全(connect-src 含 localhost 兜底)
- http allow `https://**` **有意保留**(apiBase 运行时可配是产品能力,updater 走 Rust 侧不受 scope 约束);取舍记录见 ticket 09 Comments 与 desktop-tauri.md §3;发布签名缺失已在流水线强制失败

### ✅ [P1] 路由守卫与 403(10)
- ~~启动仅凭持久化 user 放行 admin~~ → 进管理路由前经 `/user/profile` 核实 role(契约新增 `role` 字段,每会话一次),非管理员进 `/403` 页;mock 对齐后端 40300;admin_path 仅 obscurity 注记入 deploy.md

### ✅ [P2] PWA 与图表体验(11)
- ~~navigateFallback 无 denylist/runtimeCaching~~ → `/api/` denylist + GET NetworkOnly;图标补 144/384/maskable;glob 补 woff/webp/webmanifest/json
- ~~AdminReports 无占位/采样~~ → 视口懒初始化(04 交付)+ dataZoom + showLoading + resize 200ms 防抖(useEcharts 收口)

### ✅ [P1] 工程债务(12)
- ~~eslint 关 no-explicit-any/no-v-html~~ → any warn + v-html error(4 处白名单注释)
- ~~`shortcuts.ts` 与 uno.config 双维护~~ → 空壳死文件已删,单源 uno.config.ts
- ~~tokens/theme 双源同步靠人~~ → scripts/check-tokens-sync.mjs 挂入 lint 链
- ~~超大文件~~ → 后端 admin_service 1014→543(+audit/order/ticket 三域文件)、order_service 758→340(+coupon/payment/fulfill);前端 AdminNodesView 796→278、AdminUsersView 629→261(composable+弹窗组件)
- ~~CI 仅 tag~~ → PR 轻量门禁(lint+typecheck),tag 全量不变;根 Makefile `make go-check` 承接后端推送检查

## Tickets

- ~~01-dead-deps → 02-qrcode-lazy → 03-markdown-singleton → 04-echarts-shared~~ ✅
- ~~05-vite-chunks → 06-layouts-lazy → 07-authpersist-submask → 08-payment-notify~~ ✅
- ~~09-csp-allowlist → 10-guards-403 → 11-pwa-echarts-ux → 12-lint-ci-debt~~ ✅
- 每 ticket 独立 commit,message 格式 `perf|fix|refactor(scope): ...`(commitlint 限定类型枚举,`sec` 并入 fix)

## 验证矩阵(收尾态)

- 全量门禁: `pnpm lint`(含 error-toast/tokens 双源校验)+ `typecheck` + `format:check` + vitest 79 例 + e2e 58 例 + `build:web`(PWA 89 entries) 全绿
- 后端: `go vet ./...` + `go test ./...`(8 包) 全绿
- 契约变更: `/user/profile` 补 `role` 字段(docs/api/README.md §5.2),先文档后实现
