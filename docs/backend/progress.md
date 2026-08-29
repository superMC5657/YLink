# 后端开发 · 进度追踪(已完成 / 未完成 / 前置条件)

> 本文档记录 `server/` 目录 Go/Gin 后端的开发状态,是 docs/backend 与 docs/api 的实现对照表。
> 更新规则:每完成一个里程碑/修复一个缺陷,同步更新本文档「已完成」;新增缺口写入「未完成」并标注依赖。
> 最后更新:2026-08-30(**管理端 handler 重构**:admin.go/admin_batch3.go/admin_batch4.go 按业务域拆分为 `internal/handler/admin/` 子包,路由与方法签名零改动;第四批 review 修复:Telegram 绑定 stale 写竞态、webhook 总开关、webhook 注册失败报错、Clash 尾随换行、gofmt;2026-08-29:**第四批 Xboard 缺口补齐:F10 订阅模板管理 + F12 Telegram 机器人**——订阅模板全文档 text/template 化(回退内置生成器)、Telegram 绑定/webhook/提醒推送;此前:F04 报表余额两字段、第三批,更早见下)

---

## 1. 已完成项

### 重构 · 管理端 handler 子包拆分(✅ 完成,2026-08-30)

| 项 | 说明 | 位置 |
|---|---|---|
| 按业务域拆分 | 原 `handler/admin.go`(1033 行)+ `admin_batch3.go` + `admin_batch4.go`(按提交批次命名,易误解)删除,新建 `package admin` 子包,10 个文件按业务域组织:`admin.go`(Admin 结构体+helper)、`dashboard.go`(仪表盘+统计 F04)、`user.go`(用户管理 F05+审计日志 F08)、`plan.go`(套餐+优惠券)、`server.go`(节点+分组+流量 F16)、`order.go`(订单+退款+提现审核 F02)、`content.go`(公告/知识库/分类 F15+邮件模板 F11+订阅模板 F10)、`ticket.go`(工单)、`agent.go`(代理+佣金)、`system.go`(设置+版本 F20+Telegram webhook 注册 F12) | `internal/handler/admin/` |
| 兼容性保证 | `Admin` 结构体、全部方法签名、路由注册体不变——router 仅改 3 行(import、字段类型 `*admin.Admin`、构造 `admin.NewAdmin`);`pageParams` 因被 `invite.go` 使用,在 handler 包保留(`helper.go`),admin 子包内自带一份;无循环依赖(admin 子包不引父包) | `internal/router/router.go`、`internal/handler/helper.go` |
| 验证 | `go build ./...`、`go vet ./...`、`go test ./...` 全绿(含 router 路由碰撞测试) | — |

### Xboard 缺口补齐 · 第四批(✅ 完成,2026-08-29,对齐 .scratch/xboard-gap-fill/spec.md)

| 项 | 说明 | 位置 |
|---|---|---|
| F10 订阅模板·生成器模板化 | 三生成器(clash/sing-box/v2ray)重构为 Go text/template 渲染,内置模板注册于 `pkg/subscribe/template.go`(内置模板渲染结果与重构前硬编码输出逐字节一致,`TestClashBuild` 等既有测试保证);数据上下文 `TemplateData`:公共 `{{.SiteName}}`/`{{.UserInfo}}`/`{{.NodeCount}}`,clash `{{.SpeedLimit}}`(B/s)+`{{.NodeBlock}}`(预渲染 proxies 块)、sing-box `{{.Outbounds}}`(预渲染 outbounds JSON 数组)、v2ray `{{.Links}}`——节点语法细节对模板作者不可破坏 | `internal/pkg/subscribe/template.go`、`clash.go`、`singbox.go`、`v2ray.go` |
| F10 订阅模板·自定义与回退 | `subscription_templates` 表(迁移 0008,name=客户端类型);`renderSubscriptionTemplate`:自定义模板优先,**自定义缺失/渲染失败自动回退内置生成器并记 warn,订阅不 5xx**(spec F10 验收要点);`Generate` 中 userinfo 先于内容生成(模板变量需要) | `internal/service/subscription_template.go`、`subscribe_service.go`、`internal/repo/subscription.go`、`migrations/0008_batch4_subtpl_telegram.{up,down}.sql` |
| F10 管理端 API | `GET /admin/subscription-templates`(内置+自定义合并,含 variables/remark)、`PUT .../{name}`(保存前示例数据渲染校验,语法错误 40000)、`DELETE .../{name}`(恢复内置)、`POST .../{name}/preview`(示例数据渲染,v2ray 返回 base64 前文本);审计 `edit_subscription_template`/`reset_subscription_template` | `internal/service/subscription_template.go`、`internal/handler/admin_batch4.go`、`internal/router/router.go` |
| F12 Telegram 服务 | settings 新键 `telegram`(`{bot_token, bot_username, webhook_secret, enabled}`,迁移 0008 seed,管理端设置页 JSON 编辑);`TelegramService` 不引 SDK,`net/http` 直调 Bot API(sendMessage/setWebhook,10s 超时),测试经 `sendFn` 注入避免外呼 | `internal/service/telegram.go`、`internal/config/config.go` |
| F12 绑定闭环 | `POST /user/telegram/bind-code`(6 位码 Redis `tg:bind:code:{code}` 10min 单次,60s 重发间隔+每日 20 次,未启用 40000)、`POST /telegram/webhook`(secret 头 `X-Telegram-Bot-Api-Secret-Token` 校验,不匹配 40300;`/bind <code>` GetDel 消费验证码后**仅条件更新 telegram_id 一列**(`WHERE id=? AND is_banned=false`,review 修复:不再用 stale 快照全量 Updates 覆盖并发修改的余额等字段;RowsAffected=0 提示账号状态变更),`uk_users_telegram` 部分唯一索引兜底一 chat 一账号;**webhook 尊重 `telegram.enabled` 总开关**(review 修复:关闭后 /bind、/start 等静默忽略,仅 /unbind 始终放行——解绑不被锁死);`/unbind` 按 chatID 反查清空;其余命令回执用法)、`POST /user/telegram/unbind`(条件更新置空——`Updates(struct)` 跳过零值无法清列);解绑后立即停止推送 | `internal/service/telegram.go`、`internal/handler/telegram.go`、`internal/router/router.go` |
| F12 webhook 注册 | `POST /admin/telegram/webhook/setup`:自动生成 `webhook_secret`(缺失时回写 settings)、调 setWebhook(URL=`{App.BaseURL}/api/v1/telegram/webhook`),返回 `{webhook_url, message}`;**Telegram API 返回 ok=false 时保留审计但返回 50000 错误**(review 修复:不再假成功);审计 `telegram_webhook_setup`;AdminService 经 `SetTelegram` 注入委托(不动构造签名) | `internal/service/telegram.go`、`internal/service/admin_service.go`、`internal/handler/admin_batch4.go` |
| F12 提醒推送 | cron `sendExpireMail`/`TrafficRemind` 邮件后对已绑定用户追加 Telegram 纯文本通知(`NotifyUser`:未绑定/未启用跳过,**失败仅记日志不阻断主流程**——spec F12 验收要点);`GET /user/profile` 响应新增 `telegram_bound` | `internal/service/cron_service.go`、`cmd/worker/main.go`、`internal/service/user_service.go` |

新增测试 14 例(`batch4_test.go`;2026-08-30 review 修复补 3 例:总开关拦截 /bind 且不消费验证码、总开关放行 /unbind、setWebhook ok=false 返回 50000):订阅渲染回退内置/自定义生效/坏模板回退(逐字节断言)、保存语法错误 40000、内置模板预览(clash/v2ray);Telegram 绑定码未启用/正常签发(重发间隔 429)、webhook 绑定成功(secret+码+写库+回执)、secret 不匹配 40300、解绑(含未绑定 40000)、推送降级(未绑定/未启用/发送失败三态)。`go test ./... -count=1` 全绿。

### F04 报表增强 · 订单趋势余额两字段(✅ 完成,2026-08-28)

