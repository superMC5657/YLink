# ADR-002 · 管理端契约 single-owner + 生成

> 状态：已接受（2026-10-12，dev-ica3）
> 相关：`docs/api/README.md` §16/§18、`server/internal/model/dto_admin.go`

## 背景

一个管理端概念（如 AdminServerItem）在 5 处手工同步、零杠杆：
`server/internal/model/dto_admin.go`（Go DTO）、`src/types/api.d.ts`（手写 TS）、
`mock/admin.ts`（手写 fixtures）、`src/api/admin.ts`、`docs/api/README.md`（契约文档）。
后端虽有 swag 依赖，但注解覆盖不全、产物已滞后，无人消费它做 TS。

## 决定

1. **Owner**：形状归 `server/internal/model/dto_admin.go`，语义归 `docs/api/README.md` §16。
   其余三处（TS 片段/mock/机器快照）只读 owner，不再手写形状。
2. **生成路线选 DTO 直解析，不选 swagger-JSON 驱动**：
   swagger 侧缺管理端全量注解、产物滞后，补齐注解本身就是一次大重构；
   DTO 直解析（`scripts/gen-admin-contract.mjs`，纯 Node、无新依赖、Win/Linux 同行为）
   确定性发射三产物：`src/types/admin.generated.d.ts`、
   `mock/admin.skeletons.generated.ts`、`docs/api/admin-contract.json`。
   幂等（复跑字节一致），命名差 4 组（View→Item 等）只做映射登记，不改线型。
3. **mock 消费骨架**：`mock/admin.ts` 各条目以 `{...AdminSkeletons.<DTO>, 手填值}` 书写；
   DTO 加字段自动带零值不断形状，语义值手填并以 `HAND-VALUE` 标记。
   生成骨架经 vite `ignore: /\.generated\.ts$/` 排除在 mock 路由加载之外。
4. **一致性检查即接口测试面**（`scripts/check-admin-contract.mjs`，串入 `pnpm lint`，
   本地与 CI PR 门禁同覆盖）：①三产物新鲜度（改 DTO 未跑 `contract:gen` 即红）；
   ②`api.d.ts` 管理端接口逐字段审计（字段存在 + 核心类型 + binding 必填 +
   TS 扩展须 `EXTENSIONS` 登记 + 孤儿接口拦截）。人工全量对账不再做。
5. **边界**：`AdminServerGroupItem`（线型=表结构 `model.ServerGroup`）、
   `AdminTicketDetail`（线型=`model.TicketDetailResp`，用户/管理共用）、
   `AdminAuditLogsResp`（组合体）、`NodeSettings`（客户端解析形）、
   `TicketReplyReq`（历史重名）暂列 `ADHOC`，只不断言形状；
   `TicketDetailResp` 折叠进 DTO 体系是后继工作，不在本轮。

## 后果

- 管理端加字段只有一条路：改 DTO + README §16 → `pnpm contract:gen` → check 全绿。
- `| null` / `?` 不做硬断言（Go nil slice 即 null、PageOK 兜底、请求省略语义所致）；
  卡的是加/删/改名/改核心类型四类真漂移。
- 新管理端类型直接引用生成片段；存量 `api.d.ts` 别名迁移择机做，不 blocking。
