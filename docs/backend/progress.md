# 后端开发 · 当前状态

> 本文档描述 `server/` 目录 Go/Gin 后端的**当前能力与状态**,是 docs/backend 与 docs/api 的实现对照表。
> 维护规则:只记录当前态(能力清单、未完成项、前置条件),不堆叠历史流水账。端点与错误码以 [docs/api/README.md](../api/README.md) 为准,表结构与 Redis Key 以 [data-model.md](data-model.md) 为准,业务状态机以 [core-flows.md](core-flows.md) 为准;历史修复明细见 [docs/reviews/](../reviews/) 与 git log。

## 1. 状态总览(2026-08-30 实测)

| 项 | 状态 |
|---|---|
| 构建/静态检查 | `go build ./...`、`go vet ./...`、`gofmt -l` 全部通过 |
| 测试 | `go test ./... -count=1` 全绿,**144 个测试函数** |
| 契约对齐 | [docs/api/README.md](../api/README.md) 全量端点已实现(含 §16 管理端与批次新增模块) |
| 迁移 | `migrations/0001` ~ `0008`(golang-migrate,均含 down) |
| 代码结构 | 管理端 handler 按业务域拆分为 `internal/handler/admin/` 子包(2026-08-30 重构,路由与方法签名零改动) |

## 2. 已完成能力

### 2.1 工程骨架

双入口 `cmd/server` + `cmd/worker`;Viper 配置(`configs/config.yaml` + `APP_` 前缀环境变量覆盖);zap JSON 日志按天切割(stdout + 文件);中间件链 RequestID → Recovery → AccessLog → CORS → RateLimit → [Auth] → [Idempotency];统一响应信封 `{code,message,data}` + 分页(分页列表 nil slice 统一兜底为 `[]`);binding 错误统一 40000;bcrypt(12);JWT access 2h / refresh 14d(TokenType 防混用,refresh 白名单存 Redis);Makefile / Dockerfile / docker-compose / Caddyfile。

### 2.2 账户与安全

邮箱验证码注册/登录/找回(60s 限频、同 IP 日上限、失败 5 次锁 10min);refresh 旋转(旧 jti 立即失效);改密吊销其他会话保留当前;**会话版本号 SV**——封禁/解封/角色变更/代理审批/降级/找回密码实时失效 access(Redis `auth:ver:{uid}`);**单会话踢下线**(`auth:kill:{uid}`,被踢会话立即 401,其余不受影响,F14);bluemonday XSS 清洗覆盖公告/知识库/套餐/工单写入;`safe_mode` 域名白名单(非白名单 403,healthz/metrics 豁免)+ 管理端/订阅路径段定制(F22)。

### 2.3 交易闭环

套餐/优惠券/订单/支付全链路:`Idempotency-Key` 幂等(24h,key 按 user_id 隔离);优惠券**下单事务内原子占用**(取消/超时关单/退款三条路径统一回退);余额直付行锁扣减;在线渠道易支付(验签 → 金额比对 → 行锁幂等 → MarkPaid → 开通/续期);开通/续期状态机(叠加/替换/清零规则见 core-flows);退款 = 余额退回 + 收回订阅 + 优惠券回退 + 佣金回滚(行锁);支付成功异步回执邮件。

### 2.4 订阅下发

`GET /client/subscribe/{token}`:flag=clash/sing-box/v2ray + UA 嗅探;token 级限流;userinfo 缓存 30s;`subscription-userinfo` 等响应头;无订阅/到期/流量用尽注入提示节点;生成器为 Go text/template 渲染(公共/客户端变量,节点语法对模板作者不可破坏),**自定义模板缺失或渲染失败自动回退内置生成器,订阅不 5xx**(F10);6 种协议(shadowsocks/vmess/vless/trojan/hysteria2/tuic)。

### 2.5 营销与代理