| 项 | 说明 |
|---|---|
| 接口 | `GET /admin/stat/orders` 的 `items[]` 新增 `balance_used`(当日完成订单余额支付部分,按 `paid_at`)与 `balance_refunded`(当日退款订单余额部分,按 `updated_at` 近似),单位元、逐日补零 |
| 口径 | `revenue`/`refunded` 为现金部分(`pay_amount`),`balance_used`/`balance_refunded` 为余额部分(`orders.balance_used`),二者相加为订单实付总额;退款时余额支付部分原路退回用户余额(`admin_service.Refund`),与字段语义一致 |
| 实现 | `repo/stat.go` 新增 `BalanceRevenueByDay`/`BalanceRefundByDay`(与既有 RevenueByDay/RefundByDay 同结构同过滤条件,仅 SUM 列换 `balance_used`);`admin_stats.go` StatOrders 填充;`dto_admin.go` AdminStatOrderPoint 加两字段 |
| 测试 | `TestStatOrdersFillsZeroDays` 补两条 sqlmock 期望与新字段断言(3.0/2.0 元),通过 |

### 缺陷修复 · 分页空列表序列化为 null 致前端页面卡死(✅ 已修复,2026-08-28)

| 问题 | 修复 |
|---|---|
| nil slice → JSON `null` | Go 零值切片经 `encoding/json` 序列化为 `null`(空表查询必现,如 `GET /admin/traffic/resets` 无记录时返回 `list:null`)。前端模板以 `list.length === 0` 判空,对 null 抛 `TypeError` 导致 Vue 渲染中断、n-spin 永久转圈。`resp.PageOK` 现统一把 nil slice 兜底为 `[]`,所有走 `PageOK` 的分页接口(用户/审计/订单/工单/公告/知识库/代理/佣金/流量重置等)一并受益 | 
| 单测 | 新增 `internal/pkg/resp/resp_test.go`:nil slice→`[]`、空 slice→`[]`、非空 slice 原样,3 用例通过;`go build ./...` 与 `go vet` 通过 | 

前端配套:`AdminTrafficImportView.loadLogs` 增加 `res.list ?? []` 防御(兼容未更新的旧后端);naive-ui 注册缺失(NTabs 等)详见 docs/frontend/progress.md 同日记录。

### B1 骨架(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| 工程初始化 | `go.mod`(module `ylink-backend`)、双入口 `cmd/server` + `cmd/worker` | `server/go.mod` |
| 配置加载 | Viper:`configs/config.yaml` + `APP_` 前缀环境变量覆盖(点转下划线) | `internal/config/config.go` |
| 日志 | zap JSON + lumberjack 按天切割,双输出(stdout + 文件);GORM 慢查询/错误日志关闭 ANSI 颜色,避免重定向到文件出现 ESC 乱码 | `internal/pkg/logger/logger.go`、`internal/repo/repo.go` |
| 中间件链 | RequestID → Recovery → AccessLog → CORS → RateLimit → [Auth] → [Idempotency] | `internal/middleware/*` |
| 统一响应 | envelope `{code,message,data}` + 分页;错误码与 HTTP 映射对齐契约第 2 节 | `internal/pkg/resp`、`internal/pkg/errs` |
| 参数校验 | binding 错误统一转 40000 + 字段级文案 | `internal/pkg/validate` |
| 密码哈希 | bcrypt cost 12 | `internal/pkg/passwd` |
| JWT | access(2h)/refresh(14d),带 `TokenType` 防两类令牌混用;refresh 白名单存 Redis | `internal/pkg/jwt` |
| 迁移 | 18 张业务表(users/plans/orders/payments/coupons/coupon_usages/invite_codes/commission_logs/server_groups/servers/notices/knowledges/tickets/ticket_messages/traffic_logs/settings/audit_logs/agent_applies)+ 节点分组/演示套餐/settings 初始化数据,golang-migrate 格式 | `server/migrations/0001_init.{up,down}.sql` |
| 健康检查 | `GET /healthz`(存活)/`GET /readyz`(DB+Redis 连通) | `internal/handler/health.go` |
| 工程化 | Makefile( run/migrate/test/build )、Dockerfile 多阶段、docker-compose、Caddyfile、.env.example | `server/` 根目录 |
| 基础包单测 | errs 映射 / jwt 签发解析 / passwd | `internal/pkg/*/*_test.go` |

### B2 账户(✅ 完成)

| 端点 | 说明 |
|---|---|
| `POST /captcha/email` | 6 位验证码,Redis 10min 有效、60s 限频、同 IP 日上限 20、异步邮件(失败重试 2 次) |
| `POST /auth/register` | 验证码一次性校验、邮箱唯一、邀请码绑定一级(`used_count+1`,自邀不绑) |
| `POST /auth/login` | 密码校验、失败 5 次锁 10 分钟(42900 含剩余秒数)、封禁拒绝 |
| `POST /auth/refresh` | refresh 白名单校验 + 旋转(旧 jti 立即失效) |
| `POST /auth/forgot` | 验证码重置密码,吊销全部会话 |
| `POST /auth/logout` | 删除当前 jti 白名单 |
| `GET /user/stat` | 余额/佣金/待支付订单/未关工单/邀请数/是否代理 |
| `PUT /user/profile` | 通知开关(remind_expire / remind_traffic) |
| `POST /user/password/change` | 旧密码校验(40101),改密后吊销其他会话保留当前 |

### B3 内容(✅ 完成)

| 端点 | 说明 |
|---|---|
| `GET /config` | settings 组装(站点/支付方式/代理政策/语言),默认值兜底,Redis 缓存 60s |
| `GET /notices` | 上架公告分页,创建时间倒序 |
| `GET /knowledges` | 按 `language`+`keyword`(标题模糊)查询,按分类保序分组 |
| `GET /knowledges/{id}` | 详情(仅上架,否则 40400) |
| `GET /servers` | 按当前用户套餐 `group_ids` 输出可见分组;**不返回** host/port/config |

### B4 交易(✅ 完成,核心链路)

| 端点/能力 | 说明 |
|---|---|
| `GET /plans` | 上架套餐,`prices` 仅含支持周期,`speed_limit:null` 表示不限速 |
| `POST /coupons/check` | 纯试算不落库;下单时服务端重算 |
| `GET /coupons/available` | 当前用户可用优惠券列表(启用/生效期/总量未满 + 每人限用 + 可选 `plan_id`/`period` 过滤;DTO 不暴露运营计数,契约 §9) |
| `POST /orders` | `Idempotency-Key` 24h 幂等(**key 按 user_id 隔离**);算价 = amount − discount;优惠券重校验 |
| `GET /orders` / `GET /orders/{order_no}` | 列表(状态过滤)、详情(兼 3s 轮询,仅本人) |
| `POST /orders/{order_no}/cancel` | 仅待支付可取消(否则 11003) |
| `POST /orders/{order_no}/checkout` | `method=balance` 余额直付(行锁扣减);在线渠道拉起易支付;30min 内重复 checkout 返回原单 |
| `POST /payment/notify/{method}` | 验签 → 金额比对 → 行锁幂等 → MarkPaid → 开通/续期 → 写佣金 → 记优惠券使用;返回纯文本 `success` |
| 开通/续期状态机 | 同套餐未过期:叠加时长+流量(u/d 不清零);换套餐/过期/无订阅:替换+清零;onetime:只加流量 |
| 支付抽象 | `pkg/payment.Driver` 接口 + 注册表;易支付驱动(md5 验签/查单)注册为 `epay_alipay`/`epay_wxpay` |

### B5 订阅(✅ 完成)

| 端点/能力 | 说明 |
|---|---|
| `GET /client/subscribe/{token}` | `flag=clash|sing-box|v2ray`,缺省 UA 嗅探(含 clash → yaml,sing-box → json,其他 → base64);不走 envelope;token 级限流 10 次/min;userinfo 缓存 30s;响应头 `subscription-userinfo`/`profile-update-interval: 24`/`content-disposition` |
| 提示节点 | 无订阅/到期/流量用尽时注入 `⚠` 提示节点引导续费 |
| `GET /user/subscribe` | 当前订阅:套餐/到期/流量/限速/设备数/订阅链接 |
| `POST /user/subscribe/reset` | 密码二次确认,重置 token 并清除旧缓存(旧链接立即失效) |
| `GET /user/traffic-logs` | 日期范围(最大 90 天)流量明细,升序 |
| 生成器 | `pkg/subscribe`:Clash YAML / sing-box JSON / v2ray base64(6 种协议:shadowsocks/vmess/vless/trojan/hysteria2/tuic) |

### B6 营销(✅ 完成)

