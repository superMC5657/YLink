# 性能·安全·可维护性优化批次

> 状态: 进行中
> 日期: 2026-09-07
> 范围: 前端首屏体积(P0)/敏感信息与支付一致性(P0)/桌面CSP(P0)/工程债务(P1)/体验细节(P2)
> 基线: main @ ae87f3b，门禁 `pnpm lint/typecheck/test/format:check` + `go test ./...`
> 约定: 一 ticket 一 commit；先改契约/类型再改实现；每个 ticket 必须过对应门禁才可提交

## Requirements

### [P0] 首屏体积：重依赖懒加载与分包
- `qrcode` 同步打入首屏(PaymentModal/SharePanel) → NQrCode 或动态 import
- `markdown-it` 三处各 `new` 且两处在首屏链路 → 单例 `useMarkdown` + 懒加载
- `echarts` 两路由重复同步引入 → 共享 `useEcharts` + 动态 import
- `vite.config` 无 manualChunks/压缩、warningLimit 1500 掩盖告警 → 拆 vendor chunks + compression +Visualizer 取数

### [P0] 敏感信息：Token 与订阅链接
- `auth.ts persist` 明文存 access+refresh → 缩小持久面 + 掩码展示 + 登出清理
- `CopyText title=text` 全文进 DOM/剪贴板 → 去全文，订阅默认掩码

### [P0] 交易一致性：回调与退款
- Notify 对业务终态回 fail 致重试风暴 → 验签失败 fail / 业务终态 success+告警
- 金额 `float*100` 差1分失败 → 字符串分位解析；`notify_payload` 落库
- 退款仅本地、无网关调用 → UI 明示 + `gateway_refund_id` 审计字段

### [P0] 桌面安全：CSP 与能力收敛
- CSP 缺 connect-src、http allow `https://**` 过宽 → 收敛到 apiBase+updater 域名

### [P1] 工程债务
- 死依赖 `@iconify/vue` / 疑似 `@vueuse/core` / 空 `presetIcons`
- 超大文件：`admin_service.go:1011` / `order_service.go:750` / `AdminNodesView:796`
- `eslint` 关 `no-explicit-any`/`no-v-html`；CI 仅 tag、Go 不进 CI
- stores 无 SWR、`storage.ts` 高频写无节流

### [P2] 体验细节
- layouts 同步进首包 → 懒加载；路由缺 chunk 命名
- 管理 guard 刷新闪跳 → 启动补 role + `/403`
- PWA `navigateFallback` 无 denylist、无 runtimeCaching；AdminReports 5图全量 render

## Tickets
- 01-dead-deps → 02-qrcode-lazy → 03-markdown-singleton → 04-echarts-shared
- 05-vite-chunks → 06-layouts-lazy → 07-authpersist-submask → 08-payment-notify
- 09-csp-allowlist → 10-guards-403 → 11-pwa-echarts-ux → 12-lint-ci-debt
- 每 ticket 独立 commit，message 格式 `perf|fix|sec(scope): ...`

## 验证矩阵
- 前端每单：`pnpm lint` + `typecheck` + 相关 vitest；收尾全量 `test/build:web`
- 后端每单：`go vet` + `go test ./internal/service/ -run TestXxx -count=1`
- 安全每单：XSS 单测 / 403 回归其一
