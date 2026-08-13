# 后端开发 · 进度追踪(已完成 / 未完成 / 前置条件)

> 本文档记录 `server/` 目录 Go/Gin 后端的开发状态,是 docs/backend 与 docs/api 的实现对照表。
> 更新规则:每完成一个里程碑/修复一个缺陷,同步更新本文档「已完成」;新增缺口写入「未完成」并标注依赖。
> 最后更新:2026-08-13(数据库 MySQL 8 → PostgreSQL 16 切换完成,端口 5433:5433,见 §1 里程碑;测试函数 60 个全绿;全部里程碑 B1–B7 + 管理端 API 完成;0.7.0 评审 2×P2 + 3×P3 已全部修复,见 docs/reviews/review-0.7.0.md)

---

## 1. 已完成项

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
| `GET`/`POST /invite/codes` | 8 位随机码列表;生成超限 13001(上限取 settings) |
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

### 第四轮评审（v0.5.0，2026-08-13）

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

### 全量审查修复(✅ 2026-08-11)

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

### 0.5.0 审查修复(✅ 2026-08-13,见 docs/reviews/review-0.5.0.md)

| 项 | 说明 |
|---|---|
| 佣金批量查询失败被静默忽略 | `ListOrders` 原 `if comms, err := ...; err == nil` 吞掉 `ListByOrderNos` 错误,查询失败仍返回成功响应、`commission_amount` 全 null,把数据缺失误显示为「无佣金」;改为失败即返回错误,补 `TestAdminListOrdersCommissionQueryError` 回归 |

### 增强(✅ 完成)

| 项 | 说明 |
|---|---|
| `GET /metrics` | promhttp:请求计数/延迟直方图/支付成功计数器(`payment_success_total`),回调成功打点 |
| `GET /swagger/*` | gin-swagger,仅 development 环境;`make swagger` 重新生成 |
| 支付回执邮件 | 在线回调与余额支付成功后异步发送(`[站点] 支付成功`),未配置 SMTP 静默跳过 |
| agent-audit cron | 每月 1 日 03:00 复核代理有效邀请数,不达标降级 role=0 |

### 测试状态(✅ 更新,2026-08-11 实测)

