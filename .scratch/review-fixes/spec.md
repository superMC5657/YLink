# Review Fixes 0.9.0 (✅ 完成)

> 状态: 全部修复完成
> 日期: 2026-08-25
> 范围: 存量节点凭证兼容、节点上报安全/幂等、demo agent 基线、Alertmanager 邮件、NSIS 打包

## Requirements

### ✅ [P1] Keep subscription credentials valid until node inbounds are provisioned

节点 config 显式开启 `per_user_credentials: true` 后才下发 `users.uuid`；未开启的存量节点继续使用 `servers.config` 共享密码/uuid，避免订阅刷新即断连。

### ✅ [P1] Use a STARTTLS-capable SMTP endpoint for Alertmanager

Alertmanager 默认走 `ALERT_SMTP_PORT=587`（STARTTLS），不再使用后端 QQ 465 隐式 SMTPS；465 场景需前置 SMTPS→STARTTLS relay。

### ✅ [P1] Add the configured NSIS hook file

已入库 `src-tauri/nsis/installer-hooks.nsh`，Windows NSIS bundle 不再因配置路径缺失失败。

### ✅ [P2] Limit reports to users in the authenticated node's group

`POST /node/report` 校验用户套餐 `group_ids` 是否包含节点分组；不匹配返回 `not_subscribed`。

### ✅ [P2] Reject duplicate UUIDs in a report payload

同一 UUID 在单次请求重复出现时，该 UUID 全部条目在查库/开事务前整体拒绝，响应 `duplicate_uuid`。

### ✅ [P2] Send the zero baseline before advancing demo counters

`node-agent` 首轮上报当前值（0 基线）建立服务端快照，发送成功后再推进下一轮随机累计值。

### ✅ [P2] Escape SMTP values before inserting them with sed

Alertmanager entrypoint 用 AWK 逐字替换模板，并按 YAML 单引号规则转义，避免用户名/密码含 `&`、`\`、`|`、`'` 时配置损坏。

## Verification

- `go test ./internal/service/... -count=1` 通过。
- `gofmt -l` 无输出。
- 文档已同步: `docs/api/README.md`、`docs/backend/*.md`、`docs/frontend/desktop-tauri.md`、`docs/reviews/review-0.9.0.md`(2026-09-03 文档重整后 reviews 单语化为中文,原英文版见 git 历史)。
