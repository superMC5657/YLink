# Code Review — YLink v0.9.0 (Legacy Node Credentials & Node Reporting / Alerting Fixes)

- **Date:** 2026-08-25
- **Scope:** Review findings on subscription credential rollout, node reporting validation/dedup, demo agent baseline, Alertmanager SMTP delivery, and Windows NSIS packaging.
- **Status:** All 3×P1 and 4×P2 findings fixed.

## Findings

### ✅ [P1] Keep subscription credentials valid until node inbounds are provisioned

Node subscription generation now only uses `users.uuid` when the server config explicitly enables `per_user_credentials: true`. Legacy nodes continue to receive the shared `servers.config` credential, so clients do not lose connectivity on the next subscription refresh before inbound provisioning is completed.

### ✅ [P1] Use a STARTTLS-capable SMTP endpoint for Alertmanager

Alertmanager now defaults to `ALERT_SMTP_PORT=587` (STARTTLS) instead of reusing the backend QQ 465 implicit-TLS port. Operators who must use port 465 are directed to place an SMTPS-to-STARTTLS relay in front of Alertmanager, or override `ALERT_SMTP_HOST/FROM`.

### ✅ [P1] Add the configured NSIS hook file

`src-tauri/nsis/installer-hooks.nsh` is now committed with the four no-op NSIS hook macros, so the `installerHooks` path in `tauri.conf.json` resolves during Windows NSIS builds.

### ✅ [P2] Limit reports to users in the authenticated node's group

`POST /node/report` now checks each user's plan `group_ids` against the authenticated server's group. Mismatches are rejected with `not_subscribed`.

### ✅ [P2] Reject duplicate UUIDs in a report payload

Duplicate UUIDs in one request are rejected as a group before user lookup or transaction work, returning `duplicate_uuid` for every occurrence.

### ✅ [P2] Send the zero baseline before advancing demo counters

`node-agent` now reports the current counters first (zero on startup) to establish the snapshot baseline, then advances randomized cumulative counters for the next round.

### ✅ [P2] Escape SMTP values before inserting them with sed

The Alertmanager entrypoint now substitutes placeholders with AWK and escapes YAML single quotes (`'` → `''`), preventing valid credentials from corrupting `alertmanager.yml`.

## Changes

- Backend: `subscribe_service.go`, `node_service.go`, `node_service_test.go`, `subscribe_service_test.go`, `cmd/node-agent/main.go`.
- Observability: `docker-compose.yml`, `alertmanager.yml.tmpl`, `server/.env.example`.
- Desktop: `src-tauri/nsis/installer-hooks.nsh` (new).
- Docs: API contract, backend core flows/data model/deploy/progress/checklist, frontend Tauri docs, this review record, and `.scratch/review-fixes/`.

## Verification

- `go test ./internal/service/... -count=1` passes.
- `gofmt -l` is clean for changed Go files.
- Docker Compose config parsing and remaining repo checks were run where available.
