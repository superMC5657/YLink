# [P1] 12 Lint与超大文件债务

Status: done
Type: task

## Finding

- `eslint.config.ts:60-61` 关 `no-explicit-any/no-v-html`;`admin_service.go:1011/order_service.go:750/AdminNodesView:796/AdminUsers:626` 超大;CI 仅 tag、Go 不进 CI;`shortcuts.ts` 疑与 uno.config 双维护;`tokens.css+theme.ts` 双源同步靠人。

## Task

1. `any` 先 warn 起步,`no-v-html` 建白名单注释约束;`shortcuts` 留一处为准;加 tokens/theme 同步校验脚本。
2. 后端按 admin_crud/stats/withdraw 模式续拆;前端 Nodes/Users 拆 table/dialog/composable(本单只拆 + 门禁绿,不改行为)。
3. `server/` 加 push main 的 `go vet/test`(即使不进 Actions,也给 Makefile+文档),前端 PR 加轻量 lint+typecheck。

## Done

- `pnpm lint/typecheck/test` + `go vet/test ./...` 全绿;CI 文档更新。

## Comments

- eslint:`no-explicit-any` warn(现有代码 0 处 any,`--max-warnings 0` 下新增即暴露);`vue/no-v-html` 升 error,4 处使用全部白名单注释化(markdown 渲染 3 处注明 DOMPurify 过滤,AppIcon 文件级豁免注明静态 path)。
- `src/styles/shortcuts.ts` 为空壳死文件已删,shortcuts 单源留 uno.config.ts。
- 新增 scripts/check-tokens-sync.mjs:tokens.css 与 theme.ts 11 组色值×亮暗双主题映射断言(比较时忽略空白差异),挂入 pnpm lint 链;首跑即抓到 dark border 的空格式漂移(误报形态),已通过归一化消除。
- 后端拆分(纯搬运,同包不改行为):admin_service.go 1014→543,拆出 admin_audit.go(214,审计日志查询+目标可读化)/admin_order.go(202,订单/退款)/admin_ticket.go(97,工单);order_service.go 758→340,拆出 order_coupon.go(162)/order_payment.go(208,收银台/回调/回执)/order_fulfill.go(89,开通/佣金)。
- 前端拆分(行为不变,58/58 e2e 回归):AdminNodesView 796→278,useAdminNodes composable + NodeFormDialog/GroupFormDialog/NodeBatchDialog/NodeSortDialog;AdminUsersView 629→261,useAdminUsers + UserBalanceDialog/UserRoleDialog/UserBatchBalanceDialog/UserMailDialog;弹窗经 defineModel/emit 与 composable 状态双向绑定,NodeFormDialog 因共享 reactive 表单对象做文件级 no-mutating-props 豁免(注明理由)。
- CI:ci.yml 加 `pull_request` 触发 → 新 frontend-pr-gate(lint+typecheck 轻量门禁);原 quality/e2e/rust 三 job 加 tag-only 条件,tag 全量语义不变。新增根 Makefile(go-vet/go-test/go-check/check 一键全量)与 server/Makefile vet 目标;顺带修复 server/Makefile 既有 bug(lint 目标被吞进 swagger recipe 行);docs/backend/README.md §9 同步。
