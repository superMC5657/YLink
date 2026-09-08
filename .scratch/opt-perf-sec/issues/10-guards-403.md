# [P1] 10 路由守卫403

Status: ready-for-agent
Type: task

## Finding

- `guards.ts` 同步读 persist，`auth.restore()` 只恢复 token 不恢复 user，刷新后 isAdmin=false 致管理员进 /portal|/admin 闪跳回 dashboard；前端隐藏非安全边界新人易误解。

## Task

1. 启动先 `fetchUser` 补 role 再放行 admin；加 `/403` 页替代静默回 dashboard。
2. E2E 加 普通token调 /admin 接口→403 回归；文档注明 admin_path 仅 obscurity。

## Done

- 刷新不闪跳；403 可达；E2E 通过。

## Comments