| 端点 | 说明 |
|---|---|
| `GET /invite/summary` | 佣金余额/比例/注册数/累计(已发放 sum)/确认中 sum |
| `GET`/`POST /invite/codes` | 8 位随机码列表;生成超限 13001(上限取 settings);`register_url_prefix` 仅返回路径后缀 `/#/register?code=`(完整链接由前端拼 origin,见 review-0.8.0.md 第五轮) |
| `GET /invite/records` | 仅展示已发放(status=1)佣金记录 |
| `POST /invite/transfer` | 行锁事务:commission_balance → balance;不足 13002 |
| `GET /agent/status` | 有效邀请统计(有已完成订单 或 注册满 N 天未封禁,N 取 settings `agent.valid_invite_days`,默认 3)、条件卡片、apply_status(none/pending/approved/rejected) |
| `POST /agent/apply` | 达标校验 15001;审核中重复 15002;被拒后可重新提交 |

### B7 工单 + 定时任务(✅ 完成)

| 端点/能力 | 说明 |
|---|---|
| `GET`/`POST /tickets` | 列表分页;创建(首条消息) |
| `GET /tickets/{id}` | 详情含消息流 |
| `POST /tickets/{id}/reply` | 用户回复→状态回 0(待回复);已关闭 14001 |
| `POST /tickets/{id}/close` | 关闭(14001 已关闭) |
| `POST /tickets/{id}/reopen` | **重开一次(2026-08-12)**:仅已关闭且未重开过可重开,状态回 0、`reopen_count+1`;未关闭 40900、已重开 14002;迁移 `0003_ticket_reopen` |
| worker cron | robfig/cron + Redis 分布式锁:`close-expired-orders`(5min,条件更新防竞态吞单)、`reconcile-payments`(10min,易支付主动查单补账)、`confirm-commissions`(每日 02:00)、`expire-remind`(10:00,前 3/1 天双窗口)、`traffic-remind`(10:30,≥80%)、`traffic-daily`(01:00,模式 B 空跑) |

### 审查修复(✅ 已修复,阻断项清零)

第一轮内置审查(骨架/账户/交易/订阅初版):

| 问题 | 修复 |
|---|---|
| 关单与支付回调竞态可能吞掉已支付订单 | 关单改用 `UpdateStatusIfPending`(WHERE status=0 条件更新) |
| access/refresh token 可互换使用(签名同) | Claims 增加 `TokenType`,Auth 中间件与 Refresh 分别校验 |
| Idempotency-Key 跨用户可回放 | 幂等缓存 key 加 user_id 维度 + repo 查询限定本人 |
| 优惠券限额从未记账(超发风险) | 初版:MarkPaid 后置记账;二轮已升级为「下单事务内原子占用」,见下方安全加固 |

第二轮审查(管理端/安全改造增量):见「安全与一致性加固」表,2 个阻断项(取消竞态误释放券、cron 关单残留券)与 3 个 should-fix(佣金确认覆盖、代理审批并发、admin 自保护)均已修复。

### 测试状态(✅ 已更新,2026-08-10 实测)

- `go build ./...` / `go vet ./...` / `gofmt -l`(0 输出)全部通过
- `go test ./... -count=1` 全绿,**47 个测试函数**,覆盖:错误码映射、JWT、密码、验证码限频/已注册、注册/登录锁定/刷新旋转、优惠券试算/超限 12001/原子占用、下单幂等、续期状态机、回调幂等、epay 验签与篡改拒绝、订阅生成(3 格式)、佣金划转、代理申请、工单流转、佣金确认竞态、超时关单(含优惠券回退)、取消并发已支付回滚、退款佣金回滚、代理审批、bluemonday 清洗

---

### B8 流量模式 A · 节点上报(✅ 完成,2026-08-22)

> 契约见 docs/api/README.md §17;业务流程见 core-flows.md §8;表结构见 data-model.md §2.1/§2.8/§2.11.1。

| 项 | 说明 | 位置 |
|---|---|---|
| 迁移 0004 | `users.uuid`(gen_random_uuid 回填,每用户订阅凭证)、`servers.node_key`(md5 回填,X-Node-Key)、新表 `node_user_stats`(server_id,user_id,last_u,last_d 快照) | `server/migrations/0004_node_report.*` |
| 每用户凭证 | 注册即生成 uuid;节点 config 显式开启 `per_user_credentials: true` 后订阅下发 `users.uuid`(老用户 Generate 懒生成兜底);未开启的存量节点继续使用 config 共享凭证,避免 inbound 未配发前断连 | `subscribe_service.go`(toNode/buildNodes)、`auth_service.go` |
| NodeAuth 中间件 | `X-Node-Key` → server_id,Redis 缓存 60s(`node:key:{k}`),缺失/未知 40100;lookup 由路由注入,中间件不依赖数据层 | `internal/middleware/node_auth.go` |
| `GET /node/users` | 节点分组下有效订阅且未封禁用户(uuid/u/d/transfer_enable/expired_at unix)+ 节点 rate;套餐含下架(存量订阅仍有效) | `internal/service/node_service.go` |
| `POST /node/report` | **累计值口径**:node_user_stats 行锁快照差分得增量(重复上报差分 0 天然幂等;累计回退视为节点重启,增量取当前值)→ ×rate(四舍五入)→ 事务内 `users.u/d` 原子累加 + `traffic_logs` 增量聚合(`ON CONFLICT DO UPDATE SET u=u+?`,与模式 B 覆盖式区分)→ 批量删 `sub:userinfo:{token}` 缓存;仅接受套餐 group_ids 含本节点分组的用户,同一 UUID 重复整体拒绝(`duplicate_uuid`);未知 uuid / 无订阅/封禁/过期跳过并在响应 `skipped` 返回;data 1–1000 条 | `node_service.go`、`repo/node.go` |
| 管理端配套 | AdminServerView 暴露 node_key;新建节点自动生成;`POST /admin/servers/{id}/node-key/reset` 重置(审计 + 旧密钥缓存即刻失效) | `admin_crud.go`、`handler/admin.go` |
| 演示 agent | `go run ./cmd/node-agent -endpoint ... -key ...`:自动拉取 /node/users,模拟累计值(1–50/1–500 MiB 随机增量)定时上报,日志打印 accepted/skipped;**首轮先上报 0 基线建立快照,再推进累计值**;真实代理后端(Xray stats 等)对接见 [node-agent-guide.md](node-agent-guide.md) | `server/cmd/node-agent/main.go` |

### 0.9.0 评审修复(✅ 已修复,2026-08-25,见 docs/reviews/review-0.9.0.md)

| 问题 | 状态/修复 |
|---|---|
| [P1] 每用户凭证在节点 inbound 未配发前替换共享凭证,存量订阅刷新即断连 | ✅ 修复:节点 config 显式开启 `per_user_credentials: true` 才下发 `users.uuid`;未开启继续使用 config 共享密码/uuid;存量节点先配发 inbound 再开开关 |
| [P1] Alertmanager 用 QQ 465 隐式 SMTPS,SMTP 邮件无法投递 | ✅ 修复:Alertmanager 固定走 STARTTLS(默认 `ALERT_SMTP_PORT=587`),465 场景需前置 SMTPS→STARTTLS relay;支持 `ALERT_SMTP_HOST/FROM` 覆盖 |
| [P1] NSIS installerHooks 指向不存在文件,Windows 打包失败 | ✅ 修复:入库 `src-tauri/nsis/installer-hooks.nsh`(当前无自定义逻辑,保留占位) |
| [P2] 节点上报未校验用户套餐分组,可跨节点计费 | ✅ 修复:`POST /node/report` 按节点分组加载允许套餐,非本分组用户返回 `not_subscribed`;新增回归测试 |
| [P2] 同请求重复 UUID 被逐个差分,可产生错误增量 | ✅ 修复:重复 UUID 在查库/开事务前整体拒绝,响应 `duplicate_uuid`;新增回归测试 |
| [P2] node-agent 首轮在建立基线前已推进随机增量,首次即计费 | ✅ 修复:先上报当前值(首轮 0),发送后再推进下一轮累计值 |
| [P2] Alertmanager SMTP 用户名/密码未转义,含 `&`/`\`/`|`/`'` 时配置损坏 | ✅ 修复:compose entrypoint 改用 AWK 逐字替换模板,并按 YAML 单引号规则转义 |

### 管理端 API(✅ 完成,契约第 16 节全量)

