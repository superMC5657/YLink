# [P1] Add the configured NSIS hook file

Status: resolved
Type: task

## Finding

`src-tauri/tauri.conf.json` 的 `installerHooks` 指向不存在的 `nsis/installer-hooks.nsh`，Windows NSIS 打包会失败。

## Resolution

新增 `src-tauri/nsis/installer-hooks.nsh`，提供 4 个 no-op NSIS hook 宏并同步 `docs/frontend/desktop-tauri.md`。

## Comments

2026-08-25: 已入库。
