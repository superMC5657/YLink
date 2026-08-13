# Code Review — YLink Admin Modules (Incremental, post-0.2.0)

- **Date:** 2026-08-11
- **Scope:** Incremental — admin-module commits since the v0.2.0 review (`7949622..a8295ce`): new admin modules (coupons, traffic import), mocks, tests, cache and port changes
- **Method:** Reviewer-model pass over the recent commits with local verification (backend builds, new tests pass, DTO/type shapes match)
- **Status:** Resolved — 2 P1 + 1 P3, all fixed (see strikethrough items below; commits `078f541`, `294c116`, and the traffic-import wording fix)

## Summary

The new admin modules, mocks, tests, and cache/port changes are otherwise consistent (backend builds, new tests pass, DTO/type shapes match), but the port bump breaks the shipped docker-compose Caddy deployment, and percentage coupons created through the new admin UI discount 100% of the order at checkout due to a unit mismatch between the stored value and the checkout math. Both P1s and the P3 wording issue have since been fixed.

## Findings

### ~~[P1] Port move 8080→8081 leaves Caddyfile proxying api:8080 — server/docker-compose.yml:35-36~~

~~This commit changes the API listen port to 8081 (`configs/config.yaml` `addr: ":8081"` is baked into the image via the Dockerfile) and publishes `8081:8081` here, but `server/deploy/Caddyfile` — mounted read-only by the `caddy` service in this same compose file — still has `reverse_proxy api:8080`. Inside the compose network Caddy reaches the container directly, so every request through the production entrypoint will fail (connection refused/502). `docs/backend/deploy.md` was updated to `reverse_proxy api:8081`, but the actual Caddyfile was missed; `server/Dockerfile:14` also still says `EXPOSE 8080` while the deploy.md snippet was updated to 8081. Severity is limited to docker-compose/Caddy deployments (dev.sh runs the binary directly and is unaffected), but that path is fully broken by this change.~~

**Resolved** in `294c116` — `server/deploy/Caddyfile` now proxies `api:8081`, `server/Dockerfile` `EXPOSE 8081`.

### ~~[P1] Percentage coupons created via new admin UI give 100% off — server/internal/service/admin_crud.go:249-251~~

~~This commit's contract (docs §16.1 "type=2 为百分比数值，如 10 表示 10%"), the mock data, `AdminCouponsView.vue`, and this `FenToYuan` display conversion all treat a percentage coupon's `value` as a raw percent. However, `CreateCoupon`/`UpdateCoupon` store `model.YuanToFen(req.Value)` for every type (10 → 1000), and checkout computes `discount = amount * coupon.Value / 100` clamped to the order amount (`order_service.go` `validateCoupon`). A "10%" coupon created or edited through the new admin page therefore yields `amount * 10` → clamped → the whole order free; any percent ≥ 1 makes orders free. The write-path conversion and checkout math predate this commit, but this commit introduces the UI and documented contract that make percentage coupons reachable, so the end-to-end feature is financially unsafe as shipped. Fix requires one consistent unit, e.g. skip `YuanToFen`/`FenToYuan` when `type == 2` (and keep checkout's `/100`), or divide by 10000 at checkout.~~

**Resolved** in `078f541` — checkout now computes `discount = amount * coupon.Value / 10000` (the stored value is percent×100 in fen), so a 10% coupon discounts exactly 10% of the order.

### ~~[P3] Traffic-import hint "累加覆盖" contradicts overwrite behavior — src/views/admin/AdminTrafficImportView.vue:128~~

~~The hint says re-importing the same user+date "将累加覆盖", but `TrafficLogRepo.Upsert` (`server/internal/repo/admin.go`, commented "按 (user_id, date) 覆盖导入") replaces `u`/`d` rather than adding to them. An admin reading the hint as accumulation would import only the delta and silently overwrite the day's previously recorded usage. Worth rewording to state that re-import overwrites the day's values.~~

**Resolved** — the hint now reads "同一用户同一天重复导入将覆盖当日已记录的上行/下行数据" (re-import overwrites that day's `u`/`d`), matching the `Upsert` behavior.