| 端点组 | 说明 |
|---|---|
| `GET /admin/stat/overview` | 用户/代理/订单/收入(总额+今日)/在售套餐统计 |
| `GET /admin/users`、`PUT /admin/users/{id}`、`POST /admin/users/{id}/balance` | 列表/封禁与角色(禁止操作自己)/调余额(审计) |
| `GET/POST/PUT/DELETE /admin/plans`、`/admin/servers`、`/admin/server-groups` | 套餐与节点 CRUD(写入侧 XSS 清洗;响应经 `AdminPlanView`/`AdminServerView` DTO,价格统一为元并展开 group_ids/is_show/host/port/config) |
| `GET /admin/orders`、`POST /admin/orders/{no}/refund` | 订单列表(**2026-08-13 增 `commission_amount` 列:按订单号批量查佣金映射,余额支付恒 null**);退款(余额退回+**收回订阅**+优惠券回退+佣金回滚+审计,行锁) |
| `GET/POST/PUT/DELETE /admin/coupons` | 优惠券 CRUD(列表返回 `AdminCouponView`:展开 type/value(元)/min_spend(元)/used_count/valid_periods/plan_ids,见 dto_admin.go) |
| `GET/POST/PUT/DELETE /admin/notices`、`/admin/knowledges` | 公告/知识库 CRUD(bluemonday 清洗;**GET 列表含隐藏**,2026-08-11 补齐) |
| `GET /admin/tickets`、`GET /admin/tickets/{id}`、`POST /admin/tickets/{id}/reply|close` | 工单管理(客服回复→已回复) |
| `GET /admin/agent/applies`、`POST /admin/agent/applies/{id}/approve|reject` | 代理审批(行锁防并发;通过→role=2,审计) |
| `GET /admin/commission-logs` | 佣金日志(含用户邮箱) |
| `POST /admin/traffic/import` | 模式 B 流量导入(audit 审计) |
| `GET/PUT /admin/settings` | 站点配置读写(写后失效 Redis 缓存) |

### 第四轮评审 v0.5.0(✅ 已修复,2026-08-13,见 docs/reviews/review-0.5.0.md)

| 项 | 状态 | 说明 |
|---|---|---|
| 管理端订单佣金查询失败透传 | ✅ 已修复(2026-08-13) | `ListOrders` 原静默忽略批量佣金查询错误,现改为失败即上抛,不再返回 `commission_amount` 全 null 的成功响应;补回归测试 `TestAdminListOrdersCommissionQueryError`;详见 [review-0.5.0](../reviews/zh-cn/review-0.5.0.md)。 |

### 安全与一致性加固(✅ 完成)

| 项 | 说明 |
|---|---|
| XSS 清洗 | `pkg/sanitize`:bluemonday UGCPolicy(富文本)+ StrictPolicy(纯文本);应用于公告/知识库/套餐内容与工单消息写入 |
| 优惠券 TOCTOU 消除 | 下单事务内 `Occupy` 原子条件更新(防超发)+ `coupon_usages` 落账;取消/超时关单/退款三条路径统一 `Release`+`DeleteUsage` 回退 |
| 取消竞态 | `UpdateStatusIfPending` 返回影响行数,0 行(并发已支付)直接 11003,不再误释放优惠券 |
| 超时关单 | 行锁读+事务内关单,并回退优惠券占用(cron 不再残留券) |
| 佣金确认竞态 | `UpdateStatusIfPending`(0→1),已被退款撤销的佣金不再发放 |
| 代理审批并发 | 行锁读取申请,重复审批返回 409 |
| 审计日志 | `audit_logs` 已接入:调余额/退款/封禁/改角色/代理审批/流量导入 |

### 全量审查修复(✅ 已修复,2026-08-11)

第三轮全仓库审查(见 docs/reviews/review-0.2.0.md)的后端修复:

| 项 | 说明 |
|---|---|
| 验证码邮件未替换 `{code}` | 模板占位符改 `{{.code}}`,注册/找回密码可收到验证码 |
| 通知开关 false 不落库 | `UpdateProfile` 改 map 更新并回读;管理端 UpdatePlan/Server/Notice/Knowledge 同改 map 更新(支持 false/空值) |
| 订单详情空指针(套餐被删) | 套餐名回退「已删除套餐」;`DeletePlan` 增加关联订单检查(11006 不可删) |
| checkout 缓存键漏支付方式 | 缓存键含 `user_id+order_no+method`,切换支付方式不再命中旧结果 |
| 优惠券 limit_per_user 非原子 | 事务内 `Occupy` 后 `CountUsageLocked`(`SELECT ... FOR UPDATE`)串行化;幂等键重复插入捕获后重查返回首单 |
| 余额支付佣金按实付 | `grantCommission` 接收实付金额;封禁订阅改 401;注册强制邀请码按站点配置校验 |
| 佣金比例逻辑重复 | 抽取 `commissionRateFor`(订单/邀请共用);优惠券释放 4 处抽取 `releaseCoupon` |
| epay 回调缺 pid 校验 | `VerifyNotify` 验签后比对配置 pid |
| 限流取 IP 不可靠 | 优先取 `X-Forwarded-For` 首跳 |
| `couponCode` 吞错 | 改返回 `(string, error)` 显式处理 |
| 到期提醒仅一次 | `ExpireRemind` 改为前 3 天与前 1 天双窗口(marker 区分) |
| 代理有效注册天数写死 | 读 `agent.valid_invite_days`(默认 3),注册/代理审计/审批共用 |
| 余额支付 content 空串 | `CheckoutResp.Content` 改 `*string`,余额支付返回 `null` |
| 百分比优惠券整单全免 | `validateCoupon` 百分比折扣 `amount*coupon.Value/100` 单位错配(Value 为「百分比×100」的分存储:10% 券存 1000 分),10% 券折出 10000 分 → 封顶全免;改 `/10000` 还原百分比,并补 `TestCouponCheckPercent`/`TestCouponCheckPercentCapped` 防回归 |
| 退款未收回订阅 | `Refund` 原只退余额/回退券/撤佣金,未处理 `users` 订阅字段,退款后用户仍可使用已退款订阅;新增 `revokeSubscriptionOnRefund`:onetime 扣回流量(下限 0),周期套餐且当前生效订阅正是该订单套餐时清除订阅(plan_id/expired_at 置空、流量清零、限速/设备清空);不同套餐/续期叠加场景行为见方法注释,补 3 个测试 |
| 死代码清理 | 删除 `ListPendingOrderNos`/`GetByNoAdmin`/`IncrUsed`/`SetString` |

### 0.5.0 审查修复(✅ 已修复,2026-08-13,见 docs/reviews/review-0.5.0.md)

| 项 | 说明 |
|---|---|
| 佣金批量查询失败被静默忽略 | `ListOrders` 原 `if comms, err := ...; err == nil` 吞掉 `ListByOrderNos` 错误,查询失败仍返回成功响应、`commission_amount` 全 null,把数据缺失误显示为「无佣金」;改为失败即返回错误,补 `TestAdminListOrdersCommissionQueryError` 回归 |

### 增强(✅ 完成)

| 项 | 说明 |
|---|---|
| `GET /metrics` | promhttp:请求计数/延迟直方图/支付成功计数器(`payment_success_total`),回调成功打点 |
| worker cron 指标 | worker `:8082/metrics`(`cron_job_runs_total`/`cron_job_duration_seconds`):`WithLock` 打点(success/error/skipped + 耗时直方图,panic 计 error),2026-08-25 |
| `GET /swagger/*` | gin-swagger,仅 development 环境;`make swagger` 重新生成 |
| 支付回执邮件 | 在线回调与余额支付成功后异步发送(`[站点] 支付成功`),未配置 SMTP 静默跳过 |
| agent-audit cron | 每月 1 日 03:00 复核代理有效邀请数,不达标降级 role=0 |

### Xboard 缺口补齐 · 第三批(✅ 完成,2026-08-28,对齐 .scratch/xboard-gap-fill/spec.md)

