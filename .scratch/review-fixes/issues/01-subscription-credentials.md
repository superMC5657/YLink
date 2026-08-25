# [P1] Keep subscription credentials valid until node inbounds are provisioned

Status: resolved
Type: task

## Finding

迁移 0004 后订阅刷新无条件把共享凭证换成 `users.uuid`，但项目未提供 proxy-inbound 配发，存量节点仍接受 `servers.config` 凭证，客户端会在下次刷新后断连。

## Resolution

`toNode` 仅在节点 config 显式开启 `per_user_credentials: true` 时使用 `users.uuid`；未开启继续下发 `servers.config` 共享密码/uuid。存量节点应先配发 inbound 再开启开关。

## Comments

2026-08-25: 已修复并补 `TestGenerateLegacySharedCredentialFallback` / `TestGeneratePerUserCredentialOptIn`。
