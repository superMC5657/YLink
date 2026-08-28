# [P1] Batch audit target must not exceed the column length

Status: resolved
Type: task

## Finding

`audit_logs.target` 为 VARCHAR(128)，但 `send_mail` 审计把最多 100 个 ID 的 `fmt.Sprint(req.IDs)` 直接写入 target（admin_service.go 未配置/正常两处分支），`traffic_reset`（admin_traffic_reset.go）同样写入最多 500 个用户 ID。ID 稍长时插入因超长失败，而返回值被 `_ =` 忽略——操作返回成功且审计静默丢失（违反资金/敏感操作审计要求）。

## Resolution

新增 `batchAuditTarget(n)`，两类批量动作 target 统一写 `batch:<count>` 摘要；完整 ID 列表移入 detail JSON（`ids` / `user_ids`）。展示端 `enrichAuditTargets` 对 `batch:` 前缀直接透出摘要不反查实体；历史 ID 列表格式（`"7"` / `"[7 8 9]"` / `"7,8,9"`）兼容解析不变。

## Comments

2026-08-28: 已修复；新增 `TestSendMailAuditTargetSummary` / `TestTrafficResetAuditTargetSummary` 以 sqlmock WithArgs 断言 INSERT 实参（target= batch:N、detail 含 IDs）。