| 项 | 说明 | 位置 |
|---|---|---|
| F02 提现提交 | `POST /invite/withdraw`(仅代理商,`role=RoleAgent` 校验,非代理商 13003/HTTP 403):同事务行锁读用户 → 校验并**扣减 commission_balance**(提交即扣减防双花,重复提交受余额拦截) → 创建 `type=1` 提现工单(首条消息为结构化提现信息) → 写 `commission_withdraws` 提现单 → 写 `commission_logs` 提现流水(`biz_type=1`,`order_no='w<提现单ID>'`,status=0);`GET /invite/withdraws` 本人提现记录分页 | `internal/service/invite_service.go`、`internal/repo/order.go`(WithdrawRepo)、`internal/handler/invite.go`、`internal/router/router.go` |
| F02 提现审核 | `POST /admin/tickets/{id}/withdraw/pay`(确认打款:流水 status→1、关闭工单、系统消息留痕)与 `POST /admin/tickets/{id}/withdraw/reject`(拒绝:**自动退回佣金**至 commission_balance、流水 status→2、关闭工单);提现单按 ticket_id 行锁读取,仅处理中可审(否则 13004);两类操作均写审计 `withdraw_pay`/`withdraw_reject`;工单列表/详情带 `type` 字段与 `withdraw` 提现单信息;佣金账本 `commission_logs` 增加 `biz_type`(0=订单佣金/1=提现流水),summary/记录页/cron 确认任务全部排除提现流水(**提现绝不能被 cron 自动确认入账**),`order_no` 唯一约束改部分索引(`WHERE biz_type=0`) | `internal/service/admin_withdraw.go`、`internal/repo/order.go`、`internal/handler/admin_batch3.go`、`migrations/0007_batch3_withdraw_mailtpl_category.{up,down}.sql` |
| F14 会话管理 | `GET /user/sessions`(SCAN refresh 白名单,Redis 元数据 `{ip,ua,ts}`,当前会话标记,历史 "1" 值降级展示)、`DELETE /user/sessions/{jti}`(删白名单 + 写踢下线标记 `auth:kill:{uid}` Hash field=jti,Auth 中间件 Pipeline 校验 SV 与踢下线标记——**单会话 access 立即失效,其余会话不受影响**;当前会话不可自踢 40000,不存在 40400) | `internal/service/user_service.go`、`internal/service/auth_service.go`(issueSession 元数据)、`internal/middleware/auth.go`、`internal/pkg/redis/redis.go`、`internal/handler/user.go` |
| F15 内容排序 | `POST /admin/notices/sort`、`POST /admin/knowledges/sort`(`{items:[{id,sort}]}`≤500,单事务,审计 `sort_notice`/`sort_knowledge`);公告用户端展示顺序改为 `sort ASC, created_at DESC`;知识库用户端分组顺序按分类 `sort` | `internal/service/admin_content_manage.go`、`internal/repo/content.go`、`internal/handler/admin_batch3.go` |
| F15 知识库分类 | `knowledge_categories` 表(迁移 0007,唯一 `(language, name)`,**存量数据按 (language, category) 去重回填**) + `knowledges.category_id` 归属;`GET/POST/PUT/DELETE /admin/knowledge-categories`(列表含文档计数/改名**级联同步**知识文档展示分类/有文档拒绝删除);知识保存支持 `category_id` 显式归类或按名称自动归并建行 | 同上 |
| F11 邮件模板 | `mail_templates` 表(迁移 0007,name PK)+ 内置模板注册表(`captcha`/`expire_remind`/`traffic_remind`,Go template 占位符);`GET/PUT/DELETE /admin/mail-templates`(自定义覆盖合并/保存前校验可解析/删除恢复默认,审计)、`POST .../{name}/test`(示例占位符渲染走真实 SMTP,失败原因原样返回);`renderMailTemplate` 统一渲染:**自定义缺失/渲染失败自动回退内置文案**;验证码/到期提醒/流量提醒三处发送接入 | `internal/service/mail_template.go`、`internal/repo/mail.go`、`internal/handler/admin_batch3.go`、`internal/service/auth_service.go`、`internal/service/cron_service.go` |
| F19 品牌配置子集 | `site` 配置新增 `primary_color`(Hex 主色)/`background_url`(背景图),`GET /config` 下发;管理端设置页 JSON 编辑即生效 | `internal/service/content_service.go`、`configs/config.yaml` |
| F20 版本检查子集 | `app.version`(部署注入,缺省 dev)+ `update.manifest_url`(可选更新源,JSON `{version, notes}`,3s 超时 + 10min Redis 缓存);`GET /admin/version` 返回 `{version, latest, has_update, notes}`,语义化版本比较,未配置/拉取失败 latest=null 不报错;**自动执行升级不立项** | `internal/service/admin_version.go`、`internal/config/config.go`、`internal/handler/admin_batch3.go`、`configs/config.yaml` |

新增测试 7 例(`batch3_test.go`):提现提交(代理商扣减/非代理商 13003/余额不足 13002)、提现拒绝退回(退佣金+流水+关单+审计)、确认打款、非提现工单审核 40900、会话列表与踢下线(miniredis,含 kill 标记/当前会话拒绝/不存在 40400)、邮件模板回退与自定义/坏模板回退、公告排序;另修正 `TestRefreshRotation`(白名单值由 "1" 变更元数据 JSON)。

### Xboard 缺口补齐 · 第一批(✅ 完成,2026-08-28,对齐 .scratch/xboard-gap-fill/spec.md)

| 项 | 说明 | 位置 |
|---|---|---|
| F08 审计日志查询 | `GET /admin/audit-logs` 只读:操作人/动作/目标/时间范围筛选 + 分页 + 去重动作列表,JOIN users 取操作人邮箱;测试 `TestListAuditLogs`(含非法日期 40000)。**2026-08-28 目标可读化**:按 action 分派 `target_kind`(user/users/server/knowledge_category/order/mail_template)并批量反查 `target_display`(**用户类一律用邮箱**;节点名/分类名,订单号与模板名原样;用户已删除时 detail 留痕 email 兜底;均失败为 null 不影响主查询);测试新增 `TestAuditTargetDisplay`(含已删除用户兜底用例) | `internal/handler/admin.go`、`internal/service/admin_service.go`、`internal/repo/admin.go` |
| F22 安全部署项 | `security.admin_path`(管理端 API 路径段,默认 admin)、`security.subscribe_path`(订阅路径段,默认 client,`subscribeURL` 同步拼接)、`safe_mode`+`safe_domains`(域名白名单,非白名单 403/40300,healthz/metrics 不受影响);启动注入不落库;测试 SafeMode 放行/拦截/带端口/大小写、normalizeHost、路由注册无冲突 | `internal/config/config.go`、`internal/middleware/safe_mode.go`、`internal/router/router.go`、`internal/service/subscribe_service.go`、`configs/config.yaml` |
| F05 CSV 导出 | `GET /admin/users/export`:与列表同 keyword 筛选,每批 500 流式写 + UTF-8 BOM;列含套餐名(批内联 plans)与邀请人邮箱(批内联 users);测试 `TestExportUsersStreamsBatches` | `internal/handler/admin.go`、`internal/service/admin_service.go`、`internal/repo/admin.go`(StreamForExport) |
| F05 批量操作 | `POST /admin/users/batch`:`ban/unban/adjust_balance`,ids≤500,逐个执行复用单用户状态机(负值保护/操作自己拒绝/SV bump/审计),返回 `{success, failed:[{id,reason}]}`;测试 3 例(封禁+不存在/负值拦截/缺 amount 40000) | `internal/handler/admin.go`、`internal/service/admin_service.go` |
| F05 发送邮件 | `POST /admin/users/mail`:ids≤100,SMTP 同步逐发,结果写 `mail_logs`(迁移 0005,失败原因截断 512),整体审计 `send_mail`;mailer 未注入/SMTP 不可达均优雅失败留痕;测试 3 例 | `internal/service/admin_service.go`、`internal/repo/admin.go`、`migrations/0005_admin_users_enhance.{up,down}.sql` |
| F05 重置订阅密钥 | `POST /admin/users/{id}/sub-token/reset`:无需用户密码,旧 token 缓存(sub:userinfo/sub:rl)即删,返回新 subscribe_url,审计 `reset_sub_token`;测试含 404 与缓存清理断言 | `internal/service/admin_service.go` |

### Xboard 缺口补齐 · 第二批(✅ 完成,2026-08-28,对齐 .scratch/xboard-gap-fill/spec.md)

