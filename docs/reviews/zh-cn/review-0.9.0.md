# 代码评审 — YLink v0.9.0（存量节点兼容 & 节点上报/告警修复）

- **日期：** 2026-08-25
- **范围：** 订阅凭证切换兼容、节点上报分组校验与重复 UUID 拒绝、demo agent 零基线、Alertmanager 邮件投递、Windows NSIS 打包。
- **状态：** 3×P1 + 4×P2 全部修复。

## 发现

### ✅ [P1] Keep subscription credentials valid until node inbounds are provisioned

订阅下发仅在节点 config 显式开启 `per_user_credentials: true` 时使用 `users.uuid`；未开启的存量节点继续使用 `servers.config` 共享凭证，inbound 未配发前不会因订阅刷新断连。

### ✅ [P1] Use a STARTTLS-capable SMTP endpoint for Alertmanager

Alertmanager 默认走 `ALERT_SMTP_PORT=587`（STARTTLS），不再复用后端 QQ 465 隐式 TLS；必须用 465 时文档要求前置 SMTPS→STARTTLS relay，或覆盖 `ALERT_SMTP_HOST/FROM`。

### ✅ [P1] Add the configured NSIS hook file

已提交 `src-tauri/nsis/installer-hooks.nsh`（含 4 个 no-op 宏），`tauri.conf.json` 的 `installerHooks` 路径在 Windows NSIS 构建时可解析。

### ✅ [P2] Limit reports to users in the authenticated node's group

`POST /node/report` 校验用户套餐 `group_ids` 是否包含节点分组，不匹配返回 `not_subscribed`。

### ✅ [P2] Reject duplicate UUIDs in a report payload

单请求重复 UUID 在查用户/开事务前整体拒绝，所有重复条目返回 `duplicate_uuid`。

### ✅ [P2] Send the zero baseline before advancing demo counters

`node-agent` 先上报当前计数（启动首轮为 0）建立快照基线，再推进下一轮随机累计值。

### ✅ [P2] Escape SMTP values before inserting them with sed

Alertmanager entrypoint 改用 AWK 逐字替换模板，并按 YAML 单引号规则转义，避免合法用户名/密码损坏配置。

## 变更

- 后端：`subscribe_service.go`、`node_service.go` 及对应测试、`cmd/node-agent/main.go`。
- 可观测性：`docker-compose.yml`、`alertmanager.yml.tmpl`、`server/.env.example`。
- 桌面端：`src-tauri/nsis/installer-hooks.nsh`（新增）。
- 文档：API 契约、后端 core-flows/data-model/deploy/progress/checklist、前端 Tauri 文档、本评审记录与 `.scratch/review-fixes/`。

## 验证

- `go test ./internal/service/... -count=1` 通过。
- 变更 Go 文件 `gofmt -l` 无输出。
- 可用环境下完成 Docker Compose 配置解析与其余仓库检查。
