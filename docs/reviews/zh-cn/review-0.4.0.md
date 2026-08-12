# 代码评审 — proxy-seller-web v0.4.0（CI/CD、构建配置与日志）

- **版本：** 0.4.0
- **日期：** 2026-08-12
- **范围：** 增量 — v0.3.0 评审之后的提交（`4a55db1..1e001e0`）：GitHub Actions CI/CD 流水线（frontend-quality / frontend-e2e / rust check + Tauri Release）、vite 构建配置重构（`index.html` → `src/`，`root`/`publicDir`/`envDir`）、GORM 日志器关闭 ANSI 颜色、updater 插件接入
- **方法：** 评审模型对上述提交的整体评审；本地验证：新 root/publicDir/envDir 配置下 vite 构建通过、dev server + mock 可用、单测 43/43、typecheck、lint、cargo check 全部通过；`pnpm format:check` 仅在新文件 `scripts/build-latest-json.mjs` 上失败
- **状态：** 已解决 — 1 项 P1、1 项 P2 均已修复（见下方删除线条目）

## 摘要

运行时改动本地验证通过（新 root/publicDir/envDir 配置下 vite 构建、dev server + mock、单测 43/43、typecheck、lint、cargo check 全部通过），但本次改动新增的 CI 流水线中 `format:check` 步骤会立刻在新文件 `scripts/build-latest-json.mjs` 上失败，在重新格式化该文件之前会阻断 PR 的主要交付。两项发现均已修复。

## 发现

### ~~[P1] 新 CI `format:check` job 在未格式化的 `scripts/build-latest-json.mjs` 上失败 — scripts/build-latest-json.mjs:35-38~~

~~`.github/workflows/ci.yml` 新增的 `frontend-quality` job（第 44-46 行）会运行 `pnpm format:check`，但新加的 `scripts/build-latest-json.mjs` 不符合 Prettier 规范：第 35、38、42、85 行超过仓库 `printWidth: 100` 需要换行。本地用 `pnpm format:check` 验证——仅此文件失败（`Code style issues found in the above file`），因此每次 push/PR 到 main 都会红 CI，直到用 `pnpm format` 重新格式化该文件。~~

**已修复** — 已用 Prettier 重新格式化 `scripts/build-latest-json.mjs`；`pnpm format:check` 现在全仓库通过。

### ~~[P2] 恢复 `IgnoreRecordNotFoundError`，避免新增错误日志刷屏 — server/internal/repo/repo.go:21-25~~

~~`newLogger` 用自建配置替换了 `gormlogger.Default.LogMode(Warn)`，其中 `IgnoreRecordNotFoundError: false`，而 GORM 的 `Default` 日志器为 `true`。在 `LogLevel: Warn` 下，每次 `gorm.ErrRecordNotFound`（例如详情端点的正常「未找到 → 404」查询）都会被按 Error 级别记录——这超出了注释所述「仅关闭 ANSI 颜色」的目标。如果意图只是 `Colorful: false`，应设置 `IgnoreRecordNotFoundError: true` 以保留原有日志行为。~~

**已修复** — `newLogger` 中设置 `IgnoreRecordNotFoundError: true`（注释说明保持 GORM `Default` 行为），仅关闭 ANSI 颜色，正常的「未找到」查询不再刷 Error 日志。

## 验证

- `pnpm format:check` — 所有匹配文件均符合 Prettier 规范（exit 0）
- `go build ./...`、`go vet ./internal/repo/`、`gofmt -l` — 全部通过