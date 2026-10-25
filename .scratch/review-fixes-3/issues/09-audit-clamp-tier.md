# [P1] 审计分页档位钳制错误（clampPageSize(100)→10）

Status: resolved
Type: task

## Finding

`src/composables/useAdminList.ts:74,130` + `src/stores/admin/system.ts:37`：前端唯一钳制源只建模 Standard 档 {10,50}，`clampPageSize(100)` 落回 10。而 ADR-0004 保留 Audit 档 {default 20, max 100}，审计页（AdminAuditLogsView.vue:207）提供 `page-sizes [20,50,100]`——用户选 100/页实际拉 10/页，静默错误。适配层头部自己承诺「数字与后端同值」（server `pkg/page` 双档）。

## Plan

1. `utils/adminList.ts` 的 `clampPageSize` 增加档位参数（或 `clampPageSizeAudit` 变体）：Standard {10,50}、Audit {20,50,100}，default 分别 10/20——与 `pkg/page` 常量同值（注释保留「与后端同值」约定）。
2. `useAdminList` 增加 audit 模式入口（或参数），审计列表 composable/transport 传 audit 档；`stores/admin/system.ts` 审计 transport 使用之。
3. 顺带裁决 `useAdminList.ts:81` 的 `res.list ?? []` 回退（Spec 轴发现：clean-architecture spec 要求直接赋值、后端 PageOK 保证 `[]`）——若保留，注释注明是 composable 通用兜底并从 spec 语义豁免；推荐直接赋值。

## Acceptance

- 审计页选 100/页时请求 `page_size=100` 且渲染 100 条；50/100 以外的值按档位收敛。
- adapter.spec 增加 audit 档断言；`pnpm test` 全绿；ADR-0004 无需变更（实现对齐）。

## Comments

- 2026-10-25: 已修复——`utils/adminList.ts` 引入 `pageTier` 双档常量（standard {10,50} / audit {20,100}，注释保留「与 pkg/page 同值」约定）与 `clampPageSizeFor(tier,…)`,导出 `clampPageSizeAudit`/`withPageAudit`;`useAdminList` 增加 `tier` 选项（缺省 standard，pageSize 缺省取档位缺省值）;`stores/admin/system.ts` 的 `auditLogs`/`trafficResets` 传输核改用 `withPageAudit`（此前 page_size=100 被标准档归一回 10，是请求侧的第二处静默错误）;审计日志视图与流量重置记录列表传 `tier: 'audit'`（视图的 `pageSize: 20` 显式值移除，由档位缺省提供）。`res.list ?? []` 按 Plan 推荐改为直接赋值（后端 PageOK 契约，ADR-0001）。测试：adminList.spec 双档钳制/withPageAudit 断言、useAdminList.spec audit 档（默认 20、100 透传、101 回落 20）；`pnpm lint` + `pnpm test`（238 用例）全绿，ADR-0004 无需变更。
