# Code Review — YLink v0.8.0 (Mobile Share Panel & Ticket Reply Notification)

- **Version:** 0.8.0
- **Date:** 2026-08-14
- **Scope:** New mobile-first share panel (`SharePanel.vue`) wired into the invite page, and an enhancement to the "ticket replied" desktop local notification (immediate check on window focus/visibility).
- **Method:** Reviewer-model review of the diff; `pnpm typecheck`, `pnpm lint`, `pnpm format:check`, and Vitest (54/54 green, +3 new SharePanel cases) all passed.
- **Status:** All findings resolved (2026-08-14).

## Summary

The frontend gap "mobile share panel" from the phase-2 backlog was implemented: a reusable bottom-sheet share panel (`SharePanel.vue`, `n-drawer placement="bottom"`, mobile-first) that renders a QR code (existing `qrcode` dep, theme-aware colors like `PaymentModal`), shows the shareable link, and offers copy-link plus system share (`navigator.share`, capability-gated and hidden on Tauri). The invite page got a "share" button that opens the panel with the register link (prefix + first invite code). The ticket-reply local notification (already implemented via `useLocalNotifications.checkTickets` + 60 s polling in `MainLayout`) was enhanced so `onFocus`/`onVisibility` also run `checkTickets()`, removing up to 60 s of latency when the user returns to the window. Documentation was synced (frontend/backend progress, backlog item moved to done).

## Changes

- `src/components/business/SharePanel.vue` (new): props `show` / `title` / `text` / optional `desc`, v-model `update:show`; QR generation watches `show`/`text`/retry tick; failure state with retry; copy via `copyText`; system share only when `navigator.share` exists.
- `src/views/invite/InviteView.vue`: "share" button in the register-link row → opens `SharePanel`; missing-code guard uses i18n key `invite.needCode`.
- `src/components/ui/AppIcon.vue`: added `share-2` (Lucide share-2 paths).
- `src/locales/{zh-CN,en-US}.ts`: added `share.*` section + `invite.shareLink/shareDesc/needCode`.
- `src/layouts/MainLayout.vue`: `onFocus`/`onVisibility` now also call `checkTickets()`.
- `src/components/business/__tests__/SharePanel.spec.ts` (new): 3 cases (renders title/link/actions, copy → `copyText` + success message, close → `update:show(false)`).

## Findings

### ✅ [P3] QR-code failure state shared the "loading" placeholder — SharePanel.vue
If `QRCode.toDataURL` threw (e.g. no canvas), the panel showed "loading" forever with no recovery. Fixed: separate `qrFailed` state with a "retry" button that re-triggers generation via a retry tick.

### ✅ [P3] QR code was not rebuilt when `text` changed while open
The watch only tracked `show`, so a stale QR could remain if the share text changed. Fixed: watch `[show, text, retryTick]` and clear on empty.

### ✅ [P3] Hard-coded Chinese warning in invite page
`message.warning('请先生成邀请码')` bypassed i18n. Fixed: `t('invite.needCode')` (zh/en added).

## Verification

- `pnpm test`: 54/54 green (9 files; 51 pre-existing + 3 new SharePanel cases).
- `pnpm typecheck`: pass; `pnpm lint`: 0 errors; `pnpm format:check`: all files Prettier-clean.

---

## Round 2 (2026-08-14): Register-link prefix fix, card-style panel, QR-image system share

Follow-up from usage: the register link was generated with the backend API address (`http://localhost:8081/register?code=…`) because the server builds `register_url_prefix` from `cfg.App.BaseURL`, which is the API base — not the front-end origin. Also requested: card-style QR presentation and system share of the QR image.

### Changes

- `src/stores/invite.ts`: new getter `effectiveRegisterUrlPrefix` — (1) `VITE_WEB_BASE_URL` injected at build time (production / Tauri packaging, see `.env.production`), (2) else `window.location.origin` (auto-distinguishes local Vite dev `:5174`, Caddy `:80`, production HTTPS `:443`), (3) else relative `/register?code=` (only reachable on Tauri without injection; avoids leaking the API `8081` address).
- `src/views/invite/InviteView.vue`: display/copy/share/guard all switch to the effective prefix; passes `:code` to the panel.
- `src/components/business/SharePanel.vue`: redesigned as a brand invite card (brand gradient + `siteName` + invite code + QR on a white tile, QR fixed dark-on-white so it stays scannable in dark mode); system share now prefers sharing the QR image as a `File` (`navigator.canShare({ files })` gate, falls back to text share).
- `.env.production`: documented `VITE_WEB_BASE_URL`.
- Tests: `invite.spec.ts` +1 (effective prefix = page origin in jsdom); `SharePanel.spec.ts` updated for pinia + invite-code assertion.

