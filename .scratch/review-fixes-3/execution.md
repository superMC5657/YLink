# Review Fixes 3 · 分段执行手册

> 用法：**一次会话只贴一段**（整块复制下方代码框内容），等该段全部完成、验证全绿后再贴下一段。
> 若某段中途失败：在同会话让 agent 修复后续跑；确认该段 tickets 全部 `resolved`、spec.md 对应行已 ✅，才进入下一段。
> 段内规则已写成自包含指令，新会话无需其他上下文；ticket 细节（证据/方案/验收）都在 `.scratch/review-fixes-3/issues/` 里，由执行方自行读取。

## 执行顺序与依赖

- 段 1（P0 安全）最先，段 2/3/4 是三条独立正确性线（先后随意，也可分不同会话并行推进，但每个会话只贴一段）。
- 段 5（渐变令牌化，纯机械大扫除）必须在段 8 之前合入，减少同文件冲突。
- 段 9 依赖段 6+7（文档对齐要等代码事实定稿）；段 10→11 有先后（先删死代码再收敛重复形状）；段 12 排在段 3/5/7/8 之后。
- P3（段 10-12）可拆散穿插到日常迭代，不阻塞前面任何段。

## 进度表

| 段 | issues | 主题 | 状态 |
|---|---|---|---|
| 1 | 01-04 | P0 安全与启动稳定 | ✅ |
| 2 | 05-08 | 后端正确性 | ✅ |
| 3 | 09-12 | 前端正确性 | ✅ |
| 4 | 13 | Tauri 深链与更新链路 | ✅ |
| 5 | 14 | 品牌渐变全量令牌化 | ✅ |
| 6 | 15 | 后端契约合规 | ✅ |
| 7 | 16 | 前端数据层合规 | ✅ |
| 8 | 17 | UI 规范批 | ✅ |
| 9 | 18 | 文档对齐批 | ✅ |
| 10 | 19 | 后端死代码清理 | ✅ |
| 11 | 20 | 后端重复形状收敛 | ✅ |
| 12 | 21 | 前端重复形状收敛 | ✅ |

---

## 段 1 · P0 安全与启动稳定（01-04）

```
你将执行 YLink 修复计划的第 1 段（共 12 段）：P0 安全与启动稳定。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（Go 后端 server/，规范见 AGENTS.md 与 docs/）。
任务：逐张完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按各自 Finding→Plan→Acceptance 执行，Acceptance 全部满足才算完成）：
  01-config-secret-failfast.md
  02-seed-uuid.md
  03-probe-hardening.md
  04-cron-lock.md
前置：无。提醒：01 完成后生产环境必须配齐真实密钥环境变量（fail-fast 会让缺失部署拒启，这是预期行为）。
规则：
- 每张 ticket 单独 commit（conventional commit 中文描述，如 fix(config): 生产密钥缺失时拒绝启动）；提交前验证命令全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入该 ticket 的 ## Comments，不顺手修。
- 无法本地验证的项目（如需 Docker）在 Comments 记录「未验证 + 原因」，不得谎报。
- 每张完成后：ticket 文件 Status 改为 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应「### [P0] 0N …」行标题打上 ✅。
- 收尾过一遍 AGENTS.md「文档同步检查表」。
验证：cd server 后 go build ./... && go vet ./... && go test ./... 全绿，gofmt -l . 无输出。
```

## 段 2 · 后端正确性（05-08）

