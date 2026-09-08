# [P0] 09 CSP与能力收敛

Status: ready-for-agent
Type: task

## Finding

- `tauri.conf.json security.csp` 缺 connect-src；`capabilities/default.json http allow https://**` 过宽；updater 经 gh-proxy 有替换风险但有 minisign 兜底。

## Task

1. CSP 加 `connect-src 'self' https: http://localhost:* http://127.0.0.1:*` + `object-src 'none'; base-uri 'self'`。
2. http allow 收敛到 apiBase 域名+2 updater 域名；桌面 deep-link 按需补 custom protocol；latest.json 签名校验入发布检查。

## Done

- Web 降级 fetch 不被 CSP 拦；XSS 外发受限；`cargo check` 通过。

## Comments
