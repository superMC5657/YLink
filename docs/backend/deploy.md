# 后端开发文档 · 配置与部署

## 1. 配置文件（configs/config.yaml）

```yaml
app:
  name: ylink-backend
  env: production            # development / production
  addr: ":8081"
  base_url: "https://api.example.com"   # 用于拼接订阅链接/支付回调地址

database:
  dsn: "${DB_DSN}"           # host=127.0.0.1 port=5433 user=ylink password=xxx dbname=ylink-backend sslmode=disable
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
  username: "${APP_SMTP_USERNAME}"
  password: "${APP_SMTP_PASSWORD}"
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
  postgres:
    image: postgres:16-alpine
    environment: { POSTGRES_USER: ${POSTGRES_USER}, POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}, POSTGRES_DB: ylink-backend }
    volumes: ["postgres_data:/var/lib/postgresql/data"]
    ports: ["127.0.0.1:5433:5433"]      # 容器内监听 5433（command: -p 5433）
    command: -p 5433
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -h localhost -p 5433 -U ${POSTGRES_USER}"]

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes: ["redis_data:/data"]

  api:
    build: ..
    env_file: ${ENV_FILE:-.env.dev}   # 生产:ENV_FILE=.env.release 启动时覆盖
    depends_on: [postgres, redis]
    ports: ["127.0.0.1:8081:8081"]     # 只对本机暴露，由 Caddy 反代

  worker:
    build: ..
    command: ["/app/bin/worker"]
    env_file: ${ENV_FILE:-.env.dev}
    depends_on: [postgres, redis]

  caddy:
    image: caddy:2-alpine
    ports: ["80:80", "443:443"]
    volumes: ["./Caddyfile:/etc/caddy/Caddyfile", "caddy_data:/data"]

volumes: { postgres_data: {}, redis_data: {}, caddy_data: {} }
```

Caddyfile：`api.example.com { reverse_proxy api:8081 }`（自动 HTTPS）。Web 版前端静态资源可同机 Caddy 托管或独立 CDN。

## 4. 上线步骤

1. 准备环境文件：本地开发 `cp .env.example .env.dev`（scripts/dev.sh 读取，DSN 会被脚本覆盖为宿主机 127.0.0.1:5433）；生产 `cp .env.example .env.release` 并填写真实 DB/Redis/JWT/SMTP/EPAY 密钥（两者均被 .gitignore 忽略）。模板默认 `APP_ENV=production`（关 Swagger/debug，生产保持默认即可；本地开发由 dev.sh 强制覆盖为 development）。**存量部署升级**：确认已有 `.env.release` 中 `APP_ENV=production`（旧模板为 development，生产会误开 Swagger/debug）。
2. 首次启动先拉起数据库容器：`ENV_FILE=.env.release docker compose up -d postgres redis`，等待 `docker compose ps` 显示 postgres/redis healthy。
3. 数据库迁移（新主机首次部署必做，api/worker 容器不会自动迁移）：`DB_URL='postgres://ylink:ylink_root@127.0.0.1:5433/ylink-backend?sslmode=disable' make migrate`（DSN 必须带 `postgres://` 前缀，否则 migrate CLI 报 `unknown driver`；`-tags 'postgres'` 已内置在 Makefile）。此时宿主机 127.0.0.1:5433 已由 postgres 容器监听，迁移才能连通。
4. 启动全部服务 `ENV_FILE=.env.release docker compose up -d`（api/worker 的 env_file 由该变量切换），验证 `GET /healthz` 返回 200、`GET /api/v1/config` 返回站点配置。
5. 登录管理接口创建/核对：节点分组与节点、套餐、支付渠道、SMTP 测试邮件。
6. 前端 `VITE_API_BASE_URL` 指向 `https://api.example.com/api/v1` 打包部署；Tauri 端构建发布。
7. 支付网关后台配置异步通知地址：`https://api.example.com/api/v1/payment/notify/{method}`。

## 5. 健康检查与可观测

- `GET /healthz`：进程存活；`GET /readyz`：DB/Redis 连通（供编排探针）。
- 日志：zap JSON → 文件按天切割（lumberjack），容器同时输出 stdout 便于 `docker logs`；错误日志含 request_id。
- 慢查询：GORM logger 记录 >200ms SQL。
- 指标（可选二期）：`promhttp` `/metrics`（QPS、延迟直方图、支付成功率、cron 执行结果），Grafana 看板。
- 告警最小集（二期前用脚本兜底）：进程存活、磁盘、PostgreSQL/Redis 连通、每日支付成功率。

## 6. 备份与恢复

- PostgreSQL：每日 `pg_dump -Fc` 全量 + WAL 归档，保留 14 天；异机存放。
- Redis：验证码/缓存类可丢，开启 AOF everysec 即可；refresh 白名单丢失的最坏结果是用户重新登录，可接受。
- 恢复演练：每月一次把备份恢复到临时库并跑 `SELECT` 抽检。

## 7. 发布与回滚（手动流程）

> 2026-08-12 项目决策：后端**不接入 GitHub Actions**（无 CI job、无镜像构建/部署流水线），发布走本机手动流程。

- **构建**：`make build` 产出 `bin/server` 与 `bin/worker`；或用 `docker compose up -d --build` 走 §3 编排（api/worker 由 Dockerfile 构建，镜像 tag 由服务器本地管理）。
- **发布**：更新代码 → `make build`（或 `docker compose build api worker`）→ 重启容器 `docker compose up -d api worker`；滚动期间短暂 502 由 Caddy 重试掩盖，或双实例蓝绿（二期）。
- **回滚**：切回上一版本二进制/上一镜像 tag；数据库迁移只向前兼容（新增列可空、不删旧列），破坏性变更分两次发布（先加后删）；已执行不兼容迁移用对应 `.down.sql` 回退。

## 8. 环境清单

| 环境 | 用途 | 说明 |
|---|---|---|
| local | 开发本机 | `server/.env.dev`（gitignore）;compose 起 postgres/redis;dev.sh 启动 api/worker 时强制 `APP_ENV=development` 开 Swagger |
| staging | 预发 | 与生产同构,复制 `.env.release` 改网关为沙箱/0.01 元实测 |
| production | 正式 | `server/.env.release`（gitignore,真实密钥）;`ENV_FILE=.env.release docker compose up -d`;关 Swagger/debug,严格 CORS 白名单与限流 |
