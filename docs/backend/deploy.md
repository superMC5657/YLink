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
  allow_origins: ["https://panel.example.com"]   # Web 前端域名(https)；Tauri 端无 CORS 需求

log:
  level: info
  dir: ./logs
```

规则：敏感值只经环境变量注入；`app.env=development` 时开启 Swagger 路由 `/swagger/index.html` 与 debug 日志，生产关闭。

## 2. Dockerfile（多阶段）

```dockerfile
FROM golang:1.26-alpine AS build
WORKDIR /app
# 国内网络直连 proxy.golang.org 常超时,构建阶段走 goproxy.cn 加速
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
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
      test: ["CMD-SHELL", "pg_isready -h localhost -p 5433 -U ${POSTGRES_USER} -d postgres"]

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes: ["redis_data:/data"]
    ports: ["127.0.0.1:6379:6379"]      # dev.sh 的 api/worker 是宿主机进程,需经本机端口访问
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]

  api:
    build: .
    env_file: ${ENV_FILE:-.env.dev}   # 生产:ENV_FILE=.env.release 启动时覆盖
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_healthy }
    ports: ["127.0.0.1:8081:8081"]     # 只对本机暴露，由 Caddy 反代

  worker:
    build: .
    command: ["/app/bin/worker"]
    env_file: ${ENV_FILE:-.env.dev}
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_healthy }

  caddy:
    image: caddy:2-alpine
    ports: ["80:80", "443:443"]
    volumes: ["./deploy/Caddyfile:/etc/caddy/Caddyfile:ro", "caddy_data:/data"]

volumes: { postgres_data: {}, redis_data: {}, caddy_data: {} }
```

Caddyfile：`api.example.com { reverse_proxy api:8081 }`（自动 HTTPS）。Web 版前端由**同一 Caddy 实例托管**（`panel.example.com` 静态 + SPA 回退），见下方 §3.1 生产 override。

### 3.1 生产 override（docker-compose.prod.yml）

基础 compose 保留 postgres/redis/api 的宿主端口映射——**dev.sh / dev-docker.sh 依赖**（宿主机进程连 127.0.0.1:5433/6379、宿主机直连 :8081）。生产叠加 `-f docker-compose.prod.yml`：

```bash
cd server
pnpm build   # 先在仓库根生成前端 dist/(.gitignore 忽略,不进 git)
ENV_FILE=.env.release docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

prod override 做的事：

| 服务 | 生产变化 | 原因 |
|---|---|---|
| redis / api | `ports: !reset []` 清空宿主端口 | 仅容器内网互访;Caddy 内网反代 api:8081,无需宿主暴露 |
| postgres | 保留 `127.0.0.1:5433:5433` | 宿主机 `make migrate` 需经该端口(deploy.md §4 步骤 3) |
| caddy | `image: caddy:2-alpine` → `build: server/deploy/Dockerfile.web`(产出 `ylink-web:latest`) | 前端 dist + 生产 Caddyfile **打进镜像**,不可变发布;`deploy/Caddyfile` 不再 bind mount |

`server/deploy/Dockerfile.web`（build context = 仓库根）：
```dockerfile
FROM caddy:2-alpine
COPY dist/ /srv/panel
COPY server/deploy/Caddyfile /etc/caddy/Caddyfile
```

生产 Caddyfile（双域名，自动 HTTPS）：
```caddyfile
api.example.com   { @swagger path /swagger /swagger/*; respond @swagger 404; reverse_proxy api:8081 }   # API(Tauri 端 + Web 端调用);Swagger 仅开发环境,生产在网关层拒绝兜底
panel.example.com { root * /srv/panel; try_files {path} /index.html; file_server }  # Web SPA
```
部署前把 `example.com` 替换为真实域名并配好 A 记录;`cors.allow_origins` 需含 `https://panel.example.com`(config.yaml,生产改后重建 api 镜像)。

> **Swagger 安全**：`/swagger/index.html` 在 `APP_ENV=production` 时后端路由不注册(router.go,直接 404);Caddyfile 的 `@swagger` 拒绝规则是纵深防御——即使 `.env.release` 误配为 development 误开 Swagger,外部也访问不到。上线后按下述 §4 步骤 4 的 curl 验证应返回 404。

全容器本地联调(`scripts/dev-docker.sh`)必须用 `VITE_API_BASE_URL=/api/v1` 构建,让 Web 经 Caddy 同域反代;否则 Web 会跨端口直连 `localhost:8081`,触发浏览器 CORS 预检且后端白名单未放行时登录失败。`configs/config.yaml` 已包含 `http://localhost` / `http://127.0.0.1` 供本地直连兜底。

## 4. 上线步骤

