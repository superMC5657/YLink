# [P3] Traffic import default date must be local

Status: resolved
Type: task

## Finding

`AdminTrafficImportView` 初始行与 `addRow()` 均用 `new Date().toISOString().slice(0, 10)` 生成默认日期：在 UTC+8 的 00:00–07:59 会默认成前一天，管理员未手动改日期时把流量导入到错误日期。

## Resolution

初始行与 `addRow()` 改用 `localDateKey(new Date())`（本地时区 YYYY-MM-DD），与日期选择器的 `yyyy-MM-dd` 值口径一致。

## Comments

2026-08-28: 已修复（与 03 同批，Vitest 时区用例覆盖工具函数）。
