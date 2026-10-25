# [P3] 后端死代码清理

Status: resolved
Type: task

## Finding

以下符号经 grep 全仓（含测试）确认零引用或空壳（ADR-0005「不得以过渡为名重引入转发层」精神）：

- `service/cron_service.go:193` `sendExpireMail`（迁移已完成的前置转发适配器）
- `service/order_coupon.go:267` `couponCode` 包装
- `service/order_service.go:152-159` `priceOf`/`planPrices`「存量薄 wrapper」；`:112-125` `orderTradeSeam` 接口仅剩 `var _` 断言、从未作类型使用
- `service/session.go:22` `bumpSessionVersion`（5 处调用全走 `NewSessionRevoker(...).RevokeAll`；且硬编码 RevokeReasonBan）
- `service/subscribe_service.go:294` `toNode`；`:215` `renderSubscription` 仅测试使用（可内联进 batch4_test）
- `service/subscription_visibility.go:25` `subscriptionVisibility` 结构体从未实例化（模块实为自由函数）
- `service/server_service.go:18`、`ticket_service.go:19` 注入的 `rdb` 字段无任何方法读取；`order_service.go:31` `set` 同
- `service/session_revocation.go:65` `RevokeAll(_ RevocationReason)` 参数被丢弃（六种 reason 行为一致）——删参或落日志，二选一
- `pkg/validate` 整包零 importers（其自称的「保证既有引用路径」已不成立）
- `pkg/payment/payment.go:67-73` `Available()` 零调用
- `pkg/subscribe/subscribe.go:16` `Node.Cipher` 无读写
- `internal/config/config.go:113-120` `SiteConfig` 零消费且与 settings 表事实重复

## Plan

逐项删除（`renderSubscription` 与 `orderTradeSeam` 按注释判断：若标注「迁移过渡」且迁移完成即删；测试内联随迁）。每删一项跑 `go build ./... && go vet ./...`。

## Acceptance

- `go build ./...`、`go vet ./...`、`go test ./...` 全绿。
- `grep -rn "sendExpireMail\|couponCode\|bumpSessionVersion\|pkg/validate" server/` 零命中。

## Comments

2026-10-25: 已修复——清单 12 项全删（sendExpireMail、couponCode、priceOf/planPrices 存量 wrapper、orderTradeSeam 空壳接口、bumpSessionVersion、toNode、renderSubscription（测试内联进 batch4_test）、subscriptionVisibility 空壳结构体、Server/Ticket 服务 rdb 与 Order 服务 set 注入未用字段、RevokeAll 走删参数并连带移除 RevocationReason 类型与六常量、pkg/validate 整包、payment.Available、Node.Cipher、SiteConfig）；`go build/vet/test` 全绿、`gofmt -l` 无输出、Acceptance grep 零命中。
