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

## Comments

- 2026-10-25: 已修复——`useAdminList` 新增非抛出的 `reload()`（复用上次筛选、失败静默、错误交 http 层 toast，loading/保旧表语义与 load 一致），`useAdminNodes`（8 处）与 `useAdminUsers` 的 `void load()` 全部改 `void reload()`；`load` 的 rethrow 语义保留给首载/筛选切换。spec 补「reload 失败静默不抛且保留旧表、load 保持重抛」断言；`pnpm lint` + `pnpm test`（239 用例）全绿。
- 2026-10-25: Finding 清单外的同款调用点一并收敛：`useAdminUsers.ts` 的 `onMounted(() => void load())`（挂载首载被 void 丢弃，失败同样产生未处理拒绝）——合计 14 处。`reload` 同时加入两个 composable 的返回值供视图后续使用。
- 2026-10-25: 范围外发现（未修，属同类后续项）：`useAdminNodes` 的 `batchShow/batchUpdate/saveSort` 三处 `await load()` 位于 try/finally 无 catch，刷新失败会经 Vue errorHandler 逃逸（非 UnhandledRejection，不违反本 ticket 验收）；视图刷新按钮 `@click="load"` 直绑 load 同理。可后续统一改 reload。
