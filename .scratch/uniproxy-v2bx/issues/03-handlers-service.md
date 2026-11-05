# 03: [Backend] 实现 UniProxy 核心服务与 Handler 处理端点

Status: resolved
Priority: P0
Blocked by: 01, 02

---

## 1. 任务背景
实现 V2bX 与 YLink 对接的三个核心端点：
1. `GET /api/v1/server/UniProxy/config`：向节点分发网络与加密参数（支持 node_type 校验与 ETag 304）。
2. `GET /api/v1/server/UniProxy/user`：向节点分发当前允许接入的用户凭证列表（支持 ETag 304）。
3. `POST /api/v1/server/UniProxy/push`：接收节点上报的增量流量并核算入账。

---

## 2. 详细工作内容

1. **新建文件 `server/internal/service/uniproxy_service.go`**：
   - `GetConfig(ctx, serverID, nodeType)`：
     - 从 Repo 读取对应节点；
     - 校验 `nodeType` 参数与节点协议一致性；
     - 调用配置解析器生成 `UniProxyConfigData`，填充端口、传输层参数、Reality/TLS 密钥。
   - `GetUsers(ctx, serverID)`：
     - 复用用户过滤逻辑，筛选出属于该节点分组（`group_ids`）、未封禁且套餐未过期的有效用户；
     - 构造为 `[]UniProxyUserItem`，其中 `id` 为 YLink 用户 ID，`uuid` 为用户订阅密钥。
   - `PushTraffic(ctx, serverID, items []UniProxyPushItem)`：
     - **特别注意**：V2bX 上报为**本周期的增量字节 (Delta)**，非累计值！
     - 开启事务：遍历每个上报条目，计算 `delta_u = item.U * server.Rate`，`delta_d = item.D * server.Rate`；
     - 更新 `users.u += delta_u`，`users.d += delta_d`；
     - 批量写入/累加 `traffic_logs`（按天聚合）；
     - 批量淘汰 Redis 中受影响用户的 `sub:userinfo` 缓存。

2. **新建文件 `server/internal/handler/uniproxy.go`**：
   - 提取中间件注入的 `server_id`；
   - 绑定并校验各端点参数，调用 `UniProxyService`；
   - 输出双重包裹的 JSON 报文并支持 ETag 304。

3. **路由挂载**：
   - 在 `server/internal/router/router.go` 中注册 `/api/v1/server/UniProxy`。

4. **单元测试与集成测试**：
   - 编写 `server/internal/service/uniproxy_service_test.go` 与 `server/internal/handler/uniproxy_test.go`。

---

## 3. 验收标准
- [x] 三个接口严格符合 V2Board 契约规范；
- [x] 流量计费计算准确（增量入账且正确乘以节点倍率）；
- [x] 自动化测试全部通过，无并发锁或数据竞争问题。
