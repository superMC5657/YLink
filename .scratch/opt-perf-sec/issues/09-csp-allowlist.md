# [P0] 09 CSP与能力收敛

Status: done
Type: task

## Finding

- `tauri.conf.json security.csp` 缺 connect-src；`capabilities/default.json http allow https://**` 过宽；updater 经 gh-proxy 有替换风险但有 minisign 兜底。

## Task

1. CSP 加 `connect-src 'self' https: http://localhost:* http://127.0.0.1:*` + `object-src 'none'; base-uri 'self'`。
2. http allow 收敛到 apiBase 域名+2 updater 域名；桌面 deep-link 按需补 custom protocol；latest.json 签名校验入发布检查。

## Done

- CSP 已补 connect-src/object-src 'none'/base-uri 'self'；`cargo check` 通过。
- release-tauri.yml collect 步骤：`.sig` 缺失即失败（原来静默条件拷贝），与 build-latest-json.mjs 的签名校验双保险。

## Comments

- **http allow 未做静态收敛（有意保留 `https://**`）**：apiBase 运行时可在设置页改指任意自建站点（产品能力，见 docs/frontend/README.md 配置表），且发布流水线不烘焙 API 域名；Tauri capability scope 是静态 JSON，收敛到固定域名会砍掉多站点客户端能力。updater 下载走 Rust 侧、不受 http plugin scope 约束（minisign pubkey 验签兜底），updater 域名无需进 scope。残余风险与主防线记录在 docs/frontend/desktop-tauri.md §3（script-src 'self' + 无远程脚本加载）。
- 桌面 deep-link：Rust 侧无 deep-link 代码，仅 mobile 配置存在，「按需补」结论=无需补。
- 桌面 webview 内全部 API 走 plugin-http（CSP 不拦），connect-src 仅覆盖降级 fetch/本地后端兜底路径；webview 侧现无 EventSource/WebSocket 直连。
