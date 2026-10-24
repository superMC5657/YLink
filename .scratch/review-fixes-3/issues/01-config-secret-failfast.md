# [P0] 生产配置不得静默使用开发密钥

Status: ready-for-agent
Type: task

## Finding

- `server/configs/config.yaml:10,16,20` 把开发密钥作为静默默认值提交：`password=ylink_root`、`ylink_redis`、jwt `dev-only-secret-…`。
- `server/internal/config/config.go` 的 Load 无生产 fail-fast：生产部署漏配 `APP_JWT_SECRET` 时，服务用公开在仓库里的密钥签发 JWT——任何人可伪造任意用户/管理员的 token。DB/Redis 密码同理。
- `server/cmd/worker/main.go:86`：`zap.String("tz", cfg.App.Name)` 把应用名记到 tz 键。

## Plan

1. config.yaml 敏感值改为 `${APP_JWT_SECRET}` 式占位（deploy.md §1 自己的示例即是此写法）；非敏感默认值保留。
2. `config.Load` 增加 `validateProd()`：`App.Env == production`（按现有环境标识字段）时，检测 JWT secret / DB 密码 / Redis 密码等于已知开发默认值（`dev-only-secret-*`、`ylink_root`、`ylink_redis`）或为空 → 返回 error 拒绝启动，错误信息指明对应环境变量名。
3. 补测试：生产模式下三种已知开发值分别拒绝启动；开发模式不受影响。
4. worker 日志键改 `zap.String("app", cfg.App.Name)`。

## Acceptance

- 生产模式 + 开发默认密钥 → 进程启动失败并指明缺失的环境变量。
- `go test ./internal/config/...` 覆盖上述分支；`go test ./...` 全绿。
- deploy.md/launch-checklist.md 无需改动（已要求 env 注入），若有描述「config.yaml 默认密钥可直接生产」的语句一并清除。

## Comments
