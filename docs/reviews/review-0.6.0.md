# Code Review — YLink v0.6.0 (Tauri Storage Initialization and Migration)

- **Version:** 0.6.0
- **Date:** 2026-08-13
- **Scope:** Tauri `plugin-store` storage migration and persisted API-base initialization.
- **Method:** Reviewer-model review of the storage backend migration.
- **Status:** All findings resolved (P1 and P2 fixed on 2026-08-13).

## Summary

The storage backend was changed to use Tauri `plugin-store` through a synchronous in-memory facade. However, the API client resolves its module-level `API_BASE` before `bootstrap()` awaits `initStorage()`. As a result, persisted API endpoints are ignored for the session. Existing WebView `localStorage` data is also not migrated into `app-settings.json`, which can log users out and discard persisted settings after upgrade.

## Completed

Introduced a Tauri `plugin-store` backend with a synchronous in-memory facade and asynchronous persistence.

Fixed P1: `API_BASE` is now resolved lazily per request (`resolveApiBase()` in `src/utils/http.ts`) instead of at module import, so a persisted custom `apiBase` takes effect after `bootstrap()` awaits `initStorage()`.

Fixed P2: `initStorage()` (`src/utils/storage.ts`) now runs a one-time migration that imports legacy `app:`-prefixed WebView `localStorage` keys into `app-settings.json`, skipping keys already present and guarded by an `app:_legacy:migrated:v1` marker.

## Findings

### ✅ [P1] Initialize storage before resolving the API base — src/utils/storage.ts:35-39

**Status:** Fixed (2026-08-13). `API_BASE` was replaced by a lazily evaluated `resolveApiBase()` that reads the persisted `apiBase` at request time (after `initStorage()` has run). Both the main request path and the 401 silent-refresh path (`/auth/refresh`) now use it. Covered by `src/utils/__tests__/http.spec.ts` (persisted custom apiBase used for requests and refresh).

### ✅ [P2] Migrate existing WebView storage into plugin-store — src/utils/storage.ts:38-43

Before this migration, Tauri users' tokens and settings were stored in WebView `localStorage`. The new initialization only reads `app-settings.json` and never imports those existing keys. Existing users can lose their login session and persisted settings on the first upgrade.

**Status:** Fixed (2026-08-13). `initStorage()` now calls `migrateLegacyLocalStorage()` after preloading `app-settings.json`: every `app:`-prefixed key still present in WebView `localStorage` is imported into the plugin-store (persisted via the async facade) unless the key already exists there; a `app:_legacy:migrated:v1` marker makes the migration one-time. Covered by `src/utils/__tests__/storage.spec.ts`.

## Verification

- Review result: P1/P2 findings reported.
- No code fix was included in this review record.
- **2026-08-13:** Both findings fixed; `npm run typecheck` / `npm run lint` green, vitest 51/51 passing (incl. new `http.spec.ts` lazy-`apiBase` case and `storage.spec.ts` migration cases).
