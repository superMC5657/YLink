# Code Review — YLink v0.8.0 (Mobile Share Panel & Ticket Reply Notification)

- **Version:** 0.8.0
- **App release:** v0.4.1 (git tag). Review docs are numbered by review cycle (0.2.0 → 0.8.0), independently of app releases — the two version schemes are unrelated.
- **Date:** 2026-08-14
- **Scope:** New mobile-first share panel (`SharePanel.vue`) wired into the invite page, and an enhancement to the "ticket replied" desktop local notification (immediate check on window focus/visibility).
- **Method:** Reviewer-model review of the diff; `pnpm typecheck`, `pnpm lint`, `pnpm format:check`, and Vitest (54/54 green, +3 new SharePanel cases) all passed.
- **Status:** Rounds 1–7 findings all resolved (round 7 via round 8, 2026-08-22).

## Summary

The frontend gap "mobile share panel" from the phase-2 backlog was implemented: ~~a reusable bottom-sheet share panel (`SharePanel.vue`, `n-drawer placement="bottom"`, mobile-first)~~ that renders a QR code (existing `qrcode` dep, theme-aware colors like `PaymentModal`), shows the shareable link, and offers copy-link plus ~~system share (`navigator.share`, capability-gated and hidden on Tauri)~~ (changed from round 6 on to a floating panel + image download, see below). The invite page got a "share" button that opens the panel with the register link (prefix + first invite code). The ticket-reply local notification (already implemented via `useLocalNotifications.checkTickets` + 60 s polling in `MainLayout`) was enhanced so `onFocus`/`onVisibility` also run `checkTickets()`, removing up to 60 s of latency when the user returns to the window. Documentation was synced (frontend/backend progress, backlog item moved to done).

## Changes

- `src/components/business/SharePanel.vue` (new): props `show` / `title` / `text` / optional `desc`, v-model `update:show`; QR generation watches `show`/`text`/retry tick; failure state with retry; copy via `copyText`; ~~system share only when `navigator.share` exists~~ (changed from round 6 on to image download, see below).
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
- `src/components/business/SharePanel.vue`: redesigned as a brand invite card (brand gradient + `siteName` + invite code + QR on a white tile, QR fixed dark-on-white so it stays scannable in dark mode); ~~system share now prefers sharing the QR image as a `File` (`navigator.canShare({ files })` gate, falls back to text share)~~ (changed from round 6 on to image download, see below).
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

---

## Round 6 (2026-08-14): Share panel becomes a floating panel; system share replaced by image download

Usage feedback: the share panel should not slide up from the bottom (bottom sheet) — it should float above the page as a centered panel; rename "system share" to "download image" and render the purple invite card (site name + invite code + QR code) as an image the user can download, purely front-end.

### Changes

- `src/components/business/SharePanel.vue`: `n-drawer placement="bottom"` → `n-modal` preset=card centered floating panel (floats above the window, width `min(92vw, 30rem)` responsive); removed `navigator.share` system share (incl. `canShare({ files })` capability check), added a "download image" button — canvas-composed purple invite card PNG (720×940: gradient background + site name + invite code + white rounded QR tile + hint text + register link; rounded corners via a hand-written `roundRectPath` for older environments, over-wide text auto-shrinks), `canvas.toBlob` + `<a download="ylink-invite-{code}.png">` triggers the download, no backend dependency.
- `src/locales/{zh-CN,en-US}.ts`: `share.systemShare` → `share.downloadImage` (下载图片 / Download image), new `share.downloadFailed` (图片生成失败,请重试 / Failed to generate image, please retry).
- `src/components/business/__tests__/SharePanel.spec.ts`: stub switched from `n-drawer` to `n-modal`; `qrcode` mocked to return a fake dataURL (jsdom has no canvas); new download-image cases (button renders + canvas-unavailable failure message + canvas-available PNG composition triggers download with invite code in the filename, revokeObjectURL delayed 1 s + over-wide link truncated with an ellipsis).

### Findings

### ✅ [P3] jsdom has no canvas 2D context; the download branch must be testable — SharePanel.spec.ts
jsdom's `HTMLCanvasElement.getContext` returns null (unimplemented), and `qrcode.toDataURL` also depends on canvas. Handled: the test mocks `qrcode` to return a fake dataURL so the QR branch is reachable; the failure branch `spyOn(getContext).mockReturnValue(null)` asserts the `downloadFailed` message; the success branch stubs `Image`/`getContext`/`toBlob`/`URL.createObjectURL` and asserts the download fires with the invite code in the filename; the over-wide case stubs `measureText` to size by character count and asserts `fillText` receives truncated text ending in an ellipsis.

### ✅ [P2] Centered canvas text had no textAlign, drifting right — SharePanel.vue
`drawCenteredText` never set `ctx.textAlign`, so the default `start` made `fillText(IMG_W/2, y)` draw from the canvas midline left-aligned — all text drifted right and long text overflowed. Fixed: `textAlign='center'` for drawing, restored afterwards; when even the minimum size overflows, truncate to the fitting width and append an ellipsis (worst case degenerates to just the ellipsis) so the canvas never overflows.

