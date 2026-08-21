# 上线前置准备清单（不部署，提前办理）

> 目标：把**长周期外部依赖**与**需真实密钥的配置**在正式部署前准备就绪,真正上线时按 [deploy.md §4](deploy.md) 直接执行即可。
> 使用方式:每完成一项打 ✅ 并填写「实际值/进度」;密钥一律只写进 `server/.env.release`(gitignore),**不入库、不写进本文档**。
> 最后更新:2026-08-22(创建)

---

## 1. 域名与 DNS(通常 10 分钟生效,提前 1 天办)

| 项 | 要求 | 状态 | 实际值/进度 |
|---|---|---|---|
| API 域名 | 如 `api.example.com`,A 记录指向服务器公网 IP | ☐ | |
| 面板域名 | 如 `panel.example.com`,同上 | ☐ | |
| Caddyfile 替换 | `server/deploy/Caddyfile` 两处 `example.com` 改真实域名(打包进 ylink-web 镜像,改后需重建) | ☐ | |
| 80/443 可达 | 服务器安全组/防火墙放行 80、443(Caddy 自动签 Let's Encrypt,无需账号) | ☐ | |
| CORS 白名单 | `server/configs/config.yaml` `cors.allow_origins` 加 `https://panel.example.com`(改后重建 api 镜像) | ☐ | |

## 2. 服务器环境(提前装好)

| 项 | 要求 | 状态 | 实际值/进度 |
|---|---|---|---|
| Docker + compose 插件 | 服务器已安装 | ☐ | |
| 构建工具 | Go ≥1.26(仅迁移用,也可在容器内跑)或本机 `make migrate` | ☐ | |
| Node + pnpm | 服务器构建前端用(或本机构建后传 dist) | ☐ | |
| 磁盘 | ≥ 20GB(镜像 + PG 数据 + WAL 归档 14 天) | ☐ | |

## 3. 第三方账号(长周期,优先办)

| 项 | 要求 | 状态 | 实际值/进度 |
|---|---|---|---|
| 易支付商户 | 彩虹协议兼容网关;拿到 gateway / pid / key;后台配置异步通知 `https://{API域名}/api/v1/payment/notify/{method}` | ☐ | |
| SMTP 发信 | 如 QQ 邮箱授权码(smtp.qq.com:465);验证码/回执邮件依赖,**未配置则注册/找回不可用** | ☐ | |
| (可选)内网穿透 | 本机联调支付回调时用,生产不需要 | ☐ | |

## 4. `.env.release` 真实值(`cp server/.env.example server/.env.release` 后逐项替换)

| 变量 | 要求 | 状态 |
|---|---|---|
| `APP_ENV` | `production`(模板默认已是,勿改) | ☐ |
| `APP_BASE_URL` | `https://{API域名}`(订阅链接/回调地址以其拼接) | ☐ |
| `APP_JWT_SECRET` | ≥32 字节随机串(`openssl rand -hex 32`) | ☐ |
| `APP_DATABASE_DSN` | 容器内 `host=postgres ... password=<强密码>` | ☐ |
| `APP_REDIS_PASSWORD` / `REDIS_PASSWORD` | 强密码(两处一致;Redis 与 api/worker 共用) | ☐ |
| `APP_SMTP_USERNAME` / `APP_SMTP_PASSWORD` | 真实邮箱账号 + 授权码 | ☐ |
| `APP_PAYMENT_EPAY_GATEWAY/PID/KEY/METHODS` | 易支付商户真实值 | ☐ |
| `POSTGRES_USER/POSTGRES_PASSWORD/POSTGRES_DB` | 强密码 | ☐ |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | 首个管理员(启动幂等创建;密码首登后建议改) | ☐ |
| `DEMO_EMAIL` / `DEMO_PASSWORD` | 生产可留空跳过演示账号 | ☐ |
| `GRAFANA_ADMIN_PASSWORD`(可选) | 启用 `--profile obs` 时 Grafana 管理密码 | ☐ |

## 5. 本地可预演项(不碰生产,提前验证)

| 项 | 命令/方式 | 状态 |
|---|---|---|
| 生产 compose 解析 | `cd server && ENV_FILE=.env.release docker compose -f docker-compose.yml -f docker-compose.prod.yml config` 无报错 | ☐ |
| 前端生产构建 | `VITE_API_BASE_URL=https://{API域名}/api/v1 pnpm build`(产物进 ylink-web 镜像) | ☐ |
| 迁移演练 | 本地/dev 库跑 `DB_URL='postgres://...' make migrate`(至 0004)确认无错 | ☐ |
| 支付沙箱 | staging:复制 `.env.release` 改网关为沙箱/0.01 元实测一笔全链路 | ☐ |
| 节点 agent 联调 | 本地起后端 + `go run ./cmd/node-agent -key <node_key>`,核对 users.u/d 累加与 traffic_logs 日聚合 | ☐ |

## 6. 上线日序列(引用,不在此展开)

按 [deploy.md §4](deploy.md) 执行:起 postgres/redis → `make migrate` → 起全量服务 → 健康检查(`/healthz` 200、`/swagger` 与裸 `/swagger`、`/metrics` 均 404)→ 管理后台配置(节点/套餐/支付/SMTP 测试邮件)→ 前端构建发布 → 易支付回调地址核对。
