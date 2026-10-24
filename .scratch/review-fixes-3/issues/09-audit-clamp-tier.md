# [P1] 审计分页档位钳制错误（clampPageSize(100)→10）

Status: ready-for-agent
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
