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

- [x] Notify 终态语义:验签失败→fail 重试;验签通过但非成功终态→落库+**success ack**+zap 告警(防重试风暴);账务失败→fail 重试。
- [x] 金额改字符串分位精确解析(`yuanStrToFen`:仅 0-2 位小数非负,拒绝科学计数法;`fenToYuanStr` 纯整数格式化不经 float);新增 epay_money_test 15 组断言(含 10.29/4.35 等经典差 1 分值)。
- [x] `NotifyResult.Raw` 携带回调原文,HandleNotify 对每条验签通过的通知落 `payments.notify_payload`(含非成功终态与幂等重放,补对账证据);`trade_no` 唯一索引迁移已在 0001_init 确认存在。
- [x] Query 改 POST form(key 不再进 URL/访问日志)。
- [x] Refund 链路:`RefundReq.gateway_refund_id`(后端 dto/handler/service 审计 detail + 前端类型),AdminOrdersView 退款改弹窗:明示「仅余额原路/现金需线下网关退/续期单退款整体收回订阅」+ 外部单号可选输入;docs/api 契约同步。
- [x] 门禁:`go vet` + `go test ./internal/...` 全绿(payment/service/router);前端 typecheck/lint 全绿。

## Comments
