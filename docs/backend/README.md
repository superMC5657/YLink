# 后端开发文档 · 总览与架构

> Go/Gin 服务端：为用户端 App 提供 REST API，同时承担支付回调、订阅配置下发、定时任务（佣金结算/到期提醒）。管理端 API 同进程暴露（`/admin` 前缀）；管理后台前端已随主 SPA 实现 18 个管理页面（M8 核心 6 + M9 二期 7 + 缺口补齐新增 5）。

## 1. 技术选型

| 层次 | 选型 | 说明 |
|---|---|---|
| 语言 | Go 1.26.1 | `server/go.mod` 声明 |
| Web 框架 | Gin | 轻量、生态成熟 |
| ORM | GORM v2 | PostgreSQL 16；事务与关联查询方便 |
| 数据库 | PostgreSQL 16 | 业务数据（JSONB/事务/索引），端口 5433 |
| 缓存/队列 | Redis 7 | 邮箱验证码、限流、Token 会话、订单幂等、热数据缓存 |
| 鉴权 | JWT（golang-jwt/v5） | access(2h) + refresh(14d)，refresh 白名单存 Redis 可吊销 |
| 配置 | Viper | `config.yaml` + 环境变量覆盖 |
| 日志 | zap + lumberjack | JSON 结构化、按天切割 |
| 参数校验 | go-playground/validator | 绑定 Gin binding |
| 接口文档 | swaggo/swag | 注解生成 Swagger，与契约文档互校验 |
| 迁移 | golang-migrate | SQL 文件版本化迁移 |
| 定时任务 | robfig/cron/v3 | 佣金确认、到期提醒、流量日结转 |
| 邮件 | gomail（SMTP） | 验证码、到期/流量提醒 |
| 二维码 | skip2/go-qrcode | 支付二维码（若网关不返回图片） |
| 测试 | testify + miniredis + go-sqlmock | 单测（70 个测试函数，全绿） |

## 2. 分层架构

```
┌────────────────────────────────────────────────────────┐
│ router/        路由注册，按模块分组，挂载中间件           │
├────────────────────────────────────────────────────────┤
│ middleware/    Recovery/Logger/CORS/RateLimit/Auth/    │
│                Idempotency/RequestID                    │
├────────────────────────────────────────────────────────┤
│ handler/       参数绑定与校验、调用 service、组装响应     │  ← 不写业务
├────────────────────────────────────────────────────────┤
│ service/       业务逻辑、事务边界、领域规则               │  ← 单测重点
├────────────────────────────────────────────────────────┤
│ repo/          GORM 数据访问，只做 CRUD 与查询构造        │  ← 不写业务
├────────────────────────────────────────────────────────┤
│ model/         表结构体、枚举、DTO（req/resp）            │
└────────────────────────────────────────────────────────┘
   pkg/（jwt, payment 驱动, subscribe 生成器, mailer, redis, resp, errs）
```

约束：
1. 依赖单向：handler → service → repo，禁止反向；跨模块调用走 service 接口。
2. 事务只在 service 层开启（`db.Transaction`），repo 方法接收 `*gorm.DB` 以便纳入事务。
3. 所有对外响应经 `pkg/resp` 统一封装为 envelope `{code, message, data}`，错误用 `pkg/errs` 的业务错误类型（携带业务码），handler 不手写 JSON。
4. 支付、订阅生成等外部差异点用接口隔离（见第 6、7 节）。

## 3. 目录结构

```
server/
├── cmd/
│   ├── server/main.go        # API 服务入口
│   └── worker/main.go        # 定时任务进程（cron 独立部署，可选）
├── internal/
│   ├── config/               # Viper 加载与结构体
│   ├── router/               # 路由聚合：user.go admin.go client.go webhook.go
│   ├── middleware/           # auth.go rate_limit.go cors.go idempotency.go ...
│   ├── handler/              # auth.go order.go invite.go ticket.go ...
│   │   └── admin/            # 管理端 handler 子包：admin.go(dashboard/user/plan/server/order/content/ticket/agent/system)
│   ├── service/              # auth_service.go order_service.go pay_service.go ...
│   ├── repo/                 # user_repo.go order_repo.go ...
│   ├── model/                # 表结构 + DTO + 枚举
│   └── pkg/
│       ├── jwt/  redis/  mailer/  resp/  errs/  validate/
│       ├── payment/          # Driver 接口 + epay/ 等实现 + notify 验签
│       ├── subscribe/        # clash.go singbox.go v2ray.go（配置生成器）
│       └── qrcode/
├── migrations/               # 0001_init.up.sql / .down.sql ...
├── configs/config.yaml
├── docs/                     # swag 生成产物
├── Makefile / go.mod / Dockerfile / docker-compose.yml
└── deploy/                   # Caddyfile（反代/HTTPS 配置）
```

## 4. 中间件链

请求顺序：`RequestID → Recovery → AccessLog(zap) → CORS → RateLimit → [Auth] → [Idempotency] → handler`

| 中间件 | 说明 |
|---|---|
| RequestID | 生成/透传 `X-Request-Id`，贯穿日志 |
| Recovery | panic 捕获 → 500 + envelope，记录堆栈 |
| AccessLog | 方法/路径/状态/耗时/用户 ID（脱敏，不记 body 与 token） |
| CORS | 白名单域名（Web 版前端域名）；Tauri 桌面端走原生 http 不经过浏览器 CORS，但订阅端点需允许任意来源（代理客户端直连） |
| RateLimit | Redis 令牌桶：登录/注册/验证码严格（如 5 次/分钟/IP），全局宽松（如 300 次/分钟/IP）；超限返回 429 |
| Auth | 解析 Bearer JWT，注入 `user_id`；`admin` 分组额外校验 `role` |
| Idempotency | 对 `POST /orders` 等写接口支持 `Idempotency-Key` 头：同 Key 在 24h 内直接返回首次结果（Redis 缓存响应），防重复下单 |

