# Code Review — YLink v0.7.0 (MySQL → PostgreSQL 16 & Server Dev/Release Split)

- **Version:** 0.7.0
- **Date:** 2026-08-13
- **Commits:** `6f9b8d5` (feat(database): migrate MySQL 8 to PostgreSQL 16, merge dev.sh start/stop, sync docs), `8a62f74` (ci(server): separate server dev/release environments)
- **Scope:** PostgreSQL 16 driver/migration switch, `scripts/dev.sh` merge, server env-file split (`.env.dev` / `.env.release`), documentation sync.
- **Method:** Manual review of both commit diffs; `go test ./...` re-run (60/60 green); `docker compose` env-file behavior and `docker compose config` parsing checked.
- **Status:** 2× P2 and 3× P3 findings reported; all fixed (2026-08-13).

## Summary

The backend was migrated from MySQL 8 to PostgreSQL 16: the GORM driver, `migrations/*.sql`, all business SQL (boolean literals, identifiers, date arithmetic, JSONB), the docker-compose stack, and the dev script were converted consistently, and the sqlmock-based unit tests were updated to the new dialect and pass. The follow-up commit split the single `server/.env` into `server/.env.dev` / `server/.env.release` (both gitignored) and made compose select the env file via `ENV_FILE`. The Go code and migration SQL are consistent and verified. The remaining issues are in the release configuration and the deployment runbook, plus two small script/doc defects.

## Completed

Migrated GORM from `gorm.io/driver/mysql` to `gorm.io/driver/postgres` and rewrote `migrations/*.sql` to PostgreSQL syntax (BIGSERIAL, BOOLEAN, TIMESTAMP(3), JSONB, COMMENT ON, `setval` sequence bump).

Converted all business SQL (backtick identifiers, `= 1/0` boolean literals, `DATE_SUB`/`INTERVAL`, `DATE()`, `type:json`/`mediumtext`) to PostgreSQL equivalents; `repo.go` opens `postgres.Open(cfg.DSN)`.

Replaced the MySQL compose service with `postgres:16-alpine` (port 5433, `command: -p 5433`, `pg_isready` healthcheck), exposed Redis on `127.0.0.1:6379`, and updated the Makefile migrate tags to `postgres`.

Merged `dev-up.sh`/`dev-down.sh` into `scripts/dev.sh` (start / `-stop`, legacy `docker run` container guard, one-shot migration check, host-process api/worker with exported DSN).

Split `server/.env` into `.env.dev`/`.env.release`, added both to `.gitignore`, and switched `dev.sh` + compose `env_file` to `${ENV_FILE:-.env.dev}`.

Updated unit tests to the PG dialect (double-quoted identifiers, `$n`, INSERT … RETURNING); `go test ./...` 60/60 green.

## Findings

### ✅ [P2] Release env template still sets `APP_ENV=development` — Swagger/debug stay enabled in production — server/.env.example:6, docs/backend/deploy.md:112

The new release flow (`cp .env.example .env.release` + `ENV_FILE=.env.release docker compose up -d`) copies a template whose `APP_ENV=development`. `router.New` only switches to `gin.ReleaseMode` and disables `/swagger/*` when `app.env == "production"`, so a production deployment following the docs runs in development mode with Swagger publicly reachable through Caddy — contradicting deploy.md §7 ("关 Swagger/debug") and progress.md's claim that `APP_ENV` distinguishes the two environments. The template header or deploy.md step 1 should default/document `APP_ENV=production` for the release file.

**Status:** Fixed (2026-08-13). `server/.env.example` now defaults `APP_ENV=production` (header note explains both usages); `scripts/dev.sh` forces `export APP_ENV=development` for the host-process api/worker so local dev keeps Swagger/debug; deploy.md §4 step 1 documents the default.

### ✅ [P2] Production runbook runs the migration before Postgres exists on a fresh host — docs/backend/deploy.md:112-113

deploy.md §4 step 1 runs `DB_URL='postgres://…@127.0.0.1:5433/…' make migrate` "首次启动前", but the Postgres container is only started in step 2 (`ENV_FILE=.env.release docker compose up -d`). On a fresh host nothing listens on `127.0.0.1:5433` at step 1, so the migration fails; the api/worker containers do not auto-migrate, so the stack starts against an empty schema (`EnsureAdmin`/`EnsureDemoUser` silently skip and business endpoints 500). The runbook should bring up `postgres`/`redis` first and only then run the migration (or document the intended order).

**Status:** Fixed (2026-08-13). deploy.md §4 reordered: bring up `postgres`/`redis` first (step 2), run `make migrate` against the listening `127.0.0.1:5433` (step 3), then start the full stack (step 4).

### ✅ [P3] dev.sh "default fallback" does not work when `server/.env.dev` is missing — scripts/dev.sh:28-31,106,53

The script states ".env.dev 缺失时用默认值兜底,保证脚本可运行", but `docker compose --env-file "$ENV_FILE"` exits with `couldn't find env file: …` when the file is absent (verified with compose v2), so `bash scripts/dev.sh` on a fresh checkout fails at the compose step, and `-stop` cannot stop the containers either. The script should check for the file and fall back to defaults before invoking compose.

**Status:** Fixed (2026-08-13). dev.sh now generates `$RUN_DIR/env.fallback` with the default infra variables when `$ENV_FILE` is missing and points compose's `--env-file` at it — both the start and `-stop` paths.

### ✅ [P3] `dev.sh -stop` stops every service in the `ylink` compose project — scripts/dev.sh:53

`docker compose --env-file "$ENV_FILE" stop` is called without service arguments, so it also stops `api`, `worker`, and `caddy` besides `postgres`/`redis`. If a release stack (same compose project and container names) is running on the host, `dev.sh -stop` will shut it down as well. Restricting the command to `stop postgres redis` would match the script's stated scope.

**Status:** Fixed (2026-08-13). The stop command is now `docker compose --env-file "$ENV_FILE" stop postgres redis`.

### ✅ [P3] docs/README.md architecture diagram corrupted — docs/README.md:29

The ASCII diagram was mangled during the edit: the Redis line and the "拉取节点" line were merged into `│  ├── Redis（验证码/限流/会话） │┌────────────┐`, breaking the box-drawing layout. It should be split back into two lines.

**Status:** Fixed (2026-08-13). The merged line was split back into the Redis line and the "拉取节点" line; box-drawing alignment restored.

## Verification

- `go test ./...` in `server/`: 60/60 green (sqlmock expectations updated to the PG dialect).
- `docker compose --env-file .env.example config` parses correctly; the `env_file: ${ENV_FILE:-.env.dev}` default resolves as intended.
- Migration SQL: no leftover MySQL syntax (`ENGINE`/`AUTO_INCREMENT`/`TINYINT`/backticks); `0002`/`0003` converted to PG `CHECK`/`SMALLINT`; `0001_init.down.sql` still valid.
- No remaining MySQL-specific SQL (`DATE_SUB`, `DATE()`, `INTERVAL`, `IFNULL`, backtick identifiers) found in Go sources.
