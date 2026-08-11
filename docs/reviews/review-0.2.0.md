# Code Review — proxy-seller-web v0.2.0 (Full Repository)

- **Version:** 0.2.0
- **Date:** 2026-08-11
- **Scope:** Full repository — Go backend (`server/`), Vue 3 frontend (`src/`), Tauri desktop shell (`src-tauri/`), e2e tests (`tests/`), configs, CI
- **Method:** Two-axis review (Standards / Spec) run as four parallel read-only sub-agents, plus local build & test verification
- **Status:** All findings fixed (2026-08-11) — see Fix Record below

## Local Verification

All green: `go build ./...`, `go vet ./...`, `go test ./...` (backend); `pnpm typecheck`, `pnpm lint` (0 warnings), `pnpm test` (31 passed, 4 files). Note: green tests do not cover the defects below.

## Fix Record (2026-08-11)

All 42 findings (Standards 29 + Spec 13) are fixed and re-verified: `go build ./...`, `go vet ./...`, `go test ./...`; `pnpm typecheck`, `pnpm lint` (0 warnings), `pnpm test` (31 passed).

Backend (`server/`):
- Captcha email: template placeholder `{code}` → `{{.code}}` so the 6-digit code is actually delivered (`auth_service.go`).
- Notification toggles persist `false`: `UpdateProfile` uses map-based update + read-back; admin `UpdatePlan/UpdateServer/UpdateNotice/UpdateKnowledge` switched to map updates too.
- Order detail no longer nil-derefs on a deleted plan (placeholder name); `DeletePlan` refuses while orders reference the plan (11006).
- Checkout cache key now includes `user_id + order_no + method`; switching payment method no longer returns a stale result.
- Coupon `limit_per_user` is serialized inside the create-order tx (`Occupy` + `CountUsageLocked`); idempotency-key races replay the first order instead of 500.
- Balance-paid orders grant commission on the actual paid amount; banned accounts get 401 on subscribe; registration enforces mandatory invite code from site settings.
- Shared `commissionRateFor`; single `releaseCoupon` helper (cancel/close/refund/expire); dead code removed; epay notify validates `pid`; rate limit prefers `X-Forwarded-For`; `couponCode` errors surfaced.
- Expiry reminders fire at both 3-day and 1-day marks; agent valid-register days are settings-driven (`agent.valid_invite_days`); balance checkout returns `content: null`.

Frontend (`src/`, `src-tauri/`, `tests/`):
- Traffic-logs response shape aligned to the contract (`{list}`), including the mock.
- Session-expiry redirect keeps the hash route; tab titles are translated; FOUC script moved to `public/theme.js` (CSP-safe, correct storage shape).
- Profile notify switches hydrate from `GET /user/profile`; views use store actions (Profile, OrderConfirmModal, InviteView); admin views keep direct `apiAdmin` (documented exception).
- Deep-link plugin registered (`Cargo.toml`/`lib.rs`/capabilities); updater placeholder removed from `tauri.conf.json`.
- QR/echarts colors read CSS variables; shared `planSavePercent`/`periodLabel`; dead `apiPlan.list` removed; CopyText display is reactive; `formatTraffic` deduped to `formatBytes`.
- Removed debug `zz-errfail.spec.ts`; `mobile.spec.ts` unasserted call fixed to `expect(...).toBeVisible()`.

Docs:
- `docs/api/README.md`: added `GET /user/profile`; public endpoint list now includes `/notices`, `/knowledges`.
- `docs/backend/data-model.md`: coupon `limit_per_user` concurrency note.
- `docs/frontend/data-layer.md`: store-layering exception, hand-rolled polling, flat i18n files.
- `docs/frontend/progress.md` / `docs/backend/progress.md`: admin tickets moved to completed; per-axis fix sections added.

---

## Standards

### Backend (`server/`)

Architecture conforms well overall (handler→service→repo direction, `resp`/`errs` envelope, fen→yuan at DTO boundary, bcrypt-12, UUID sub-tokens), but several correctness defects remain, including all four previously-reported areas.