## 5. 统一响应与错误

- envelope：`{"code": 0, "message": "ok", "data": ...}`；错误码段位见 [../api/README.md](../api/README.md) 第 2 节。
- `pkg/errs`：`errs.New(40001, "邮箱已注册")` → handler 返回时被中间件翻译为 envelope + 对应 HTTP 状态；未知错误统一 50000，**不外泄内部错误细节**。
- 校验失败由 binding 错误统一转换为 40000 + 字段级 message。

## 6. 支付抽象（`pkg/payment`）

```go
type Driver interface {
    Name() string
    CreatePayment(ctx, p *Payment) (*CreateResult, error) // 返回跳转URL或二维码内容
    VerifyNotify(r *http.Request) (*NotifyResult, error)  // 验签+解析回调
    Query(ctx, tradeNo string) (*QueryResult, error)      // 主动查单（兜底对账）
}
// 注册表：map[method_code]Driver，method_code 如 "epay_alipay" "epay_wxpay"
```

- 一期实现易支付（epay，彩虹易支付兼容协议）一个驱动即可覆盖支付宝/微信；新增渠道 = 新增 Driver 实现 + 配置。
- 回调入口统一 `POST /api/v1/payment/notify/{method}`：验签 → 幂等（trade_no 唯一约束 + 状态机校验）→ 调 `OrderService.MarkPaid`。
- 金额以「分」int64 全链路流转，仅序列化给前端时转「元」。

## 7. 订阅生成器（`pkg/subscribe`）

```go
type Generator interface { Format() string; Build(u *User, nodes []Server) ([]byte, string, error) }
// clash → YAML；sing-box → JSON；v2ray → base64 分享链接列表
```

- 入口 `GET /api/v1/client/subscribe/{token}`：按 `flag` 参数或 User-Agent 嗅探选择生成器（Clash UA → yaml；sing-box → json；默认 base64）。
- 响应头携带 `subscription-userinfo: upload=..; download=..; total=..; expire=..` 与 `profile-update-interval: 24`，客户端可显示用量并自动更新。
- 节点含倍率（rate）用于流量倍乘；用户流量单位统一字节。

## 8. 配置管理

- `configs/config.yaml` + 环境变量（`APP_` 前缀，`.` 转 `_` 覆盖），敏感项（DB 密码、SMTP、支付密钥、JWT 密钥）只走环境变量或 secret 注入，不进 git。
- 站点运营配置（站点名、公告弹窗、TG 链接、邀请佣金比例、代理条件、支付开关等）存 `settings` 表，管理端可改，运行时 Redis 缓存 60s。

## 9. 测试与工程化

- **单测**：service 层为主（repo 用 sqlmock 或以接口 mock；Redis 用 miniredis）；重点：下单算价、优惠券、佣金、订阅状态机、回调验签幂等。
- **集成测试**：一期未引入 dockertest；核心链路以 service 层单测覆盖（当前 144 个测试函数，见 progress.md §1 状态总览）。
- **Makefile**：`make run / migrate / swagger / lint / test / build`。
- **CI**：本机 `make lint / test / build` 可用；按项目决策，后端**不接入 GitHub Actions**（代码仓库 private，后端构建/部署由本机或内部流程完成；前端 CI 见 docs/frontend）。
- **Swagger**：handler 注解维护，`make swagger` 生成；接口变更时与契约文档同步 PR 评审。

## 10. 安全清单

1. 密码 bcrypt cost=12 哈希（`internal/pkg/passwd`）；登录错误次数锁定（Redis 计数 5 次/10 分钟）。
2. 邮件验证码 6 位数字、10 分钟有效、同邮箱 60s 限频、同 IP 每日上限。
3. 订阅 token = UUIDv4，与用户解耦可重置；订阅端点单独限流（防刷流量统计接口）。
4. 支付回调只信验签结果与网关主动查单，不信前端任何「已支付」声明。
5. 全量参数化查询（GORM 绑定）；Markdown 内容入库前 `bluemonday` 白名单清洗（防存储型 XSS）。
6. 管理接口与普通接口同进程不同分组，强制 `role=admin`；敏感操作（退款、改余额）记审计日志表。
7. HTTPS 强制（Caddy/Nginx 终止 TLS）；安全响应头（X-Content-Type-Options 等）。

## 11. 性能与容量预估

- 目标量级：单实例支撑 1–5 万注册用户、峰值 200 QPS（面板类业务足够）。
- 手段：套餐/节点/配置 Redis 缓存；列表接口全部分页；`subscription-userinfo` 读缓存 30s；PostgreSQL 合理索引（见 data-model.md）；GORM 预加载防 N+1；慢查询日志 >200ms 告警。
- 横向扩容：服务无状态（会话在 Redis），多实例前置负载均衡即可；cron 进程单实例运行（或加分布式锁）。

## 12. 里程碑

| 阶段 | 内容 |
|---|---|
| B1 骨架 | 工程初始化、配置/日志/中间件、迁移、健康检查、统一响应 |
| B2 账户 | 验证码邮件、注册/登录/刷新/找回、用户信息、JWT 与限流 |
| B3 内容 | 公告、知识库、站点配置、节点列表 |
| B4 交易 | 套餐、优惠券、下单、易支付接入与回调、余额支付、订单状态机 |
| B5 订阅 | 订阅下发（Clash/sing-box/v2ray）、重置订阅、流量统计与明细 |
| B6 营销 | 邀请码、佣金结算 cron、划转、代理申请审核 |
| B7 工单与其他 | 工单、到期提醒邮件、管理端 API、压测与上线 |
