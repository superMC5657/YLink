# ADR-0004 · 分页钳制 single-owner（ADR-0001 的对称收敛）

> 状态：已接受（2026-10-12，dev-ica3）
> 相关：ADR-0001、`server/internal/pkg/page/`、`server/internal/handler/helper.go`

## 背景

分页钳制无归属：10 处 service List 手写 `page<1→1/pageSize 越界→10`，
2 处 repo（审计日志/重置记录）另写 `100/20` 分叉规则，4 处用户端 handler
手 parse 不钳制（响应 envelope 甚至回显未钳制值）。改上限要动 N 处；
helper.go 注释还明示“钳制仍由 service 逐 List 负责”。

## 决定

1. **唯一 owner 为 `internal/pkg/page`**：缺省与上限成对（`Standard{10,50}` /
   `Audit{20,100}`），`PageParams` 解析即钳制（标准档），`PageParamsAudit`
   覆盖审计类两端点；service/repo 逐 List 手写规则全删，只透传。
2. **审计类 100/20 保留而非拉平**：swag 注解、管理端审计页 `page-sizes [20,50,100]`
   三处同指翻阅场景，拉平是行为变更；缺省/上限成对进表后，分叉以显式档存在，
   不再是散落数字。契约 §1.3 补 carve-out 一行。
3. helper.go 旧注释（“钳制仍由 service 逐 List 负责”）同步改写；本 ADR 与
   ADR-0001 互补：PageOK 拥有 nil→[]，本包拥有缺省/越界回落。

## 后果

- 改上限只动 `pkg/page` 一处（前端 `withPage` 薄适配的同值常量须同步，见其头注）。
- 新 List 端点：handler 走 `pageParams` 即自动钳制；审计类走 `pageParamsAudit`；
  service/repo 不再写钳制，测试只测 owner 与缝。