| 项 | 说明 | 位置 |
|---|---|---|
| F09 批量操作 | `POST /admin/servers/batch`:`action ∈ {delete, update}`,ids≤500,`update` 至少一项公共字段(status/is_show/group_id/rate,rate 须>0);整批单事务、逐节点汇总 `{success, failed:[{id,reason}]}`(不存在/失败不中断);审计 `batch_server_delete`/`batch_server_update` | `internal/service/admin_node_batch.go`、`internal/handler/admin.go`、`internal/router/router.go` |
| F09 复制节点 | `POST /admin/servers/{id}/copy`:全字段复制、名称追加 `-copy`(64 字符截断)、**重新生成 node_key**(不与源节点共享);返回新 AdminServerView,审计 `copy_server` | 同上 |
| F09 排序 | `POST /admin/servers/sort`:`{items:[{id,sort}]}`≤500,单事务批量更新 sort,审计 `sort_server` | 同上 |
| F16 流量重置 | `POST /admin/traffic/reset`:`{user_ids≤500, mode ∈ {clear_usage, reset_quota}}`,逐用户单事务(行锁)清零 u/d、reset_quota 另按当前套餐额度重置 transfer_enable(无套餐记失败),写 `traffic_reset_logs`(迁移 0006);**保留 node_user_stats 快照**防节点累计值重复计费;审计 `traffic_reset` | `internal/service/admin_traffic_reset.go`、`migrations/0006_admin_batch_stats.{up,down}.sql` |
| F16 重置记录 | `GET /admin/traffic/resets`:`user_id?` 筛选 + 分页,JOIN users 取邮箱,含 before/after 字段(字节) | `internal/repo/admin.go`(TrafficResetRepo)、`internal/handler/admin.go` |
| F04 订单统计 | `GET /admin/stat/orders?days=1..365(默认30)`:逐日 order_count(按 created_at)/completed_count+revenue(按 paid_at,已完成)/refunded(按 updated_at 近似,已退款);金额分→元,无数据日补零 | `internal/repo/stat.go`、`internal/service/admin_stats.go`、`internal/handler/admin.go`、`internal/router/router.go` |
| F04 用户统计 | `GET /admin/stat/users?days=`:注册趋势(逐日补零)+ 套餐分布(当前生效订阅按套餐聚合,联套餐名) | 同上 |
| F04 流量统计 | `GET /admin/stat/traffic?days=`:用户流量 Top10(traffic_logs 时间范围 u+d 合计联邮箱)+ 节点流量 Top10(node_user_stats 上报累计,未乘倍率,全周期口径) | 同上 |
| F04 聚合索引 | 迁移 0006 补 `traffic_logs(date)`、`orders(paid_at) WHERE paid_at IS NOT NULL`(部分索引)、`users(created_at)`,时间范围 GROUP BY 避免全表扫描 | `migrations/0006_admin_batch_stats.{up,down}.sql` |

新增测试 6 例(`admin_batch_stats_test.go`):批量删除汇总(不存在记失败)、update 缺字段/rate≤0 拒绝、批量更新、复制节点新 node_key、排序单事务、流量重置保留快照(期望序列无 node_user_stats DELETE)、reset_quota 无套餐失败、报表补零/TopN。

### Spec 二次调整 · 需求变更(📝 文档调整,未实现代码,2026-08-28,对齐 .scratch/xboard-gap-fill/spec.md)

| 项 | 说明 | 状态 |
|---|---|---|
| F02 佣金提现重新入选 | 用户决策恢复原被排除的佣金提现,范围收窄为最小闭环:**仅代理商**(`role=RoleAgent`,代理申请审批通过)可发起;用户端发送「佣金提现」工单(方式/收款账号/金额),提交时同事务防双花(推荐提交即扣减 commission_balance + commission_logs 流水 withdraw_submit);管理端工单页手动**确认打款**(线下发放,流水记 withdraw_paid)或**拒绝**(自动退回佣金 withdraw_refund);均写 audit_logs。设置项(开关/白名单/最低额/限额)先不引入。**批次已定:并入第三批(2026-08-28 用户确认)** | 📝 需求已立项,未实现 |
| F13 快速登录不做 | 用户明确不需要,从第三批移出并删除线标识 | 📝 已关闭 |
| 第三批名单更新 | 移出 F13、并入 F02,余:F02 佣金提现、F15 公告/知识库排序与分类、F14 会话管理界面、F11 邮件模板管理、F19 品牌配置子集、F20 版本检查+变更日志子集;**第三批全部待做(2026-08-28 用户确认)** | 📝 待实现 |

### 测试状态(✅ 已更新,2026-08-30 实测)

- `go build ./...` / `go vet ./...` / `gofmt -l`(0 输出)全部通过
- `go test ./... -count=1` 全绿;**144 个测试函数**(2026-08-30 第四批 review 修复新增 3 例——Telegram 总开关拦截 /bind 且不消费验证码、总开关放行 /unbind、setWebhook ok=false 返回 50000;2026-08-29 第四批新增 11 例——订阅模板回退/自定义/坏模板回退/保存校验/预览 + Telegram 绑定码/webhook 绑定/secret 校验/解绑/推送降级;2026-08-28 第三批新增 7 例——提现提交/拒绝退回/确认打款/非提现工单拒绝/会话列表与踢下线/邮件模板回退/公告排序,并修正 TestRefreshRotation 断言;2026-08-28 第二批新增 9 例——批量节点删除汇总/update 参数校验/批量更新/复制节点新 node_key/排序单事务/流量重置保留快照/reset_quota 无套餐失败/报表补零/流量 TopN;2026-08-25:0.9.0 新增 3 例——重复 UUID 整体拒绝、节点分组不匹配 `not_subscribed`、存量共享凭证回退;此前 2026-08-22 模式 A 新增 11 例——NodeUsers 同步/onetime 无到期、首次上报全量、同值重报幂等(不写 users/traffic)、计数器回退重启判定、0.5 倍率乘算、unknown_user+not_subscribed 跳过、servers 错误上抛、cumDelta/scaleRate 边界、NodeAuth(无头 401/未知 401/有效注入+缓存命中不再查库)、每用户凭证下发(opt-in 时 config 共享密码不下发断言)、admin 重置密钥(含 404)),此前覆盖:错误码映射、JWT(含 SV 会话版本号)、密码、验证码限频/已注册、注册/登录锁定/刷新旋转、优惠券试算(固定/百分比/封顶)/超限 12001/原子占用/每人限用、下单幂等、续期状态机、回调幂等、epay 验签与篡改拒绝、订阅生成(3 格式)、佣金划转、代理申请、工单流转、工单重开、佣金确认竞态、超时关单(含优惠券回退)、取消并发已支付回滚、退款(余额/券/佣金/订阅收回/onetime/异套餐)、代理审批、bluemonday 清洗、Auth 中间件、余额调整负值拒绝、管理端订单佣金查询、CORS

---

## 2. 未完成项

### 一期小缺口收尾(✅ 完成,2026-08-11)

| 项 | 说明 |
|---|---|
| 封禁/降级后 JWT 实时失效 | 会话版本号机制:Claims 增加 `SV`(签发时快照,`jwt.Generate` 带参);Redis `auth:ver:{uid}` 存当前版本;封禁/解封/角色变更/代理审批通过/代理商降级/找回密码/登出均 `INCR` bump;Auth 中间件比对 SV 不一致即 401(Key 不存在视为 0,Redis 异常不阻断退化为 TTL)。access 2h 内无需等过期 |
| 余额负值保护 | `AdjustBalance` 服务层拒绝调整后余额为负(40000)+ 迁移 `0002_balance_check` 加 `CHECK (balance >= 0)`(PostgreSQL CHECK 强制执行);佣金回滚减 `commission_balance` 不受约束 |
| 迁移 | `server/migrations/0002_balance_check.{up,down}.sql`(golang-migrate 格式,新增约束/回滚) |
| 测试 | 新增 `middleware/auth_test.go`(6 场景:无头/无效/refresh 混用/有效/SV 匹配/bump 后立即失效+重新签发恢复)、admin_service 3 例(负值拒绝/封禁 bump/角色 bump);**测试函数 47 → 60 全绿** |
| 既有 format:check 告警(前端) | ✅ 已解决(2026-08-12):前端 `pnpm format:check` 全仓通过(见 docs/reviews/review-0.4.0.md);后端 `gofmt -l` 0 输出 |

### 数据库切换:MySQL → PostgreSQL(✅ 完成,2026-08-13)

