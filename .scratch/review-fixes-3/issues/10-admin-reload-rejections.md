# [P1] 管理端变更后刷新的未处理 Promise 拒绝

Status: resolved
Type: task

## Finding

`useAdminList.load` 设计为 rethrow（useAdminList.ts:110-112，供视图首载 distinguish loading/error），但 13 个调用点用 `void load()` 触发变更后的刷新：

- `src/composables/useAdminNodes.ts:153,168,183,209,296,366,382,386`（8 处）
- `src/composables/useAdminUsers.ts:80,102,130,165,197`（5 处）

刷新失败（http 已 toast）→ 未处理 Promise 拒绝，污染控制台并可能触发全局错误上报。

## Plan

在 `useAdminList` 增加非抛出的 `reload()`（内部 `load().catch(() => {})`，语义即「刷新，错误交给 http 层」），13 处 `void load()` 全部改 `void reload()`。保留 `load` 的 rethrow 语义给首载场景。

## Acceptance

- 管理端任一变更动作（封禁/调余额/批删等）后断网或后端 500，控制台无 UnhandledRejection。
- `pnpm test` 全绿（useAdminList spec 补 reload 不抛断言）。