- `go build ./...` / `go vet ./...` / `gofmt -l`(0 输出)全部通过
- `go test ./... -count=1` 全绿;**68 个测试函数**(2026-08-13 实测:源码 `func Test` 68 个,含工单重开 4 例 + 优惠券每人限用 1 例 + 0.5.0 佣金查询失败回归 1 例,0 失败/跳过),覆盖:错误码映射、JWT(含 SV 会话版本号)、密码、验证码限频/已注册、注册/登录锁定/刷新旋转、优惠券试算(固定/百分比/封顶)/超限 12001/原子占用/**每人限用(同用户同券第二次下单被拒)**、下单幂等、续期状态机、回调幂等、epay 验签与篡改拒绝、订阅生成(3 格式)、佣金划转、代理申请、工单流转、**工单重开(重开成功/未关闭拒绝/仅一次 14002/并发 0 行拒绝)**、佣金确认竞态、超时关单(含优惠券回退)、取消并发已支付回滚、**退款(余额/券/佣金/订阅收回/onetime/异套餐)**,代理审批、bluemonday 清洗、Auth 中间件(无头/无效/refresh 混用/SV 匹配/bump 立即失效)、余额调整负值拒绝、**管理端订单佣金查询(成功映射/失败上抛)**

---

## 2. 未完成项

### 二期 / 明确标注待办

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| 流量模式 A(节点上报 `POST /node/report`) | 未实现,一期为模式 B(手工导入) | 需节点 agent 端实现与节点密钥鉴权 |
| 移动端深链/一键导入(前端侧) | 属于前端,后端无需改动 | 订阅端点已就绪 |
| 订阅「重开一次」工单 | ~~未实现~~ **✅ 已实现(2026-08-12)**:`POST /tickets/{id}/reopen` + `reopen_count` 字段(迁移 0003);前端详情页已关闭且未重开时显示「重新打开」 | core-flows 第 7 节 |
| 订单超时主动查单后关闭待支付支付单 | ✅ 已完善(2026-08-13):超时关单同步关闭该订单残留待支付支付单;查单任务发现订单已非待支付时关闭支付单并跳过查单(防残留反复轮询) | `cron_service.go` + `PaymentRepo.ClosePendingByOrderNo` + 2 测试 |
| 工单用户侧「已回复」桌面端本地通知 | 前端轮询已具备数据(状态变化),本地通知属前端 | — |
| Grafana 看板 | `/metrics` 已暴露,看板配置未做 | 依赖运维侧导入 dashboards |
| 后端 CI / Release 接入 | ❌ 不接入(项目决策,2026-08-12 确认) | 后端不走 GitHub Actions——无 CI job、无镜像构建/部署流水线;`backend` job 与 `deploy-backend.yml` 已删除;构建/部署走本机 `make` + 手动流程(见 deploy.md) |

### 一期小缺口收尾(✅ 2026-08-11)

| 项 | 说明 |
|---|---|
| 封禁/降级后 JWT 实时失效 | 会话版本号机制:Claims 增加 `SV`(签发时快照,`jwt.Generate` 带参);Redis `auth:ver:{uid}` 存当前版本;封禁/解封/角色变更/代理审批通过/代理商降级/找回密码/登出均 `INCR` bump;Auth 中间件比对 SV 不一致即 401(Key 不存在视为 0,Redis 异常不阻断退化为 TTL)。access 2h 内无需等过期 |
| 余额负值保护 | `AdjustBalance` 服务层拒绝调整后余额为负(40000)+ 迁移 `0002_balance_check` 加 `CHECK (balance >= 0)`(PostgreSQL CHECK 强制执行);佣金回滚减 `commission_balance` 不受约束 |
| 迁移 | `server/migrations/0002_balance_check.{up,down}.sql`(golang-migrate 格式,新增约束/回滚) |
| 测试 | 新增 `middleware/auth_test.go`(6 场景:无头/无效/refresh 混用/有效/SV 匹配/bump 后立即失效+重新签发恢复)、admin_service 3 例(负值拒绝/封禁 bump/角色 bump);**测试函数 47 → 60 全绿** |
| 既有 format:check 告警(前端) | **已解决(2026-08-12)**：前端 `pnpm format:check` 全仓通过(见 docs/reviews/review-0.4.0.md)；后端 `gofmt -l` 0 输出 |

### 数据库切换:MySQL → PostgreSQL(✅ 2026-08-13)

| 项 | 说明                                                                                                                                                                                                                                                                                                                                                       |
|---|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 驱动 | `gorm.io/driver/mysql` → `gorm.io/driver/postgres` v1.6.2(pgx v5);`internal/repo/repo.go` 切 `postgres.Open`                                                                                                                                                                                                                                              |
| 迁移 SQL | `migrations/*.sql` 全部重写为 PG 语法:BIGSERIAL 主键、SMALLINT、BOOLEAN、TIMESTAMP(3)、JSONB、TEXT、COMMENT ON、CONSTRAINT/CREATE INDEX;初始化数据改 JSON 字面量并 `setval` 推进序列                                                                                                                                                                                                   |
| 业务 SQL | 反引号标识符 → 双引号;bool 列 `= 1/0` → `= true/false`;`DATE_SUB(NOW(), INTERVAL ? DAY)` → `now() - (? * interval '1 day')`;`DATE(paid_at)` → `paid_at::date`;`gorm:"type:json"` → `type:jsonb`、`mediumtext` → `text`                                                                                                                                              |
| 部署 | docker-compose `mysql` 服务 → `postgres:16-alpine`,端口 **127.0.0.1:5433:5433**(容器内 `-p 5433`),`pg_isready` 健康检查;`redis` 补 `127.0.0.1:6379:6379` 发布(dev.sh 的 api/worker 为宿主机进程);Makefile 迁移 `-tags 'postgres'`;`scripts/dev.sh` 合并启停(无参=启动,`-stop`=关闭含 `docker compose stop` 容器;`--env-file` 读 `server/.env.dev`,变量单源;旧 docker run 容器检测拦截;dev-down.sh 已删除) |
| 测试 | 8 个 service 测试文件 sqlmock 期望切 PG 语法(双引号/`$n`/INSERT RETURNING),驱动 `postgres.New(Conn+PreferSimpleProtocol)`;**60 个测试函数全绿**                                                                                                                                                                                                                                |
| 实测 | postgres:16 容器实测:迁移、注册(INSERT RETURNING)、JSONB 读写(/config、优惠券)、事务下单、admin dashboard、worker 启动全部通过                                                                                                                                                                                                                                                        |

### 环境文件拆分:server/.env → .env.dev / .env.release(✅ 2026-08-13)

| 项 | 说明 |
|---|---|
| 背景 | 原单一 `server/.env`(含真实 SMTP 密钥)不再区分环境,且 `scripts/dev.sh` 与 `docker-compose.yml` 均直接引用 `.env` |
| 方案 | 拆为 `server/.env.dev`(本地联调,含真实 SMTP)与 `server/.env.release`(生产发布,密钥占位待填);两者均加入 `.gitignore` **不入库**;`.env.example` 保留为无敏感模板(`cp .env.example .env.dev` / `.env.release`) |
| 引用改造 | `scripts/dev.sh` 的 `ENV_FILE` 改读 `server/.env.dev`(注释同步);`docker-compose.yml` api/worker `env_file: .env` → `${ENV_FILE:-.env.dev}`,生产用 `ENV_FILE=.env.release docker compose up -d` 一条命令切换 |
| DSN 差异 | dev 的 api/worker 为宿主机进程(DSN 127.0.0.1:5433,dev.sh 显式 export 覆盖);release 容器内用服务名 `host=postgres`;`APP_ENV` 分别 development/production(控制 Swagger/debug) |
| 验证 | `docker compose --env-file .env.dev config` 与 `ENV_FILE=.env.release docker compose config` 均正确解析对应环境变量 |

### 0.7.0 评审（✅ 2026-08-13 出评审;发现 2×P2 + 3×P3 已全部修复,见 docs/reviews/review-0.7.0.md）

| 项 | 说明 | 状态 |
|---|---|---|
| 评审范围 | commit `6f9b8d5`（MySQL→PostgreSQL 16 + dev.sh 合并）+ `8a62f74`（server dev/release 环境拆分）;`go test ./...` 60/60 全绿;compose config 解析正常 | ✅ 已完成 |
| [P2] release 模板 `APP_ENV=development` | `.env.example` 复制为 `.env.release` 后仍为 development,生产会开 Swagger/debug(`router.New` 仅 production 关闭) | ✅ 已修复(2026-08-13):模板默认 `APP_ENV=production`,dev.sh 强制宿主机进程 `export APP_ENV=development`,deploy.md 同步说明 |
| [P2] deploy.md 迁移顺序 | §4 步骤 1 在 postgres 启动前执行 `make migrate`,新主机 5433 无监听、迁移失败且容器不自动迁移 | ✅ 已修复(2026-08-13):§4 重排为先起 postgres/redis → 迁移 → 起全部服务 |
| [P3] dev.sh 缺 `.env.dev` 兜底失效 | `docker compose --env-file` 对缺失文件直接报错(实测 `couldn't find env file`),与注释承诺不符,`-stop` 同样受影响 | ✅ 已修复(2026-08-13):缺失时生成 `$RUN_DIR/env.fallback`(默认基础设施变量)并让 compose 指向它,启动/-stop 均生效 |
| [P3] dev.sh `-stop` 停全项目 | `docker compose stop` 无服务参数,会连 api/worker/caddy 一起停 | ✅ 已修复(2026-08-13):改为 `stop postgres redis` |
| [P3] docs/README.md 架构图错行 | Redis 行与「拉取节点」行合并,框线排版错乱 | ✅ 已修复(2026-08-13):拆回两行,框线对齐恢复 |

---

## 3. 前置条件(运行 / 联调 / 上线)

### 3.1 本地运行

| 前置 | 说明 |
|---|---|
| Go ≥ 1.26.1（`go.mod` 声明） | ✅ 已满足:本机 1.26.1(2026-08-12 实测) |
| PostgreSQL 16 | ⚠️ 运行时前置(2026-08-13 起由 MySQL 8 切换),端口 **5433**(容器内 5433:5433);库 `ylink-backend`(默认用户 `ylink`/密码 `ylink_root`);DSN 见 `configs/config.yaml` 或 `APP_DATABASE_DSN`;本机有 Docker 时 `bash scripts/dev.sh` 经 `server/docker-compose.yml` 编排 postgres+redis(**变量读 `server/.env.dev`**,`docker compose --env-file`) |
| Redis 7 | ⚠️ 运行时前置,2026-08-12 本机实测 **6379 未监听且 `redis-server` 命令不在 PATH**,需先启动本地 Redis;本地 `127.0.0.1:6379`(默认无密码);生产必须设密码;`server/docker-compose.yml` 可一键起 |
| 全容器联调 | `bash scripts/dev-docker.sh`(2026-08-14 新增):**api/worker 也跑 Docker compose**,前端 `pnpm build` 产物由 Caddy 容器托管(`http://localhost` 静态 + `/api/*` 同域反代 api:8081);与 dev.sh(宿主机进程 + Vite dev)二选一;`-stop` 停全部容器;容器模式 env 生成于 `.dev/env.docker`(DSN 用服务名 `host=postgres`/`redis:6379`),`server/.env.dev` 仅作密钥源。网络与构建加固(2026-08-14):预拉镜像统一走 daoCloud 加速源,`golang:1.26-alpine`/`alpine:3.20` 构建基础镜像已纳入预拉;本机 Docker Desktop `daemon.json` 已配 `registry-mirrors: https://docker.m.daocloud.io`(重启 Docker Desktop 后全局生效,备份 `daemon.json.bak-20260814-012449`);`server/Dockerfile` 修复:①`COPY go.mod go.sum ./ && RUN go mod download` 同行错误拆分(COPY 与 RUN 不可用 `&&` 同行,原写法会把 `download` 误当 COPY 目标报 `cannot copy to non-directory`),②构建镜像 `golang:1.24-alpine` → `golang:1.26-alpine`(与 go.mod `go 1.26.1` 匹配),③`ENV GOPROXY=https://goproxy.cn,direct`(容器内直连 proxy.golang.org 超时);新增 `server/.dockerignore`(排除 `.env*` 密钥/`.codegraph`/`logs` 等,构建上下文 6.15MB → 5KB);④Caddyfile.dev 补 `root * /srv/panel`(缺 root 时 file_server 从工作目录 /srv 找文件,前端实际 404;加后 http://localhost/ 实测 200) |
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
| TLS | Caddy 反代终止 HTTPS;`app.env=production` 关闭 Swagger/debug |
| Web 前端托管 | (2026-08-14 落地)双域名:`api.example.com` 反代 api:8081 + `panel.example.com` 托管 SPA;`docker-compose.prod.yml` override(清 redis/api 宿主端口、caddy 改 build `deploy/Dockerfile.web` 把 dist+Caddyfile 打进 ylink-web 镜像);部署命令 `ENV_FILE=.env.release docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build`,先 `pnpm build` 生成 dist |
| CORS 白名单 | `cors.allow_origins` 配 Web 版域名 `https://panel.example.com`(config.yaml,生产改后重建 api 镜像;订阅端点已豁免任意来源) |
| worker 单实例 | cron 已带 Redis 分布式锁,多实例部署安全 |
| 备份 | 按 deploy.md 第 6 节:pg_dump -Fc 全量 + WAL 归档保留 14 天,Redis AOF everysec |

---

## 4. 与契约文档的对照基准

- 端点、错误码、信封格式:以 [docs/api/README.md](../api/README.md) 为准(本实现已对齐第 2 节错误码、1.1 信封、1.4 单位约定)
- 表结构与 Redis Key:以 [docs/backend/data-model.md](../backend/data-model.md) 为准(实现额外新增 `agent_applies` 表承载代理申请)
- 业务状态机:以 [docs/backend/core-flows.md](../backend/core-flows.md) 为准
- 部署:以 [docs/backend/deploy.md](../backend/deploy.md) 为准