```
你将执行 YLink 修复计划的第 2 段（共 12 段）：后端正确性。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（Go 后端 server/，规范见 AGENTS.md 与 docs/）。
任务：逐张完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按各自 Finding→Plan→Acceptance 执行）：
  05-stale-struct-updates.md
  06-tx-and-races.md
  07-async-and-crypto.md
  08-migrations-down.md
前置：第 1 段已完成（可查 spec.md 前四行是否 ✅）。
规则：
- 每张 ticket 单独 commit（conventional commit 中文描述）；提交前验证命令全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入该 ticket 的 ## Comments，不顺手修。
- 08 的 up→down→up 数据库演练若无本地 Docker/Postgres 环境，在 Comments 记录「未验证 + 原因」，不得谎报。
- 每张完成后：ticket 文件 Status 改为 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
- 收尾过一遍 AGENTS.md「文档同步检查表」（07 若调整邮件模板语义注意 docs/backend/core-flows.md）。
验证：cd server 后 go build ./... && go vet ./... && go test ./... 全绿，gofmt -l . 无输出。
```

## 段 3 · 前端正确性（09-12）

```
你将执行 YLink 修复计划的第 3 段（共 12 段）：前端正确性。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（Vue3+TS+Pinia+UnoCSS 前端 src/、pnpm，规范见 AGENTS.md 与 docs/frontend/）。
任务：逐张完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按各自 Finding→Plan→Acceptance 执行）：
  09-audit-clamp-tier.md
  10-admin-reload-rejections.md
  11-frontend-bug-batch.md
  12-node-install-dialog.md
前置：无（与后端段互不依赖）。
规则：
- 每张 ticket 单独 commit（conventional commit 中文描述，如 fix(admin): 审计分页档位钳制）；提交前验证命令全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入该 ticket 的 ## Comments，不顺手修。
- 每张完成后：ticket 文件 Status 改为 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
- 收尾过一遍 AGENTS.md「文档同步检查表」；11 若触碰契约形状（mock/类型）须对齐 docs/api/README.md 与 types/api.d.ts。
验证：pnpm lint && pnpm test 全绿（lint 已含 check-tokens-sync / check-admin-contract / check-error-toast）；有浏览器环境时跑 pnpm exec playwright test tests/e2e/withdraw.spec.ts tests/e2e/admin.spec.ts，无环境在 Comments 记录。
```

## 段 4 · Tauri 深链与更新链路（13）

```
你将执行 YLink 修复计划的第 4 段（共 12 段）：Tauri 桌面端。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（Tauri v2 在 src-tauri/，发布工作流 .github/workflows/release-tauri.yml，规范见 docs/frontend/desktop-tauri.md 与 AGENTS.md）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  13-tauri-deeplink-updater.md
前置：无。
规则：
- 单独 commit（conventional commit 中文描述，如 fix(tauri): 注册桌面深链并接通国内更新清单）；提交前验证全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入 ## Comments，不顺手修。
- 需真机/安装包才能验证的项目（ylink:// 实测、release dry-run）在 Comments 记录「未验证 + 原因」，不得谎报。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
- 本段含 desktop-tauri.md §3/§7 文档同步（AGENTS 检查表）。
验证：cd src-tauri 后 cargo fmt --check && cargo clippy 通过；根目录 pnpm lint && pnpm test 全绿；有环境时 pnpm tauri build 走通（无环境记录）。
```

## 段 5 · 品牌渐变全量令牌化（14）——机械大扫除，须在段 8 之前

```
你将执行 YLink 修复计划的第 5 段（共 12 段）：品牌渐变全量令牌化（纯机械替换，独立大 PR，先于其他 UI 段合入以减少冲突）。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（前端 src/、uno.config.ts、src/styles/，规范见 docs/frontend/design-system.md 与 AGENTS.md）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  14-gradient-token-sweep.md
前置：无（但必须排在第 8 段之前）。
规则：
- 单独 commit（如 refactor(theme): 品牌渐变与激活色全量令牌化）；提交前验证全绿。
- 只做 ticket 范围内的改动（约 30 处替换 + uno shortcut + theme.ts + check-tokens-sync MAPPING + design-system 数值对齐，清单都在 ticket 里）。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
验证：pnpm lint && pnpm test 全绿；ticket Acceptance 中的 grep（#6558f5 等字面量）零命中；暗色主题下主渐变/激活色随主题切换（截图或组件测试佐证）。
```

