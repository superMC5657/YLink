# [P1] 12 Lint与超大文件债务

Status: ready-for-agent
Type: task

## Finding

- `eslint.config.ts:60-61` 关 `no-explicit-any/no-v-html`；`admin_service.go:1011/order_service.go:750/AdminNodesView:796/AdminUsers:626` 超大；CI 仅 tag、Go 不进 CI；`shortcuts.ts` 疑与 uno.config 双维护；`tokens.css+theme.ts` 双源同步靠人。

## Task

1. `any` 先 warn 起步，`no-v-html` 建白名单注释约束；`shortcuts` 留一处为准；加 tokens/theme 同步校验脚本。
2. 后端按 admin_crud/stats/withdraw 模式续拆；前端 Nodes/Users 拆 table/dialog/composable（本单只拆 + 门禁绿，不改行为）。
3. `server/` 加 push main 的 `go vet/test`（即使不进 Actions，也给 Makefile+文档），前端 PR 加轻量 lint+typecheck。

## Done

- `pnpm lint/typecheck/test` + `go vet/test ./...` 全绿；CI 文档更新。

## Comments
