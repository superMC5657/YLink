# Code Review — YLink v0.4.0 (CI/CD, Build Config & Logging)

- **Version:** 0.4.0
- **Date:** 2026-08-12
- **Scope:** Incremental — commits since the v0.3.0 review (`4a55db1..1e001e0`): GitHub Actions CI/CD pipeline (frontend-quality / frontend-e2e / rust check + Tauri release), vite build config restructure (`index.html` → `src/`, `root`/`publicDir`/`envDir`), GORM logger ANSI-color fix, updater plugin integration
- **Method:** Reviewer-model pass over the recent commits with local verification (vite build with the new root/publicDir/envDir config, dev server + mock, unit tests 43/43, typecheck, lint, and cargo check all pass; `pnpm format:check` fails only on `scripts/build-latest-json.mjs`)
- **Status:** Resolved — 1 P1 + 1 P2, both fixed (see strikethrough items below)

## Summary

The runtime changes verify locally (vite build with the new root/publicDir/envDir config, dev server + mock, unit tests 43/43, typecheck, lint, and cargo check all pass), but the patch adds a CI pipeline whose `format:check` step fails immediately on the newly added `scripts/build-latest-json.mjs`, blocking the PR's main deliverable until that file is reformatted. Both findings have since been fixed.

## Findings

### ~~[P1] New CI `format:check` job fails on the unformatted `scripts/build-latest-json.mjs` — scripts/build-latest-json.mjs:35-38~~

~~The new `frontend-quality` job added in `.github/workflows/ci.yml` (lines 44-46) runs `pnpm format:check`, but the newly added `scripts/build-latest-json.mjs` is not Prettier-clean: lines 35, 38, 42 and 85 exceed the repo's `printWidth: 100` and need wrapping. Verified locally with `pnpm format:check` — it fails only on this file (`Code style issues found in the above file`), so every push/PR to main will get a red CI until the file is reformatted with `pnpm format`.~~

**Resolved** — `scripts/build-latest-json.mjs` reformatted with Prettier; `pnpm format:check` now passes repo-wide.

### ~~[P2] Restore `IgnoreRecordNotFoundError` to avoid new error-log spam — server/internal/repo/repo.go:21-25~~

~~`newLogger` replaces `gormlogger.Default.LogMode(Warn)` with a hand-built config that sets `IgnoreRecordNotFoundError: false`, while GORM's `Default` logger uses `true`. With `LogLevel: Warn`, every `gorm.ErrRecordNotFound` (e.g., normal "not found → 404" lookups in detail endpoints) will now be logged at Error level — a behavior change beyond the comment's stated goal of just disabling ANSI colors. If the intent was only `Colorful: false`, set `IgnoreRecordNotFoundError: true` to preserve the previous logging behavior.~~

**Resolved** — set `IgnoreRecordNotFoundError: true` in `newLogger` (documented in a comment as keeping GORM `Default` behavior), so only ANSI colors are turned off and normal not-found lookups stay silent.

## Verification

- `pnpm format:check` — all matched files use Prettier code style (exit 0)
- `go build ./...`, `go vet ./internal/repo/`, `gofmt -l` — pass