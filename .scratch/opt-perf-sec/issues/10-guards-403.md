# [P1] 10 路由守卫403

Status: done
Type: task

## Finding

- `guards.ts` 同步读 persist,`auth.restore()` 只恢复 token 不恢复 user,刷新后 isAdmin=false 致管理员进 /portal|/admin 闪跳回 dashboard;前端隐藏非安全边界新人易误解。

## Task

1. 启动先 `fetchUser` 补 role 再放行 admin;加 `/403` 页替代静默回 dashboard。
2. E2E 加 普通token调 /admin 接口→403 回归;文档注明 admin_path 仅 obscurity。

## Done

- 刷新不闪跳;403 可达;E2E 通过。

## Comments

- 结论修正:ticket 07 重构持久面后 `restore()` 已恢复 user(含 role),刷新闪跳的主因已消除;本单补的是「持久化 user 过期」场景(如被降权的管理员仍凭旧 role 进后台)。
- 契约先行:`/user/profile` GET/PUT 响应补 `role` 字段(model.UserProfileResp + service 两处 + docs/api §5.2 + ProfileResp 类型 + mock),后端 user 对象本就取回,改动极小。
- 守卫:进 meta.admin 路由前 `auth.ensureRole()` 向服务端核实 role(每会话一次,失败回退持久化 role、后端 403 兜底);非管理员访问 /admin/* 进新 `/403` 页,门户分流页保持仪表板兜底(A6 语义不变)。核实途中会话失效则回登录页。
- mock 对齐后端:管理接口对非管理员返回 40300(原来误用 40100);mock refresh 对管理员 refresh 续签 admin 前缀 access(否则静默刷新会把管理员降级)。
- E2E:admin 页 403 断言改写 + 新增「普通 token 调 /admin/stat/overview → 40300」接口级回归;全量 58/58 通过。admin_path 仅 obscurity 注记写入 docs/backend/deploy.md §1.1。