邀请码(8 位,生成上限)/佣金划转(行锁)/佣金记录;佣金 cron 每日确认(**严格排除提现流水**);代理申请/审批(行锁防并发)/有效邀请复核 cron;**佣金提现**(F02):仅代理商(`role=RoleAgent`)经工单提交,提交即扣减 `commission_balance` 防双花,管理端确认打款或拒绝(自动退回佣金),三态全程审计。

### 2.6 工单与定时任务

工单创建/回复/关闭/重开一次(`reopen_count`);worker cron(Redis 分布式锁 + 指标打点):`close-expired-orders`(5min,条件更新防竞态)、`reconcile-payments`(10min 主动查单补账)、`confirm-commissions`(每日 02:00)、`expire-remind`(10:00 前 3/1 天双窗口)、`traffic-remind`(10:30 ≥80%)、`traffic-daily`(01:00 模式 B 空跑);到期/流量提醒在邮件后对已绑定用户追加 Telegram 推送(失败仅记日志不阻断,F12)。

### 2.7 管理端

契约 §16 全量端点:概览(含全体用户余额)、用户(CSV 流式导出/批量操作/发邮件 mail_logs 留痕/重置订阅密钥)、套餐、节点(批量/复制/排序/node-key 重置)、订单(佣金列/退款)、优惠券、公告/知识库(排序/分类管理)、工单(提现审核)、代理审批、佣金日志、流量(导入/重置/重置记录)、审计日志(只读查询 + target 可读化)、站点设置、邮件模板(编辑/测试发送/恢复默认,失败回退内置文案)、订阅模板(编辑/预览/恢复内置)、版本检查(`GET /admin/version` 语义化比较)、Telegram webhook 注册。

handler 按业务域组织于 `internal/handler/admin/` 子包:`admin.go`(Admin 结构体+helper)、`dashboard.go`、`user.go`、`plan.go`、`server.go`、`order.go`、`content.go`、`ticket.go`、`agent.go`、`system.go`。

### 2.8 节点上报(流量模式 A)

`X-Node-Key` 鉴权(Redis 缓存 60s);`GET /node/users` 按节点分组下发有效订阅用户;`POST /node/report` **累计值差分**幂等(重复上报差分 0;累计回退视为节点重启)→ ×rate → 事务内 `users.u/d` 原子累加 + `traffic_logs` 增量聚合;套餐分组校验(非本分组 `not_subscribed`)与重复 UUID 整体拒绝(`duplicate_uuid`);每用户订阅凭证(节点 config 显式开启 `per_user_credentials` 才下发,存量节点共享凭证不断连);演示 agent `cmd/node-agent`;真实代理后端(Xray stats 等)对接见 [node-agent-guide.md](node-agent-guide.md)。

### 2.9 Xboard 缺口补齐(四批全部完成)

需求与决策见 [.scratch/xboard-gap-fill/spec.md](../../.scratch/xboard-gap-fill/spec.md);2026-08-28 第一批(F08/F22/F05)、第二批(F09/F16/F04)、第三批(F02/F15/F14/F11/F19/F20)、2026-08-29 第四批(F10/F12)。

### 2.10 可观测性

`GET /metrics`(promhttp:请求计数/延迟直方图/支付成功计数)+ worker `:8082/metrics`(cron 运行/耗时);`docker compose --profile obs` 一键 Prometheus/Grafana/Alertmanager/node-exporter,看板 provisioning 自动加载,数据保留 180 天,告警经 Alertmanager 邮件(STARTTLS);生产 Caddy 拦截公网 /metrics。

## 3. 未完成与已决策

| 项 | 状态 |
|---|---|
| F17 节点机器管理 | 缓排(见 xboard spec);当前手工对接 + node-agent-guide 已可运转 |
| 后端 CI / Release 流水线 | ❌ 不接入(项目决策,2026-08-12 确认):构建/部署走本机 `make` + 手动流程,见 deploy.md |
| F18 节点路由 / F21 插件系统 | ❌ 不做(xboard spec 决策) |
| 完整主题包 / 订阅自动升级 | ❌ 不做(仅取品牌配置、版本检查最小子集) |