| 项 | 说明                                                                                                                                                                                                                                                                                                                                                       |
|---|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 驱动 | `gorm.io/driver/mysql` → `gorm.io/driver/postgres` v1.6.2(pgx v5);`internal/repo/repo.go` 切 `postgres.Open`                                                                                                                                                                                                                                              |
| 迁移 SQL | `migrations/*.sql` 全部重写为 PG 语法:BIGSERIAL 主键、SMALLINT、BOOLEAN、TIMESTAMP(3)、JSONB、TEXT、COMMENT ON、CONSTRAINT/CREATE INDEX;初始化数据改 JSON 字面量并 `setval` 推进序列                                                                                                                                                                                                   |
| 业务 SQL | 反引号标识符 → 双引号;bool 列 `= 1/0` → `= true/false`;`DATE_SUB(NOW(), INTERVAL ? DAY)` → `now() - (? * interval '1 day')`;`DATE(paid_at)` → `paid_at::date`;`gorm:"type:json"` → `type:jsonb`、`mediumtext` → `text`                                                                                                                                              |
| 部署 | docker-compose `mysql` 服务 → `postgres:16-alpine`,端口 **127.0.0.1:5433:5433**(容器内 `-p 5433`),`pg_isready` 健康检查;`redis` 补 `127.0.0.1:6379:6379` 发布(dev.sh 的 api/worker 为宿主机进程);Makefile 迁移 `-tags 'postgres'`;`scripts/dev.sh` 合并启停(无参=启动,`-stop`=关闭含 `docker compose stop` 容器;`--env-file` 读 `server/.env.dev`,变量单源;旧 docker run 容器检测拦截;dev-down.sh 已删除) |
| 测试 | 8 个 service 测试文件 sqlmock 期望切 PG 语法(双引号/`$n`/INSERT RETURNING),驱动 `postgres.New(Conn+PreferSimpleProtocol)`;**71 个测试函数全绿(2026-08-14 实测)**                                                                                                                                                                                                                                 |
| 实测 | postgres:16 容器实测:迁移、注册(INSERT RETURNING)、JSONB 读写(/config、优惠券)、事务下单、admin dashboard、worker 启动全部通过                                                                                                                                                                                                                                                        |

### 环境文件拆分:server/.env → .env.dev / .env.release(✅ 完成,2026-08-13)

| 项 | 说明 |
|---|---|
| 背景 | 原单一 `server/.env`(含真实 SMTP 密钥)不再区分环境,且 `scripts/dev.sh` 与 `docker-compose.yml` 均直接引用 `.env` |
| 方案 | 拆为 `server/.env.dev`(本地联调,含真实 SMTP)与 `server/.env.release`(生产发布,密钥占位待填);两者均加入 `.gitignore` **不入库**;`.env.example` 保留为无敏感模板(`cp .env.example .env.dev` / `.env.release`) |
| 引用改造 | `scripts/dev.sh` 的 `ENV_FILE` 改读 `server/.env.dev`(注释同步);`docker-compose.yml` api/worker `env_file: .env` → `${ENV_FILE:-.env.dev}`,生产用 `ENV_FILE=.env.release docker compose up -d` 一条命令切换 |
| DSN 差异 | dev 的 api/worker 为宿主机进程(DSN 127.0.0.1:5433,dev.sh 显式 export 覆盖);release 容器内用服务名 `host=postgres`;`APP_ENV` 分别 development/production(控制 Swagger/debug) |
| 验证 | `docker compose --env-file .env.dev config` 与 `ENV_FILE=.env.release docker compose config` 均正确解析对应环境变量 |

### 0.7.0 评审(✅ 已修复,2026-08-13;发现 2×P2 + 3×P3 已全部修复,见 docs/reviews/review-0.7.0.md)

| 项 | 说明 | 状态 |
|---|---|---|
| 评审范围 | commit `6f9b8d5`（MySQL→PostgreSQL 16 + dev.sh 合并）+ `8a62f74`（server dev/release 环境拆分）;`go test ./...` 60/60 全绿;compose config 解析正常 | ✅ 已完成 |
| [P2] release 模板 `APP_ENV=development` | `.env.example` 复制为 `.env.release` 后仍为 development,生产会开 Swagger/debug(`router.New` 仅 production 关闭) | ✅ 已修复(2026-08-13):模板默认 `APP_ENV=production`,dev.sh 强制宿主机进程 `export APP_ENV=development`,deploy.md 同步说明 |
| [P2] deploy.md 迁移顺序 | §4 步骤 1 在 postgres 启动前执行 `make migrate`,新主机 5433 无监听、迁移失败且容器不自动迁移 | ✅ 已修复(2026-08-13):§4 重排为先起 postgres/redis → 迁移 → 起全部服务 |
| [P3] dev.sh 缺 `.env.dev` 兜底失效 | `docker compose --env-file` 对缺失文件直接报错(实测 `couldn't find env file`),与注释承诺不符,`-stop` 同样受影响 | ✅ 已修复(2026-08-13):缺失时生成 `$RUN_DIR/env.fallback`(默认基础设施变量)并让 compose 指向它,启动/-stop 均生效 |
| [P3] dev.sh `-stop` 停全项目 | `docker compose stop` 无服务参数,会连 api/worker/caddy 一起停 | ✅ 已修复(2026-08-13):改为 `stop postgres redis` |
| [P3] docs/README.md 架构图错行 | Redis 行与「拉取节点」行合并,框线排版错乱 | ✅ 已修复(2026-08-13):拆回两行,框线对齐恢复 |

### 一期遗留缺口(候补项,排在二期之前)

> 以下候补项不属于一期交付承诺,列为候补;统一排在一期收尾之后、二期启动之前补齐,不再单独设档。

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| Grafana 看板 | ✅ 已实现(2026-08-22;2026-08-25 扩展):`docker compose --profile obs up -d` 一键启动 prometheus+grafana+alertmanager+node-exporter(compose 内网抓取 api:8081/worker:8082/node-exporter:9100,默认 up 不含);看板「YLink API」与「YLink 基础设施」(worker cron + 机器层)经 provisioning 自动加载(`deploy/obs/`);**数据保留半年**(retention.time=180d,原 15d);告警规则 rules.yml 经 alertmanager 邮件通知(ALERT_EMAIL_TO);Grafana 绑 127.0.0.1:3000;生产 Caddyfile `@metrics` 拦截公网 /metrics(纵深防御) | `server/deploy/obs/`、`docker-compose.yml`、`deploy/Caddyfile` |
| 工单用户侧「已回复」桌面端本地通知 | ✅ 已实现(属前端,2026-08-14):前端轮询/聚焦即时检查已具备并增强,后端无需改动 | 前端 `useLocalNotifications.ts` + `MainLayout.vue`(见 frontend/progress.md) |
| 移动端深链/一键导入(前端侧) | ✅ 已实现(属前端,2026-08-13):订阅端点已就绪 | — |
| 上线前置准备清单 | ✅ 已创建(2026-08-22):域名/DNS、服务器环境、第三方账号(易支付/SMTP)、.env.release 真实值逐项 checklist + 本地预演项,不部署提前办理 | `docs/backend/launch-checklist.md` |

### 二期 / 明确标注待办

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| 流量模式 A(节点上报 `POST /node/report`) | ✅ 已实现(2026-08-22,二期):每用户凭证(节点 config 开启后) + X-Node-Key 鉴权 + 累计值差分幂等累加,详见 §1 B8;演示 agent `cmd/node-agent` | 契约 §17;真实代理后端(Xray stats 等)对接见 [node-agent-guide.md](node-agent-guide.md) |
| 订阅「重开一次」工单 | ✅ 已实现(2026-08-12):`POST /tickets/{id}/reopen` + `reopen_count` 字段(迁移 0003);前端详情页已关闭且未重开时显示「重新打开」 | core-flows 第 7 节 |
| 订单超时主动查单后关闭待支付支付单 | ✅ 已完善(2026-08-13):超时关单同步关闭该订单残留待支付支付单;查单任务发现订单已非待支付时关闭支付单并跳过查单(防残留反复轮询) | `cron_service.go` + `PaymentRepo.ClosePendingByOrderNo` + 2 测试 |
| 后端 CI / Release 接入 | ❌ 不接入(项目决策,2026-08-12 确认) | 后端不走 GitHub Actions——无 CI job、无镜像构建/部署流水线;`backend` job 与 `deploy-backend.yml` 已删除;构建/部署走本机 `make` + 手动流程(见 deploy.md) |

---

## 3. 前置条件(运行 / 联调 / 上线)

### 3.1 本地运行