### ✅ [P3] revokeObjectURL right after download aborts it on Safari/iOS — SharePanel.vue
Calling `URL.revokeObjectURL(url)` immediately after `a.click()` is known to abort in-flight downloads on Safari/iOS. Fixed: delay the revoke 1000 ms, and `appendChild` the anchor to the DOM before `a.click()` / `remove` after (old Safari requires the anchor to be attached to trigger the download).

### Verification (round 6)

- `pnpm test`: 58/58 green (9 files; SharePanel 6 cases).
- `pnpm typecheck` / `pnpm lint` / `pnpm build` pass; Playwright e2e (desktop/mobile) 14/14 pass.

---

## Round 7 (2026-08-22): Two-axis review of the range `1ccacb8...HEAD`

Whole-range review of the six commits `99016a7..88b8656` (mobile share panel & prefix adaptivity, suffix-only `register_url_prefix`, floating panel + image download, docs sync, Swagger gate, version bump), run as two parallel reviewer passes: **Standards** (documented repo standards — `docs/frontend/design-system.md`, `docs/frontend/README.md` §8, `docs/README.md` architecture, API contract — plus a Fowler code-smell baseline) and **Spec** (rounds 1–6 of this document, `docs/frontend/progress.md` / `docs/backend/progress.md`, commit intents). The axes are reported separately so one cannot mask the other. All findings were fixed in round 8. The uncommitted working-tree change to `src-tauri/tauri.conf.json` is outside the reviewed diff.

### Standards findings (resolved in round 8)

- **✅ [P2] Hard-coded brand gradient in business code — SharePanel.vue.** Template `style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"` violates design-system.md ("所有视觉决策以 CSS 变量令牌落地，业务代码不写死任何颜色") and frontend/README.md §8. Design-system §2.3 endorses this exact gradient but also specifies a dark variant (`#7C72FF → #A78BFA`) — a hard-coded literal can't switch, so dark mode shows the light-mode card.
- **✅ [P3] Canvas color literals** (`#6558f5`, `#8b5cf6`, `#FFFFFF`, `QR_DARK = '#1F2430'`): judgement call — the exported PNG is deliberately theme-independent, and canvas can't consume CSS vars without `getComputedStyle`.
- **✅ [P3] SharePanel header comment** cites no screenshot page / contract interface (frontend/README.md §8). Judgement call — new generic component.
- **✅ [P3] Duplicated Code** — gradient hexes appear in both the template style and canvas `addColorStop`; extract shared constants.
- **✅ [P3] Duplicated Code** — `InviteView.vue` builds `prefix + codes[0]` in three places (`registerLinkText` computed, `copyRegisterLink`, template interpolation); reuse the computed.
- **✅ [P3] Dead guards / Speculative Generality** — `openShare` and `copyRegisterLink` check `!effectiveRegisterUrlPrefix`, but the getter never returns falsy (final `return '/#/register?code='`); delete the checks.
- **✅ [P3] Duplicated Code** — `server/docker-compose.dev.yml`: api and worker `environment` blocks are copy-paste identical (a YAML anchor would dedupe); the `'/#/register?code='` literal appears 3× inside the prefix getter.
- **✅ [P3] Duplicated Code (pre-existing, extended)** — `MainLayout.vue` `onFocus`/`onVisibility` now have identical bodies after round 1 added `checkTickets()` to both; extract one `onVisible()`.

### Spec findings (resolved in round 8)

- **✅ [P2] Tauri fallback never triggers on Windows — invite.ts:37.** `if (origin && !origin.startsWith('tauri:'))` assumes Tauri origins start with `tauri:`, but Windows Tauri (the only packaging target, per progress.md) uses origin `http://tauri.localhost`, which passes the guard. A packaged build without `VITE_WEB_BASE_URL` yields an unusable `http://tauri.localhost/#/register?code=` link instead of the relative fallback.
- **✅ [P2] `.env.production` documentation is not in the repo.** Rounds 2–3 claim "`.env.production`: documented `VITE_WEB_BASE_URL`", but the file is gitignored (`.gitignore:32`) and absent from the diff — the packaging requirement lives only in an untracked local file; a fresh clone gets none of it. Move the guidance into a tracked doc (deploy.md / frontend README) or ship a `.env.production.example`.
- **✅ [P3] Swagger gate misses bare `/swagger`.** Caddyfile `@swagger path /swagger/*` doesn't match `/swagger` without a trailing slash — that path is still proxied. Minor: deploy.md's verification curl uses `/swagger/index.html`, which is covered.
- **✅ [P3] Review record incomplete.** backend/progress.md documents the dev-docker.sh rewrite, the MSYS path-conversion fix, and `docker-compose.dev.yml`, but this document has no round for any of it.
- **✅ [P3] Scope creep.** The dev-docker.sh full rewrite (`require_env`, REDIS/APP_REDIS consistency check, MSYS guards) plus DEMO_EMAIL/DEMO_PASSWORD in `server/.env.example` go beyond 99016a7's stated scope and anything in this document; documented after the fact in progress.md only.
- **✅ [P3] Version scheme mismatch.** 88b8656 bumps package.json/Cargo.toml/tauri.conf.json to 0.4.1 (mutually consistent), but this cycle's docs are titled v0.8.0 — code and doc version schemes disagree.
- **✅ [P3] Dead guard** — same item as on the Standards axis: `openShare`'s `!effectiveRegisterUrlPrefix` is always false.

