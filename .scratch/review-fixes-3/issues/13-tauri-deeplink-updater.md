# [P1] Tauri 深链不可达与更新双清单半接线

Status: resolved
Type: task

## Finding

1. `src-tauri/tauri.conf.json:50-53`：`plugins.deep-link` 只声明 `mobile`，无 `desktop.schemes`，且 `lib.rs` 未调 `register_all()`——NSIS 模板的协议注册块（installer.nsi:680-685，由 `deep_link_protocols` 驱动）不写注册表，`ylink://` 桌面深链整条链路（desktop-tauri.md §3/§4 的单实例 argv 转发，lib.rs:64-78）不可达。
2. `.github/workflows/release-tauri.yml:181-198` + `tauri.conf.json:56-59`：`latest.json` 以 `--proxy-prefix ""` 构建（直连 GitHub asset），`latest-cn.json`（gh-proxy asset，build-latest-json.mjs:69）没有任何 updater endpoint 引用——端点只拉 `latest.json`，desktop-tauri.md §7 承诺的国内下载加速到不了二进制。
3. `src-tauri/capabilities/default.json:8-13`：六个 `core:window:*` 授权（set-theme/show/set-focus/minimize/close/start-dragging）零 JS 调用（主题走自定义命令 set_window_theme 无需 core 权限；show/focus 在 Rust 侧不受 capability 门控）——违反 §3 最小授权。
4. `src-tauri/Cargo.toml:31-33`：serde/serde_json 未使用；`log::info!`（lib.rs:77）无 logger 插件注册为静默 no-op。

## Plan

1. deep-link 补 `"desktop": {"schemes": ["ylink"]}` 并在 setup 调 `app.deep_link().register_all()`（Windows 注册表由 NSIS 块 + 运行时注册双保险）。
2. updater endpoints：endpoint #1 指向 `latest-cn.json`（保留 fallback endpoint 指向 `latest.json`），或恢复 latest.json 的 proxy 前缀——以「国内用户默认走代理清单」为目标，与 build-latest-json.mjs 的双清单意图对齐。
3. capabilities 删除 6 个未用授权，保留 `core:default`。
4. Cargo 删 serde/serde_json；`log::info!` 要么接 tauri-plugin-log，要么改 eprintln/删除。
5. desktop-tauri.md §3/§7 同步当前真实行为（深链注册方式、endpoint 布局）。

## Acceptance

- Windows 安装后 `ylink://` 打开应用并转发参数（手动 regedit 或安装实测）。
- release dry-run 产出的 endpoint 配置实际指向存在的清单文件；CN/直连两条路径 URL 均可下载。
- `cargo clippy` 无未用依赖告警；capabilities 语法通过 `tauri build`。
- desktop-tauri.md 与实现一致（AGENTS 文档同步清单）。

## Comments

- 勘误 Finding 4：`serde_json` 不能删——`tauri::generate_context!` 宏展开直接引用 `::serde_json`（解析 tauri.conf.json/capabilities），删除后 clippy 报 E0433；已保留并加注释说明。`serde` 与 `log` 确认零引用已删（`log::info!` 改 `eprintln!`，未引 tauri-plugin-log）。
- 本地验证：cargo fmt --check / cargo clippy 零警告 / pnpm lint / pnpm test 242 全绿；`pnpm tauri build` 走通（`--config` 临时关闭 createUpdaterArtifacts，本地无 TAURI_SIGNING_PRIVATE_KEY），渲染后 NSIS 脚本含 ylink 注册表写入（target/release/nsis/x64/installer.nsi:656-660）与卸载清理（:779-781），capabilities 语法通过构建。
- 未验证（需真机/CI）：① 安装 NSIS 产物后 `ylink://` 实际唤起与参数转发（本地未执行安装实测）；② release dry-run 与 CN/直连两条下载 URL（需打 tag 触发 Actions + RELEASES_PAT，本地无环境）；③ 签名 updater 产物（私钥仅在 CI secret）。
- 范围外新问题（待 18 号文档对齐批）：desktop-tauri.md §7 首行「Windows-only Release」、README.md:56、docs/frontend/README.md:90 仍写「Windows 打包 / latest.json gh-proxy 前缀」，与工作流双平台矩阵 + 双清单现状不符。
- 2026-10-25: 已修复 deep-link desktop schemes + NSIS/运行时 register_all 双保险、updater 端点 CN 双清单（latest-cn.json 镜像优先 / latest.json 直连兑底）、capabilities 删 6 个未用 core:window:*、Cargo 清理 serde/log。
