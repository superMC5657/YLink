# ✅ 节点机器管理与中台部署脚本配置 (F17)

> 状态: 已完成
> 日期: 2026-09-11
> 范围: 探针心跳与硬件监控 / 中台部署脚本动态配置 / 一键安装命令 / 生产 Agent 增强

## 1. 需求背景

当前 YLink 服务端已具备流量模式 A 契约（`GET /node/users`、`POST /node/report`）与密钥鉴权，但节点机管理仍处于手动运维状态：
1. **状态无实时感知**：面板无法感知节点 VPS 的实时心跳与在线/离线状态。
2. **缺乏硬件负载指标**：无法获知节点的 CPU、内存、实时网络带宽与连接数。
3. **缺乏一键部署体验**：新节点上线缺少一键安装/对接流程。
4. **中台脚本指定需求**：用户要求**部署脚本可以手动在中台更换指定**（如企业自建脚本源、国内镜像加速源或自定义安装脚本）。

## 2. 方案设计

### 2.1 存储与通信设计 (Redis 优先)
- 节点通过 `POST /node/report` 上报流量时同时携带可选的 `system` 探针数据（`cpu_percent`, `mem_total`, `mem_used`, `mem_percent`, `disk_total`, `disk_used`, `disk_percent`, `net_rx_rate`, `net_tx_rate`, `tcp_conns`, `udp_conns`, `uptime_sec`）。放宽 `data` 为 `omitempty`，支持纯心跳探针上报。
- 采用 Redis 记录探针状态（`node:heartbeat:{id}` TTL 90s 判定在线，`node:stats:{id}` TTL 180s 缓存硬件快照），避免高频心跳写库 PostgreSQL。
- 管理端 `GET /admin/servers` 聚合 Redis 缓存批量注入 `is_online`、`last_heartbeat_at` 和 `system` 数据返回前端。

### 2.2 中台部署脚本可配置
- 中台站点设置（`settings` 表）新增 `node` 键：
  ```json
  {
    "deploy_script_url": "https://raw.githubusercontent.com/superMC5657/YLink/main/deploy/install-agent.sh",
    "command_template": "curl -fsSL {script_url} | sudo bash -s -- --endpoint {api_url} --node-id {node_id} --node-key {node_key}"
  }
  ```
- 管理端「节点管理」行操作增加「部署」按钮，唤起 `NodeInstallDialog`：
  - 自动根据当前节点的 `node_id`、`node_key` 与面板 `api_base_url` 渲染完整安装命令。
  - 支持一键复制命令与一键复制 Node Key。
  - 支持就地查看与自定义部署脚本 URL 和命令模板（实时持久化至 settings）。

### 2.3 生产 Agent 与部署脚本
- 编写通用 Linux 部署脚本 `deploy/install-agent.sh`。
- 增强 `server/cmd/node-agent`，基于 Linux `/proc`（Windows fallback）实现无 Cgo 的系统指标采集与上报。

## 3. 验收标准

- [x] 后端单元测试通过，`make go-check` 全绿
- [x] 前端类型检查 `pnpm typecheck` 与 `pnpm lint` 全绿
- [x] 前端单元测试 `pnpm test` 全绿
- [x] 管理后台节点列表展示在线/离线状态灯与 CPU/内存/带宽悬浮信息
- [x] 管理后台可唤起部署命令弹窗，可复制命令并动态更换/配置部署脚本
- [x] 接口契约文档 `docs/api/README.md` 与进度文档更新完成