### Verified OK

Caddyfile Swagger rejection for `/swagger/*` + deploy.md 404 curl step; MainLayout focus/visibility calling `checkTickets()`; CORS https allow-list + 5-case `cors_test.go`; `register_url_prefix` = `/#/register?code=` consistent across service/dto/mock/api README; SharePanel floating `n-modal` + canvas PNG download with Safari/iOS revoke handling; all new copy through i18n (zh/en); test counts match the claims (frontend 58, SharePanel 6, backend 71).

### Summary

Standards: 8 findings (1 hard violation, 7 judgement calls) — worst: the hard-coded gradient breaking the token rule and the dark-mode card variant. Spec: 7 findings (3 missing/partial, 1 scope creep, 3 wrong) — worst: the Tauri register-link fallback dead on Windows, the only packaged platform.

---

## Round 8 (2026-08-22): Round-7 findings fixed; dev-docker work retroactively recorded

All 15 round-7 findings fixed (2×P2, 13×P3; the dead-guard item is counted once per axis). Also documents here, retroactively, the dev-docker.sh rewrite that had no round of its own.

### Changes

- `src/styles/tokens.css`: new `--c-primary-grad-end` token (`#8b5cf6` light / `#a78bfa` dark, design-system §2.3). SharePanel's card gradient is now `linear-gradient(135deg, var(--c-primary), var(--c-primary-grad-end))`, so dark mode finally gets the §2.3 dark variant; the download-image canvas reads the same tokens at draw time via a `cssColor()` `getComputedStyle` helper (jsdom falls back to the light values). design-system.md §2.3 notes the tokenized form. Follow-up (out of round-7 scope, not done here): the same gradient literal still exists in 6 pre-existing components (AppSidebar, CustomerServiceFab, OrderCardList, OrderTable, SubscribeCard, UpdateCard) — migrate to the token when convenient.
- `src/stores/invite.ts`: Tauri fallback now keyed on `isTauri()` (`__TAURI_INTERNALS__`, `utils/platform.ts`) instead of matching the `tauri:` origin prefix — Windows packaged builds (`http://tauri.localhost` origin) now get the relative `/#/register?code=` fallback instead of an unusable link. Path suffix extracted to a `REGISTER_URL_SUFFIX` constant (was 3× inline). +1 test (`isTauri` mocked true → relative fallback).
- `src/views/invite/InviteView.vue`: display via new `registerLinkDisplay` computed (`------` placeholder preserved); `copyRegisterLink` reuses the `registerLinkText` computed; dead `!effectiveRegisterUrlPrefix` guards removed from `openShare`/`copyRegisterLink` (getter never returns falsy).
- `src/layouts/MainLayout.vue`: `onFocus`/`onVisibility` (identical bodies) merged into one `onWindowActive` handler registered for both events.
- `src/components/business/SharePanel.vue`: header comment now names the caller (invite page, 截图4) and states there is no contract interface (pure display component).
- `server/deploy/Caddyfile` + `docs/backend/deploy.md`: swagger matcher is now `path /swagger /swagger/*` — the bare path is also rejected; the deploy verification step gained the bare-path curl.
- `server/docker-compose.dev.yml`: api/worker `environment` blocks deduped via a `x-api-worker-env` YAML anchor (anchor resolution verified).
- `.env.production.example` (new, tracked): documents `VITE_WEB_BASE_URL` (incl. the Tauri Windows-origin caveat) and the Tauri signing vars; `docs/frontend/README.md` tree, `desktop-tauri.md`, and `progress.md` now reference it instead of only the gitignored `.env.production`.
- Version schemes clarified (header above): review docs are numbered per review cycle; app releases are git tags (currently v0.4.1). Code at 0.4.1 is correct — no version bump made.

### Retroactive record: dev-docker.sh rewrite (commit 99016a7, previously undocumented here)

That commit also rewrote `scripts/dev-docker.sh` beyond its title: `.env.dev` as the sole env source (`require_env`, no built-in defaults), a REDIS/APP_REDIS consistency check, MSYS path-conversion guards for Git Bash on Windows (cygpath-exported `DEV_CADDYFILE`/`DEV_DIST`), and `DEMO_EMAIL`/`DEMO_PASSWORD` added to `server/.env.example` (demo account). Motivation: the old script's built-in defaults silently diverged from `dev.sh`'s env handling. The work is kept; this record closes the round-7 spec findings "review record incomplete" and "scope creep".

### Verification (round 8)

- `pnpm test`: 59/59 green (9 files; +1 Tauri-fallback case in `invite.spec.ts`).
- `pnpm typecheck` / `pnpm lint` / `pnpm format:check`: pass.
- `server/docker-compose.dev.yml`: YAML parses and the anchor resolves to identical api/worker env (validated).
- Server Go code untouched this round (Caddyfile/compose/docs only), so `go test` not re-run.
