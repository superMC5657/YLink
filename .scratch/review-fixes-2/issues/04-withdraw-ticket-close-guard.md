# [P2] Users must not close/reopen withdraw tickets

Status: resolved
Type: task

## Finding

提现工单提交即扣减佣金，生命周期依赖管理员 pay/reject；但 `TicketService.Close` 允许工单所有者关闭任意自己的工单。用户关闭提现工单后管理端 UI 禁用审核按钮，佣金可能长期停留在已扣减状态。`Reopen` 同理：管理员审核关闭后用户可重开，已完结的资金工单回到「待回复」误导管理端。

## Resolution

`Close` 与 `Reopen` 对 `type=1`（TicketTypeWithdraw）返回新错误码 `14003 提现工单不可手动关闭`（HTTP 409）；用户端 `TicketDetailView` 对提现工单隐藏「关闭工单」「重新打开」按钮。普通工单行为不变；管理员 pay/reject 审核闭环（含自动关闭工单）不受影响。

## Comments

2026-08-28: 已修复；新增 `TestTicketCloseWithdrawRejected` / `TestTicketReopenWithdrawRejected` / `TestTicketCloseNormalStillAllowed`（ticketRow 测试 helper 补 type 列）。