### Findings

### ✅ [P3] QR colors followed the theme and inverted in dark mode — SharePanel.vue
`--c-text` resolves to a light color in dark mode, producing a light-on-white QR inside the white tile and hurting scannability. Fixed: QR `dark`/`light` are now fixed (`#1F2430` / `#FFFFFF`) since the tile background is always white.

### ✅ [P3] Tauri fallback still leaked the API `8081` prefix — invite.ts
Without `VITE_WEB_BASE_URL`, the desktop packaged app fell back to the backend-built prefix (`http://localhost:8081/register?code=`), keeping the original bug alive. Fixed: fallback is now the relative path `/register?code=`; packaging requires `VITE_WEB_BASE_URL` (documented in `.env.production`).

### Verification (round 2)

- `pnpm test`: 55/55 green (9 files; +1 invite-store case).
- `pnpm typecheck` / `pnpm lint` / `pnpm format:check`: all pass.

---

## Round 3 (2026-08-14): Hash-route register link (`#/register`)

The generated link missed the hash segment — the app uses `createWebHashHistory()`, so the register page URL is `…/#/register?code=…`, not `…/register?code=…`.

### Changes

- `src/stores/invite.ts`: `effectiveRegisterUrlPrefix` now emits `…/#/register?code=` in all three branches (injected base, page origin, relative fallback).
- `server/internal/service/invite_service.go`: placeholder `register_url_prefix` updated to `…/#/register?code=` for contract consistency (front-end no longer consumes it).
- `docs/api/README.md`: contract example updated to `https://panel.example.com/#/register?code=` with a note that the front-end builds the prefix from its own origin.
- `.env.production` comment updated with the hash-route link shape.
- Tests: `invite.spec.ts` expectation updated (`http://localhost:3000/#/register?code=`); `SharePanel.spec.ts` LINK updated.

### Verification (round 3)

- `pnpm test`: 55/55 green; `pnpm typecheck` / `pnpm lint` / `pnpm format:check`: pass.
- `server`: `go test ./internal/service/` pass.

---

## Round 4 (2026-08-14): CORS support for https origins

The CORS allow-list only contained `http` variants, so `https` front-end origins (Vite https / Caddy local TLS / production `https://panel.example.com`) were rejected by the browser preflight.

### Changes

- `server/configs/config.yaml`: allow-list now also includes `https://localhost`, `https://127.0.0.1`, `https://localhost:5174`, `https://localhost:1420` (comment explains local https usage; production uses `https://panel.example.com` per deploy.md).
- `server/internal/middleware/cors_test.go` (new): 5 cases — https origins allowed, http origins allowed, non-allow-listed origin rejected (no `Access-Control-Allow-Origin`), OPTIONS preflight returns 204 with full headers, no-Origin requests unaffected.
- `docs/backend/deploy.md`: production example annotated as https.

### Verification (round 4)

- `server`: `go test ./internal/middleware/ -run TestCORS` 5/5 pass; `go test ./...` all pass.

---

## Round 5 (2026-08-14): `register_url_prefix` returns a path suffix only

Review feedback: the backend field still carried a full URL (`http://localhost:8081/register?code=…`, built from `cfg.App.BaseURL` — the API address, not the front-end origin). Since the front-end already builds the prefix from its own origin, a full URL is misleading; the field should carry only the path suffix.

### Changes

- `server/internal/service/invite_service.go`: `register_url_prefix` now returns the constant `/#/register?code=` (path suffix only, no domain); comment explains API base ≠ front-end site and that the front-end assembles the full link via `effectiveRegisterUrlPrefix`.
- `server/internal/model/dto_invite.go`: field documented as contract-only placeholder (suffix, not consumed by the front-end).
- `src/stores/__tests__/invite.spec.ts`: removed the misleading dead `registerUrlPrefix = 'http://localhost:8081/register?code='` assignment (the getter never reads the field); test now directly asserts origin-based assembly in jsdom; fetchCodes mock updated to suffix form.
- `mock/business.ts`: `register_url_prefix` mock updated to `/#/register?code=`.
- `docs/api/README.md`: contract example now `/#/register?code=` with a note that the full prefix is assembled front-end side.

### Verification (round 5)

- `server`: `go test ./internal/service/ ./internal/model/ ./internal/middleware/` pass.
- `pnpm test`: invite + SharePanel specs green; `pnpm typecheck` pass.
