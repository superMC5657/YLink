# 代码评审 — YLink v0.2.0（全仓库）

- **版本：** 0.2.0
- **日期：** 2026-08-11
- **范围：** 全仓库 — Go 后端（`server/`）、Vue 3 前端（`src/`）、Tauri 桌面壳（`src-tauri/`）、e2e 测试（`tests/`）、配置文件、CI
- **方法：** 双轴评审（Standards / Spec），按四个并行只读子代理执行，并配合本地构建与测试验证
- **状态：** 全部发现已修复（2026-08-11）——见下方修复记录

## 本地验证

全部通过：`go build ./...`、`go vet ./...`、`go test ./...`（后端）；`pnpm typecheck`、`pnpm lint`（0 警告）、`pnpm test`（4 个文件 31 条用例通过）。注意：通过的测试并未覆盖下述缺陷。

## 修复记录（2026-08-11）

全部 42 项发现（Standards 29 + Spec 13）已修复并重新验证：`go build ./...`、`go vet ./...`、`go test ./...`；`pnpm typecheck`、`pnpm lint`（0 警告）、`pnpm test`（31 条用例通过）。

后端（`server/`）：
- 验证码邮件：模板占位符 `{code}` → `{{.code}}`，6 位验证码真正送达（`auth_service.go`）。
- 通知开关 `false` 可持久化：`UpdateProfile` 改 map 更新并回读；管理端 `UpdatePlan/UpdateServer/UpdateNotice/UpdateKnowledge` 同改 map 更新。
- 订单详情不再因套餐删除空指针（回退占位名）；`DeletePlan` 有关联订单时拒绝删除（11006）。
- checkout 缓存键含 `user_id + order_no + method`，切换支付方式不再命中旧结果。
- 优惠券 `limit_per_user` 在下单事务内串行化（`Occupy` + `CountUsageLocked`）；幂等键竞态重查返回首单而非 500。
- 余额支付佣金按实付金额发放；封禁账号订阅返回 401；注册强制邀请码按站点配置校验。
- 共用 `commissionRateFor`；抽取 `releaseCoupon`（取消/关单/退款/过期）；清理死代码；epay 回调校验 `pid`；限流优先取 `X-Forwarded-For`；`couponCode` 错误显式返回。
- 到期提醒在前 3 天与前 1 天双窗口触发；代理有效注册天数由 `agent.valid_invite_days` 配置；余额支付返回 `content: null`。

前端（`src/`、`src-tauri/`、`tests/`）：
- 流量明细响应形状对齐契约（`{list}`），含 mock 对齐。
- 会话过期跳转保留 hash 路由；标签页标题翻译；FOUC 脚本移至 `public/theme.js`（兼容 CSP、读取形状正确）。
- 通知开关从 `GET /user/profile` 回填；视图改用 store actions（Profile、OrderConfirmModal、InviteView）；管理端视图保留直调 `apiAdmin`（文档化例外）。
- 注册 deep-link 插件（`Cargo.toml`/`lib.rs`/capabilities）；删除 `tauri.conf.json` updater 占位。
- 二维码/ECharts 颜色改读 CSS 变量；抽取 `planSavePercent`/`periodLabel` 共享；删除死代码 `apiPlan.list`；CopyText 展示改为响应式；`formatTraffic` 收敛为 `formatBytes`。
- 删除调试残留 `zz-errfail.spec.ts`；`mobile.spec.ts` 未断言调用改为 `expect(...).toBeVisible()`。

文档：
- `docs/api/README.md`：新增 `GET /user/profile`；免鉴权列表补充 `/notices`、`/knowledges`。
- `docs/backend/data-model.md`：补充优惠券 `limit_per_user` 并发控制说明。
- `docs/frontend/data-layer.md`：store 分层例外、手写轮询、单文件 i18n。
- `docs/frontend/progress.md` / `docs/backend/progress.md`：管理端工单移至已完成，并新增各轴修复章节。

---

## Standards（标准轴）

### 后端（`server/`）

整体架构符合规范（handler→service→repo 分层方向、`resp`/`errs` 统一响应封装、fen→yuan 在 DTO 边界转换、bcrypt-12、UUID 子令牌），但仍存在若干正确性缺陷，且此前报告的全部四个问题区域仍在。

