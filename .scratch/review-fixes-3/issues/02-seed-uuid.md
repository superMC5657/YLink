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

## Comments

- 2026-10-25: 已修复——`EnsureAdmin`/`EnsureDemoUser` 两处种子创建补 `UUID: uuid.NewString()`（与同结构 `SubToken` 同款写法）；新增 `internal/repo/admin_test.go`：gorm create 回调捕获落库前模型态，断言 admin+demo 先后创建成功、两账号 uuid 均为合法 UUID 且互不相同（uuid 列 NOT NULL+UNIQUE，非空互异值构造上不可能撞 `uq_users_uuid`），并保留幂等跳过分支测试。`go build/vet/test ./...` 全绿。
- 2026-10-25: 真实库端到端（全新 PostgreSQL 建库→迁移→启动）未验证——本机 Docker daemon 未运行，未擅自启动；已以 sqlmock 断言 INSERT 前模型态作为替代覆盖。
