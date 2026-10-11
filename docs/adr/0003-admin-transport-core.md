# ADR-0003 · 管理端传输核归属与列表深模块

> 状态：已接受（2026-10-12，dev-ica3）
> 相关：`src/stores/admin/`、`src/api/admin.ts`、`src/composables/useAdminList.ts`、ADR-0002

## 背景

管理端前端数据流（视图 → 组合式函数 → apiAdmin → http → types）呈 god object 形：
`src/api/admin.ts` 独家手写 69 个方法（URL 与形态无第二归属）；
`useAdminUsers`/`useAdminNodes`/6 个直调视图各抄一份取数/分页/多选/批量汇总骨架，
行为靠人工对齐。 composition 处的真 bug（如分组保存复位错 flag）无测试可拦。
店内已有先例：ticket 域传输核（`src/stores/ticket.ts` + 7 行适配缝，ica-E）。

## 决定

1. **传输核按域归 `src/stores/admin/<domain>.ts`**（overview/users/plans/nodes/orders/
   tickets/content/agent/system + 共用 `path.ts` 路径段）；
   `src/api/admin.ts` 收薄为纯适配缝（导出名全量稳定，`adapter.spec` 锁定）。
   新管理端端点只加传输核方法 + 适配缝接入，不再进 god object。
2. **列表骨架归 `src/composables/useAdminList.ts`**（取数/钳制/多选/批量汇总/失败保旧表/
   空页单步回退）；`useAdminSelection.ts` 删除，逻辑逐字搬入（删除测试=新单测全覆盖）。
   确认弹窗等域语义留在各组合式函数。
3. **组合层 bug 修在组合层**：本轮顺手修分组保存 `finally` 错置 flag
  （`batchSaving`→`groupSaving`，此前成功后分组按钮永久 saving），回归单测锁定。

## 后果

- 管理端加接口三步：DTO+README（ADR-0002）→ 传输核方法 → 视图/组合消费；
  列表行为改一处，18 视图同生效。
- `| null`/`?` 宽松策略沿用 ADR-0002；传输核只锁 URL 与调用形态，不重做契约审计。
