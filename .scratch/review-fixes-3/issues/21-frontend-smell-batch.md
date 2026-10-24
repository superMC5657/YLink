# [P3] 前端重复形状与死代码收敛

Status: ready-for-agent
Type: task

## Finding

- **重复形状**：
  - AdminNoticesView:80-109,233-277 ≡ AdminKnowledgesView:86-115,448-492 排序弹窗逐字重复（NodeSortDialog 已存在）
  - AdminPlans/Coupons/Notices/Knowledges 四份非分页 CRUD 脚手架（loading/list/modal/editingId/saving/open*/save/remove）同形，表单缺省字面量写两遍
  - RegisterView:102-117 ≡ ForgotView:59-74 邮箱验证码块；NotFoundView ≡ ForbiddenView（仅文案不同）
  - `.slice(0,N).replace('T',' ')` 日期原语 ×5（AdminOrdersView:161、AdminTicketsView:214,217,261,343）
  - ThemeToggle:19-35 ≡ LanguageToggle:19-33 滑块常量+位移；SubscribeCard:92-139 五格 5 份同块；PaymentModal:174-182 ≡ 197-205 倒计时胶囊；QuickActionGrid:36-132 动作表+switch 双记账 + 193-215 三个复制按钮
  - mock `ok()/unauthorized()` ×6 文件、分页仅 admin.ts 有 `paged()`；e2e admin.spec:13-36 ≡ withdraw.spec:10-28 loginAs/enterPortal/openAdminMenu；dev.sh:92-100 ≡ dev-docker.sh:89-97 ensure_image；http.ts:152-158 重实现 buildQueryStr；useAdminUsers:224-231 重复实现 useAdminList 的批量明细格式化
  - 6 份手写 loading/try/assign 列表骨架（stores/{notice,knowledge,plan,ticket,order,server}）
- **死代码**：`router/nav.ts:98` ALL_NAV_PATHS、`stores/ticket.ts:90` levelOf、`useAdminList.ts:194-207` reportBatchResult 仅自测消费（改私有或内联）、`utils/adminList.ts:33-37` batchAuditTarget 零生产调用（mock 手写 `'batch:2'`——注释声明失实：采纳进 mock 或内联删注释）。
- **杂项**：i18n 测试无 zh↔en 键齐平断言；server-status 选项数组两处内联；CustomerServiceFab watch 镜像改 computed；ProfileView 592 行/InviteView 568 行多域混杂（可选拆卡片组件，admin 侧已示范）。

## Plan

按组小 PR：① 排序弹窗/CRUD 工厂（useAdminCrud + emptyForm，参照 useAdminList）② ErrorPage/useEmailCode/formatDateTime 单源化 ③ 组件内小去重（滑块 composable、v-for 化）④ mock `_envelope.ts` + e2e helpers 入 fixtures + ensure_image 抽出 ⑤ 死代码删除 ⑥ i18n 键齐平测试（`expect(Object.keys(zh)).toEqual(Object.keys(en))` 递归形状）。ProfileView/InviteView 拆分为可选，不在本 ticket 验收内。

## Acceptance

- `pnpm test` + e2e 全绿；删重复后行数净减（排序弹窗/CRUD 各收敛为一份实现）。
- i18n 键齐平测试落位且当前通过（或先落测试暴露差异并补齐缺失键）。
- `grep -rn "ALL_NAV_PATHS\|levelOf" src/` 零命中。

## Comments
