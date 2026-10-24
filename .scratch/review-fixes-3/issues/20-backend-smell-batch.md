# [P3] 后端重复形状收敛与分层渗漏回收

Status: ready-for-agent
Type: task

## Finding

- **重复形状**：repo 层 11 处 `Count + Order("id DESC") + Offset/Limit`（admin.go:18/153/202/214/248、order.go:54/242/254/313、ticket.go:24、content.go:13）；`order_coupon.go:278-301` 手写 `containsInt64/containsStr/min64`（Go 1.26 有 slices.Contains/min）；settings-JSON-int 默认值解析 4+ 处两式访问（invite/cron/mail_template/setting）；SortNotices/Knowledges/Servers 三份同形（content_manage:20-41 vs node_batch:106-115）；GB→bytes 两处（admin_order.go:172、admin_traffic_reset.go:68）；handler 可选 body bind ×4；user/admin order 状态解析块重复。
- **分层渗漏**：service 直写 GORM 绕过 repo——`admin_crud.go:272-295,330-369`（Coupon List/Update 缺口）、`admin_content_manage.go:118,271-299`、`admin_stats.go:41-66`（Overview 8 查询）、`telegram.go:109,180,197`、`ticket_withdraw_lifecycle.go:183`；零值 `(repo.XRepo{}).Get` 绕过注入 seam 3 处（setting_service:66、subscribe_service:223、ticket_withdraw_lifecycle:60）。
- **缓存旁路**：`setting_service.go:66 commissionRateFor` + `telegram.go:68 settings()` 直读 DB，绕过同文件 SettingService 的 60s Redis 缓存；commissionRateFor 每笔支付订单执行。
- **N+1**：`admin_order.go:40-44`、`admin_agent.go:31-35`、`admin_stats.go:139`。
- **杂项**：工单状态/申请状态魔法数字（缺 model 常量）；`middleware/rate_limit.go:19` 手拼 `rl:` key（应走 redispkg.Key）；`middleware/auth.go:14-18` 角色常量重复 model；~10 处 `_ = m.audit.write` 静默丢失败；`node_service.go:250` rate≤0 静默计费 0；context key 字符串（auth 两处）；`payment.go:37-40` QueryResult 缺 Amount（对账无法核对金额）；`resp.go:78` 硬编码 40000；`admin_ticket.go:28` Message Chain `m.withdraw.audit.emailsOf`；`node_service.go:167,198` 为取 SubToken 拷全量 `[]model.User`；测试错置（sanitize/coupon 测试在 admin_service_test、userRows 重复、TestServerListWithoutPlan 错文件）。

## Plan

1. repo 抽 `paginate` helper 收敛 11 处；手写 contains/min 换 stdlib；settings-int 单 helper 单访问式；sort 三合一；GB→bytes 入 model 小函数。
2. 缺口 repo 方法补齐（CouponRepo.List/Update、Notice/Knowledge ListAll、Overview stats 查询组、ClearTelegramID、MarkWithdrawLogStatus），service 全部经注入 seam 调用。
3. commissionRateFor/settings() 走 SettingService 缓存路径。
4. N+1 改批量查询（照 emailsOf 形状）。
5. 杂项按清单逐条（常量入 model、key 走 owner、audit 失败记 log、QueryResult 补 Amount 并在 service 比对、魔法数字常量化、测试归位）。

分多个小 PR，按上述 5 组各一个。

## Acceptance

- `go test ./...` 全绿；新增 paginate/settings helper 有单测。
- service 层 `grep -n "\.Table(\|db\.\|gorm\." server/internal/service/*.go` 仅剩有注释依据的豁免点。
- Overview/订单列表 SQL 数量断言（sqlmock 期望语句数不随行数增长）。

## Comments
