# 规范文档: YLink V2Board / UniProxy 协议兼容与 V2bX 节点守护对接 ✅

> 状态: 已完成 (Completed)  
> 责任模块: `server/internal/` (Handler, Service, Middleware, Model), `src/components/admin/NodeInstallDialog.vue`, `src/locales/` & `docs/`  
> 关联 Issue: `.scratch/uniproxy-v2bx/issues/` (01 ~ 04)

---

## 1. 需求与架构背景

### 1.1 背景与现状
当前 YLink 拥有完整的用户端 SPA、Tauri 桌面应用与 Go/Gin 服务端业务体系（包含套餐、订单、工单、支付、订阅下发）。
在节点侧，YLink 原生定义了流量模式 A 协议（`GET /node/users`、`POST /node/report`），且提供了演示/探针 Agent（`cmd/node-agent`）。但该 Agent 未真正对接底层的代理核心（Xray-core / Sing-box）。
如果自研代理核心驱动 Agent，需要自行维护 gRPC 动态注入用户、证书申请（ACME）、热重载以及多协议适配，工程成本与维护负担极高。

### 1.2 目标
在 YLink 服务端实现 **V2Board UniProxy 通讯协议兼容层**：
1. 使 YLink 服务端作为控制中心，原生支持接入社区最主流的高性能多协议节点守护程序 **V2bX**。
2. 借力 V2bX 成熟的底层能力，开箱即用支持 **Xray-core / Sing-box / Hysteria2 / Trojan-Go**，支持自动 ACME 证书续期、用户热更新动态增删与本地审计拦截。
3. 保持 YLink 现有的计费、倍率计算、用户到期校验与日聚合逻辑不变，复用已有的领域模型与服务。

---

## 2. 总体架构设计

```text
                  [ 海外 / 国内 VPS 节点 ]
                ┌──────────────────────────┐
                │     V2bX 守护进程        │
                │ ┌──────────────────────┐ │
                │ │  Xray / Sing-box /   │ │
                │ │     Hysteria2 核心   │ │
                │ └──────────────────────┘ │
                └────────────┬─────────────┘
                             │
     HTTP/HTTPS 请求         │ 定时心跳与轮询 (默认 60s)
     Header: Token: {key}    │ 或 Query: ?token={key}&node_id={id}
                             ▼
              ┌──────────────────────────────┐
              │      YLink 服务端 (Go)        │
              │  /api/v1/server/UniProxy/*   │
              └──────────────┬───────────────┘
                             │
            ┌────────────────┴────────────────┐
            ▼                                 ▼
   [ 鉴权与上下文解析 ]              [ 业务逻辑处理与计费 ]
   middleware.UniProxyAuth           handler.UniProxy & service.UniProxyService
   - Token 与 node_key 校验           - GET  /config -> 映射 servers 表字段
   - Redis 缓存加速 (60s)             - GET  /user   -> 过滤有效用户列表
   - 解析注入 server_id               - POST /push   -> 增量累加与日聚合计费
```

---

## 3. 详细接口契约规范

### 3.1 路由前缀与鉴权
* 路由前缀：`/api/v1/server/UniProxy`
* 鉴权方式：
  * **Query 参数**：`node_id` (int64), `token` (string，对应 `servers.node_key`)
  * **请求头**：`Token: <node_key>`
  * 鉴权规则：校验 `token` 与对应 `node_id` 在 `servers` 表中的 `node_key` 是否一致。校验通过后将 `server_id` 注入 Gin 上下文；失败返回 HTTP 401（错误码 40100）。

---

### 3.2 节点配置下发：`GET /api/v1/server/UniProxy/config`
* **Query 参数**：
  * `node_id`: 节点 ID（必需）
  * `node_type`: 协议类型（`v2ray` / `shadowsocks` / `trojan` / `hysteria` 等；可选，若提供则校验与节点类型一致，否则 400）
  * `token`: 节点密钥（必需）
