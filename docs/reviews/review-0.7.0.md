# 代码评审 — YLink v0.7.0（MySQL → PostgreSQL 16 与服务端 dev/release 环境拆分）

- **版本：** 0.7.0
- **日期：** 2026-08-13
- **提交：** `6f9b8d5`（feat(database)：迁移 MySQL 8 至 PostgreSQL 16，dev.sh 合并启停并同步文档）、`8a62f74`（ci(server)：区分 server dev/release 环境）
- **范围：** PostgreSQL 16 驱动/迁移切换、`scripts/dev.sh` 合并、服务端环境文件拆分（`.env.dev` / `.env.release`）、文档同步。
- **方法：** 人工审查两个 commit 的 diff；重跑 `go test ./...`（60/60 全绿）；核对 `docker compose` env-file 行为与 `docker compose config` 解析。
- **状态：** 报告 2 项 P2、3 项 P3；均已修复（2026-08-13）。

## 摘要

后端从 MySQL 8 迁移至 PostgreSQL 16：GORM 驱动、`migrations/*.sql`、全部业务 SQL（布尔字面量、标识符、日期运算、JSONB）、docker-compose 栈与开发脚本均做了统一转换，sqlmock 单测同步切到新方言并通过。后续 commit 将单一 `server/.env` 拆分为 `server/.env.dev` / `server/.env.release`（均 gitignore），并让 compose 通过 `ENV_FILE` 选择环境文件。Go 代码与迁移 SQL 一致且已验证；剩余问题集中在发布配置与部署文档，另有 2 处脚本/文档小缺陷。

## 已完成

GORM 驱动由 `gorm.io/driver/mysql` 切换为 `gorm.io/driver/postgres`，`migrations/*.sql` 全部重写为 PostgreSQL 语法（BIGSERIAL、BOOLEAN、TIMESTAMP(3)、JSONB、COMMENT ON、`setval` 推进序列）。

业务 SQL 全部转换（反引号标识符、`= 1/0` 布尔字面量、`DATE_SUB`/`INTERVAL`、`DATE()`、`type:json`/`mediumtext` → PG 等价写法）；`repo.go` 改用 `postgres.Open(cfg.DSN)`。

docker-compose 的 mysql 服务替换为 `postgres:16-alpine`（端口 5433、`command: -p 5433`、`pg_isready` 健康检查），Redis 暴露到 `127.0.0.1:6379`，Makefile 迁移 tag 改为 `postgres`。

`dev-up.sh`/`dev-down.sh` 合并为 `scripts/dev.sh`（启动 / `-stop`、旧 `docker run` 容器检测拦截、一次性迁移检查、宿主机进程 api/worker 并导出 DSN）。

`server/.env` 拆分为 `.env.dev`/`.env.release`，两者加入 `.gitignore`，`dev.sh` 与 compose `env_file` 切换为 `${ENV_FILE:-.env.dev}`。

单测同步到 PG 方言（双引号标识符、`$n`、INSERT … RETURNING）；`go test ./...` 60/60 全绿。

## 发现

### ✅ [P2] 发布环境模板仍为 `APP_ENV=development`，生产会开启 Swagger/debug — server/.env.example:6、docs/backend/deploy.md:112

新的发布流程（`cp .env.example .env.release` + `ENV_FILE=.env.release docker compose up -d`）复制出的模板 `APP_ENV=development`。`router.New` 仅在 `app.env == "production"` 时才切换 `gin.ReleaseMode` 并关闭 `/swagger/*`，因此按文档部署的生产环境会以开发模式运行，Swagger 经 Caddy 公网可达——与 deploy.md §7「关 Swagger/debug」以及 progress.md 中「APP_ENV 区分两套环境」的表述相矛盾。模板头部或 deploy.md 步骤 1 应把 release 文件的 `APP_ENV` 默认/说明为 `production`。