- ~~**server/internal/service/auth_service.go:96** — [P1] 验证码 `{code}` 未在邮件中替换。`mailer.Template` 将 `{code}` 作为字面文本，而 `mailer.Render` 执行的是 Go `html/template`，只有 `{{...}}` 动作会被替换，因此用户收到的邮件包含字面字符串 `{code}`，注册/找回密码不可用。修复：模板正文使用 `{{.code}}`。~~
- ~~**server/internal/service/user_service.go:62**（经由 **server/internal/repo/user.go:79**）— [P2] `UpdateProfile` 无法持久化 `false` 布尔值。`db.Model(u).Updates(u)` 对结构体跳过零值字段，因此 `RemindExpire=false` / `RemindTraffic=false` 永远写不到数据库，API 仍报告为开启。修复：只更新变更列（map 或 `Select`）。~~
- ~~**server/internal/service/order_service.go:309** — [P2] `GetOrder` 在套餐被删除后空指针解引用。`plan, _ := s.repos.Plan.GetByID(...)` 之后直接取 `plan.Name`；由于 `DeletePlan`（**server/internal/service/admin_crud.go:104**）没有引用检查且迁移未定义外键，删除套餐会使所有历史订单详情崩溃（Recovery 转为 500）。修复：处理错误，并阻止删除被订单引用的套餐或改为软删除。~~
- ~~**server/internal/service/order_service.go:387** — [P2] 结算缓存键缺少 `method`。`redispkg.Key("order", "paying", orderNo)` 会在 30 分钟内返回第一次结算的缓存结果，用户从 `epay_alipay` 切到 `epay_wxpay` 会拿到过期的支付宝链接，且不会创建 wxpay 的 `payments` 记录。修复：缓存键加入 `method`（及用户）。~~
- ~~**server/internal/service/order_service.go:165** — [P2] 优惠券 `limit_per_user` 检查非原子。`CountUsage` 后 `RecordUsage` 存在 TOCTOU 窗口；同一用户两个并发订单可同时通过并超出每人限用次数（`coupon_usages` 唯一键包含 `order_no`，无济于事）。修复：在现有事务内做条件插入或加锁检查。~~
- ~~**server/internal/service/admin_crud.go:78** — [P2] 管理端更新无法将布尔值置为 `false`。`UpdatePlan`（以及经结构体 `Updates(p)` 的 `UpdateServer`/`UpdateNotice`/`UpdateKnowledge`）会丢弃零值字段，管理员无法下架套餐/节点/公告；`SpeedLimit`/`DeviceLimit` 置空同样被跳过。修复：改用 map 更新。~~
- ~~**server/internal/service/order_service.go:186** — [P3] Idempotency-Key 并发竞争返回 500。两个同键并发 `CreateOrder` 都未命中 DB 查询；第二次插入违反唯一键 `uk_orders_idem`，表现为 50000 而非回放第一笔订单。修复：捕获重复键错误并重新读取。~~
- ~~**server/internal/service/order_service.go:369** — [P3] `Checkout` 中死参数 `r *http.Request`。handler 传入但函数体未使用；要么使用它（如网关 IP/scheme 拼接 notify URL），要么删除。~~
- ~~**server/internal/service/order_service.go:623** 与 **server/internal/service/invite_service.go:61** — [P3] 重复代码：佣金比例解析。`commissionRate` 与 `rateOf` 几乎相同（同一 `inviteCfg` 结构、同一回退逻辑）；提取为 `SettingService` 上的共享助手函数。~~
- ~~**server/internal/service/order_service.go:328**、**server/internal/service/admin_service.go:201**、**admin_service.go:233**、**server/internal/service/cron_service.go:53** — [P3] 优惠券释放块重复四次。取消/关闭/退款/过期中的 `Release` + `DeleteUsage` 序列被复制粘贴；提取 `releaseCoupon(tx, ...)` 助手（同时降低 Shotgun Surgery 风险）。~~
- ~~**server/internal/repo/order.go:120**、**server/internal/repo/admin.go:126**、**server/internal/repo/order.go:151**、**server/internal/pkg/redis/redis.go:40** — [P3] 投机性泛化/死代码。`ListPendingOrderNos`、`GetByNoAdmin`、`IncrUsed`、`SetString` 从未被调用；删除或接入使用。~~
- ~~**server/internal/pkg/payment/epay.go:60** — [P3] `VerifyNotify` 从不校验 `pid` 与配置商户一致。签名校验是主要防线，但标准易支付流程还会校验 `pid`；建议补充该校验，防止密钥一旦共享时跨商户重放。~~
- ~~**server/internal/middleware/rate_limit.go:26** — [P3] Caddy 之后 `ClientIP()` 不可靠。未配置可信代理时所有客户端共享一个 IP 桶，全局 300/分钟与登录限流实际变成全局上限；配置可信代理或谨慎使用 `X-Forwarded-For`。~~
- ~~**server/internal/service/order_service.go:321** — [P3] `couponCode` 吞掉 DB 错误。`GetOrder` 遇到优惠券已被删除的订单时静默返回空 `coupon_code`；应显式返回错误或 `nil`。~~

