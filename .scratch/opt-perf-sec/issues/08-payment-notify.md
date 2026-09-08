# [P0] 08 支付Notify+金额+退款

Status: ready-for-agent
Type: task

## Finding

- `handler/order.go:192 Notify` 对验签通过但非成功回 400 fail → 网关重试风暴。
- `epay.go yuanStrToFen float*100+0.5` 差1分整单失败；`Query` key 放 GET query；`notify_payload` 未落库。
- `admin_service.go:650 Refund` 仅本地余额+收订阅，无网关退款调用。

## Task

1. Notify 区分 验签失败→fail重试 / 业务终态→success+告警；`payments.notify_payload TEXT` 存原文（确认 trade_no 唯一索引迁移）。
2. 金额改字符串分位解析；Query 改 POST/form。
3. Refund UI 明示“仅余额原路，现金需线下网关退并填外部单号”，审计 detail 加 `gateway_refund_id`；同套餐续期退款整体清订阅加二次确认文案。

## Done

- `go test ./internal/service -run TestOrder|TestRefund|TestNotify` 全绿；对账证据可查。

## Comments
