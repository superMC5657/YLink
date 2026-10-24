# [P0] 生产配置不得静默使用开发密钥

Status: resolved
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

- 2026-10-25: 已修复——config.yaml 三类敏感值改 `${APP_*}` 占位；`config.Load` 增 `validateProd()`（生产模式下 JWT/DB/Redis 密码为空、`${...}` 占位符或已知开发默认值 `dev-only-secret-*`/`please-change-me*`/`ylink_root`/`ylink_redis` 一律拒绝启动并指明环境变量名）；worker 日志键 `tz`→`app`；补 `internal/config` 测试（9 个拒绝分支 + 开发放行 + Load 集成 + DSN 口令提取）。`go build/vet/test ./...` 全绿。
- 2026-10-25: 实现中发现并一并修复（fail-fast 的前置条件）：viper 把 `app.env` 映射为 `APP_APP_ENV`，而 `.env.example`/dev 脚本/compose 用的 `APP_ENV` 从未生效——生产容器实际一直按 development 运行（Swagger 常开、本次校验也永远不会触发）。已在 `Load` 中 `BindEnv("app.env","APP_ENV")` 显式绑定，`APP_APP_ENV` 长名仍兼容。
- 2026-10-25: 范围外新问题（未修）：`app.base_url` 存在同样的短名失配——`.env` 模板的 `APP_BASE_URL` 映射不到 `app.base_url`（实际生效名 `APP_APP_BASE_URL`），生产部署若只设短名，订阅链接/支付回调会拼接成 config.yaml 的默认值。建议后续 ticket 处理（同法 `BindEnv` 或模板改长名）。
- 2026-10-25: 行为变化提示：config.yaml 不再内置可用的开发 DSN/密码，裸 `make run`/`go run` 需先导出 `APP_DATABASE_DSN` 等变量；`scripts/dev.sh`（自动导出全部变量）仍是本地一键路径。`docs/backend/progress.md` 基础设施表的 DSN 提示已同步。