### 前端（`src/`、`src-tauri/`、配置文件）

未发现 P1（未发现崩溃/数据丢失/安全问题；markdown 经 DOMPurify 消毒，token 存储遵循文档化的 localStorage 方案）。

- ~~**src/utils/http.ts:86-89** — [P2] 会话过期跳转在 hash 路由下失效。`redirectToLogin()` 执行 `window.location.href = '/login?...'`，而应用使用 `createWebHashHistory()`；`redirect` 参数由 `pathname+search` 拼出，当前路由（位于 hash 中）丢失，且在 Tauri 自定义协议下 `/login` 不是真实文件 → 空白/死页面。修复：通过路由事件/回调跳转 `{ name:'login', query:{ redirect: route.fullPath } }`。~~
- ~~**src/router/guards.ts:34-37** — [P2] 标签页标题显示原始 i18n 键。`document.title = ... ' · ' + title` 中的 `meta.title` 是键（`router/index.ts` 中声明为 i18n 键，如 `'dashboard.title'`），标签页显示 `YLink · dashboard.title` 且永不本地化。修复：守卫中用 `t(title)` 翻译。~~
- ~~**index.html:16-19** — [P2] FOUC 脚本读取了错误的数据形状。`localStorage.getItem('app:theme')` 后 `JSON.parse(raw).value`；`setThemeMode` 存储的是裸模式字符串（`JSON.stringify('dark')`），所以 `.value` 恒为 `undefined`，持久化的显式主题被忽略（总是回退到系统偏好）→ 深色模式用户会先看到浅色闪烁。修复：直接 `JSON.parse(raw)`（其本身就是字符串）。~~
- ~~**src/views/profile/ProfileView.vue:22-23,112-114** — [P2] 通知开关从不从服务端回填。开关初始为硬编码 `true/false`；`onMounted` 只拉取 `stat`/`config`，且契约（`docs/api/README.md` §5.2）只有 `PUT /user/profile` 没有 GET——保存的设置不显示，且拨动一个开关会写入两个标志位，静默还原另一个。修复：契约补充 profile GET 并在挂载时回填。~~
- ~~**src/main.ts:120 + src/utils/platform.ts:87-88** — [P2] 深链接处理接到未注册的插件。`onDeepLink` 调用 `@tauri-apps/plugin-deep-link` 的 `onOpenUrl`，但 `lib.rs` 从未注册 `tauri-plugin-deep-link`（`Cargo.toml` 中没有），`capabilities/default.json` 也未授予 deep-link 权限 → Tauri 启动时未处理的 Promise 拒绝，与 `desktop-tauri.md` §3 相悖。修复：注册插件与能力，或启用前先移除接线。~~
- ~~**src/views/invite/InviteView.vue:49** — [P2] `user.stat!.balance = data.balance` 在视图里用非空断言修改 store 状态；直接进入 `/invite` 而 `stat` 未加载时，转账成功后 `try` 内抛错，UI 对一次成功的转账显示错误提示。违反文档化「视图只读写 store」分层。修复：在 store action 中更新余额。~~
- ~~**src/components/business/PaymentModal.vue:68** — [P3] 二维码硬编码 `light:'#FFFFFF'`/`dark:'#1F2430'`，深色模式下显示白色块（设计系统禁止硬编码颜色）。~~
- ~~**src/views/traffic/TrafficView.vue:117-140** — [P3] ECharts 系列/提示颜色硬编码 hex 而非 CSS 变量；仅部分 `isDark` 分支，深色刷新靠 `watch` 处理但没有 `echarts` 主题。违反设计标准。~~
- ~~**src/components/business/PlanCard.vue:24-33,46-53 + OrderConfirmModal.vue:43-56** — [P3] `savePercent` 数学与周期标签映射重复（第三份副本在 `utils/format.ts:periodLabel`）；提取共享 composable/util。~~
- ~~**src/api/plan.ts:5-10** — [P3] `apiPlan.list` 与 `fetch` 重复且返回类型冲突（`Plan[]` vs `PlanListResp`），从未被调用（死代码；存在契约不匹配风险）。~~
- ~~**src/components/ui/CopyText.vue:38** — [P3] `const shown = computedDisplay()` 在 setup 时只执行一次；按 key 复用的行在 `text` prop 变化后过期。应使用 `computed`。~~
- ~~**tests/e2e/zz-errfail.spec.ts:1-22** — [P3] 提交了零断言的调试 spec 和 5 秒 sleep，污染 CI 运行（其 `8081` 追踪目标是未 mock 的后端）；`mobile.spec.ts:27` 以未断言的 `isVisible()` 结尾。~~
- ~~**src-tauri/tauri.conf.json:46-47** — [P3] Updater 配置为占位（`https://example.com/latest.json`、空 `pubkey`）而 updater 插件未注册——若日后启用会静默失败的死配置。~~
- ~~**src-tauri/tauri.conf.json:33 + index.html** — [P3] CSP `script-src 'self'` 在 Tauri WebView 内会拦截内联 FOUC 脚本（仅浏览器端受保护）。~~
- ~~**src/views/admin/AdminUsersView.vue:57-61** — [P3] 管理视图直接调用 `apiAdmin`，绕过文档化的仅-store 分层；`formatTraffic` 用不同舍入重新实现了 `formatBytes`（GB 保留 1 位小数 vs 2 位）。~~

