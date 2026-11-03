# [P0] 种子账号必须携带订阅凭证 UUID

Status: resolved
Type: task

## Finding

`server/internal/repo/admin.go:422-428,447-453`：`EnsureAdmin` / `EnsureDemoUser` 创建用户时不填 `UUID`。migration 0004 起 `users.uuid` 为 `NOT NULL + UNIQUE`：

- Go 零值插入 `uuid=''`：第一个账号写空串成功后，第二个账号必然撞 `uq_users_uuid` 唯一约束（启动顺序 admin→demo，demo 创建失败）。
- 两个账号的订阅凭证为空，节点无法按 `users.uuid` 归因其流量（CONTEXT.md：订阅凭证=每用户独立 users.uuid）。

同文件 `SubToken` 生成处已有 `uuid.NewString()` 正确写法可对照。

## Plan

1. 两处种子创建补 `UUID: uuid.NewString()`。
2. 新增/扩展 repo 测试：连续 EnsureAdmin + EnsureDemoUser 不报唯一约束错误，且两账号 UUID 均非空。

## Acceptance

- 全新库上启动（admin+demo 先后创建）无错误；两账号 uuid 非空且互不相同。
- `go test ./internal/repo/...` 全绿。
