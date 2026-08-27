# 节点 Agent 部署对接说明（Xray 流量上报）

> 面向：VPS 节点侧的运维 / 对接开发。
> 目的：让节点上的代理后端（Xray）与 YLink 面板对接，实现「流量模式 A」——节点自动上报每用户流量，面板侧差分累加、计费、日聚合。
> 状态：**面板服务端协议已全部就绪**（契约 §17，`server/` 内 `GET /node/users`、`POST /node/report` 均已实现），本文档只讲**节点侧**如何对接，面板服务端无需改动。

---

## 1. 总体流程

```
面板管理端 ── 下发 node_key ──────────────────────────┐
                                                      │
VPS 节点 agent（自研脚本/程序）                        ▼
  ① GET /node/users   拉取有效订阅用户（uuid / u / d / transfer_enable / expired_at）
  ② 生成 Xray 配置（api + stats + 每用户 client，email=uuid）并重载
  ③ 定时读取 Xray stats，取每用户累计 uplink / downlink
  ④ POST /node/report 上报每用户累计值
```

面板侧收到上报后：快照差分得增量 → 增量 × 节点 `rate`（倍率）→ 累加 `users.u/d` + `traffic_logs` 日聚合 → 失效 `subscription-userinfo` 缓存。**重复上报天然幂等**（差分 0）。

---

## 2. 前置条件

| 项 | 说明 |
|---|---|
| 面板 API 地址 | 即 `APP_BASE_URL`，如 `https://panel.example.com/api/v1`；节点需能访问该域名 |
| 节点密钥 `node_key` | 管理端「节点」列表查看/重置（`POST /admin/servers/{id}/node-key/reset`，重置后旧密钥立即失效）。32 位十六进制字符串 |
| `per_user_credentials` | 节点 `servers.config` 需为 `true`，订阅端才会下发每用户独立凭证 `users.uuid`（模式 A 归因依据）。**先在节点配发好每用户 client，再开启该开关**，否则存量用户订阅刷新会断连 |
| Xray | 已安装并可由 agent 控制配置与重载（systemd 托管更稳妥） |

---

## 3. 鉴权

两个端点都走请求头：

```
X-Node-Key: <node_key>
```

- 密钥 → 节点 id 的映射在面板侧经 Redis 缓存 60s（`node:key:{k}`）。
- 无效或缺失 → 错误码 `40100`，HTTP 401。
- 重置密钥后旧密钥立即失效（面板会删除对应缓存），节点需同步更新。

---

## 4. 端点对接

### 4.1 `GET /node/users` — 拉取有效订阅用户

```bash
curl -s https://panel.example.com/api/v1/node/users \
  -H 'X-Node-Key: <node_key>'
```

响应（统一信封）：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "rate": 1.0,
    "users": [
      {
        "uuid": "5f2b7c9e-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "u": 1073741824,
        "d": 10737418240,
        "transfer_enable": 107374182400,
        "expired_at": 1767225600
      }
    ]
  }
}
```

字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `rate` | float | 节点倍率（面板计费时用，节点无需自行乘） |
| `users[].uuid` | string | 用户订阅凭证。vmess/vless/tuic 即 uuid；shadowsocks/trojan/hysteria2 作为密码下发。节点 inbound 按此区分用户 |
| `users[].u` / `d` | int64 | 用户当前已用流量（字节，已含倍率），仅作节点本地参考 |
| `users[].transfer_enable` | int64 | 用户总流量额度（字节），用于节点本地掐断（可选） |
| `users[].expired_at` | int64/null | 到期时间（unix 秒），`null` 表示不限期 |

返回的是该节点分组下**所有有效订阅且未封禁**的用户（套餐 `group_ids` 含本节点分组、未过期）。节点据此生成 inbound 每用户凭证，并可选做本地限速/到期掐断。

### 4.2 `POST /node/report` — 上报每用户累计流量

```bash
curl -s https://panel.example.com/api/v1/node/report \
  -H 'X-Node-Key: <node_key>' \
  -H 'Content-Type: application/json' \
  -d '{"data":[{"uuid":"5f2b7c9e-xxxx","u":2147483648,"d":21474836480}]}'
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "accepted": 1,
    "skipped": [
      { "uuid": "some-other-uuid", "reason": "not_subscribed" }
    ]
  }
}
```

**口径（务必遵守）**：

| 项 | 说明 |
|---|---|
| 累计值 | `u`/`d` 为**自 agent 启动起单调递增的累计字节**，**非增量**。面板与上次快照差分得增量 |
| 幂等 | 重复上报同一累计值，差分 0，不重复计费，重试安全 |
| 计数器回退 | 某字段 `cur < 上次快照` 视为节点计数器重启，该字段增量取当前值（未回退字段仍按差分） |
| 计费 | 面板按「增量 × 节点 `rate`」累加。**agent 上报原始字节即可，不要自己乘倍率** |
| 数据量 | `data` 1–1000 条；建议上报周期 60s |
| 重复 UUID | 同一请求内同一 `uuid` 出现多次 → 该 uuid 全部条目拒绝（`duplicate_uuid`），务必每用户每请求只报一次 |

`skipped[].reason` 取值：

| reason | 含义 |
|---|---|
| `unknown_user` | uuid 在系统中不存在 |
| `not_subscribed` | 用户无订阅 / 套餐分组不包含本节点 / 封禁 / 已过期 |
| `duplicate_uuid` | 同一请求内 uuid 重复 |

---

## 5. Xray 对接（参考实现）

> 本节为对接方提供参考，非面板侧功能。核心目标：让 Xray 按用户归因统计流量，并读取每用户累计 uplink/downlink 上报。

### 5.1 Xray 配置：启用 API + stats + 每用户 client（email=uuid）

推荐**每用户一个 client，`email` 设为 uuid**，用 `user>>>{email}>>>traffic>>>*` 归因（无需每用户独立 inbound，省资源）：

```jsonc
{
  "api": {
    "tag": "api",
    "services": ["StatsService"]
  },
  "stats": {},
  "policy": {
    "system": {
      "statsInboundUplink": true,
      "statsInboundDownlink": true
    }
  },
  "inbounds": [
    {
      "tag": "api",
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": { "address": "127.0.0.1" }
    },
    {
      "tag": "user-in",
      "listen": "0.0.0.0",
      "port": 443,
      "protocol": "vmess",
      "settings": {
        "clients": [
          { "id": "5f2b7c9e-xxxx", "email": "5f2b7c9e-xxxx" }
          // ... 每个有效用户一个 client，email 与 uuid 一致
        ]
      }
    }
  ],
  "routing": {
    "rules": [
      { "type": "field", "inboundTag": ["api"], "outboundTag": "api-out" }
    ]
  },
  "outbounds": [
    { "tag": "api-out", "protocol": "freedom" }
  ]
}
```

说明：

- `api.services` 必须含 `StatsService`；`stats: {}` 开启统计。
- `api` inbound 是一个 `dokodemo-door`（本地 gRPC 入口，供 agent 查询 stats，监听 `127.0.0.1:10085`）。
- 用户 inbound 的每个 client 用 `email` = `uuid`（shadowsocks 协议的 client 用 `password`=uuid + `email`=uuid 同理）。
- 生成配置后重载 Xray（Xray 无内置运行时热重载，通常 `systemctl restart xray`）。

### 5.2 读取每用户累计流量

stats 查询 pattern（u=上行，d=下行，均为累计字节）：

```
user>>>{uuid}>>>traffic>>>uplink
user>>>{uuid}>>>traffic>>>downlink
```

**命令行方式**（Xray 自带）：

```bash
# 列出所有统计
xray api stats query --server=127.0.0.1:10085 --pattern=''

