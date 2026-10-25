# [P2] 文档对齐批（README/launch-checklist/deploy/data-model/desktop-tauri/Makefile/compose）

Status: resolved
Type: task
Blocked by: 15, 16

## Finding

违反 AGENTS.md 文档同步检查表的陈旧描述与两处小代码错位：

1. `docs/backend/README.md:118 + §3`：`Generator.Build` 文档签名 `([]byte, string, error)/nodes []Server` ≠ 实际 `([]byte, error)/[]Node`；列出不存在的 `pkg/qrcode`。
2. `docs/backend/launch-checklist.md:58`、`deploy.md:80`：「迁移演练…(至 0004)」/「升级注意 0007」——现为 0008（本批后含 down 修复不变号）。
3. `data-model.md:289`：「迁移插入首个管理员」≠ 实际运行时 `repo.EnsureAdmin`（cmd/server/main.go:55）；`:3`「所有表含 created_at/updated_at」对 traffic_logs/ticket_messages/audit_logs/mail_logs/traffic_reset_logs/settings 不成立。
4. `docs/frontend/desktop-tauri.md §7`：Windows-only、guard job、单 latest.json+gh-proxy、CI「三 job」均与现双 OS 矩阵/双清单/四 job 不符；`§4` 托盘「按系统主题切换亮暗两套资源」未实现（静态 default_window_icon）。§4/§7 由 ticket 13 一并落地后此处核对。
5. `server/Makefile:17`：`swag init -o doc` 产物在 `doc/`，仓库实际 `docs/`——每次重生成产出孤儿目录。
6. `server/docker-compose.prod.yml:19-27`：caddy override 只声明 data 卷，基座 Caddyfile bind 经卷合并仍生效，deploy.md §3.1「不再 bind mount/不可变发布」不成立 → 需 `!reset`。
7. 邀请分成默认值双源分叉：`configs/config.yaml:48-54`（30/40）vs `0001_init.up.sql:379-380` 种子（40/50）——同一事实两处维护。
8. spec 文本勘误：admin-console-split A6（`/admin/*→dashboard` 已被 opt-perf-sec ticket10 超越为 `/403`）、A3（13 项→18 项）；opt-perf-sec ticket09「collect 步骤缺 .sig 即失败」实为 build-latest-json.mjs:88 兜底。

## Plan

1-4. 逐条把文档改为当前态（以代码为准；desktop-tauri §3/§7 在 ticket 13 落地后核对）。
5. Makefile `-o doc` → `-o docs` 并验证 `make swagger` 产物落位。
6. prod compose caddy service 对基座 bind 加 `!reset`（或重声明目标），与 deploy.md §3.1 对齐。
7. 分成默认值收敛单源：settings 表种子为准，config.yaml 删除该项或注释指向种子（按 README §8 settings 表为运行时来源的定位）。
8. 三个 spec 文本勘误直接改 spec 行（保留 ✅ 状态，只修陈述）。

## Acceptance

- `grep -rn "0004\|pkg/qrcode\|至 0004" docs/backend` 无陈旧迁移计数/幽灵包引用。
- `make swagger` 产物 diff 仅限 docs/ 且内容与 swagger.json 一致。
- `docker compose -f docker-compose.yml -f docker-compose.prod.yml config` 中 caddy 无宿主 Caddyfile bind。
- 全部文档改动遵循「同一事实只在一处维护」。

## Comments

- 2026-10-25: 已修复——backend README §1/§3/§7（删幽灵 `pkg/qrcode` 与 go-qrcode 选型行、pkg 目录树对齐实际包清单、Generator 签名改 `Build(u *User, nodes []Node) ([]byte, error)`）；launch-checklist 迁移演练至 0009、deploy §1.2 升级注记改 0007~0009；data-model 头部时间戳改为分类例外陈述、§4 管理员/演示账号改述为启动时 `repo.EnsureAdmin/EnsureDemoUser`（迁移不插账号，删未实现的「首登强制改密」）；desktop-tauri §2 targets/§4 托盘静态图标/§7 重写为 ci 四 job（PR 门禁 + tag 全量 + rust 双 OS 矩阵）与 release 三段（ci 复用 → 双平台 build → publish 内联 SemVer/release.env 校验 + 双清单）；Makefile swagger 落位 `docs/` 并再生成（本机无 swag 二进制，经 `go run github.com/swaggo/swag/cmd/swag@v1.16.6`（go.mod 锁定版本）执行等价命令验证，产物 diff 仅 server/docs/ 三件，为补齐入库陈旧产物，JSON 82 paths 含 /admin/audit-logs）；compose prod caddy `volumes: !override`（docker compose v5.1.3 config 实测仅剩 caddy_data:/data，无宿主 Caddyfile bind）；config.yaml 删除整段 `site:` 块收敛 settings 单源（cfg.Site 零消费，struct 删除归 #19）、core-flows 佣金键名对齐种子 `invite.commission_rate/agent_commission_rate`；spec 勘误 ×3（admin-console-split A3 18 项、A6 /403 收紧注记、opt-perf-sec 09 .sig 卡点归属 build-latest-json.mjs）。Acceptance grep 复验：docs/backend 剩余 0004 均为「迁移 0004 新增」列级溯源注记，非陈旧迁移计数。
