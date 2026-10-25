# [P2] 前端数据层合规批（store 下沉 / silent / api seam ADR / AppIcon / api.d.ts）

Status: resolved
Type: task

## Finding

1. **用户端视图直调 api**：`views/profile/ProfileView.vue:16,153,179,198,213`（telegramBindCode/Unbind、sessions、revokeSession）与 `composables/useLocalNotifications.ts:34` 直调 `apiUser.*`，违反 data-layer.md §2「用户端业务视图一律经 store」；`useUserStore` 恰好缺这些 actions。
2. **双重反馈**：`OrderConfirmModal.vue:145` 内联渲染优惠券错误，但 `stores/order.ts:23` couponTransport.check 未加 `silent: true` → http 层又 toast 一遍（§7 约定）。
3. **api seam 倒置未记录**：c47bfb0 后 `api/{agent,invite,knowledge,notice,order,plan,server,ticket,user}.ts` 九模块变为从 `@/stores/*` 反向 re-export 的 seam，分层方向与 README §3/data-layer §1.2 相反且无 ADR；data-layer.md §2 仍写「store 是唯一允许调用 api 模块的地方」并列出不存在的 `useSubscribeStore`；auth.ts 保留旧方向（两种模式并存）。
4. **AppIcon 标准二选一**：README §2/design-system §8 规定 @iconify/vue（Solar/MingCute），实际为内联 Lucide（头部注释声明有意为之），文档未更新。
5. **api.d.ts 契约陈旧**：`:466` 会话 `created_at: string | null` + 「历史会话可能无元数据」注释——clean-architecture spec 已删除该历史兼容（后端 dto.go:101 已同步「恒有值」）。

## Plan

1. `useUserStore` 补 sessions/telegram/revoke actions（transport 已在 store 内），ProfileView 与 useLocalNotifications 全部改走 store。
2. couponTransport.check 加 `silent: true`。
3. **已拍板：代码为准，文档随改**：保留 api 模块「契约名出口 + store 持有实现」的 seam 现状（c47bfb0 引入的既成架构）——新增 `docs/adr/0006` 记录该方向与理由（契约名稳定、transport 随 store 收敛），README §3 与 data-layer.md §1.2、§2 按当前实现改写（删 useSubscribeStore 幽灵条目、改「store 是唯一 transport 持有者」表述）；auth.ts 的双模式在 ADR 中说明（顺手统一亦可，非硬性）。
4. **已拍板：代码为准，文档随改**：保留 AppIcon 内联 Lucide（零网络依赖契合桌面端），README §2 与 design-system §8 改写为内联图标规范（AppIcon PATHS 为单源、新增图标=补一条 path），删除 @iconify/vue 表述。
5. api.d.ts 会话 `created_at: string`，删除历史注释。

## Acceptance

- `grep -rn "apiUser\|api/admin" src/views src/composables src/components` 在用户端/组件层零命中（admin 视图豁免除外）。
- 优惠券错误只出现一次（内联，无 toast）——check-error-toast 通过。
- docs/adr/0006 + data-layer.md/README 与代码一致；`pnpm test`（含 adapter.spec）全绿。

## Comments

2026-10-25: 用户拍板——Plan 3/4 两个决策点一律按「代码为准、文档随改」执行：api seam 保留现状并补 ADR-0006；AppIcon 保留内联 Lucide，文档改写为内联图标规范。

2026-10-25: 已修复 ProfileView/useLocalNotifications 全部改走 store（useUserStore 补 sessions/telegramBindCode/telegramUnbind/revokeSession actions、useTicketStore 补 fetchStatusSnapshot 快照轮询动作[不写列表态]）、couponTransport.check 加 silent:true、api.d.ts 会话 created_at 收敛为 string 并删历史注释；新增 docs/adr/0006 并按当前实现改写 data-layer.md §1.2/§2（删 useSubscribeStore 幽灵条目）、README §2/§3/§4、design-system §6.3/§8；progress 能力行同步。

2026-10-25: 范围外发现（未顺手修）——① 8 个用户端 seam（apiAgent/apiInvite/apiKnowledge/apiNotice/apiCoupon/apiOrder/apiPlan/apiServer）连同本次归零消费方的 apiTicket 已无直接调用方，是否删除归 ticket 21 死代码收敛；② README §4 composables 行仍列 usePolling/useTrafficFormat/useMediaQuery（实际 useMediaQuery 来自 @vueuse/core、usePolling 不存在），归 ticket 18 文档对齐批；③ server/docs/swagger.json 为陈旧部分快照（无 /user/sessions 等端点，Makefile swag 输出目录问题），created_at 契约在 Go 注解源（dto.go 恒有值）已正确，swagger 重生成归 ticket 18。
