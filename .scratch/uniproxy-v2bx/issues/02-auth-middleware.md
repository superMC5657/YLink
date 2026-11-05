# 02: [Backend] 实现 UniProxy 节点通信鉴权中间件

Status: resolved
Priority: P0
Blocked by: 01

---

## 1. 任务背景
V2bX 与面板通信时，可能通过 URL Query 参数传递 `?token={token}&node_id={id}`，也可能在 HTTP Header 中传递 `Token: {token}`。
为了保证节点与面板通讯的安全性，必须对每个进入 `/api/v1/server/UniProxy/*` 的请求进行严格校验。

---

## 2. 详细工作内容

1. **新建文件 `server/internal/middleware/uniproxy_auth.go`**：
   - 提取 `node_id`：优先从 Query 参数 `node_id` 解析；若缺失则直接报错拒绝。
   - 提取 `token`：优先从 Query 参数 `token` 读取，若为空则从 Header `Token`、`X-Token`、`X-Node-Key` 读取。
   - 校验机制：
     - 查询 Redis 缓存 `node:uniproxy:token:{node_id}`（TTL 60s），命中且匹配则直接放行；若缓存命中但 token 不匹配，Fail-fast 直接返回 401（不回源 DB）；
     - 未命中缓存时，通过 Repo 查询 `servers` 表中该节点的 `node_key`；
     - 若 `servers.node_key == token` 且节点 `status == 1 || status == 2`，将结果写入 Redis，并将 `server_id` 注入 Gin 上下文；
     - 校验失败返回 HTTP 401（JSON：`{"code": 40100, "message": "unauthorized", "data": null}`）。

2. **单元测试与异常覆盖**：
   - 新建 `server/internal/middleware/uniproxy_auth_test.go`；
   - 覆盖以下分支：
     - 缺失 `node_id` ➔ 拦截并返回 401 (40100)
     - 缺失 `token` ➔ 拦截并返回 401 (40100)
     - `token` 与节点密钥不匹配 ➔ 拦截并返回 401 (40100)
     - 节点不存在或已禁用 ➔ 拦截并返回 401 (40100)
     - 鉴权成功且命中 Redis 缓存 ➔ 成功放行并正确设置上下文。

---

## 3. 验收标准
- [x] 中间件能同时兼容 Header `Token` 与 Query `?token=` 两种传参方式；
- [x] 具备 Redis 缓存鉴权结果能力，不匹配时 Fail-fast，避免高频请求击穿数据库；
- [x] 单元测试 100% 覆盖关键鉴权分支。
