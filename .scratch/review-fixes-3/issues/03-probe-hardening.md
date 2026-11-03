# [P0] 探针与 Telegram Webhook 加固

Status: resolved
Type: task

## Finding

- `server/internal/handler/health.go:46-52`：`dbPing` 用 `context.Background()` 调 `Ping`，`/readyz` 的 2s 超时（health.go:32）只约束 Redis——DB 挂起时探针无限挂起，k8s/compose 健康检查超时但进程内 goroutine 泄漏。
- `server/internal/handler/health.go:36,40`：`/readyz` 把 `err.Error()` 原文返回给调用方（含 DB 主机/端口等内部拓扑）；该端点在 router.go:76-89 的 SafeMode/api 组之外，公网可达。
- `server/internal/handler/telegram.go:57-63`：Webhook 先 `ShouldBindJSON` 再校验 `X-Telegram-Bot-Api-Secret-Token`——带错误/缺失 secret 的坏 JSON 请求得到 200 `ok`，而 docs/api/README.md §19 要求 secret 校验失败返回 403（「secret 校验失败除外」语义被 bind 错误掩盖）。

## Plan

1. `dbPing` 接收请求 ctx（或 `context.WithTimeout(ctx, 2*time.Second)`）并改用 `db.PingContext`。
2. `/readyz` 响应只回 `{"status":"unavailable"}` 类通用文案（沿用现有 envelope 形状），具体错误进日志。
3. Telegram Webhook：secret 校验提前到 body 读取之前，不匹配直接 403；bind 错误维持原契约行为。

## Acceptance

- DB 不可达时 `/readyz` 在 2s 内返回，且响应体不含 DSN/主机名等内部细节。
- 错误/缺失 secret 的任意 body → 403；正确 secret + 坏 JSON → 原有 bind 错误行为。
- handler 测试覆盖上述分支；`go test ./internal/handler/...` 全绿；docs/api §19 无需变更（实现对齐契约）。
