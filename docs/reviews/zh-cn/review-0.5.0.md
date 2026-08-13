# 代码评审 — YLink v0.5.0（管理端订单佣金数据）

- **版本：** 0.5.0
- **日期：** 2026-08-13
- **范围：** 管理端订单列表新增 `commission_amount` 的近期变更。
- **方法：** 评审模型审查；前端类型检查、Vitest（43 项）及 server Go 测试均通过。
- **状态：** 全部发现已解决（P2 已于 2026-08-13 修复）。

## 摘要

本次主要行为变更已实现，现有验证均通过。P2 发现（批量查询佣金失败被静默忽略）已修复：`ListOrders` 现在会透传批量佣金查询的错误，管理端订单接口不再在查询失败时返回 `commission_amount` 全为 null 的"成功"响应。

## 已完成

~~管理端订单列表已按订单号批量加载佣金记录，并增加 `commission_amount` 字段。~~
~~透传佣金查询失败（P2）— `ListOrders` 改为返回 `ListByOrderNos` 的错误而非静默忽略；新增回归测试 `TestAdminListOrdersCommissionQueryError`。~~

## 发现

### [P2] 透传佣金查询失败 — server/internal/service/admin_service.go:196-197

~~`ListOrders` 忽略了批量佣金查询的错误：~~

```go
if comms, err := s.repos.Commission.ListByOrderNos(s.db, orderNosOf(list)); err == nil {
```

~~数据库或查询失败时，接口仍使用空佣金映射返回订单列表。应直接返回该查询错误，避免管理端 UI 将错误的财务数据当作成功结果展示。~~

**状态：** ✅ 已解决（2026-08-13）。`ListOrders` 现在直接返回批量佣金查询的错误，删除了原先 `err == nil` 的静默吞错；由 `TestAdminListOrdersCommissionQueryError` 覆盖。

## 验证

- 前端类型检查 — 通过
- Vitest — 43 项通过
- server Go 测试 — 通过（测试函数 67 → 68，含新增 P2 回归测试）
