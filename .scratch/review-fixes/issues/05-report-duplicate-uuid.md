# [P2] Reject duplicate UUIDs in a report payload

Status: resolved
Type: task

## Finding

单请求可包含同一 UUID 的多个累计值，逐个差分会导致错误计费。

## Resolution

查库/开事务前用 `dedupeReportItems` 整体拒绝重复 UUID 条目，响应 `duplicate_uuid`。

## Comments

2026-08-25: 已修复并补 `TestNodeReportDuplicateUUID`。
