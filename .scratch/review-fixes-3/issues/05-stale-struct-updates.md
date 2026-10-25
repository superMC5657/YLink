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

## Comments

- 2026-10-25: 已修复——repo 新增 `UpdatePasswordHash`/`UpdateSubToken` 按列更新；用户端改密与重置订阅改走按列更新；测试以锚定正则断言 UPDATE 仅含目标列（+gorm 自动 updated_at），全结构体回写会因首列不匹配而失败。core-flows.md 无「全量保存」类语句，无需变更；`go build/vet/test ./...` 全绿。
- 2026-10-25: Finding 遗漏的同语义调用点一并收敛（仍属本 ticket「改密/重置订阅」主题）：`auth_service.Forgot`（忘记密码改密）与 `admin_user.ResetUserSubToken`（管理端重置订阅密钥）此前的全结构体 Updates 同款陈旧快照风险，已同范式改按列更新。至此 `repos.User.Update`（全结构体）在 service 层无非测试调用方。
