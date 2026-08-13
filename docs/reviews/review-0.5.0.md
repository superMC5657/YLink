# Code Review — YLink v0.5.0 (Admin Order Commission Data)

- **Version:** 0.5.0
- **Date:** 2026-08-13
- **Scope:** Recent changes for exposing `commission_amount` in the admin order list.
- **Method:** Reviewer-model review; frontend typecheck, Vitest (43 tests), and server Go tests passed.
- **Status:** All findings resolved (P2 fixed 2026-08-13).

## Summary

The main behavior change is implemented and the available validation passes. The P2 finding (commission lookup failure silently ignored) has been fixed: `ListOrders` now propagates the batch commission query error, so the admin order endpoint no longer returns successful-looking data with `commission_amount: null` for every order when the lookup fails.

## Completed

~~Added `commission_amount` to the admin order list by batch-loading commission records by order number.~~
~~Propagate commission lookup failures (P2) — `ListOrders` returns the error from `ListByOrderNos` instead of silently ignoring it; regression test `TestAdminListOrdersCommissionQueryError` added.~~

## Findings

### [P2] Propagate commission lookup failures — server/internal/service/admin_service.go:196-197

~~`ListOrders` ignores errors from the batch commission query:~~

```go
if comms, err := s.repos.Commission.ListByOrderNos(s.db, orderNosOf(list)); err == nil {
```

~~When the database/query fails, the endpoint still returns the order list with an empty commission map. Return the lookup error instead, so the admin UI does not display inaccurate financial data as a successful response.~~

**Status:** ✅ Resolved (2026-08-13). `ListOrders` now returns the error from the batch commission query; the old `err == nil` swallow is removed. Covered by `TestAdminListOrdersCommissionQueryError`.

## Verification

- Frontend typecheck — passed
- Vitest — 43 tests passed
- Server Go tests — passed (67 → 68 test functions, including the new P2 regression test)
