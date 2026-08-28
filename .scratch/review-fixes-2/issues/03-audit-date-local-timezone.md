# [P2] Audit log date filter must use the user's local timezone

Status: resolved
Type: task

## Finding

`AdminAuditLogsView` 用 `new Date(...).toISOString().slice(0, 10)` 生成 from/to：在 UTC+8，本地 8 月 28 日 00:00 转 UTC 为 8 月 27 日，后端再按本地时间解析，筛选范围整体前移一天。

## Resolution

`src/utils/format.ts` 新增 `localDateKey(value: Date | number | string)`（dayjs `format('YYYY-MM-DD')`，按用户时区取日期），`from`/`to` 改用它；并补 format.spec.ts 时区用例（本地构造 00:30 断言日期键不前移）。

## Comments

2026-08-28: 已修复；与 03 同用 `localDateKey`。
