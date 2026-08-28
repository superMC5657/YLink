# [P1] Withdraw/transfer amount must be overflow-safe

Status: resolved
Type: task

## Finding

`WithdrawCreateReq.Amount` 只校验 `required,gt=0`，`YuanToFen` 将 float64 转 int64 无溢出防护：金额约 > MaxInt64/100 时转换回绕为负的 MinInt64，`CommissionBalance < amountFen` 的余额校验因此通过，`CommissionBalance -= amountFen` 反而大幅增加佣金余额（资损）。`POST /invite/transfer` 在 handler 直接 `YuanToFen(req.Amount)`，存在同类问题。

## Resolution

新增 `validMoneyFen(fen)`：`0 < fen ≤ 2^52`（远离 int64 溢出边界，且 < 2^53 保证 float64 精度），应用于 `SubmitWithdraw`（转换后校验）与 `Transfer`（入口校验）。违规返回新错误码 `13005 金额无效或超出可处理范围`。定位为溢出/精度防护，非业务限额——spec F02 明确暂不引入提现限额设置项。

## Comments

2026-08-28: 已修复；新增 `TestTransferAmountOverflow`（1e19 元回绕负值 / 负数 / 0 / maxMoneyFen+1 → 13005）与 `TestSubmitWithdrawAmountOverflow`（1e19 / 1e14 元 → 13005）。
