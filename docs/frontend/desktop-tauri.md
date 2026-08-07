# 前端开发文档 · Tauri 2 桌面集成与发布

> Tauri 2 负责把同一套 Vue SPA 包装为 Windows/macOS/Linux 桌面应用，并提供原生能力。原则：**Rust 侧尽量薄**——只承担系统能力，业务逻辑全部在前端。

## 1. 工程结构

```
src-tauri/
├── capabilities/
│   └── default.json        # v2 权限声明（最小授权）
├── gen/                    # 自动生成的平台工程（不手改）
├── icons/                  # 各尺寸应用图标
├── src/
│   ├── main.rs             # 入口
│   └── lib.rs              # 插件注册、托盘/菜单/深链接处理
├── Cargo.toml
├── build.rs
└── tauri.conf.json
```

## 2. tauri.conf.json 关键配置

| 配置 | 值 | 说明 |
|---|---|---|
| `productName` / `identifier` | 站点名 / `com.nanocloud.app` | identifier 全局唯一 |
| `app.windows[0]` | 1280×800，min 1024×680，title 跟随站点名 | 保留系统原生标题栏 |
| `beforeDevCommand` / `devUrl` | `pnpm dev` / `http://localhost:5173` | 开发联动 |
| `beforeBuildCommand` / `frontendDist` | `pnpm build` / `../dist` | 打包联动 |
| `app.security.csp` | `default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'` | 接口走 http 插件，无需放宽 connect-src |
| `bundle.targets` | `nsis`（Win）/ `dmg`（macOS）/ `appimage, deb`（Linux） | — |
| `plugins.updater.endpoints` | `https://<release-host>/latest.json` | 自动更新清单 |

## 3. 插件清单与用途

| 插件 | 用途 | 对应页面能力 |
|---|---|---|
| `tauri-plugin-http` | 原生栈发请求，绕开 WebView CORS/CSP | 全部接口 |
| `tauri-plugin-store` | 设置与 token 持久化（JSON 文件） | storage 适配层 |
| `tauri-plugin-clipboard-manager` | 写剪贴板 | 复制订阅链接/邀请码/订单号 |
| `tauri-plugin-opener` | 打开外部 URL 与自定义 scheme | TG 链接、支付跳转、一键导入 |
| `tauri-plugin-deep-link` | 注册 `nanocloud://` 协议 | 从网页/TG 唤起 App 并定位页面（如 `nanocloud://plans`） |
| `tauri-plugin-single-instance` | 单实例运行 | 重复启动聚焦已有窗口并转发深链接参数 |
| `tauri-plugin-autostart` | 开机自启 | 设置页开关（默认关） |
| `tauri-plugin-notification` | 系统通知 | 订阅到期、工单回复提醒（轮询发现变化时触发） |
| `tauri-plugin-updater` + `tauri-plugin-process` | 检查更新、下载安装、重启 | 启动时静默检查 + 设置页手动检查 |
| `tauri-plugin-window-state` | 记忆窗口尺寸/位置 | 体验细节 |
| `tauri-plugin-os` | 系统信息 | 关于页、埋点 UA |

capabilities/default.json 采用最小授权：逐项声明上述插件权限，`http` 权限的 `scope` 限定为已配置的后端域名（`https://api.example.com/**`），深链接插件仅注册 `nanocloud` scheme。

## 4. 窗口、托盘与系统行为

- **窗口**：单主窗口；关闭按钮默认最小化到托盘（可在设置改为直接退出）；托盘菜单：显示主窗口 / 检查更新 / 退出。
- **主题跟随**：监听前端主题切换事件（`emit` 到 Rust），调用窗口 `set_theme` 让标题栏亮暗一致；托盘图标按系统主题切换亮暗两套资源。
- **单实例 + 深链接**：第二个实例启动时，把 argv/深链接 URL 转发给已有实例，前端监听 `deep-link://new-url` 事件做路由跳转。
- **通知触发点**：订阅剩余 ≤3 天（每日一次）、工单状态变为已回复、订单支付成功（App 内轮询发现后触发本地通知）。

## 5. 自动更新

1. Release 流水线生成 `latest.json`（版本、各平台产物 URL、签名）并上传至固定地址（GitHub Releases 或自建静态服务）。
2. App 启动后 10s 静默检查；发现新版本弹出自制更新卡片（版本号 + 更新日志），用户确认后下载、校验签名、安装并重启。
3. 签名密钥对：`tauri signer generate` 生成；私钥存 CI Secret（`TAURI_SIGNING_PRIVATE_KEY` + 密码），公钥写入 `tauri.conf.json`。
4. 降级策略：更新检查失败静默忽略，不打扰用户；设置页显示当前版本号与「检查更新」按钮。

## 6. 安全要点

- CSP 收紧到 `'self'`；生产构建禁用 devtools（Cargo feature 控制）与右键菜单。
- token 仅存 store 文件，不进日志；Rust 侧不接触业务数据。
- 所有外部 URL 打开前做协议白名单校验（仅 `https:`、`mailto:`、已知的客户端 scheme）。
- 依赖审计：CI 跑 `cargo audit` 与 `pnpm audit`。

## 7. 构建与发布流水线（GitHub Actions）

- **PR CI**：`pnpm lint` → `vue-tsc` → `vitest` → `pnpm build`；Rust 侧 `cargo check`。
- **Release（打 tag `v*` 触发）**：矩阵构建（windows-latest / macos-latest / ubuntu-latest），`tauri-action` 产出 NSIS/DMG/AppImage/deb + updater 签名产物，上传 GitHub Release 并生成 `latest.json`。
- **macOS 公证**（可选二期）：Apple 证书 + notarize 步骤；无证书时文档注明用户需在「安全性与隐私」中放行。
- 版本号策略：语义化版本，`package.json`、`Cargo.toml`、`tauri.conf.json` 三处由脚本同步。

## 8. 与纯 Web 部署的关系

- `pnpm build` 产物即完整 Web 版，可独立部署到任意静态托管（Nginx/Caddy/对象存储 + CDN）。
- Web 版隐藏桌面专属入口（托盘/自启/更新卡片），其余功能一致；一键导入、复制等能力自动降级（见 [data-layer.md](data-layer.md)）。
- Web 部署需后端开启 CORS（允许 Web 域名），Tauri 版无此要求。

## 9. 移动端策略（Tauri 2 Mobile）

一期决策：**不打包 Tauri Android/iOS 应用**，手机端体验由响应式 Web 完整承载（见 [design-system.md](design-system.md) 第 6 节）。理由：

1. 手机浏览器 + PWA（可后续加入主屏图标、离线壳）已覆盖「查流量/买套餐/复制订阅/一键导入」全部场景；scheme 唤起在移动浏览器表现最好。
2. Tauri Mobile 的 iOS 分发需要 Apple 开发者账号与审核，代理类应用上架风险高，投入产出比低。

架构预留（若二期评估后要做）：
- 能力适配层（`utils/platform.ts`）已抽象，新增 `isTauriMobile()` 分支即可；
- 深链接、通知、剪贴板插件均有 mobile 实现；
- 需补的工作：移动端窗口/安全区适配（响应式已覆盖）、`tauri android/ios init` 工程、签名与分发渠道、移动端专属能力（如分享面板）。
