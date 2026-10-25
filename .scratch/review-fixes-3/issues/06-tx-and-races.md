# [P1] 工单回复竞态、流量导入双插与 UpdateUser 事务

Status: resolved
Type: task

## Finding

1. `server/internal/service/admin_ticket.go:59-79`：ReplyTicket 在事务外检查 `t.Status==2`，随后无条件 `UpdateStatus(tx, id, 1)`——检查后并发关闭的工单被写入回复并翻回「已回复」，绕过 reopen-once 状态机（review-fixes-2/04 刚修过的生命周期又被开了口子）。`CloseOrder` 已有 `UpdateStatusIfPending` 条件更新范式。
2. `server/internal/repo/admin.go:323-331`：`TrafficLogRepo.Upsert` 是 Count→Update/Create 的 check-then-write，并发导入双插触发 `uk_traffic_user_date` 唯一约束 → 500。同文件 `AdditiveUpsert`（node.go:46）已用 `clause.OnConflict`。
3. `server/internal/service/admin_user.go:42-77`：UpdateUser 依次 SetBanned + UpdateRole + 2 次审计写，无事务——中途失败留下「已封禁未改角色」的部分状态；兄弟方法 `AdjustBalance` 已用 `repo.WithTx`。

## Plan

1. ReplyTicket 改条件更新：仅当 `status IN (已开启/已回复)` 时写入并更新状态（`UPDATE ... WHERE id=? AND status<>2`，受影响行数=0 则走已关闭分支的返回语义），回复与状态更新同事务。
2. Upsert 改 `clause.OnConflict{Columns: user_id+date, DoUpdates: 覆盖列}`（模式 B 同日覆盖语义不变，见 CONTEXT.md 流量模式 B）。
3. UpdateUser 包 `repo.WithTx`，审计写纳入同一事务（顺带消除该路径 `_ =` 忽略）。

## Acceptance

- 并发「关闭 vs 回复」交错后工单不可能处于「已关闭却有新回复且状态=已回复」。
- 并发同日导入不产生唯一约束错误（测试用重复调用断言）。
- UpdateUser 中途失败无部分状态残留。
- `go test ./internal/...` 全绿。

## Comments

- 2026-10-25: 已修复——repo 新增 `TicketRepo.UpdateStatusIfNotClosed`（`WHERE status<>2` 条件更新、0 行=已关闭）；管理端 ReplyTicket 在事务内以条件更新为守卫（0 行走已关闭分支语义，回复/状态/last_reply_at 同事务）；`TrafficLogRepo.Upsert` 改 `clause.OnConflict`（user_id+date）单语句覆盖式 upsert（模式 B 语义不变）；UpdateUser 整体包 `repo.WithTx`，审计写纳入同一事务且不再 `_ =` 忽略，会话吊销（Redis 非事务资源）移至库事务成功后。测试：管理端/用户侧回复竞态关闭（0 行→回滚+14001）、同日重复导入断言单语句 ON CONFLICT 无 Count 前置、审计失败整体回滚且不 bump 会话版本。`go build/vet/test ./...` 全绿。
- 2026-10-25: Finding 遗漏的同款竞态一并收敛（仍属本 ticket 回复竞态主题）：用户侧 `TicketService.Reply`（ticket_service.go）与管理端 ReplyTicket 存在相同的「检查后关闭」绕过 reopen-once 问题，已同范式加条件更新守卫。
- 2026-10-25: 真实并发交错（两连接同库竞争）未做真实库演练——本机 Docker daemon 未运行；以 sqlmock 断言条件更新 SQL 形状（WHERE status<>2 + RowsAffected 分支）作为替代覆盖。