## 段 6 · 后端契约合规（15）

```
你将执行 YLink 修复计划的第 6 段（共 12 段）：后端契约与规范合规。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（Go 后端 server/，规范见 AGENTS.md、CONTEXT.md、docs/backend/data-model.md）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  15-backend-contract-conformance.md
前置：第 2 段已完成（同批 service/handler 文件先落正确性修复）。
规则：
- 单独 commit（如 fix(admin): 批量审计 target 统一 batch 摘要并收敛 IDParam）；提交前验证全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入 ## Comments，不顺手修。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
- 本段改审计 target 语义与 models tag → 必同步 docs/backend/data-model.md（补 agent_applies、used_count，索引对齐），过 AGENTS 文档同步检查表。
验证：cd server 后 go build ./... && go vet ./... && go test ./... 全绿，gofmt -l . 无输出；ticket Acceptance 的 grep 仅剩 helper.go 一处 ParseInt(c.Param。
```

## 段 7 · 前端数据层合规（16）

```
你将执行 YLink 修复计划的第 7 段（共 12 段）：前端数据层合规（两个决策点已拍板为「代码为准、文档随改」，照 ticket 内「已拍板」措辞执行，不要重新发起讨论）。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（前端 src/，规范见 AGENTS.md、docs/frontend/data-layer.md、docs/adr/）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  16-frontend-data-layer-conformance.md
前置：第 3 段已完成（ProfileView/useAdminNodes 相关改动先落）。
规则：
- 单独 commit（如 refactor(api): 补 ADR-0006 记录 seam 方向并对齐数据层文档）；提交前验证全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入 ## Comments，不顺手修。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
- 本段产出 docs/adr/0006 + data-layer.md/README §2 改写 + api.d.ts 契约同步，过 AGENTS 文档同步检查表（改契约须对齐 mock 与 types）。
验证：pnpm lint && pnpm test 全绿（含 adapter.spec 与 check-error-toast）；ticket Acceptance 的 grep 在用户端/组件层零命中。
```

## 段 8 · UI 规范批（17）

```
你将执行 YLink 修复计划的第 8 段（共 12 段）：UI 规范批（侧栏宽度决策已拍板为「代码为准、文档随改」：保留 200px/72px，改 pages.md）。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（前端 src/，规范见 docs/frontend/design-system.md、pages.md 与 AGENTS.md）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  17-ui-standards-batch.md
前置：第 5 段已完成（渐变已令牌化，避免同文件冲突）。
规则：
- 单独 commit（如 fix(ui): 响应式判定走 useMediaQuery 并收敛提现状态映射）；提交前验证全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入 ## Comments，不顺手修。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
- 本段含 pages.md §2.1 宽度描述更新（240px→200px），过 AGENTS 文档同步检查表。
验证：pnpm lint && pnpm test 全绿；有浏览器环境时 pnpm exec playwright test tests/e2e/withdraw.spec.ts，无环境在 Comments 记录；Acceptance 的 grep（bg-emerald- 等语义色）零命中、pages.md 无 240px 残留。
```

## 段 9 · 文档对齐批（18）——依赖段 6+7

```
你将执行 YLink 修复计划的第 9 段（共 12 段）：文档对齐批（以代码当前态为准改写全部陈旧文档）。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（docs/、server/Makefile、server/docker-compose.prod.yml，规范见 AGENTS.md、CONTEXT.md）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  18-docs-drift-batch.md
前置：第 6、7 段均已完成（Blocked by: 15, 16——文档要描述的代码事实已定稿；desktop-tauri §3/§7 还依赖第 4 段已完成）。
规则：
- 单独 commit（如 docs: 全量对齐后端/前端/桌面端文档至当前实现）；文档类改动以代码为准逐一核实后落笔，遵循「同一事实只在一处维护」。
- 无法本地验证的项目（make swagger 产物、docker compose config）有环境就跑，无环境在 Comments 记录「未验证 + 原因」。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
验证：ticket Acceptance 的 grep（至 0004/pkg/qrcode 等陈旧引用）零命中；make -C server swagger 产物落位 server/docs/；docker compose -f server/docker-compose.yml -f server/docker-compose.prod.yml config 中 caddy 无宿主 Caddyfile bind（无 Docker 环境则人工核对卷合并语义并记录）。
```