| 前置 | 说明 |
|---|---|
| Go ≥ 1.26.1（`go.mod` 声明） | ✅ 已满足:本机 1.26.1(2026-08-12 实测) |
| PostgreSQL 16 | ⚠️ 运行时前置(2026-08-13 起由 MySQL 8 切换),端口 **5433**(容器内 5433:5433);库 `ylink-backend`(默认用户 `ylink`/密码 `ylink_root`);DSN 见 `configs/config.yaml` 或 `APP_DATABASE_DSN`;本机有 Docker 时 `bash scripts/dev.sh` 经 `server/docker-compose.yml` 编排 postgres+redis(**变量读 `server/.env.dev`**,`docker compose --env-file`) |
| Redis 7 | ⚠️ 运行时前置,2026-08-12 本机实测 **6379 未监听且 `redis-server` 命令不在 PATH**,需先启动本地 Redis;本地 `127.0.0.1:6379`(默认无密码);生产必须设密码;`server/docker-compose.yml` 可一键起 |
| 全容器联调 | `bash scripts/dev-docker.sh`(2026-08-14 新增):**api/worker 也跑 Docker compose**,前端 `pnpm build` 产物由 Caddy 容器托管(`http://localhost` 静态 + `/api/*` 同域反代 api:8081);与 dev.sh(宿主机进程 + Vite dev)二选一;`-stop` 停全部容器;env 以 `server/.env.dev` 为**唯一来源、缺失即报错**(2026-08-14 完全重写,脚本**不内置任何默认值、不生成 env 文件**):`.env.dev` 不存在直接报错并提示 `cp server/.env.example server/.env.dev`;必需基础设施变量(`POSTGRES_USER/POSTGRES_PASSWORD/POSTGRES_DB/REDIS_PASSWORD`)缺失同样报错。api/worker 的 env_file 与 compose 插值统一读 `.env.dev`,应用配置(`APP_*`/`ADMIN_*`/`DEMO_*`/SMTP 等)全部由 env_file 导入;override 仅覆盖容器模式差异:DSN/Redis 用服务名 `host=postgres`/`redis:6379` + `APP_ENV` 强制 development。演示账号 `DEMO_EMAIL/DEMO_PASSWORD` 亦从 `.env.dev` 读取(`.env.example` 模板已含,缺失时后端跳过创建)。网络与构建加固(2026-08-14):预拉镜像统一走 daoCloud 加速源,`golang:1.26-alpine`/`alpine:3.20` 构建基础镜像已纳入预拉;本机 Docker Desktop `daemon.json` 已配 `registry-mirrors: https://docker.m.daocloud.io`(重启 Docker Desktop 后全局生效,备份 `daemon.json.bak-20260814-012449`);`server/Dockerfile` 修复:①`COPY go.mod go.sum ./ && RUN go mod download` 同行错误拆分(COPY 与 RUN 不可用 `&&` 同行,原写法会把 `download` 误当 COPY 目标报 `cannot copy to non-directory`),②构建镜像 `golang:1.24-alpine` → `golang:1.26-alpine`(与 go.mod `go 1.26.1` 匹配),③`ENV GOPROXY=https://goproxy.cn,direct`(容器内直连 proxy.golang.org 超时);新增 `server/.dockerignore`(排除 `.env*` 密钥/`.codegraph`/`logs` 等,构建上下文 6.15MB → 5KB);④Caddyfile.dev 补 `root * /srv/panel`(缺 root 时 file_server 从工作目录 /srv 找文件,前端实际 404;加后 http://localhost/ 实测 200)。**MSYS 路径转换修复(2026-08-14)**:构建命令改 `MSYS_NO_PATHCONV=1 VITE_API_BASE_URL=/api/v1 pnpm build`——Git Bash 会把 `/api/v1` 转换成 `C:/Program Files/Git/api/v1` 写进产物默认 apiBase,页面以 file:// 打开时请求变成 `file:///C:/Program%20Files/Git/api/v1/auth/login` 报 "Not allowed to load local resource";加保护后产物默认值恢复 `/api/v1`(实测无残留)。 | `scripts/dev-docker.sh` |
| Web 登录 CORS | ✅ 已修复(2026-08-14):`dev-docker.sh` 构建前端时强制 `VITE_API_BASE_URL=/api/v1`,Web 经 Caddy 同域反代,不再跨端口直连 `localhost:8081`;`configs/config.yaml` CORS 白名单补充 `http://localhost` / `http://127.0.0.1`,直连 8081 的本地 Web 也能通过预检。**https 支持(2026-08-14)**:白名单同步补充 `https://localhost` / `https://127.0.0.1` / `https://localhost:5174` / `https://localhost:1420`(Vite https / Caddy 本地 TLS 联调),生产用 `https://panel.example.com`(见 deploy.md);新增 `middleware/cors_test.go`(https/http 放行、非白名单拒绝、OPTIONS 预检 204、无 Origin 不受影响)。Tauri 走 plugin-http 原生栈,不受 CORS 影响。已有浏览器若持久化过旧 `app:apiBase`,需在登录页点「重置后端接口地址」 | `scripts/dev-docker.sh`、`server/configs/config.yaml`、`server/internal/middleware/cors_test.go` |
| 迁移 | `DB_URL='...' make migrate` 执行 `migrations/0001_init.*`、`0002_balance_check.*`;或 docker-compose 内首启前执行 |
| SMTP | 验证码/提醒邮件需要可用 SMTP;未配置时**注册/找回流程无法完成**(验证码发送失败仅记日志) |
| 管理员账号 | 启动时经环境变量 `ADMIN_EMAIL`/`ADMIN_PASSWORD` 幂等创建首个 role=1 用户 |

### 3.2 支付联调(易支付)

| 前置 | 说明 |
|---|---|
| 网关配置 | `APP_PAYMENT_EPAY_GATEWAY/PID/KEY/METHODS`(methods 如 `alipay,wxpay`);未配置则在线支付渠道不可用(余额支付不受影响) |
| 回调地址 | 网关后台配置 `https://{APP_BASE_URL}/api/v1/payment/notify/{method}`;本机联调可用内网穿透 |
| 验签一致性 | 驱动按彩虹协议:除 sign/sign_type 外 key 升序拼接 + `&key={密钥}` md5 小写;与网关配置的 MD5 密钥一致 |
| 金额单位 | 请求网关为「元」(两位小数),服务端内部/库内为「分」;回调按分比对 |

### 3.3 订阅下发

| 前置 | 说明 |
|---|---|
| 节点数据 | 管理端录入节点分组与节点(`servers.config` 存协议参数 JSON:password/uuid/sni/network/path 等) |
| 套餐 group_ids | 套餐需关联节点分组 JSON 数组 |
| 外网可达 | `APP_BASE_URL` 必须为代理客户端可访问的域名,订阅链接/回调地址均以其拼接 |

### 3.4 上线(生产)

| 前置 | 说明 |
|---|---|
| 密钥注入 | DB/Redis/JWT(≥32 字节)/SMTP/EPAY 全部经环境变量或 secret,不进 git(见 `.env.example`) |
| TLS | Caddy 反代终止 HTTPS;`app.env=production` 关闭 Swagger/debug(网关兜底 `@swagger path /swagger /swagger/*`,裸路径同样 404,2026-08-22) |
| Web 前端托管 | (2026-08-14 落地)双域名:`api.example.com` 反代 api:8081 + `panel.example.com` 托管 SPA;`docker-compose.prod.yml` override(清 redis/api 宿主端口、caddy 改 build `deploy/Dockerfile.web` 把 dist+Caddyfile 打进 ylink-web 镜像);部署命令 `ENV_FILE=.env.release docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build`,先 `pnpm build` 生成 dist |
| CORS 白名单 | `cors.allow_origins` 配 Web 版域名 `https://panel.example.com`(config.yaml,生产改后重建 api 镜像;订阅端点已豁免任意来源) |
| worker 单实例 | cron 已带 Redis 分布式锁,多实例部署安全 |
| 备份 | 按 deploy.md 第 6 节:pg_dump -Fc 全量 + WAL 归档保留 14 天,Redis AOF everysec |

---

## 4. 与契约文档的对照基准

- 端点、错误码、信封格式:以 [docs/api/README.md](../api/README.md) 为准(本实现已对齐第 2 节错误码、1.1 信封、1.4 单位约定)
- 表结构与 Redis Key:以 [docs/backend/data-model.md](../backend/data-model.md) 为准(实现额外新增 `agent_applies` 表承载代理申请)
- 业务状态机:以 [docs/backend/core-flows.md](../backend/core-flows.md) 为准
- 部署:以 [docs/backend/deploy.md](../backend/deploy.md) 为准;上线前置准备(不部署提前办理)见 [launch-checklist.md](launch-checklist.md)
