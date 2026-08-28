# Review Fixes 2 (✅ 完成)

> 状态: 全部修复完成
> 日期: 2026-08-28
> 范围: 近期 13 次提交引入的正确性问题——审计丢失、资金金额溢出、提现流程阻塞、时区偏移、零值时间展示
> 关联: 上一轮见 `.scratch/review-fixes/`（0.9.0 基础设施批次）

## Requirements

### ✅ [P1] Batch audit target must not exceed the column length

`send_mail`（2 处）与 `traffic_reset` 审计的 target 原为 `fmt.Sprint(IDs)`（最多 100/500 个 ID），超出 `audit_logs.target` VARCHAR(128) 时插入失败且返回值被忽略——操作成功但审计静默丢失。现改为 `batch:<count>` 摘要，完整 ID 列表留痕在 detail JSON（`ids`/`user_ids`）；展示端对 `batch:` 前缀直接透出摘要，历史 ID 列表格式仍兼容解析。

### ✅ [P1] Withdraw/transfer amount must be overflow-safe

提现金额只校验 `required,gt=0`，`YuanToFen` 对超大金额（约 > MaxInt64/100）回绕为负 MinInt64——`CommissionBalance < 负金额` 恒不成立，后续扣减反而**增加**佣金余额（资损）。现新增 `validMoneyFen`（`0 < fen ≤ 2^52`，远离溢出边界且保证 float64 精度）应用于 `SubmitWithdraw` 与 `Transfer`（划转在 handler 转分，同类漏洞一并修复），违规返回新错误码 **13005**。此为溢出防护，非业务限额（spec F02 不引入限额设置）。

### ✅ [P2] Audit log date filter must use the user's local timezone

`AdminAuditLogsView` 用 `toISOString().slice(0,10)` 生成 from/to——UTC 日期在东八区会把本地 8/28 00:00–07:59 的筛选起点前移到 8/27。现新增 `localDateKey`（dayjs `format('YYYY-MM-DD')`）统一处理。

### ✅ [P2] Users must not close/reopen withdraw tickets

提现工单提交即扣减佣金，生命周期依赖管理员 pay/reject；`TicketService.Close` 却允许工单所有者关闭任意自己的工单——关闭后管理端审核按钮禁用，佣金长期停留在已扣减状态。现 `Close`/`Reopen` 均拒绝 type=1 工单（新错误码 **14003**），用户端对提现工单隐藏关闭/重开按钮；管理员审核闭环（审核后自动关闭）不受影响。

### ✅ [P3] Sessions without metadata must return null created_at

升级前 refresh 白名单值为字符串 `"1"`，元数据解析失败时 `CreatedAt` 保持 Go 零值并序列化为 `0001-01-01T00:00:00Z`，前端显示 0001/1/1。现 `UserSessionItem.CreatedAt` 改为 `*time.Time`，无元数据返回 **null**，前端 `api.d.ts` 同步为 `string | null`（ProfileView 的 truthiness 判断天然兼容，显示「--」）。

### ✅ [P3] Traffic import default date must be local

`AdminTrafficImportView` 初始行与 `addRow()` 用 `toISOString().slice(0,10)` 生成默认日期——东八区 0:00–7:59 会默认到前一天，管理员不改动就会导错日期。现改用 `localDateKey(new Date())`。

## Verification

- `cd server && go test ./... -count=1` 通过（service/errs/router 等全绿）。
- 新增 Go 测试：`TestSendMailAuditTargetSummary`、`TestTrafficResetAuditTargetSummary`（断言 INSERT audit_logs 实参 target= batch:N + detail 留痕）、`TestTransferAmountOverflow`、`TestSubmitWithdrawAmountOverflow`、`TestTicketCloseWithdrawRejected`、`TestTicketReopenWithdrawRejected`、`TestTicketCloseNormalStillAllowed`；`TestListAndRevokeSessions` 补历史会话 nil 断言。
- `gofmt`（本次改动文件）/ `go vet` clean；`npm run typecheck`、`npm run lint`（--max-warnings 0）、`npm run test`（Vitest 63 用例，含新增 localDateKey 时区用例）全部通过。
- 文档已同步: `docs/api/README.md`（错误码 13005/14003、§5.7 会话 created_at 可 null、§11.5 提现金额防护、§13.2 工单关闭限制、§16 审计 batch 摘要格式）。
- 备注: 全量 e2e 未跑（沿用上轮结论：本地 5174 端口残留非 mock dev server，不作为代码结论依据）。