## 4. 前置条件(运行 / 联调 / 上线)

### 4.1 本地运行

| 前置 | 说明 |
|---|---|
| Go ≥ 1.26.1 | `go.mod` 声明 |
| PostgreSQL 16 | 端口 **5433**(容器内 5433);库 `ylink-backend`(默认用户 `ylink`/密码 `ylink_root`);DSN 见 `configs/config.yaml` 或 `APP_DATABASE_DSN` |
| Redis 7 | 本地 `127.0.0.1:6379`(默认无密码);生产必须设密码 |
| 一键编排 | `bash scripts/dev.sh`:compose 起 postgres+redis,api/worker 为宿主机进程;`-stop` 只停基础设施容器 |
| 全容器联调 | `bash scripts/dev-docker.sh`:api/worker 也跑 compose,前端 `pnpm build` 产物由 Caddy 容器托管(`http://localhost` 静态 + `/api/*` 同域反代);`-stop` 停基础设施。**env 以 `server/.env.dev` 为唯一来源,缺失即报错**(提示 `cp server/.env.example server/.env.dev`,必需基础设施变量缺失同样报错)。Windows 构建前端需 `MSYS_NO_PATHCONV=1 VITE_API_BASE_URL=/api/v1 pnpm build`(防 Git Bash 路径转换写坏产物默认 apiBase) |
| 环境文件 | `server/.env.dev`(本地联调,含真实密钥,不入库)/ `server/.env.release`(生产,密钥占位待填),均 gitignore;`.env.example` 为无敏感模板;compose 经 `${ENV_FILE:-.env.dev}` 选择,生产 `ENV_FILE=.env.release docker compose up -d` |
| 迁移 | `DB_URL='...' make migrate`,或 compose 首启前执行 |
| SMTP | 验证码/提醒邮件需要;未配置时**注册/找回流程无法完成**(发送失败仅记日志) |
| 管理员账号 | 启动时经环境变量 `ADMIN_EMAIL`/`ADMIN_PASSWORD` 幂等创建首个 role=1 用户 |

### 4.2 支付联调(易支付)

网关配置 `APP_PAYMENT_EPAY_GATEWAY/PID/KEY/METHODS`,未配置则在线渠道不可用(余额支付不受影响);回调地址 `https://{APP_BASE_URL}/api/v1/payment/notify/{method}`(本机联调用内网穿透);验签按彩虹协议(除 sign/sign_type 外 key 升序拼接 + `&key={密钥}` md5 小写);金额:请求网关「元」、服务端/库内「分」、回调按分比对。

### 4.3 订阅下发

节点录入 `servers.config` 存协议参数 JSON(password/uuid/sni/network/path 等);套餐 `group_ids` 关联节点分组;`APP_BASE_URL` 必须为代理客户端可访问的域名(订阅链接/回调均以其拼接)。

### 4.4 上线(生产)

密钥(DB/Redis/JWT≥32 字节/SMTP/EPAY)全部经环境变量或 secret 注入,不进 git;Caddy 反代终止 HTTPS,`app.env=production` 关闭 Swagger/debug(网关兜底拦 `/swagger`);双域名部署:`api.example.com` 反代 api:8081 + `panel.example.com` 托管 SPA(`docker-compose.prod.yml` override,部署命令见 deploy.md);CORS 白名单配 Web 域名(订阅端点豁免任意来源);worker 多实例安全(cron 已带分布式锁);备份按 deploy.md §6(pg_dump -Fc 全量 + WAL 归档 14 天,Redis AOF everysec);上线前置检查清单见 [launch-checklist.md](launch-checklist.md)。

## 5. 历史记录指引

- 版本评审与修复明细:[docs/reviews/](../reviews/)(review-0.2.0 ~ review-0.9.0,冻结快照)
- 需求立项与决策:[.scratch/](../../.scratch/)(各 feature 的 spec.md,review-fixes 批次含 issues/ 工单)
- 逐次变更:git log(Conventional Commits)
