# 代码评审 — YLink 全量（前端 web + Tauri + 后端 Go）

- **日期：** 2026-10-25
- **范围：** 整仓全量（非增量 diff）：后端 Go 185 文件、前端 web 208 文件、Tauri 12 文件（icons、swagger 生成物、go.sum、pnpm-lock 按约定略读）。
- **方式：** 15 个并行分片逐文件审查，无抽样。Standards 轴以 AGENTS.md、CONTEXT.md、ADR-0001~0005、docs/{frontend,backend,api}/* 为准，Fowler 12 味基线作判断类补充；工具已强制项（gofmt/eslint/stylelint/prettier）跳过。Spec 轴核对 `.scratch/` 全部 11 个 spec + 25 个 issue。
- **状态：** 待修复 —— 修复台账见 `.scratch/review-fixes-3/`（21 张 ticket：P0×4 / P1×9 / P2×5 / P3×3）。
- **总结：** Spec 轴无严重不符（所有 ✅ 项均核实了代码机制而非仅标记；删除线项确认已从代码移除）。Standards 轴发现 [bug] 级 23 项、[hard] 硬违规约 35 项、[judgement] 异味约 55 项。最严重单项：`server/configs/config.yaml` 将开发 JWT 密钥作为静默默认值且生产无 fail-fast。

## 发现

### 横切系统性问题（[hard]，跨前端多片）

1. **品牌主渐变写死约 30 处，绕过已令牌化的颜色变量。** design-system.md §2.3 明言该渐变「已令牌化为 `var(--c-primary), var(--c-primary-grad-end)`，暗色自动切换」，README §8 禁止写死颜色；现状暗色主题下永远渲染亮色渐变。涉及：
   - `src/views/`：NotFoundView:17、ForbiddenView:17、DocsView:77、PortalView:55,59,74,95、TicketDetailView:137,151、AgentView:62,96、AdminTicketsView:322
   - `src/components/app/`：AppSidebar:47、AdminSidebar:51、DrawerMenu:64,103、AppHeader:122、CustomerServiceFab:33、UpdateCard:127
   - `src/components/ui|business/`：ThemeToggle:53、LanguageToggle:51、OrderTable:69、OrderCardList:58、SubscribeCard:40、AuthLayout:24,28,43、BannerStatCard:18,53,62、NoticePanel:78
   - `uno.config.ts:14`：`btn-primary` 一处波及 53 个用法
   - `ToastBridge.vue:60`：danger 渐变混用亮暗两套端点（最糟）
   - 正确范例：SharePanel.vue:249
2. **theme.ts 与 tokens.css 双源漂移，守门脚本漏检。** `src/styles/theme.ts:60` bodyColor `#F5F6FB` ≠ `--c-bg-app #eceef5`；`:13` 字体 Inter 优先 ≠ `--font-ui` YaHei 优先；`:29-37` `Pagination.itemColorActive`/`Switch.railColorActive` 钉在共享 `base`，暗色仍用亮色主色。`scripts/check-tokens-sync.mjs` MAPPING 未映射上述键，门禁放行漂移。
3. **文档↔代码漂移若干**（违反 AGENTS.md 文档同步检查表）：
   - data-model.md 缺 `agent_applies` 表（migration 0001:346 有表）与 `Coupon.used_count` 字段
   - docs/backend/README.md §3：`Generator.Build` 签名与现实不符；列出不存在的 `pkg/qrcode`
   - desktop-tauri.md §7（Windows-only、CI 三 job 等过时）、§4 托盘亮暗双资源未实现（实为静态单图标）
   - launch-checklist.md:58 / deploy.md:80：「迁移至 0004/0007」实为 0008
   - data-model.md:289「迁移插入首个管理员」实际是运行时 `EnsureAdmin`；`:3`「所有表含 created_at/updated_at」对 traffic_logs 等不成立

### 后端 Go — 正确性/安全缺陷（[bug]）

| # | 位置 | 问题 |
|---|---|---|
| 1 | `server/configs/config.yaml:10,16,20` | 开发密钥（`ylink_root`/`ylink_redis`/`dev-only-secret`）作为静默默认值提交，`config.Load` 无生产 fail-fast：漏配 `APP_JWT_SECRET` 即用公开密钥签发 token |
| 2 | `repo/admin.go:422-428,447-453` | `EnsureAdmin`/`EnsureDemoUser` 未填 `UUID`：0004 后 `uuid NOT NULL+UNIQUE`，首次插 `''`、第二次撞唯一约束（demo 必失败）；两账号订阅凭证为空、模式 A 归因失效 |
| 3 | `handler/health.go:46-52` | `dbPing` 用 `context.Background()`，`/readyz` 2s 超时只管 Redis，DB 挂起探针无限挂起 |
| 4 | `handler/health.go:36,40` | `/readyz` 把 `err.Error()`（含 DB 主机端口）返回给公网未鉴权调用方 |
| 5 | `handler/telegram.go:57-63` | Webhook 先 bind body 再校验 secret：坏 JSON + 错 secret 得 200 而非契约 §19 的 403 |
| 6 | `service/cron_schedule.go:43-49` | 分布式锁两缺陷：`SetNX` 错误与锁被占混为一谈（Redis 故障静默杀死全部 cron）；`defer Del` 无属主 token，超 TTL 后删掉继任者的锁导致并发双跑 |
| 7 | `service/user_service.go:91` + `subscribe_service.go:113` | `Update(u)` 全结构体陈旧快照落库：改密/重置订阅期间并发的余额/流量/佣金写被覆盖（telegram.go:178 已文档化并规避同风险） |
| 8 | `service/admin_ticket.go:59-79` | ReplyTicket 事务外检查 `status==2` 后无条件置 1：并发关闭的工单被翻回「已回复」，绕过 reopen-once 状态机 |
| 9 | `repo/admin.go:323-331` | `TrafficLogRepo.Upsert` check-then-write，并发导入双插触发 `uk_traffic_user_date` 500 |
| 10 | `service/order_payment.go:188` | `sendReceiptMailAsync` 无 `recover`，panic 带崩 API 进程（repo 自身 deliverAsync 政策要求带 recover） |
| 11 | `service/auth_service.go:114-122` | 验证码 Get→比对→Del 并发窗口可复用一次性验证码（tg-bind 流程已用 `GetDel` 正确写法） |
| 12 | `service/auth_service.go:286-291` | `issueSession` 忽略 `rdb.Set` 错误：refresh token 可能未入白名单 |
| 13 | `service/invite_service.go:132` | `rand.Int` 出错时 `n` 为 nil → panic |
| 14 | `pkg/mailer/mailer.go:38-41` | `Send` 无 context/超时，调用方均 fire-and-forget：SMTP 挂起 → goroutine 无限泄漏 |
| 15 | `migrations/0008.down` | 不删 up 种入的 `settings.telegram` 行 |
| 16 | `migrations/0007.down:11` | 重建全量 `UNIQUE(order_no)` 而提现行 `w<N>` 仍存留 → 回滚后约束冲突 |
| 17 | `cmd/worker/main.go:86` | `zap.String("tz", cfg.App.Name)` 把应用名记到 tz 键 |

### 后端 Go — 硬违规（[hard]）

- `service/admin_node_batch.go:70,113`、`admin_content_manage.go:27,39`、`admin_traffic_reset.go:25` — 批量动作审计 target 写 `""`，违反 CONTEXT.md「批量 target 只存 `batch:<count>`」（仅 send_mail/traffic_reset 合规）。
- `handler/ticket.go:71,96,121,144` + `handler/content.go:81` — 5 处手写 `ParseInt...40000` 绕过 `IDParam`（helper.go 自称唯一 owner）。
- `model/models.go` — tag 与 data-model/migration 漂移 6 处：KnowledgeCategory 唯一索引缺 language（:286）、TrafficLog 缺 UNIQUE 复合（:339-341）、order_no 注释/索引与部分唯一约束矛盾（:208,213）、users/orders/mail_logs/traffic_reset_logs/commission_withdraws/knowledges 复合索引仅声明首列。
- `server/Makefile:17` — `swag init -o doc` 输出到 `doc/`，产物实际在 `docs/`，每次重生成产出孤儿目录。
- `server/docker-compose.prod.yml:19-27` — caddy 卷合并后宿主 Caddyfile bind 仍生效，deploy.md §3.1「不再 bind mount」不成立（需 `!reset`）。
- `middleware/metrics.go:61-63` — 未匹配路由以原始 `URL.Path` 作 Prometheus label → 404 扫描造成无基数上限。
- `internal/config/config.go:113-120` — `SiteConfig` 零消费方（Speculative Generality，硬违规边缘）。

### 后端 Go — 异味（[judgement]，择要）

- **死代码**：`cron_service.go:193` sendExpireMail、`order_coupon.go:267` couponCode、`order_service.go:152-159` priceOf/planPrices、`session.go:22` bumpSessionVersion、`subscribe_service.go:294` toNode、`subscription_visibility.go:25` 空壳结构体、`pkg/validate` 整包零引用、`payment.go:67-73` Available()、`subscribe.go:16` Node.Cipher、`session_revocation.go:65` 丢弃的 reason 参数、`order_service.go:31`/`server_service.go:18`/`ticket_service.go:19` 注入未用字段。
- **重复形状**：repo 层 11 处 `Count+Order+Offset/Limit` 样板；`order_coupon.go:278-301` 手写 contains/min（Go 1.26 有 slices.Contains/min）；settings-JSON-int 解析 4+ 种写法；SortNotices/Knowledges/Servers 三份同形；GB→bytes 两处；handler 用户/管理端 order 状态解析块重复、可选 body bind ×4；`epay.go:149-171` verify 复刻 sign 且非常量时间比较。
- **分层渗漏**：`admin_crud.go:272-295,330-369`/`admin_stats.go:41-66`/`telegram.go:109,180,197`/`ticket_withdraw_lifecycle.go:183` service 直写 GORM 绕过 repo；`setting_service.go:66`+`telegram.go:68` 零值 `SettingRepo{}.Get` 绕过自家 60s 缓存且每笔支付订单跑 DB。
- **N+1**：`admin_order.go:40-44`、`admin_agent.go:31-35`、`admin_stats.go:139`。
- **事务/审计**：`admin_user.go:42-77` UpdateUser 无事务；~10 处 `_ = m.audit.write(...)` 静默丢审计失败。
- **其他**：工单/申请状态魔法数字；`rate_limit.go:19` 手拼 Redis key；`node_service.go:250` rate≤0 静默计费 0；`payment.go:37-40` QueryResult 无 Amount（对账无法核对金额）；字符串 context key（auth 两处）；`subscribe_service.go:175,210` "YLink" 硬编码；config.yaml:48-54 与 0001 种子邀请分成默认值分叉（30/40 vs 40/50）；node-agent 计数器 rng 伪造但标「生产/演示」；docker-compose 全服务 container_name 阻止扩容、无 restart 策略；Dockerfile 以 root 运行；`resp.go:78` 硬编码 40000 重复 errs.ErrParam；测试错置若干（sanitize/coupon 测试在 admin_service_test、userRows 重复）。

### 前端 web — 正确性缺陷（[bug]）

- `composables/useAdminList.ts:74,130` + `stores/admin/system.ts:37` — 钳制只建模 Standard{10,50}，`clampPageSize(100)→10`；审计页提供 100/页（ADR-0004 保留 Audit 档），选 100 实际拉 10。
- `useAdminNodes.ts`（8 处）+ `useAdminUsers.ts`（5 处）— `void load()` 而 load 设计为 rethrow：刷新失败即未处理 Promise 拒绝。
- `mock/admin.ts:756-761` — `command_template` 默认值生成 `--node-id/--node-key {x}` 命令，`deploy/install-agent.sh:36-44` 不识别会 exit 1；后端 0009 规范为 `--endpoint {{endpoint}} --key {{node_key}}`。
- `TrafficView.vue:120`、`AdminReportsView.vue:306` — `cssVar` 回落值与令牌不符（pink 回落成渐变紫、olive/warning/blue 各偏色）。
- `AdminTrafficImportView.vue:68` — 重置行 `toISOString().slice(0,10)`（UTC）与同文件 `localDateKey` 混用，东八区 8 点前差一天；`:321` `toLocaleString()` 绕过 formatTime。
- `OrderDetailModal.vue:159-163` — 支付方式硬编码，无视 `config.paymentMethods`（OrderConfirmModal 是配置驱动）。
- `ImportClientSheet.vue:71` — icon key `box` 不存在于 AppIcon PATHS，sing-box 回落 info 图标。
- `NodeInstallDialog.vue:38-42` — 本地重实现 `resolveApiBase` 且忽略运行时 apiBase 覆盖（http.ts:38 文档化的 P1 陷阱）：运行时改地址后安装命令指向错误端点。

### 前端 web — 硬违规（[hard]）

- `ProfileView.vue:16,153,179,198,213` + `useLocalNotifications.ts:34` — 用户端视图直调 `apiUser`，违反 data-layer.md §2（`useUserStore` 缺 sessions/telegram actions）。
- `InviteView.vue:42-49` + `AdminTicketsView.vue:124-131,281-287` — 提现状态本地映射表 ×2，绕开 `utils/statusMeta.ts` 唯一归属。
- `NodeInstallDialog.vue:10,63,88` — 组件直调 `apiAdmin`（data-layer 的 admin 豁免仅限 views/admin/*）；`:110` 硬编码中文 toast。
- `AppHeader.vue:27` — `computed(() => window.innerWidth < 768)` 无响应源；design-system §6.1 要求 useMediaQuery。
- `AppSidebar.vue:3,38` — 宽度 200px vs pages.md §2.1 记载 240px/72px。
- `OrderConfirmModal.vue:145` — 优惠券错误双重反馈：内联渲染但 transport 未加 `silent: true`（§7 约定）。
- `ForgotView.vue:100` — 缺 `:show-feedback="false"` 与 4px 间距（pages.md §3.2「与登录页一致」）。
- `AdminNodesView.vue:175,178,231` — 在线点用 `bg-emerald-400/500` 调色板类（README §8 禁止）。
- `AppIcon.vue` vs README §2/design-system §8 — 文档规定 @iconify/vue，实际内联 Lucide（有意为之但文档未更新）。
- `api/{agent,invite,knowledge,notice,order,plan,server,ticket,user}.ts` — c47bfb0 后 9 个 api 模块变成从 stores 反向 re-export 的 seam，分层方向倒置且无 ADR；data-layer.md §2 仍写「store 是唯一允许调用 api 模块的地方」并列出不存在的 `useSubscribeStore`；auth.ts 又保留旧方向（两种模式并存）。

### 前端 web — 异味（[judgement]，择要）

- **重复**：AdminNotices≡AdminKnowledges 排序弹窗逐字重复（NodeSortDialog 已在）；非分页 CRUD 脚手架四份同形；Register≡Forgot 邮箱验证码块；NotFound≡Forbidden；`.slice(0,N).replace('T',' ')` ×5；payMethodLabel 三处实现（format.ts 为正源）；LanguageToggle≡ThemeToggle 滑块逻辑；SubscribeCard 五格同块；PaymentModal 倒计时胶囊 ×2；QuickActionGrid 动作表+switch 双记账与 3 个复制按钮；mock `ok()/unauthorized()` ×6；e2e loginAs/enterPortal 复制；dev.sh/dev-docker.sh ensure_image 重复；http.ts:152-158 重实现 buildQueryStr；useAdminUsers 重复实现批量明细格式化。
- **死代码**：`nav.ts:98` ALL_NAV_PATHS、`ticket.ts:90` levelOf、`useAdminList.ts:194-207` reportBatchResult 仅测试消费、`adminList.ts:33-37` batchAuditTarget 零生产调用（mock 手写 `'batch:2'`，注释声明失实）。
- **杂项**：ProfileView 592 行/InviteView 568 行多域混杂；`agent.ts` 传输层 string 后强转；6 份手写列表骨架；UserBalanceDialog 缺 null 守卫；UserMailDialog 手写 v-model；NodeInstallDialog 3 处未翻译文案；CustomerServiceFab watch 可改 computed；AppSidebar 陈旧注释「13 项」（实 18）；i18n 测试无 zh↔en 键齐平断言；server-status 选项数组两处内联；AdminTicketsView:65 `openDetail(t)` 遮蔽 i18n `t`。

### Tauri

- [bug] `tauri.conf.json:50-53` — deep-link 只声明 `mobile`，无 `desktop.schemes` 也未调 `register_all()`：NSIS 模板（installer.nsi:680-685）不写注册表，`ylink://` 桌面深链整条链路（desktop-tauri.md §3/§4）不可达。
- [bug] `release-tauri.yml:181-198` — 双清单半接线：`latest-cn.json`（gh-proxy 加速）无任何 updater endpoint 引用，端点只拉 `latest.json` → 国内下载加速到不了二进制。
- [hard] `capabilities/default.json:8-13` — 六个 `core:window:*` 授权零 JS 调用，违反 §3「最小授权」。
- [hard] desktop-tauri.md §7 陈旧、§4 托盘亮暗双资源未实现（见横切第 3 条）。
- [judgement] Cargo.toml serde/serde_json 未使用；`log::info!` 无 logger 插件为静默 no-op；`lib.rs:92` unwrap → expect；987 行 vendored NSIS 模板漂移风险（头部已有 re-diff 约定）。

### 合规确认（重点核实项）

ADR-0001（PageOK nil→[] 唯一 owner）、ADR-0002（admin DTO 单 owner + 生成链 + mock/HAND-VALUE 对齐）、ADR-0003（transport/core 分层，api/admin.ts 纯 seam 74 名锁定）、ADR-0004（后端钳制单源，service 层零手写钳制，审计档路由正确）、ADR-0005（门面已退役）全部成立；Redis key 命名与 data-model §3 全对齐（含 TTL）；模式 A 差分计费/模式 B 覆盖校准语义一致；migration 0001-0007 up/down 与 data-model §2 逐列核对一致；obs 全栈指标名与 middleware 导出精确匹配；版本号 0.1.1 四处同步由 bump-version.mjs 单点负责；e2e/.env 均为占位符无真实密钥；无 authz 缺口（用户端按 UserID 圈定、admin 组 Auth+RequireRole(1)、node 走 NodeAuth）。

### Spec 轴（.scratch/ 全量核对）

**无严重不符。** 所有 ✅ 项核实了代码机制，删除线项确认缺席。轻微发现：

1. clean-architecture「流量重置列表直接赋值 `res.list`」被 useAdminList 重构稀释：`useAdminList.ts:81` 回退 `res.list ?? []` 回来了（可辩护为 composable 兼容 PageOK 空列表，但与 spec 字面不符）。
2. clean-architecture「会话 CreatedAt 恒有值」后端已同步，但 `src/types/api.d.ts:466` 仍是 `string | null` + 已删除的「历史白名单值」注释（违反 AGENTS 契约同步清单）。
3. opt-perf-sec ticket09「collect 步骤缺 .sig 即失败」不准确：实际兜底在 `build-latest-json.mjs:88`（publish 阶段），整体保证仍成立。
4. spec 文本过时（非缺陷）：admin-console-split A6 的 `/admin/*→dashboard` 已被 ticket10 超越为 `/403`；A3「13 项菜单」现为 18 项。

## 修复台账

见 `.scratch/review-fixes-3/`（spec.md 为状态台账，issues/01~21 为可执行 ticket）：

| 优先级 | tickets | 主题 |
|---|---|---|
| P0 | 01-04 | 密钥 fail-fast、种子 UUID、探针/Webhook 加固、cron 锁 |
| P1 | 05-08（后端）、09-12（前端）、13（Tauri） | 正确性缺陷 |
| P2 | 14-18 | 渐变令牌化扫荡、契约/分层/UI 规范、文档对齐 |
| P3 | 19-21 | 死代码清理、重复形状收敛 |

## 验证

本轮为只读评审，未改代码。修复验证门槛（每张 ticket 的 Acceptance）：`go test ./...` + `gofmt -l` 无输出；`pnpm lint && pnpm test`（含 check-tokens-sync / check-admin-contract / check-error-toast）与 e2e；涉及 Tauri 的跑 `tauri build` 或 release dry-run。
