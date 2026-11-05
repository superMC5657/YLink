# V2Board / UniProxy 协议兼容与 V2bX 节点守护对接规范

> 目标：为 YLink 服务端实现 V2Board UniProxy 通讯协议兼容层，支持无缝接入社区高性能多协议节点守护程序 V2bX。  
> 详尽任务拆解与工单跟踪见项目根目录：`.scratch/uniproxy-v2bx/`

---

## 1. 方案选型与背景

YLink 原生定义了流量模式 A 协议（`GET /node/users`、`POST /node/report`），且提供了探针 Agent（`cmd/node-agent`）。但为了在真实海外节点上实现完善的代理核心守护、多协议（VLESS-Reality / Hysteria2 / Trojan / Shadowsocks）支持以及自动 ACME 证书申请，推荐兼容 **V2Board UniProxy 事实标准协议**，由社区成熟的 **V2bX** 承担节点端的底层代理驱动工作。

### 优势对比
* **开发成本**：无需自研复杂的 Xray/Sing-box gRPC 动态注入与证书自动化，服务端仅需实现 3 个标准端点（工作量从 1 个月压缩至 1 天）。
* **协议生态**：开箱即用支持 Xray、Sing-box、Hysteria2、Trojan-Go。
* **运维便利**：社区提供完善的 Linux Systemd 一键安装部署脚本。

---

## 2. 接口契约与计费规范

接口端点定义、鉴权规则（含 HTTP 401 与 40100 错误码）、双重包裹与增量流量计算规则以 **[接口契约文档 (docs/api/README.md §18)](../api/README.md)** 为单一事实来源：
- 契约细节与响应示例见 `docs/api/README.md §18`。
- 节点鉴权缓存与状态流转见 `docs/backend/data-model.md`。
- 详细 VPS 部署指引与 V2bX 配置示例见 `docs/backend/v2bx-deployment-guide.md`。