# 按用户查询
xray api stats query --server=127.0.0.1:10085 \
  --pattern='user>>>5f2b7c9e-xxxx>>>traffic>>>uplink'
```

**Go 客户端方式**（对接方若用 Go 写 agent，可直接用 Xray 官方 client）：

```go
import (
    statsCmd "github.com/xtls/xray-core/app/stats/command"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

conn, _ := grpc.Dial("127.0.0.1:10085", grpc.WithTransportCredentials(insecure.NewTransportCredentials()))
client := statsCmd.NewStatsServiceClient(conn)
resp, _ := client.GetStats(ctx, &statsCmd.GetStatsRequest{
    Name:   "user>>>" + uuid + ">>>traffic>>>uplink",
    Reset_: false,
})
// resp.Stat.Value 即该用户累计上行字节
```

> 也可用 `inbound>>>{tag}>>>traffic>>>uplink` 做整节点统计，但**每用户归因请用 `user>>>{email}>>>traffic>>>*`**（email = uuid）。

---

## 6. 上报节奏与注意事项

- **周期**：建议 60s（契约建议值）；数据条数 1–1000。
- **累计值单调递增**：agent 需保证 `u`/`d` 单调递增。若 agent 重启导致计数器归零，面板按「回退」语义处理（增量取当前值），不会错计，但会损失重启前未上报的部分——建议 agent 持久化累计值或启动后先上报一次基线。
- **只报原始字节**：倍率由面板按节点 `rate` 计算，agent 切勿自行乘倍率。
- **每用户每请求只报一次**：避免触发 `duplicate_uuid`。
- **本地掐断（可选）**：可用 `GET /node/users` 返回的 `transfer_enable` / `expired_at` 在节点侧提前掐断用户，与面板计费互不冲突。
- **密钥轮换**：重置 `node_key` 后，节点 agent 需同步更换 `X-Node-Key`，否则旧密钥 40100。
- **缓存**：面板对受影响用户的 `subscription-userinfo` 缓存（30s）会立即失效，客户端下次拉订阅即见新用量。

---

## 7. 常见问题（FAQ）

**Q：为什么开启 `per_user_credentials` 后客户端订阅刷新会断连？**
A：开关打开后，订阅下发会把共享密码/uuid 换成 `users.uuid`（每用户独立凭证）。若节点 inbound 还没配发好这些每用户 client，客户端就无凭证可用。正确顺序：**先在节点配发每用户 client → 再开 `per_user_credentials: true`**。

**Q：上报返回 `skipped` 要不要重试？**
A：`unknown_user` / `not_subscribed` 说明该 uuid 当前不属于本节点有效用户，无需重试（可能是刚退订/过期/封禁，拉取 `/node/users` 后按列表上报即可）。`duplicate_uuid` 是请求内重复，检查去重逻辑。

**Q：面板服务端还需要改什么？**
A：不需要。`GET /node/users`、`POST /node/report`、X-Node-Key 鉴权、差分累加、倍率、幂等、缓存失效均已实现并通过测试。本文档即对接方唯一需要的协议说明。