**状态：** 已修复（2026-08-13）。`server/.env.example` 默认值改为 `APP_ENV=production`（头部注释说明两种用法）；`scripts/dev.sh` 对宿主机 api/worker 强制 `export APP_ENV=development`（本地联调保留 Swagger/debug）；deploy.md §4 步骤 1 同步说明默认值。

### ✅ [P2] 生产上线步骤在新主机上先迁移、后启动数据库 — docs/backend/deploy.md:112-113

deploy.md §4 步骤 1 在「首次启动前」执行 `DB_URL='postgres://…@127.0.0.1:5433/…' make migrate`，但 Postgres 容器要到步骤 2（`ENV_FILE=.env.release docker compose up -d`）才启动。新主机上步骤 1 时 `127.0.0.1:5433` 无任何监听，迁移必然失败；api/worker 容器不会自动迁移，于是服务在空库上启动（`EnsureAdmin`/`EnsureDemoUser` 静默跳过，业务接口 500）。上线步骤应先拉起 `postgres`/`redis` 再执行迁移（或写明预期顺序）。

**状态：** 已修复（2026-08-13）。deploy.md §4 重排：先拉起 `postgres`/`redis`（步骤 2），再对已监听的 `127.0.0.1:5433` 执行 `make migrate`（步骤 3），最后启动全部服务（步骤 4）。

### ✅ [P3] `server/.env.dev` 缺失时 dev.sh 的「默认值兜底」不生效 — scripts/dev.sh:28-31,106,53

脚本注释称「.env.dev 缺失时用默认值兜底,保证脚本可运行」，但 `docker compose --env-file "$ENV_FILE"` 在文件不存在时直接报 `couldn't find env file: …`（compose v2 实测），因此全新检出后直接 `bash scripts/dev.sh` 会在 compose 步骤失败，`-stop` 也无法停止容器。应在调用 compose 前检查文件并回退默认值。

**状态：** 已修复（2026-08-13）。dev.sh 在 `$ENV_FILE` 缺失时生成 `$RUN_DIR/env.fallback`（写入默认基础设施变量）并让 compose 的 `--env-file` 指向它，启动与 `-stop` 两条路径均生效。

### ✅ [P3] `dev.sh -stop` 会停止 `ylink` compose 项目全部服务 — scripts/dev.sh:53

`docker compose --env-file "$ENV_FILE" stop` 未指定服务参数，除 `postgres`/`redis` 外还会连 `api`、`worker`、`caddy` 一起停。若同一主机上运行着发布栈（同一 compose 项目与容器名），`dev.sh -stop` 会将其一并关闭。限定为 `stop postgres redis` 更符合脚本所述范围。

**状态：** 已修复（2026-08-13）。停止命令改为 `docker compose --env-file "$ENV_FILE" stop postgres redis`。

### ✅ [P3] docs/README.md 架构图错行 — docs/README.md:29

ASCII 图在本次编辑中被改乱：Redis 行与「拉取节点」行合并成 `│  ├── Redis（验证码/限流/会话） │┌────────────┐`，破坏了框线排版。应拆回两行。

**状态：** 已修复（2026-08-13）。合并行已拆回 Redis 行与「拉取节点」两行，框线对齐恢复。

## 验证

- `server/` 下 `go test ./...`：60/60 全绿（sqlmock 期望已切到 PG 方言）。
- `docker compose --env-file .env.example config` 解析正常；`env_file: ${ENV_FILE:-.env.dev}` 默认值按预期解析。
- 迁移 SQL 无残留 MySQL 语法（`ENGINE`/`AUTO_INCREMENT`/`TINYINT`/反引号）；`0002`/`0003` 已转为 PG `CHECK`/`SMALLINT`；`0001_init.down.sql` 仍然有效。
- Go 源码中未发现残留 MySQL 专属 SQL（`DATE_SUB`、`DATE()`、`INTERVAL`、`IFNULL`、反引号标识符）。
