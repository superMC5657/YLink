# YLink · 代理订阅售卖系统

> 一套仿 YLink 面板风格的代理订阅售卖系统，包含 **用户端应用**（响应式 Web + Tauri 2 桌面端）与 **Go/Gin 服务端**，管理后台随主 SPA 实现全部 13 模块。

## 系统组成

| 端 | 形态 | 技术栈 |
|---|---|---|
| 用户端 | 响应式 Web（桌面/平板/手机浏览器）+ Tauri 2 桌面应用（正式打包仅 Windows，见 desktop-tauri.md §7） | Vue 3.5 + TS + Vite 6 + Naive UI + UnoCSS + Pinia |
| 服务端 | REST API + 订阅下发 + 支付回调 + 定时任务 | Go 1.26 + Gin + GORM + PostgreSQL 16 + Redis 7 |
| 管理端 | 运营后台（同仓 SPA 内 13 模块：M8 核心 6 + M9 二期 7） | Vue 3 SPA（同仓库） |

## 功能范围

- **账户体系**：注册 / 登录 / 找回密码（邮箱验证码）、JWT 会话（refresh 旋转、封禁实时失效）
- **仪表板**：余额与佣金、公告、快捷操作、当前订阅与流量统计
- **交易闭环**：套餐购买（优惠券固定/百分比、余额抵扣、在线支付）、订单状态机、退款收回订阅
- **营销**：邀请赚钱（邀请码、佣金、划转）、申请代理、公告优惠码一键复制
- **订阅**：节点状态、一键导入 10 款客户端（Clash/sing-box 等）、订阅配置下发
- **用户**：个人信息（改密、通知开关、Telegram）、我的工单、流量明细（ECharts）
- **管理后台**：套餐/节点/用户/订单/优惠券/公告/知识库/工单/流量导入/代理审批等 13 模块

## 快速开始

### 前端（Web + Mock）

```bash
pnpm install
pnpm dev            # http://localhost:5174（默认 VITE_USE_MOCK=true）
pnpm test           # Vitest 单测
pnpm e2e            # Playwright E2E（固定 Mock 环境）
```

### 后端（Go/Gin）

```bash
cd server
cp .env.example .env.dev     # 本地开发（.env.dev/.env.release 均被 gitignore,不入库）
make migrate                 # 执行迁移
make run                     # API 服务（:8081）
# 生产发布: cp .env.example .env.release 并填写密钥;先 pnpm build 生成前端 dist(打进 ylink-web 镜像);
# ENV_FILE=.env.release docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
# 详见 docs/backend/deploy.md
```

### Tauri 桌面端

```bash
pnpm tauri:dev       # 开发联动
pnpm tauri:build     # Windows 打包（nsis，含签名；2026-08-12 起仅 Windows）
```

## CI / CD

- `.github/workflows/ci.yml`：仅发布 tag（`v*`）触发——`frontend-quality`（lint→typecheck→format:check→test→build:web）+ `frontend-e2e` + `rust`（cargo check，windows-latest）；日常检查走本地 `pnpm lint/typecheck/test/format:check`；Go 后端按项目决策不走 GitHub Actions
- `.github/workflows/release-tauri.yml`：打 tag `v*` 触发 Windows 打包签名（nsis）→ `latest.json` 合并（gh-proxy.com 加速前缀）→ 发布到公开产物仓库，供公网下载与 Tauri 自动更新

## 目录结构

```
YLink/
├── src/            # Vue 3 SPA（api/components/composables/layouts/stores/views/...）
├── src-tauri/      # Tauri 2 Rust 工程（托盘/深链接/updater 等插件）
├── mock/           # vite-plugin-mock 数据（严格按契约）
├── server/         # Go/Gin 后端（cmd/internal/migrations/deploy）
├── scripts/        # 构建/发布脚本（build-latest-json.mjs、dev.sh 等）
├── tests/          # Playwright E2E
├── docs/           # 开发文档（frontend/backend/api/reviews）
└── .github/workflows/  # CI/CD
```

## 文档

| 文档 | 内容 |
|---|---|
| [docs/README.md](docs/README.md) | 开发文档总览与全局约定 |
| [docs/frontend/README.md](docs/frontend/README.md) | 前端：技术选型、架构、目录结构、工程化 |
| [docs/backend/README.md](docs/backend/README.md) | 后端：技术选型、分层架构、中间件、工程化 |
| [docs/api/README.md](docs/api/README.md) | 接口契约：通用约定、错误码、全量端点（前后端唯一事实来源） |
| [docs/frontend/progress.md](docs/frontend/progress.md) | 前端进度追踪（已完成/未完成/前置条件） |
| [docs/backend/progress.md](docs/backend/progress.md) | 后端进度追踪（已完成/未完成/前置条件） |
| [docs/reviews/](docs/reviews/) | 代码评审记录（review-0.2.0 ~ 0.7.0，中英文对照） |

## 环境要求

- Node >= 20、pnpm（lockfileVersion 9.0）
- Rust >= 1.77.2（`src-tauri/Cargo.toml` rust-version）
- Go 1.26（`server/go.mod`）
- PostgreSQL 16、Redis 7