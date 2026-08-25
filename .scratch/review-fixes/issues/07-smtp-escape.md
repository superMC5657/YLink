# [P2] Escape SMTP values before inserting them with sed

Status: resolved
Type: task

## Finding

Alertmanager entrypoint 直接 sed 替换，用户名/密码含 `&`、`\`、`|`、`'` 时会破坏配置。

## Resolution

改用 AWK 逐字替换模板占位符，并按 YAML 单引号规则转义（单引号 → `''`），避免 sed/awk 替换语法误解 `&`、`\`、`|`。

## Comments

2026-08-25: 已修复 `server/docker-compose.yml` 与 `alertmanager.yml.tmpl` 注释。
