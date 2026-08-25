# [P2] Limit reports to users in the authenticated node's group

Status: resolved
Type: task

## Finding

有效节点密钥可上报任何活跃未封禁用户的 UUID，未校验用户套餐 `group_ids` 是否包含节点分组。

## Resolution

`Report` 先加载节点分组允许的套餐集合，只接受 `group_ids` 含本节点分组的用户；不匹配返回 `not_subscribed`。

## Comments

2026-08-25: 已修复并补 `TestNodeReportGroupMismatch`。
