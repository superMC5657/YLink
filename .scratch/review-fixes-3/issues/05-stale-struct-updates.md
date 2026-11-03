# [P1] 改密/重置订阅不得覆盖并发写

Status: resolved
Type: task

## Finding

- `server/internal/service/user_service.go:91`（修改密码）与 `subscribe_service.go:113`（重置订阅凭证）走 `repos.User.Update(u)` 全结构体 `Updates`——u 是请求开始时读出的完整快照。
- 用户操作期间并发的余额/流量/佣金/到期时间写会被陈旧快照整体覆盖（丢钱、丢流量）。
- `telegram.go:178` 对同一风险已有注释并改用 `Update("telegram_id", …)` 按列更新——范式已在库内。

## Plan

1. user_service 改密：`Update("password_hash", …)`（必要时同列带上 SV bump 前置）。
2. subscribe_service 重置订阅：`Update("sub_token", …)`（uuid/token 相关列）。
3. 各自补「并发修改 balance 不被覆盖」的测试（sqlmock 断言 UPDATE 列集合不含 balance）。

## Acceptance

- UPDATE 语句只含目标列；并发字段不被覆盖（测试断言实参）。
- `go test ./internal/service/...` 全绿；core-flows.md 相关流程描述如有「全量保存」语句改为按列更新。
