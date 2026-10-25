# [P2] 后端契约与规范合规批（审计 target / IDParam / models tag / metrics 基数）

Status: resolved
Type: task

## Finding

1. **批量审计 target 违规**：`admin_node_batch.go:70,113`（batch_server_delete/update）、`admin_content_manage.go:27,39`（sort_server/notice/knowledge）、`admin_traffic_reset.go:25`（traffic_import）写 `""`，违反 CONTEXT.md/docs/api §16.1「批量 target 只存 `batch:<count>`」（仅 send_mail/traffic_reset 合规，`batchAuditTarget` 已存在）。
2. **IDParam 收敛被绕过**：`handler/ticket.go:71,96,121,144` + `handler/content.go:81` 五处手写 `strconv.ParseInt(c.Param("id"))…40000`，helper.go:12 自称「param 解析唯一 owner」。
3. **models.go tag 与 data-model/migration 漂移**：KnowledgeCategory 唯一索引缺 language（:286，应为 `(language,name)`）；TrafficLog 是普通 index 而文档/迁移为 `UNIQUE(user_id,date)`（:339-341）；order_no「全表唯一」注释与非唯一 tag 自相矛盾（:208,213，实际是 `WHERE biz_type=0` 部分唯一）；users/orders/mail_logs/traffic_reset_logs/commission_withdraws/knowledges 六处复合索引仅声明首列。
4. **文档缺表**：data-model.md 缺 `agent_applies` 表（0001:346）与 `Coupon.used_count` 字段。
5. **metrics 基数**：`middleware/metrics.go:61-63` 未匹配路由以原始 `URL.Path` 作 label——404 扫描造成无基数上限的时序爆炸。

## Plan

1. 5 处批量动作改 `batchAuditTarget(len(ids))`，完整 ID 列表入 detail JSON（照 send_mail 既有形状）。
2. 5 处 handler 改 `id, ok := IDParam(c)`。
3. models.go tag 按迁移真实定义逐一对齐（唯一索引/复合索引完整声明），order_no 注释改为部分唯一表述。
4. data-model.md 补 `agent_applies` 小节与 `used_count` 字段行。
5. metrics 未匹配路由 label 归一为常量 `unmatched`（或 hash）。

## Acceptance

- 新增测试：上述批量动作的审计 target=`batch:<N>`、detail 含 ids（照 TestSendMailAuditTargetSummary 形状）。
- `grep -n "ParseInt(c.Param" server/internal/handler` 仅剩 helper.go 一处。
- `go test ./...` 全绿；data-model.md 与迁移/结构体三方一致（抽查六处索引）。

## Comments

- 范围外备忘（未顺手修）：单列普通索引 `agent_applies(status)`、`commission_logs(biz_type)` 在 models.go tag 中仍未声明（本次对齐范围为唯一/复合索引）；`auditTargetKind` 未收录 `batch_server_*`/`sort_*`/`traffic_import`，这些动作的 `batch:<N>` target 在审计列表原样展示（kind=null，不影响主查询）。
- 2026-10-25: 已修复 批量审计 target 统一 `batch:<count>`（batch_server_delete/update、sort_server/notice/knowledge、traffic_import 共 5 处，完整 ID 列表入 detail JSON，新增 TestBatchActionsAuditTargetBatchSummary）；ticket.go×4 + content.go×1 改走 `IDParam`（grep 仅剩 helper.go 一处 ParseInt）；models.go tag 对齐迁移（users/orders 复合、coupon_usages 唯一、knowledge_categories `(language,name)`、traffic_logs `UNIQUE(user_id,date)`、mail_logs/traffic_reset_logs/commission_withdraws/knowledges 复合完整声明，order_no 注释改部分唯一表述）；data-model.md 补 agent_applies 小节与 Coupon.used_count；metrics 未匹配路由 path label 归一常量 `unmatched`。go build/vet/test 全绿，gofmt 无输出。
