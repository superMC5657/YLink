# [P3] 前端重复形状与死代码收敛

Status: resolved
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

- 2026-10-25: 已修复，按 6 组各一 commit（b122739/0a74b93/d5601be/89432e2/b475221/14e9058）：① 排序弹窗数据域 `useSortDialog` + 呈现 `SortDialog`（NodeSortDialog 更名参数化，节点/公告/知识库三份归一）、非分页 CRUD 四视图迁 `useAdminCrud`（emptyForm 单源，分类联动走 onOpen/beforeSave 钩子，净减 132 行）② `ui/ErrorPage` 收敛 404/403、`useEmailCode`+`views/auth/EmailCodeField` 收敛注册/找回验证码块、`formatDateTime` 替换 Orders/Tickets 五处 slice 原语 ③ `useSlideOffset` 收敛主题/语言滑块、SubscribeCard 五格 v-for、`CountdownPill` 收敛倒计时胶囊、QuickActionGrid 动作 run 入表删 switch + 下载按钮 v-for、CustomerServiceFab watch→computed、`serverStatusOptions` 收敛节点状态选项两处内联 ④ `mock/_envelope.ts`（ok/unauthorized/forbidden/paged ×6 文件）、e2e loginAs/enterPortal/openAdminMenu 入 fixtures、`scripts/lib/ensure-image.sh` 抽出 dev.sh/dev-docker.sh 重复、http.ts request() 复用 buildQueryStr、useAdminUsers 发邮件汇总复用 reportBatch（文案与批量动作同形，mailSent/mailPartial→mailDone）⑤ 死代码：ALL_NAV_PATHS/levelOf 删除、reportBatchResult 收私有（删冗余直测用例）、batchAuditTarget 采纳进 mock（注释声明现成立）⑥ i18n zh↔en 键树齐平测试落位（当前两份语言包同构，一次通过）。验证：pnpm lint/typecheck/test（244 绿）+ pnpm exec playwright test（66 绿，浏览器环境已跑）+ grep ALL_NAV_PATHS/levelOf 零命中。未做项：stores 六份手写 loading/try/assign 列表骨架（Finding 提及但 Plan 六组未分配，抽象收益低）；ProfileView/InviteView 拆卡片组件（可选项，按票不做）。