1. 准备环境文件：本地开发 `cp .env.example .env.dev`（scripts/dev.sh 与 dev-docker.sh 均读取，**dev-docker.sh 以 .env.dev 为唯一来源、缺失即报错**；dev.sh 的 DSN 会被脚本覆盖为宿主机 127.0.0.1:5433，dev-docker.sh 的容器内 DSN/Redis 地址由 compose override 覆盖为服务名）；生产 `cp .env.example .env.release` 并填写真实 DB/Redis/JWT/SMTP/EPAY 密钥（三者均被 .gitignore 忽略）。模板默认 `APP_ENV=production`（关 Swagger/debug，生产保持默认即可；本地开发由 dev.sh / dev-docker.sh 强制覆盖为 development），模板含 `DEMO_EMAIL/DEMO_PASSWORD`（演示账号，dev-docker.sh 亦从 .env.dev 读取）。**存量部署升级**：确认已有 `.env.release` 中 `APP_ENV=production`（旧模板为 development，生产会误开 Swagger/debug）。
2. 首次启动先拉起数据库容器：`ENV_FILE=.env.release docker compose up -d postgres redis`，等待 `docker compose ps` 显示 postgres/redis healthy。
3. 数据库迁移（新主机首次部署必做，api/worker 容器不会自动迁移）：`DB_URL='postgres://ylink:ylink_root@127.0.0.1:5433/ylink-backend?sslmode=disable' make migrate`（DSN 必须带 `postgres://` 前缀，否则 migrate CLI 报 `unknown driver`；`-tags 'postgres'` 已内置在 Makefile）。此时宿主机 127.0.0.1:5433 已由 postgres 容器监听，迁移才能连通。
4. 启动全部服务 `ENV_FILE=.env.release docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build`（api/worker 的 env_file 由该变量切换;caddy 构建 ylink-web 镜像含前端与生产 Caddyfile）。api 宿主端口已清空,**验证改走域名**:`curl https://api.example.com/healthz` 返回 200、`GET https://api.example.com/api/v1/config` 返回站点配置。**Swagger 关闭验证**:`curl -i https://api.example.com/swagger/index.html | head -1` 与裸路径 `curl -i https://api.example.com/swagger | head -1` 均应返回 `HTTP/1.1 404`(代码层 `APP_ENV=production` 不注册路由 + Caddyfile `@swagger` 拒绝兜底,两道防线任一命中即 404;若返回 200 说明 `.env.release` 里 `APP_ENV` 不是 production,立即排查)。
5. 登录管理接口创建/核对：节点分组与节点、套餐、支付渠道、SMTP 测试邮件。
6. 前端打包：`VITE_API_BASE_URL=https://api.example.com/api/v1 pnpm build` 生成 `dist/`（.gitignore 忽略）→ 已被 §3.1 的 `--build` 打进 `ylink-web` 镜像;Tauri 端构建发布。
7. 支付网关后台配置异步通知地址：`https://api.example.com/api/v1/payment/notify/{method}`。

## 5. 健康检查与可观测

- `GET /healthz`：进程存活；`GET /readyz`：DB/Redis 连通（供编排探针）。
- 日志：zap JSON → 文件按天切割（lumberjack），容器同时输出 stdout 便于 `docker logs`；错误日志含 request_id。
- 慢查询：GORM logger 记录 >200ms SQL。
- 指标（✅ 已落地,2026-08-22 首版;2026-08-25 扩展）：`promhttp` `/metrics`（api：QPS、延迟直方图、支付成功计数、Go 运行时;worker：`cron_job_runs_total`/`cron_job_duration_seconds`,见下）。
  - **数据链路**：api `:8081/metrics`、worker `:8082/metrics`、node-exporter `:9100/metrics` 均经 compose 内网被 Prometheus 抓取(15s);Prometheus 数据**保留半年**(`--storage.tsdb.retention.time=180d`,原默认 15d,另有 `--storage.tsdb.retention.size=20GB` 兜底;指标存于 `prometheus_data` 卷,注意其磁盘占用随保留期增长)。
  - **Grafana 看板**：`docker compose --profile obs up -d` 启动 prometheus + grafana(compose 内网抓取,Grafana 绑 `127.0.0.1:3000`,首登 admin / `GRAFANA_ADMIN_PASSWORD`(默认 admin)即改密);看板经 provisioning 自动加载(`deploy/obs/`),无需手动导入,共两张:`YLink API`(QPS/状态码/p50-p95/支付成功/错误率/Top 路径/Go 运行时)与 `YLink 基础设施`(worker: cron 执行量/失败跳过/耗时 p50-p95;机器: 负载/CPU/内存/磁盘/网络)。默认 `up -d` 不启动 obs 服务。
  - **告警**：`--profile obs` 同时启动 alertmanager(绑 `127.0.0.1:9093`),规则见 `deploy/obs/rules.yml`(进程存活 API/worker、5xx 错误率>10%、CPU/内存>90%、根分区磁盘>85%),命中后**邮件通知**:SMTP 复用 `env_file` 的 `APP_SMTP_*`,收件人取 `ALERT_EMAIL_TO`(未配置默认发 admin@example.com,生产必须显式配置;`alertmanager.yml.tmpl` 占位符由容器 entrypoint 的 sed 替换,取值含 `|` 需调整模板分隔符)。DB/Redis 连通性与每日支付成功率仍由既有脚本兜底。
  - **公网收口**：生产 Caddyfile api 域名 `@metrics` 拦截 `/metrics` 返回 404（与 Swagger 同款纵深防御）;Prometheus/Alertmanager 走内网抓取不受影响,node-exporter 亦不映射公网端口。
- 告警（2026-08-25 起由 Prometheus rules + alertmanager 邮件承担进程/错误率/资源告警,见上）;DB/Redis 连通性与每日支付成功率仍由既有脚本兜底。

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
| local | 开发本机 | `server/.env.dev`（gitignore，dev.sh / dev-docker.sh 的唯一 env 源;dev-docker.sh 缺失即报错、不内置默认值）;compose 起 postgres/redis;dev.sh 启动 api/worker 时强制 `APP_ENV=development` 开 Swagger(dev-docker.sh 同,容器内 DSN/Redis 由 override 覆盖为服务名) |
| staging | 预发 | 与生产同构,复制 `.env.release` 改网关为沙箱/0.01 元实测 |
| production | 正式 | `server/.env.release`（gitignore,真实密钥）;`ENV_FILE=.env.release docker compose up -d`;关 Swagger/debug,严格 CORS 白名单与限流 |
