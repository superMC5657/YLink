# [P1] Use a STARTTLS-capable SMTP endpoint for Alertmanager

Status: resolved
Type: task

## Finding

Alertmanager SMTP 客户端只支持 STARTTLS，文档默认 QQ 465 是隐式 SMTPS，启用 obs profile 后邮件无法投递。

## Resolution

Alertmanager 默认使用 `ALERT_SMTP_PORT=587`，支持 `ALERT_SMTP_HOST/FROM` 覆盖；465 场景在文档中标注需前置 SMTPS→STARTTLS relay。

## Comments

2026-08-25: 已修复并同步 `.env.example`、`deploy.md`、`launch-checklist.md`。