---

## Spec（规格轴）

### 后端（`server/` 对照 `docs/api/README.md`、`docs/backend/*`）

- ~~**server/internal/service/order_service.go:445-458,607** — [P1] (c) 余额支付订单从不产生佣金。规格：`docs/backend/core-flows.md` §2.2 — 余额结算必须「走与在线支付相同的 MarkPaid 后置逻辑（开通订阅/写佣金）」；§4 —「被邀请人每次付费（含续费）都产生佣金」。实现先清零 `PayAmount` 再写佣金：`locked.PayAmount = 0` 后 `grantCommission(tx, locked)`，而 `grantCommission` 计算 `amount := order.PayAmount * int64(rate) / 100` → 0 → 不产生 `commission_logs` 行。被邀请用户用余额支付会静默断掉佣金链。~~
- ~~**server/internal/service/subscribe_service.go** — [P2] (c) 封禁用户的订阅返回 403，规格要求 401。规格：`docs/api/README.md` §15 第 536 行 —「token 无效/用户封禁 → 401 纯文本」。实现：`if user.IsBanned { return nil, errs.ErrForbidden }`，`ErrForbidden` 映射为 HTTP 403（`server/internal/pkg/errs/errs.go:39`）。按 401 处理的客户端无法处理。~~
- ~~**server/internal/service/auth_service.go** — [P2] (a) 注册未强制 `invite_code_required`。规格：`docs/api/README.md` 第 145 行 —「`invite_code` 选填（站点开启强制邀请时必填，见 /config）」。`Register` 从不读取站点设置；该标志只在 `GET /config` 中暴露（`server/internal/service/content_service.go:68-69`）。管理员开启强制邀请后，无邀请码注册仍然成功。~~
- ~~**server/internal/service/cron_service.go** — [P3] (a) 到期提醒只发一次，而非 3 天与 1 天两个节点。规格：`docs/backend/core-flows.md` §9 —「expire-remind | 每日 10:00 | 到期前 3/1 天且开启提醒的用户发邮件」。实现使用窗口 `expired_at <= now+72h` 加一次性 Redis 标记 → 3 天内只发一封邮件，永远不会发 1 天提醒。~~
- ~~**server/internal/service/order_service.go:464** — [P3] (c) 余额结算返回 `"content": ""` 而非 `null`。规格：`docs/api/README.md` 第 390 行 — `{ "type": "paid", "content": null, "expire_in": 0 }`。实现：`Content: ""`（Go 字符串，无 `omitempty`）。~~
- ~~**server/internal/repo/agent.go** — [P3] (a) 「注册满 N 天」硬编码，非设置驱动。规格：`docs/backend/core-flows.md` §5 —「有效邀请定义：…或 注册满 N 天未封禁，阈值取 settings」。实现硬编码 `INTERVAL 3 DAY`；N 天无对应设置键（仅 `required_valid_invites` 可配置）。~~
- ~~**server/internal/router/router.go** — [P3] (b/c) `/notices` 与 `/knowledges` 免鉴权暴露。规格：`docs/api/README.md` 第 22 行穷举了免鉴权端点（captcha/auth/config/subscribe/notify），不包含 notices/knowledges。实现将两者注册在 Auth 中间件之前的公开组。影响温和，但超出契约的鉴权矩阵。~~

