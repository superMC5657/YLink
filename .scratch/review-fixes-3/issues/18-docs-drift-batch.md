# [P2] 文档对齐批（README/launch-checklist/deploy/data-model/desktop-tauri/Makefile/compose）

Status: ready-for-agent
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