## 段 10 · 后端死代码清理（19）

```
你将执行 YLink 修复计划的第 10 段（共 12 段）：后端死代码清理（P3，可穿插日常迭代）。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（Go 后端 server/）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行；清单内每项删除前自行 grep 复核零引用）：
  19-backend-dead-code.md
前置：第 2、6 段已完成（相关 service 文件的功能性修改先落，再删码）。
规则：
- 单独 commit（如 chore(server): 清理零引用死代码与空壳抽象）。
- 清单外新发现的死代码只记入 ## Comments，不扩scope。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
验证：cd server 后 go build ./... && go vet ./... && go test ./... 全绿，gofmt -l . 无输出；ticket Acceptance 的 grep 零命中。
```

## 段 11 · 后端重复形状收敛（20）

```
你将执行 YLink 修复计划的第 11 段（共 12 段）：后端重复形状收敛与分层渗漏回收（P3，建议按 ticket 内 5 组拆多个 commit）。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（Go 后端 server/，规范见 docs/backend/README.md §2 分层约定）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  20-backend-smell-batch.md
前置：第 10 段已完成（死代码已删，重构面最小化）。
规则：
- 按 ticket 的 5 个分组各一个 commit（refactor(server): …）；提交前验证全绿。
- 只做 ticket 范围内的改动；范围外发现的新问题写入 ## Comments，不顺手修。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
- QueryResult 补 Amount 属契约增强 → 同步 docs/api/README.md 与前端类型，过 AGENTS 文档同步检查表。
验证：cd server 后 go build ./... && go vet ./... && go test ./... 全绿，gofmt -l . 无输出；N+1 修复有 sqlmock 语句数断言。
```

## 段 12 · 前端重复形状收敛（21）

```
你将执行 YLink 修复计划的第 12 段（共 12 段）：前端重复形状与死代码收敛（P3，建议按 ticket 内 6 组拆多个 commit）。
仓库：C:\Users\supermc\Workplaces\goland_workplace\YLink（前端 src/、mock/、tests/、scripts/，规范见 docs/frontend/data-layer.md 与 AGENTS.md）。
任务：完整执行以下 ticket（在 .scratch/review-fixes-3/issues/ 下，按 Finding→Plan→Acceptance 执行）：
  21-frontend-smell-batch.md
前置：第 3、5、7、8 段均已完成（相关组件/store 的功能与令牌化修改先落）。
规则：
- 按 ticket 的分组各一个 commit（refactor(frontend): …）；提交前验证全绿。
- 只做 ticket 范围内的改动；ProfileView/InviteView 大拆分是可选项，不做不阻塞验收；范围外发现的新问题写入 ## Comments。
- 完成后：ticket Status 改 resolved，## Comments 追加一行「<执行当天日期>: 已修复 <要点>」；在 .scratch/review-fixes-3/spec.md 把对应需求行标题打上 ✅。
验证：pnpm lint && pnpm test 全绿；有浏览器环境时跑 pnpm exec playwright test，无环境在 Comments 记录；ticket Acceptance 的 grep（ALL_NAV_PATHS/levelOf）零命中。
```

---

## 全部完成后的收尾

1. spec.md 标题行打 ✅、状态改为「全部修复完成」；进度表 12 格全部勾选。
2. docs/reviews/review-round-09.md 的「状态」行改为「已修复」，并在「变更/验证」节补记实际提交范围与验证结果。
3. 按 AGENTS.md 约定不追加流水账：progress.md 只更新能力行。
