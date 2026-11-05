# V2bX 节点守护程序对接与部署指南

本文档介绍如何在 Linux VPS 节点上部署 **V2bX**，并通过 YLink 提供的 **V2Board UniProxy 兼容协议**（`/api/v1/server/UniProxy/*`）实现多协议代理核心（Xray / Sing-box / Hysteria2）的自动化配置、用户凭证动态同步与增量流量计费。

---

## 1. 准备工作

1. **YLink 面板准备**：
   - YLink 服务端已正常部署，外网可访问其 API 域名（例如 `https://ylink.example.com`）。
   - 登录 YLink 管理控制台进入「节点管理」，新建对应协议的节点。
   - 获取该节点的 **Node ID**（数字 ID）与 **Node Key**（节点密钥）。

2. **VPS 节点准备**：
   - 推荐系统：Ubuntu 20.04+ / Debian 11+ / CentOS 8+ / AlmaLinux 9+。
   - 开放对应业务端口（例如 443、8443）及防火墙。

---

## 2. V2bX 一键安装

在目标 VPS 上以 root 权限运行官方一键安装脚本：

```bash
wget -N --no-check-certificate https://raw.githubusercontent.com/v2bx/v2bx/master/install.sh && bash install.sh
```

或使用 curl：

```bash
curl -fsSL https://raw.githubusercontent.com/v2bx/v2bx/master/install.sh | bash
```

安装完成后，主程序与配置文件默认路径为：
* 配置文件：`/etc/V2bX/config.json`
* Systemd 服务名：`V2bX`

---

## 3. 配置 V2bX (`/etc/V2bX/config.json`)

编辑 `/etc/V2bX/config.json`，根据节点类型配置对应的 Node 项。

### 3.1 核心配置模板

```json
{
  "Log": {
    "Level": "warning",
    "Output": ""
  },
  "Nodes": [
    {
      "Core": "xray",
      "ApiHost": "https://ylink.example.com",
      "ApiKey": "YOUR_SERVER_NODE_KEY",
      "NodeID": 1,
      "NodeType": "v2ray",
      "Timeout": 30,
      "ListenIP": "0.0.0.0",
      "SendIP": "0.0.0.0",
      "EnableProxyProtocol": false,
      "EnableFallback": false,
      "FallBackConfigs": []
    }
  ]
}
```

> **注意**：
> * `ApiHost`：填写 YLink 面板的根域名（**不包含** `/api/v1` 前缀），V2bX 会自动请求 `/api/v1/server/UniProxy/*`。
> * `ApiKey`：填入 YLink 管理后台该节点的 `node_key`。
> * `NodeID`：填入 YLink 管理后台该节点的数字 ID。

---

### 3.2 各协议推荐配置

| YLink 节点类型 | V2bX `NodeType` | V2bX `Core` | 说明 |
| :--- | :--- | :--- | :--- |
| **VLESS-Reality** | `v2ray` | `xray` 或 `sing` | 自动下发 Reality 密钥与目标域名伪装 |
| **VMess** | `v2ray` | `xray` | 支持 TCP/WS/gRPC 传输层 |
| **Shadowsocks** | `shadowsocks` | `xray` 或 `sing` | 自动下发 AEAD / 2022 密码与加密套件 |
| **Trojan** | `trojan` | `xray` 或 `sing` | 支持自动 ACME 证书申请 |
| **Hysteria 2** | `hysteria` | `hysteria2` 或 `sing` | 下发 UDP 端口、上下行速率与混淆口令 |

---

## 4. 服务管理与启动

配置保存后，启动或重启 V2bX 服务：

```bash
# 重启服务
v2bx restart
# 或使用 systemctl
systemctl restart V2bX

# 查看运行状态
systemctl status V2bX

# 实时查看节点日志
v2bx log
# 或 journalctl
journalctl -u V2bX -f
```

---

## 5. 端到端闭环验证流程

1. **配置下发验证**：
   查看 V2bX 启动日志，确认出现类似以下信息且无 401 报错：
   ```text
   [INFO] Get node info success, start core...
   ```
   若返回 HTTP 304，表示配置未变更，已复用本地缓存。

2. **用户同步验证**：
   查看日志，确认拉取到用户凭证：
   ```text
   [INFO] User count: 12, added: 12, deleted: 0
   ```

3. **代理翻墙测试**：
   使用 Clash Verge / Sing-box / v2rayN 客户端拉取个人订阅并连接该节点，成功访问测试网页（如 `https://www.google.com` 或 `https://ipinfo.io`）。

4. **增量流量扣费验证**：
   - 产生约 50MB 流量。
   - 等待 60 秒（一个 Push 周期）后，V2bX 输出日志：
     ```text
     [INFO] Report user traffic success
     ```
   - 在 YLink 用户端「仪表盘」或管理后台「流量明细」中，确认已用流量准确增加了 $50\text{MB} \times \text{节点倍率}$。
   - 确认在 YLink 管理后台「节点列表」中，该节点心跳指示灯为绿色「在线」。

---

## 6. 常见问题排查

* **HTTP 401 `token invalid`**：
  检查 `/etc/V2bX/config.json` 中的 `ApiKey` 是否与后台该节点的 `node_key` 完全一致；检查后台节点是否处于「正常（状态 1）」；若重置了节点密钥，请同步更新 VPS 配置。
* **节点显示离线**：
  YLink 节点在线判定依据为 90 秒内的上报心跳。确认 V2bX 能够正常与 YLink 接口通讯，并且端口无网络阻断。
* **ACME 证书申请失败 (Trojan/Hysteria2)**：
  确认 VPS 的 80/443 端口未被其他服务占用，且节点的域名解析已正确指向该 VPS 的公网 IP。
