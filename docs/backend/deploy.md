# 后端开发文档 · 配置与部署

## 1. 配置文件（configs/config.yaml）

```yaml
app:
  name: ylink-api
  env: production            # development / production
  addr: ":8081"
  base_url: "https://api.example.com"   # 用于拼接订阅链接/支付回调地址

database:
  dsn: "${DB_DSN}"           # user:pass@tcp(mysql:3306)/ylink?charset=utf8mb4&parseTime=true&loc=Local
  max_open: 50
  max_idle: 10

redis:
  addr: "redis:6379"
  password: "${REDIS_PASSWORD}"
  db: 0

jwt:
  secret: "${JWT_SECRET}"    # >= 32 字节随机串
  access_ttl: 2h
  refresh_ttl: 336h          # 14d

smtp:
  host: "smtp.qq.com"
  port: 465
  username: "${SMTP_USER}"
  password: "${SMTP_PASS}"
  from_name: "YLink"

payment:
  epay:                      # 易支付（彩虹兼容）
    gateway: "https://pay.example.com"
    pid: "${EPAY_PID}"
    key: "${EPAY_KEY}"
    methods: ["alipay", "wxpay"]

cors:
  allow_origins: ["https://panel.example.com"]   # Web 前端域名；Tauri 端无 CORS 需求

log:
  level: info
  dir: ./logs
```

规则：敏感值只经环境变量注入；`app.env=development` 时开启 Swagger 路由 `/swagger/index.html` 与 debug 日志，生产关闭。

## 2. Dockerfile（多阶段）

```dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./ && RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server ./cmd/server \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/worker ./cmd/worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /app/bin/ /app/bin/
COPY configs/config.yaml /app/configs/
EXPOSE 8081
ENTRYPOINT ["/app/bin/server"]   # worker 容器覆盖为 /app/bin/worker
```

## 3. docker-compose 编排

```yaml
services:
  mysql:
    image: mysql:8.0
    environment: { MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASS}, MYSQL_DATABASE: ylink }
    volumes: ["mysql_data:/var/lib/mysql"]
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes: ["redis_data:/data"]

  api:
    build: ..
    env_file: .env
    depends_on: [mysql, redis]
    ports: ["127.0.0.1:8081:8081"]     # 只对本机暴露，由 Caddy 反代

  worker:
    build: ..
    command: ["/app/bin/worker"]
    env_file: .env
    depends_on: [mysql, redis]

  caddy:
    image: caddy:2-alpine
    ports: ["80:80", "443:443"]
    volumes: ["./Caddyfile:/etc/caddy/Caddyfile", "caddy_data:/data"]

volumes: { mysql_data: {}, redis_data: {}, caddy_data: {} }
```

Caddyfile：`api.example.com { reverse_proxy api:8081 }`（自动 HTTPS）。Web 版前端静态资源可同机 Caddy 托管或独立 CDN。

## 4. 上线步骤

1. 准备 `.env`（DB/Redis/JWT/SMTP/EPAY 密钥），首次启动前执行迁移：`DB_URL='mysql://user:pass@tcp(host:3306)/ylink?charset=utf8mb4&parseTime=true&loc=Local' make migrate`（DSN 必须带 `mysql://` 前缀，否则 migrate CLI 报 `unknown driver`；`-tags 'mysql'` 已内置在 Makefile）。
2. `docker compose up -d`，验证 `GET /healthz` 返回 200、`GET /api/v1/config` 返回站点配置。
3. 登录管理接口创建/核对：节点分组与节点、套餐、支付渠道、SMTP 测试邮件。
4. 前端 `VITE_API_BASE_URL` 指向 `https://api.example.com/api/v1` 打包部署；Tauri 端构建发布。
5. 支付网关后台配置异步通知地址：`https://api.example.com/api/v1/payment/notify/{method}`。

## 5. 健康检查与可观测

- `GET /healthz`：进程存活；`GET /readyz`：DB/Redis 连通（供编排探针）。
- 日志：zap JSON → 文件按天切割（lumberjack），容器同时输出 stdout 便于 `docker logs`；错误日志含 request_id。
- 慢查询：GORM logger 记录 >200ms SQL。
- 指标（可选二期）：`promhttp` `/metrics`（QPS、延迟直方图、支付成功率、cron 执行结果），Grafana 看板。
- 告警最小集（二期前用脚本兜底）：进程存活、磁盘、MySQL/Redis 连通、每日支付成功率。

## 6. 备份与恢复

- MySQL：每日 `mysqldump --single-transaction` 全量 + binlog，保留 14 天；异机存放。
- Redis：验证码/缓存类可丢，开启 AOF everysec 即可；refresh 白名单丢失的最坏结果是用户重新登录，可接受。
- 恢复演练：每月一次把备份恢复到临时库并跑 `SELECT` 抽检。

## 7. 发布与回滚

- CI 构建镜像打 `git sha` 与 `latest` 双标签；发布 = 更新 compose 镜像 tag + `up -d`（滚动期间短暂 502 由 Caddy 重试掩盖，或双实例蓝绿，二期；⚠ 后端 CI 当前未接入 GitHub Actions，见 progress.md §2）。
- 数据库迁移只向前兼容（新增列可空、不删旧列），保证回滚旧版本仍可运行；破坏性变更分两次发布（先加后删）。
- 回滚：切回上一镜像 tag；若已执行不兼容迁移，用对应 `.down.sql` 回退（迁移设计时保证 down 可用）。

## 8. 环境清单

| 环境 | 用途 | 说明 |
|---|---|---|
| local | 开发本机 | compose 起 mysql/redis；`app.env=development` 开 Swagger |
| staging | 预发 | 与生产同构，支付用网关沙箱/0.01 元实测 |
| production | 正式 | 关 Swagger/debug，严格 CORS 白名单与限流 |