* **响应格式**：具备双重包裹（顶层与 `data` 均提供配置字段，同时支持 ETag / 304 缓存）
```json
{
  "data": {
    "server_port": 443,
    "network": "tcp",
    "network_settings": {},
    "tls": 1,
    "tls_settings": {
      "server_name": "hk01.example.com",
      "reality_settings": {
        "show": false,
        "dest": "www.apple.com:443",
        "server_names": ["www.apple.com"],
        "private_key": "...",
        "short_ids": [""]
      }
    },
    "protocol": "vless",
    "base_config": {
      "push_interval": 60,
      "pull_interval": 60
    }
  }
}
```
* **实现逻辑**：
  从 `servers` 表取出该节点记录，解析 `servers.config` (JSONB) 与 `servers.type`，拼装为 V2bX 各核心所需的配置结构。

---

### 3.3 有效用户下发：`GET /api/v1/server/UniProxy/user`
* **Query 参数**：同上。
* **响应格式**：支持双重包裹与 ETag / 304 缓存。
```json
{
  "data": {
    "users": [
      {
        "id": 1001,
        "uuid": "5f2b7c9e-1111-2222-3333-444455556666",
        "speed_limit": 0,
        "device_limit": 0
      }
    ]
  }
}
```
* **实现逻辑**：
  1. 根据 `server_id` 获取该节点所属分组 `group_id`。
  2. 获取拥有包含该 `group_id` 套餐、且未过期、未封禁的所有有效用户。
  3. 映射为 V2bX 需要的 `id` 与 `uuid`（Shadowsocks/Trojan/Hysteria2 作为连接密码）。

---

### 3.4 用户流量上报：`POST /api/v1/server/UniProxy/push`
* **Query 参数**：`node_id` (int64), `token` (string)
* **请求体**：支持 JSON 数组 `[{"user_id": 1001, "u": 2097152, "d": 20971520}]` 或 V2bX 原生 Map 报文 `{"1001": [2097152, 20971520]}`。
* **响应格式**：
```json
{
  "data": true
}
```
* **核心口径（与原有模式 A 的差异）**：
  * **增量上报 (Delta)**：V2bX 上报的 `u` 与 `d` 为**该周期内产生的增量字节**（V2bX 上报后本地计数清零），**不是单调递增累计值**。
  * **计费累加**：服务端直接计算：
    - `delta_u_billed = item.u * server.rate`
    - `delta_d_billed = item.d * server.rate`
  * 累加到 `users.u`、`users.d`，写入 `traffic_logs` 日记账，并实时失效该用户的 `subscription-userinfo` 缓存。

---

## 4. 关键文件与模块变更清单

| 模块 | 涉及文件 | 变动说明 |
| :--- | :--- | :--- |
| **Model / DTO** | `server/internal/model/dto_uniproxy.go` | 新增 UniProxy 相关的请求/响应 DTO 结构与双格式流量解析 |
| **Middleware** | `server/internal/middleware/uniproxy_auth.go` | 新增 `Token` 与 `node_id` 鉴权中间件（40100 错误码与 Redis 快速校验） |
| **Handler** | `server/internal/handler/uniproxy.go` | 实现 `Config`、`User`、`Push` 三个端点 |
| **Service** | `server/internal/service/uniproxy_service.go` | 核心适配逻辑、配置序列化、协议校验与增量流量入账 |
| **Router** | `server/internal/router/router.go` | 注册 `/api/v1/server/UniProxy` 路由组 |
| **Frontend UI** | `src/components/admin/NodeInstallDialog.vue` | 节点管理弹窗新增 V2bX 安装标签、自动生成配置文件与部署命令 |
| **Locales** | `src/locales/zh-CN.ts`, `src/locales/en-US.ts` | 节点安装与 V2bX 配置指引中英文国际化文本 |
| **Docs** | `docs/backend/uniproxy-v2bx-spec.md` | 技术规范与部署指南同步归档 |

---

## 5. 验收标准

1. **协议正确性**：
   - 单元测试覆盖各协议节点的配置映射输出（VLESS-Reality、Hysteria2、Trojan、Shadowsocks）；
   - 单元测试覆盖正常、伪造 Token 及缓存命中与失效拦截；
   - 单元测试覆盖多用户增量流量的批量入账（数组与 Map 双格式）与倍率计算。
2. **联调与部署验证**：
   - 提供本地自动化测试集成套件验证握手、用户下发与增量上报流程；
   - 官方 V2bX 节点部署文档覆盖 systemd 与配置文件生成；
   - 前端安装弹窗经单元测试覆盖核心映射与多语言文本。
