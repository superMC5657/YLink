# 代理售卖系统 · 开发文档总览

> 一套仿 YLink 面板风格的代理订阅售卖系统，包含 **用户端应用**（响应式 Web + Tauri 2 桌面端）与 **Go/Gin 服务端**。本文档是全部开发文档的导航与全局约定。

## 1. 系统组成

| 端 | 形态 | 技术栈 | 文档 |
|---|---|---|---|
| 用户端 | 响应式 Web（桌面/平板/手机浏览器）+ Tauri 2 桌面应用（正式打包仅 Windows，见 desktop-tauri.md §7） | Vue 3.5 + TS + Vite 6 + Naive UI + UnoCSS + Pinia | [frontend/](frontend/README.md) |
| 服务端 | REST API + 订阅下发 + 支付回调 | Go 1.26.1 + Gin + GORM + PostgreSQL 16 + Redis 7 | [backend/](backend/README.md) |
| 接口契约 | 前后端唯一事实来源 | REST + JSON，OpenAPI 风格描述 | [api/README.md](api/README.md) |
| 管理端 | 运营后台（同仓 SPA，18 个管理页面：M8 核心 6 + M9 二期 7 + 缺口补齐新增 5） | Vue 3 SPA（同仓库） | 见 [api/README.md](api/README.md) §16 |

## 2. 总体架构

```
                        ┌───────────────────────────────┐
                        │           用户端 App           │
                        │  Vue3 SPA（响应式，一套代码）   │
                        │  ├── 浏览器（桌面/手机）        │
                        │  └── Tauri 2 WebView（桌面壳）  │
                        └──────────────┬────────────────┘
                                       │ HTTPS  /api/v1（Bearer Token）
                                       ▼
┌────────────┐   支付异步通知   ┌───────────────────────────────┐
│ 支付网关    │ ─────────────▶ │         Go/Gin 服务端          │
│ (易支付等)  │ ◀───────────── │  handler → service → repo     │
└────────────┘   创建支付单     │  ├── PostgreSQL 16（业务数据）  │
                                │  ├── Redis（验证码/限流/会话）  │
┌────────────┐   拉取节点       │  └── 订阅生成器（Clash/sing-   │
│ 代理客户端  │ ◀───────────── │      box/v2ray 配置）          │
│ Clash 等   │  /subscribe/xx  └───────────────────────────────┘
└────────────┘
```

## 3. 功能范围

面向访客的功能清单维护在[根 README](../README.md)「功能范围」一节,此处不重复;各端实现细节见 frontend/backend 文档。

## 4. 文档导航

| 文档 | 内容 |
|---|---|
| [frontend/README.md](frontend/README.md) | 前端：技术选型、架构、目录结构、工程化、里程碑 |
| [frontend/progress.md](frontend/progress.md) | 前端：开发进度追踪（已完成 / 未完成 / 前置条件） |
| [frontend/design-system.md](frontend/design-system.md) | 前端：设计令牌、暗色模式、响应式与移动端适配、组件规范 |
| [frontend/pages.md](frontend/pages.md) | 前端：路由表与逐页组件拆解（对照截图） |
| [frontend/data-layer.md](frontend/data-layer.md) | 前端：HTTP 封装、状态管理、i18n、深链接一键导入 |
| [frontend/desktop-tauri.md](frontend/desktop-tauri.md) | 前端：Tauri 2 集成、插件、打包发布、移动端策略 |
| [backend/README.md](backend/README.md) | 后端：技术选型、分层架构、目录结构、中间件、工程化 |
| [backend/data-model.md](backend/data-model.md) | 后端：数据库表结构、索引、Redis Key 设计 |
| [backend/core-flows.md](backend/core-flows.md) | 后端：注册登录、下单支付、佣金、订阅下发等核心流程 |
| [backend/deploy.md](backend/deploy.md) | 后端：配置、Docker 部署、运维 |
| [backend/progress.md](backend/progress.md) | 后端：开发进度追踪（已完成 / 未完成 / 前置条件） |
| [backend/node-agent-guide.md](backend/node-agent-guide.md) | 后端：节点 agent（Xray 流量上报）部署对接说明 |
| [backend/launch-checklist.md](backend/launch-checklist.md) | 后端：上线前置检查清单（域名/环境/第三方账号/预演） |
| [api/README.md](api/README.md) | 接口契约：通用约定、错误码、全量端点定义 |
| [reviews/](reviews/) | 代码评审记录（review-0.2.0 ~ 0.9.0，中文，按版本冻结的历史快照） |
| [.scratch/](../.scratch/)（仓库根） | 需求立项与决策：各 feature 的 spec.md（review-fixes 批次含 issues/ 工单） |

## 5. 全局约定

1. **接口契约唯一来源**：`docs/api/README.md`。前后端并行开发时，前端以 Mock 按契约造数，后端以契约为准实现并以 Swagger 自校验。
2. **接口变更流程**：先改契约文档 → 双方评审 → 再改实现。禁止实现先行、文档后补。
3. **单位约定**：金额接口传输单位为「元」（保留两位小数的 number），服务端数据库存储为「分」（bigint）；流量接口传输单位为「字节」（int64），前端负责格式化为 G/Mbps；时间统一 RFC3339（带时区）。
4. **鉴权约定**：除订阅下发、支付回调、站点配置外，所有接口需要 `Authorization: Bearer <access_token>`。
5. **分支约定**：`main` 稳定分支，`feat/*` 功能分支，PR 合并；提交信息遵循 Conventional Commits。
