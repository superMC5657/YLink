# [P1] 异步投递与密码学卫生批修

Status: resolved
Type: task

## Finding

1. `service/auth_service.go:114-122`：验证码 Get→比对→Del 三步存在窗口，一次性验证码可被并发请求复用（tg-bind 流程已用 `GetDel` 正确写法）。
2. `service/auth_service.go:286-291`：`issueSession` 忽略 `rdb.Set` 错误——refresh token 可能未入白名单即返回成功。
3. `service/invite_service.go:132`：`n, _ := rand.Int(rand.Reader, …)` 出错时 `n` 为 nil → `n.Int64()` panic。
4. `service/order_payment.go:188`：`sendReceiptMailAsync` goroutine 无 `recover`，render/mailer panic 带崩 API 进程（repo 自身 `cron_schedule.go:165 deliverAsync` 政策要求带 recover）。
5. `pkg/mailer/mailer.go:38-41`：`Send` 无 context/超时（net/smtp 无 deadline），调用方均 fire-and-forget——SMTP 挂起 → goroutine 无限泄漏。
6. `pkg/payment/epay.go:149-171`：`verify()` 逐字复刻 `sign()`（Duplicated Code）且用非常量时间 `==` 比较 MD5。

## Plan

1. 验证码校验改 `GetDel`（先取后删一步完成再比对）。
2. `issueSession` 检查 Set 错误并返回（登录失败优于「不在白名单的会话」）。
3. `rand.Int` 错误传播（返回 err）。
4. `sendReceiptMailAsync` 套用 deliverAsync 的 recover 形状。
5. mailer：Dialer 增加 `ConnTimeout`/deadline（gomail `Dialer` 底层 net.Dialer 可注入 context；或包一层 `net.Conn` SetDeadline）；调用点传入带超时的 ctx 或统一 30s deadline。
6. epay：`verify` 改为 `subtle.ConstantTimeCompare([]byte(e.sign(form)), []byte(sign))`，同时消灭重复实现。

## Acceptance

- 并发重复使用同一验证码只有一个成功（测试）。
- SMTP 不可达时登录/支付回调仍正常返回，无 goroutine 泄漏（可用 goleak 或超时断言）。
- epay verify 单元测试不变全绿（行为等价、实现单一）。
- `go test ./...` 全绿。