- ~~**server/internal/service/auth_service.go:96** — [P1] Verification code `{code}` is never substituted in email. `mailer.Template` uses `{code}` as literal text, but `mailer.Render` executes Go `html/template` where only `{{...}}` actions substitute, so users receive the literal string `{code}` and registration/forgot become unusable. Fix: use `{{.code}}` in the template body.~~
- ~~**server/internal/service/user_service.go:62** (via **server/internal/repo/user.go:79**) — [P2] `UpdateProfile` cannot persist `false` booleans. `db.Model(u).Updates(u)` with a struct skips zero values, so `RemindExpire=false` / `RemindTraffic=false` never reach the DB and the API reports them as still on. Fix: update only the changed columns (map or `Select`).~~
- ~~**server/internal/service/order_service.go:309** — [P2] `GetOrder` nil-pointer deref on deleted plan. `plan, _ := s.repos.Plan.GetByID(...)` then `plan.Name`; since `DeletePlan` (**server/internal/service/admin_crud.go:104**) has no referential check and migrations define no FKs, deleting a plan makes every historical order detail panic (recovered as 500). Fix: handle the error, and block/soft-delete plans referenced by orders.~~
- ~~**server/internal/service/order_service.go:387** — [P2] Checkout cache key omits `method`. `redispkg.Key("order", "paying", orderNo)` returns the first checkout's cached result for 30 min, so a user switching from `epay_alipay` to `epay_wxpay` gets the stale alipay URL and no wxpay `payments` row is created. Fix: include `method` (and user) in the key.~~
- ~~**server/internal/service/order_service.go:165** — [P2] Coupon `limit_per_user` check is not atomic. `CountUsage` then `RecordUsage` is a TOCTOU window; two concurrent orders by one user can both pass and exceed the per-user limit (the `coupon_usages` unique key includes `order_no`, so it doesn't help). Fix: conditional insert or a locked check inside the existing tx.~~
- ~~**server/internal/service/admin_crud.go:78** — [P2] Admin updates can't set booleans to `false`. `UpdatePlan` (and `UpdateServer`/`UpdateNotice`/`UpdateKnowledge` via `Updates(p)` with structs) drops zero-value fields, so admins cannot unpublish a plan/server/notice; `SpeedLimit`/`DeviceLimit` null-out is also skipped. Fix: map-based updates.~~
- ~~**server/internal/service/order_service.go:186** — [P3] Idempotency-Key race returns 500. Two concurrent `CreateOrder` calls with the same key both miss the DB lookup; the second insert violates unique `uk_orders_idem` and surfaces as 50000 instead of replaying the first order. Fix: catch the duplicate-key error and re-read.~~
- ~~**server/internal/service/order_service.go:369** — [P3] Dead parameter `r *http.Request` in `Checkout`. The request is passed from the handler but never used in the body; either use it (e.g., gateway IP/scheme for notify URL) or drop it.~~
- ~~**server/internal/service/order_service.go:623** vs **server/internal/service/invite_service.go:61** — [P3] Duplicated Code: commission-rate resolution. `commissionRate` and `rateOf` are near-identical (same `inviteCfg` struct, same fallbacks); extract one shared helper on `SettingService`.~~
- ~~**server/internal/service/order_service.go:328**, **server/internal/service/admin_service.go:201**, **admin_service.go:233**, **server/internal/service/cron_service.go:53** — [P3] Duplicated coupon-release block repeated four times. The `Release` + `DeleteUsage` sequence for cancel/close/refund/expire is copy-pasted; extract a `releaseCoupon(tx, ...)` helper (also reduces Shotgun Surgery risk).~~
- ~~**server/internal/repo/order.go:120**, **server/internal/repo/admin.go:126**, **server/internal/repo/order.go:151**, **server/internal/pkg/redis/redis.go:40** — [P3] Speculative Generality / dead code. `ListPendingOrderNos`, `GetByNoAdmin`, `IncrUsed`, and `SetString` are never called; remove or wire them in.~~
- ~~**server/internal/pkg/payment/epay.go:60** — [P3] `VerifyNotify` never checks `pid` matches the configured merchant. Signature verification is the main defense, but the standard epay flow also validates `pid`; add the check to prevent cross-merchant replay if the key is ever shared.~~
- ~~**server/internal/middleware/rate_limit.go:26** — [P3] `ClientIP()` is unreliable behind Caddy. With no trusted proxies configured, all clients share one IP bucket, so the global 300/min and login limits become global ceilings; configure trusted proxies or use `X-Forwarded-For` deliberately.~~
- ~~**server/internal/service/order_service.go:321** — [P3] `couponCode` swallows its DB error. `GetOrder` on an order whose coupon was deleted returns an empty `coupon_code` silently; return an error or `nil` explicitly.~~

### Frontend (`src/`, `src-tauri/`, configs)

No P1 findings (no crash/data-loss/security issue found; markdown is DOMPurify-sanitized, token storage follows the documented localStorage scheme).

- ~~**src/utils/http.ts:86-89** — [P2] Session-expiry redirect breaks under hash routing. `redirectToLogin()` does `window.location.href = '/login?...'` while the app uses `createWebHashHistory()`; the `redirect` param is built from `pathname+search`, so the current route (living in the hash) is lost, and on Tauri's custom protocol `/login` is not a real file → dead/blank page. Fix: push `{ name:'login', query:{ redirect: route.fullPath } }` via a router event/callback.~~
- ~~**src/router/guards.ts:34-37** — [P2] Tab title shows raw i18n key. `document.title = ... ' · ' + title` where `meta.title` is the key `'dashboard.title'` (declared as i18n key in `router/index.ts`), so tabs read `YLink · dashboard.title` and never localize. Fix: translate with `t(title)` in the guard.~~
- ~~**index.html:16-19** — [P2] FOUC script reads the wrong shape. `localStorage.getItem('app:theme')` then `JSON.parse(raw).value`; `setThemeMode` stores a bare mode string (`JSON.stringify('dark')`), so `.value` is always `undefined` and the persisted explicit theme is ignored (always falls back to system preference) → dark-mode users get a light flash. Fix: `JSON.parse(raw)` (the string itself).~~
- ~~**src/views/profile/ProfileView.vue:22-23,112-114** — [P2] Notify switches never hydrate from server. They start at hardcoded `true/false`; `onMounted` only fetches `stat`/`config`, and the contract (`docs/api/README.md` §5.2) has only `PUT /user/profile`, no GET — so saved settings aren't shown, and toggling one switch writes both flags, silently reverting the other. Fix: add a profile GET to the contract and hydrate on mount.~~
- ~~**src/main.ts:120 + src/utils/platform.ts:87-88** — [P2] Deep-link handler wired to an unregistered plugin. `onDeepLink` calls `onOpenUrl` from `@tauri-apps/plugin-deep-link`, but `lib.rs` never registers `tauri-plugin-deep-link` (not in `Cargo.toml`) and `capabilities/default.json` grants no deep-link permission → unhandled promise rejection at startup on Tauri, contradicting `desktop-tauri.md` §3. Fix: register plugin + capability, or skip wiring until enabled.~~
- ~~**src/views/invite/InviteView.vue:49** — [P2] `user.stat!.balance = data.balance` mutates store state from a view with a non-null assertion; entering `/invite` directly without `stat` loaded throws inside `try` after a successful transfer, so the UI shows an error toast for a succeeded transfer. Violates the documented "views only read/write store" layering. Fix: apply the balance in a store action.~~
- ~~**src/components/business/PaymentModal.vue:68** — [P3] QR code hardcodes `light:'#FFFFFF'`/`dark:'#1F2430'`, producing a white block in dark mode (design system forbids hardcoded colors).~~
- ~~**src/views/traffic/TrafficView.vue:117-140** — [P3] ECharts series/tooltip colors hardcoded hex instead of CSS variables; partial `isDark` branching only, dark refresh handled via `watch` but no `echarts` theme. Design-standard violation.~~
- ~~**src/components/business/PlanCard.vue:24-33,46-53 + OrderConfirmModal.vue:43-56** — [P3] Duplicated `savePercent` math and period-label maps (a third copy lives in `utils/format.ts:periodLabel`); extract one shared composable/util.~~
- ~~**src/api/plan.ts:5-10** — [P3] `apiPlan.list` duplicates `fetch` with a conflicting return type (`Plan[]` vs `PlanListResp`) and is never called (dead code; contract-mismatch risk).~~
- ~~**src/components/ui/CopyText.vue:38** — [P3] `const shown = computedDisplay()` runs once at setup; rows reused by key go stale when `text` prop changes. Use `computed`.~~
- ~~**tests/e2e/zz-errfail.spec.ts:1-22** — [P3] Committed debug spec with zero assertions and 5s sleeps pollutes CI runs (its `8081` tracing targets a non-mock backend); `mobile.spec.ts:27` ends with an unasserted `isVisible()`.~~
- ~~**src-tauri/tauri.conf.json:46-47** — [P3] Updater config is a placeholder (`https://example.com/latest.json`, empty `pubkey`) while the updater plugin isn't registered — dead config that will silently fail if enabled later.~~
- ~~**src-tauri/tauri.conf.json:33 + index.html** — [P3] CSP `script-src 'self'` blocks the inline FOUC script inside the Tauri WebView (browser-only protection).~~
- ~~**src/views/admin/AdminUsersView.vue:57-61** — [P3] Admin views call `apiAdmin` directly, bypassing the documented store-only layering; `formatTraffic` re-implements `formatBytes` with different rounding (`1 decimal GB` vs `2`).~~

---

## Spec

### Backend (`server/` vs `docs/api/README.md`, `docs/backend/*`)

- ~~**server/internal/service/order_service.go:445-458,607** — [P1] (c) Balance-paid orders never generate commission. Spec: `docs/backend/core-flows.md` §2.2 — balance checkout must "走与在线支付相同的 MarkPaid 后置逻辑（开通订阅/写佣金）"; §4 — "被邀请人每次付费（含续费）都产生佣金". Implementation zeroes `PayAmount` before writing commission: `locked.PayAmount = 0` then `grantCommission(tx, locked)`, and `grantCommission` computes `amount := order.PayAmount * int64(rate) / 100` → 0 → no `commission_logs` row. Invited users paying by balance silently break the commission chain.~~
- ~~**server/internal/service/subscribe_service.go** — [P2] (c) Banned user subscription returns 403, spec requires 401. Spec: `docs/api/README.md` §15 line 536 — "token 无效/用户封禁 → 401 纯文本". Implementation: `if user.IsBanned { return nil, errs.ErrForbidden }`, and `ErrForbidden` maps to HTTP 403 (`server/internal/pkg/errs/errs.go:39`). Clients keyed on 401 won't handle it.~~
- ~~**server/internal/service/auth_service.go** — [P2] (a) `invite_code_required` not enforced at registration. Spec: `docs/api/README.md` line 145 — "`invite_code` 选填（站点开启强制邀请时必填，见 /config）". `Register` never reads the site setting; the flag is only surfaced in `GET /config` (`server/internal/service/content_service.go:68-69`). When an admin enables mandatory invites, registration still succeeds without a code.~~
- ~~**server/internal/service/cron_service.go** — [P3] (a) Expire reminder sends once, not at both 3-day and 1-day marks. Spec: `docs/backend/core-flows.md` §9 — "expire-remind | 每日 10:00 | 到期前 3/1 天且开启提醒的用户发邮件". Implementation uses window `expired_at <= now+72h` plus a one-shot Redis mark → only a single email within 3 days, never the 1-day reminder.~~
- ~~**server/internal/service/order_service.go:464** — [P3] (c) Balance checkout returns `"content": ""` instead of `null`. Spec: `docs/api/README.md` line 390 — `{ "type": "paid", "content": null, "expire_in": 0 }`. Implementation: `Content: ""` (Go string, no `omitempty`).~~
- ~~**server/internal/repo/agent.go** — [P3] (a) "注册满 N 天" hardcoded, not settings-driven. Spec: `docs/backend/core-flows.md` §5 — "有效邀请定义：…或 注册满 N 天未封禁，阈值取 settings". Implementation hardcodes `INTERVAL 3 DAY`; no settings key for N days (only `required_valid_invites` is configurable).~~
- ~~**server/internal/router/router.go** — [P3] (b/c) `/notices` and `/knowledges` exposed without auth. Spec: `docs/api/README.md` line 22 lists免鉴权 endpoints exhaustively (captcha/auth/config/subscribe/notify) — notices/knowledges absent. Implementation registers both under the public group before the Auth middleware. Benign but outside the contract's auth matrix.~~

### Frontend (`src/` vs `docs/api/README.md`, `docs/frontend/*`)

- ~~**src/api/user.ts:29 + src/stores/user.ts:30** — [P1] (c) Traffic-logs response shape mismatch (contract break). Spec: `docs/api/README.md` §5.6 — `GET /user/traffic-logs` returns `"data": { "list": [ { "date", "u", "d", "total" } ] }`. Implementation types it as `http.get<TrafficLog[]>` (bare array) and assigns it straight to `trafficLogs`. Backend matches the doc (`server/internal/handler/subscribe.go:83` `resp.OK(c, gin.H{"list": list})`), but `mock/user.ts:274-277` also returns `ok(trafficLogs)` (array) — so the mock itself deviates from the contract, masking the bug in E2E. Against the real backend, `TrafficView.vue` gets `{list: [...]}` and renders an empty table/chart (`user.trafficLogs.reduce` on a non-array object) → traffic page broken. Fix: parse `.list`; align mock to §5.6.~~
- ~~**src/api/plan.ts:5-8** — [P2] (c) `GET /plans` duplicated with contradictory type. `fetch()` returns `PlanListResp` (`{list}`), while `list()` declares `Plan[]` for the same URL. The doc (`docs/api/README.md` §8) says data is `{ "list": [...] }`, so `list()` would return the envelope object as if it were a plan array. It's never called (dead code), so impact is latent, but it's a wrong-typed public API surface contradicting the contract.~~
- ~~**src/router/index.ts:152-156 + src/api/admin.ts:70-75** — [P2] (b) Scope creep: Admin tickets module not in documented completed scope. `docs/frontend/progress.md` §1 M8: "管理后台核心 5 模块(总览/用户/套餐/节点/订单)"; §2 lists "剩余:…工单…" as not done. Code registers `/admin/tickets` and `tests/e2e/admin.spec.ts` asserts it. The API appendix (`docs/api/README.md` §16) does include `/admin/tickets`, so it's additive scope the spec pages didn't claim as built — docs and code disagree on M8/M9 state.~~
- ~~**src/locales/zh-CN.ts / en-US.ts** — [P3] (a) i18n module structure differs from spec. `docs/frontend/data-layer.md` §5: "语言包按模块拆分：`locales/zh-CN/{common,auth,dashboard,order,invite,...}.ts`". Implementation is single flat files (`src/locales/index.ts:8`). Functionally lazy-loaded and complete, but not the documented module split.~~
- ~~**src/views/profile/ProfileView.vue + src/components/business/OrderConfirmModal.vue:10** — [P3] (a) Store-layer convention violated by views calling api directly. `docs/frontend/data-layer.md` §2: "store 是唯一允许调用 api 模块的地方". `ProfileView.vue` calls `apiUser.updateProfile` / `changePassword` / `resetSubscribe` directly; `OrderConfirmModal.vue` imports `apiCoupon` directly (`checkCoupon`), bypassing `useOrderStore`. Functionally fine, violates the documented layering.~~
- ~~**src/composables/** — [P3] (a) `usePolling` composable documented but absent. `docs/frontend/data-layer.md` §4 lists `usePolling` (generic polling with `visibilityState` pause). No `usePolling` exists — polling is hand-rolled in `stores/order.ts:60` and `stores/server.ts:29` with no visibility-pause. Minor divergence; behavior mostly equivalent.~~

---

## Summary

- **Standards axis:** 29 findings total — worst: P1 (verification code `{code}` never substituted, breaking registration/password recovery) in backend `auth_service.go:96`.
- **Spec axis:** 13 findings total — worst: P1 in each half (balance-paid orders never generate commission, backend `order_service.go:445-458`; traffic-logs response shape mismatch breaking the traffic page, frontend `src/api/user.ts:29`).
