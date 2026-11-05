# 01: [Backend] 数据模型与 UniProxy DTO 定义

Status: resolved
Priority: P0

---

## 1. 任务背景
V2bX 节点守护程序在调用 `/api/v1/server/UniProxy/*` 时，对 `/config`、`/user`、`/push` 端点的请求与返回报文格式有严格的结构定义。需要将 YLink 的 `Server`、`User` 领域模型与 V2bX 的数据结构进行解耦与类型映射。

---

## 2. 详细工作内容

1. **新建文件 `server/internal/model/dto_uniproxy.go`**：
   - 定义配置响应结构体 `UniProxyConfigResp`：
     - `ServerPort int`
     - `Network string`
     - `NetworkSettings map[string]any`
     - `TLS int`
     - `TLSSettings UniProxyTLSSettings`（包含 `ServerName`、`RealitySettings` 等）
     - `Protocol string`
     - `BaseConfig UniProxyBaseConfig`（`PushInterval`、`PullInterval`）
   - 定义用户列表响应结构体 `UniProxyUserResp`：
     - `Users []UniProxyUserItem`，每个元素包含 `ID int64`、`UUID string`、`SpeedLimit int64`、`DeviceLimit int`
   - 定义流量上报请求结构体 `UniProxyPushItem`：
     - `UserID int64`
     - `U int64`
     - `D int64`
   - 实现双格式流量上报解析器 `ParseUniProxyPushTraffic`（支持数组格式与 Map 格式）。

2. **协议映射函数**：
   - `ParseUniProxyConfig(srv *Server) (*UniProxyConfigData, error)`：
     - 解析 `srv.Config` JSONB 字段并根据 `srv.Type` 映射。
     - 支持 Reality（dest, server_names, private_key, short_ids）。
     - 支持 WebSocket / gRPC / TCP 传输模式。

---

## 3. 验收标准
- [x] 正确覆盖 VLESS-Reality、Hysteria2、Trojan、Shadowsocks 等主流协议的映射转换；
- [x] 单元测试 `server/internal/model/dto_uniproxy_test.go` 100% 通过。
