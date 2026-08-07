# 后端开发 · 进度追踪(已完成 / 未完成 / 前置条件)

> 本文档记录 `server/` 目录 Go/Gin 后端的开发状态,是 docs/backend 与 docs/api 的实现对照表。
> 更新规则:每完成一个里程碑/修复一个缺陷,同步更新本文档「已完成」;新增缺口写入「未完成」并标注依赖。
> 最后更新:2026-08-11(全部里程碑 B1–B7 实现完成;2026-08-11 全量核对:go build/vet/test 实测全绿;全量审查 33 项修复完成,见「全量审查修复」;同日文档核对:与代码实况比对,待办见第 2 节)

---

## 1. 已完成项

### B1 骨架(✅ 完成)

| 项 | 说明 | 位置 |
|---|---|---|
| 工程初始化 | `go.mod`(module `ylink`)、双入口 `cmd/server` + `cmd/worker` | `server/go.mod` |
| 配置加载 | Viper:`configs/config.yaml` + `APP_` 前缀环境变量覆盖(点转下划线) | `internal/config/config.go` |
| 日志 | zap JSON + lumberjack 按天切割,双输出(stdout + 文件) | `internal/pkg/logger/logger.go` |
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
| `GET /admin/orders`、`POST /admin/orders/{no}/refund` | 订单列表;退款(余额退回+优惠券回退+佣金回滚+审计,行锁) |
| `GET/POST/PUT/DELETE /admin/coupons` | 优惠券 CRUD(列表返回 `AdminCouponView`:展开 type/value(元)/min_spend(元)/used_count/valid_periods/plan_ids,见 dto_admin.go) |
| `GET/POST/PUT/DELETE /admin/notices`、`/admin/knowledges` | 公告/知识库 CRUD(bluemonday 清洗;**GET 列表含隐藏**,2026-08-11 补齐) |
| `GET /admin/tickets`、`GET /admin/tickets/{id}`、`POST /admin/tickets/{id}/reply|close` | 工单管理(客服回复→已回复) |
| `GET /admin/agent/applies`、`POST /admin/agent/applies/{id}/approve|reject` | 代理审批(行锁防并发;通过→role=2,审计) |
| `GET /admin/commission-logs` | 佣金日志(含用户邮箱) |
| `POST /admin/traffic/import` | 模式 B 流量导入(audit 审计) |
| `GET/PUT /admin/settings` | 站点配置读写(写后失效 Redis 缓存) |

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
| 死代码清理 | 删除 `ListPendingOrderNos`/`GetByNoAdmin`/`IncrUsed`/`SetString` |

### 增强(✅ 完成)

| 项 | 说明 |
|---|---|
| `GET /metrics` | promhttp:请求计数/延迟直方图/支付成功计数器(`payment_success_total`),回调成功打点 |
| `GET /swagger/*` | gin-swagger,仅 development 环境;`make swagger` 重新生成 |
| 支付回执邮件 | 在线回调与余额支付成功后异步发送(`[站点] 支付成功`),未配置 SMTP 静默跳过 |
| agent-audit cron | 每月 1 日 03:00 复核代理有效邀请数,不达标降级 role=0 |

### 测试状态(✅ 更新,2026-08-10 实测)

- `go build ./...` / `go vet ./...` / `gofmt -l`(0 输出)全部通过
- `go test ./... -count=1` 全绿;**47 个测试函数**,新增覆盖:bluemonday 清洗、优惠券超限 12001、优惠券原子占用、超时关单(含优惠券回退)、取消并发已支付(0 行回滚)、佣金确认竞态、退款佣金回滚、代理审批

---

## 2. 未完成项

### 二期 / 明确标注待办

| 项 | 状态 | 依赖/说明 |
|---|---|---|
| 流量模式 A(节点上报 `POST /node/report`) | 未实现,一期为模式 B(手工导入) | 需节点 agent 端实现与节点密钥鉴权 |
| 移动端深链/一键导入(前端侧) | 属于前端,后端无需改动 | 订阅端点已就绪 |
| 订阅「重开一次」工单 | 未实现(core-flows 第 7 节标注二期可做) | — |
| 订单超时主动查单后关闭待支付支付单 | 查单兜底已实现;查单失败/支付单长期待支付的关闭策略可二期完善 | — |
| 工单用户侧「已回复」桌面端本地通知 | 前端轮询已具备数据(状态变化),本地通知属前端 | — |
| Grafana 看板 | `/metrics` 已暴露,看板配置未做 | 依赖运维侧导入 dashboards |
| 后端 CI / Release 接入 | 未接入 GitHub Actions | 当前 `.github/workflows/ci.yml` 仅前端 quality + e2e;`server/` 无独立 CI job,镜像构建/发布未配流水线 |

### 一期内已知缺口(可后补)

| 项 | 说明 |
|---|---|
| 封禁/降级对已签发 JWT 无效 | Auth 中间件只校验 token 快照,封禁后最多 2h(access TTL)内旧 token 仍可用;严格实时生效需中间件查库或 Redis 黑名单(量级小可接受) |
| 余额支付负余额保护 | 划转/调余额允许结果出现负值(佣金扣回场景允许为负,记账审计);如需强约束可加 CHECK 约束(二期) |

---

## 3. 前置条件(运行 / 联调 / 上线)

### 3.1 本地运行

| 前置 | 说明 |
|---|---|
| Go ≥ 1.26.1（`go.mod` 声明） | 本机 1.26.1 已满足 |
| MySQL 8.0 | 库 `ylink`(utf8mb4);DSN 见 `configs/config.yaml` 或 `APP_DATABASE_DSN` |
| Redis 7 | 本地 `127.0.0.1:6379`(默认无密码);生产必须设密码 |
| 迁移 | `DB_URL='...' make migrate` 执行 `migrations/0001_init.*`;或 docker-compose 内首启前执行 |
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
| CORS 白名单 | `cors.allow_origins` 配 Web 版域名(订阅端点已豁免任意来源) |
| worker 单实例 | cron 已带 Redis 分布式锁,多实例部署安全 |
| 备份 | 按 deploy.md 第 6 节:mysqldump + binlog 保留 14 天,Redis AOF everysec |

---

## 4. 与契约文档的对照基准

- 端点、错误码、信封格式:以 [docs/api/README.md](../api/README.md) 为准(本实现已对齐第 2 节错误码、1.1 信封、1.4 单位约定)
- 表结构与 Redis Key:以 [docs/backend/data-model.md](../backend/data-model.md) 为准(实现额外新增 `agent_applies` 表承载代理申请)
- 业务状态机:以 [docs/backend/core-flows.md](../backend/core-flows.md) 为准
- 部署:以 [docs/backend/deploy.md](../backend/deploy.md) 为准