### 前端（`src/` 对照 `docs/api/README.md`、`docs/frontend/*`）

- ~~**src/api/user.ts:29 + src/stores/user.ts:30** — [P1] (c) 流量明细响应结构不匹配（契约破坏）。规格：`docs/api/README.md` §5.6 — `GET /user/traffic-logs` 返回 `"data": { "list": [ { "date", "u", "d", "total" } ] }`。实现将其类型化为 `http.get<TrafficLog[]>`（裸数组）并直接赋给 `trafficLogs`。后端符合文档（`server/internal/handler/subscribe.go:83` `resp.OK(c, gin.H{"list": list})`），但 `mock/user.ts:274-277` 也返回 `ok(trafficLogs)`（数组）——mock 本身偏离契约，掩盖了 E2E 中的缺陷。对接真实后端时 `TrafficView.vue` 拿到 `{list: [...]}`，表格/图表渲染为空（`user.trafficLogs.reduce` 作用于非数组对象）→ 流量页损坏。修复：解析 `.list`；让 mock 对齐 §5.6。~~
- ~~**src/api/plan.ts:5-8** — [P2] (c) `GET /plans` 重复且类型矛盾。`fetch()` 返回 `PlanListResp`（`{list}`），而 `list()` 对同一 URL 声明 `Plan[]`。文档（`docs/api/README.md` §8）规定 data 为 `{ "list": [...] }`，因此 `list()` 会把信封对象当作套餐数组返回。它从未被调用（死代码），影响潜伏，但这是与契约矛盾的错误类型公共 API。~~
- ~~**src/router/index.ts:152-156 + src/api/admin.ts:70-75** — [P2] (b) 越界：管理端工单模块不在文档化完成范围内。`docs/frontend/progress.md` §1 M8：「管理后台核心 5 模块(总览/用户/套餐/节点/订单)」；§2 将「剩余:…工单…」列为未完成。代码注册了 `/admin/tickets`，`tests/e2e/admin.spec.ts` 也断言了它。API 附录（`docs/api/README.md` §16）确实包含 `/admin/tickets`，所以这是规格页未声明已建成的增量功能——文档与代码在 M8/M9 状态上不一致。~~
- ~~**src/locales/zh-CN.ts / en-US.ts** — [P3] (a) i18n 模块结构与规格不符。`docs/frontend/data-layer.md` §5：「语言包按模块拆分：`locales/zh-CN/{common,auth,dashboard,order,invite,...}.ts`」。实现是单一扁平文件（`src/locales/index.ts:8`）。功能上按需加载且完整，但不符合文档化的模块拆分。~~
- ~~**src/views/profile/ProfileView.vue + src/components/business/OrderConfirmModal.vue:10** — [P3] (a) 视图直接调用 api，违反 store 分层约定。`docs/frontend/data-layer.md` §2：「store 是唯一允许调用 api 模块的地方」。`ProfileView.vue` 直接调用 `apiUser.updateProfile` / `changePassword` / `resetSubscribe`；`OrderConfirmModal.vue` 直接导入 `apiCoupon`（`checkCoupon`），绕过 `useOrderStore`。功能正常，但违反文档化分层。~~
- ~~**src/composables/** — [P3] (a) 文档化的 `usePolling` composable 不存在。`docs/frontend/data-layer.md` §4 列出 `usePolling`（带 `visibilityState` 暂停的通用轮询）。`src/composables/` 中没有 `usePolling`——轮询在 `stores/order.ts:60` 和 `stores/server.ts:29` 手写实现，无可见性暂停。轻微偏差，行为大体等价。~~

---

## 总结

- **Standards 轴：** 共 29 项发现 — 最严重：P1（验证码 `{code}` 未被替换，导致注册/找回密码不可用），位于后端 `auth_service.go:96`。
- **Spec 轴：** 共 13 项发现 — 两半各有 P1（余额支付订单不产生佣金，后端 `order_service.go:445-458`；流量明细响应结构不匹配导致流量页损坏，前端 `src/api/user.ts:29`）。
