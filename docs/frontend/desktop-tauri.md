# 前端开发文档 · Tauri 2 桌面集成与发布

> Tauri 2 负责把同一套 Vue SPA 包装为 Windows 桌面应用（Android 见 §9），并提供原生能力。原则：**Rust 侧尽量薄**——只承担系统能力，业务逻辑全部在前端。

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
| `productName` / `identifier` | 站点名 / `com.ylink.app` | identifier 全局唯一 |
| `app.windows[0]` | 1280×800，min 1024×680，title 跟随站点名 | 保留系统原生标题栏 |
| `beforeDevCommand` / `devUrl` | `pnpm dev` / `http://localhost:5174` | 开发联动 |
| `beforeBuildCommand` / `frontendDist` | `pnpm build` / `../dist` | 打包联动 |
| `app.security.csp` | `default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'` | 接口走 http 插件，无需放宽 connect-src |
| `bundle.targets` | `"targets": "nsis"` | 仅 Windows NSIS，不产 MSI；Android APK 由 `tauri android build` 独立控制，不受此影响 |
| `bundle.windows.nsis.installerHooks` | `nsis/installer-hooks.nsh` | NSIS 自定义安装/卸载钩子文件已入库（含 4 个 no-op 宏）；当前无自定义逻辑，文件必须保留否则 Windows NSIS 构建失败 |
| `plugins.updater.endpoints` | 已配置（含 pubkey；endpoints 指向公开产物仓库 `superMC5657/ylink-releases` 的 latest.json，gh-proxy.com 优先 + 直连兑底） | updater 与前端更新卡片已接入（见 §5） |

## 3. 插件清单与用途

| 插件 | 用途 | 对应页面能力 |
|---|---|---|
| `tauri-plugin-http` | 原生栈发请求，绕开 WebView CORS/CSP | 全部接口 |
| `tauri-plugin-store` | 设置与 token 持久化（JSON 文件） | storage 适配层 |
| `tauri-plugin-clipboard-manager` | 写剪贴板 | 复制订阅链接/邀请码/订单号 |
| `tauri-plugin-opener` | 打开外部 URL 与自定义 scheme | TG 链接、支付跳转、一键导入 |
| `tauri-plugin-deep-link` | 注册 `ylink://` 协议 | 从网页/TG 唤起 App 并定位页面（如 `ylink://plans`） |
| `tauri-plugin-single-instance` | 单实例运行（仅桌面） | 重复启动聚焦已有窗口并转发深链接参数 |
| `tauri-plugin-autostart` | 开机自启（仅桌面） | 已注册但**前端不暴露开关** |
| `tauri-plugin-notification` | 系统通知 | 订阅到期、工单回复提醒（轮询发现变化时触发） |
| `tauri-plugin-updater` + `tauri-plugin-process` | 检查更新、下载安装、重启（✅ 已接入） | 启动时静默检查 + 设置页手动检查（更新卡片，见 §5） |
| `tauri-plugin-window-state` | 记忆窗口尺寸/位置（仅桌面） | 体验细节 |
| `tauri-plugin-os` | 系统信息 | 关于页、埋点 UA |

capabilities/default.json 采用最小授权：逐项声明上述插件权限，`http` 权限的 `scope` 实际为 `https://**` + `http://localhost:**` + `http://127.0.0.1:**`（而非文档示例的 `https://api.example.com/**`），深链接插件仅注册 `ylink` scheme。

## 4. 窗口、托盘与系统行为

- **窗口**：单主窗口；关闭即退出（未做最小化到托盘设置）；托盘菜单：显示主窗口 / 退出（检查更新入口在设置页/更新卡片，见 §5）。
- **托盘**：由代码手动创建（`TrayIconBuilder::with_id("main-tray")` + 菜单），`tauri.conf.json` 不配 `app.trayIcon`——配置与代码各建一个会出现双图标，其一无菜单点击无反应。
- **主题跟随**：监听前端主题切换事件（`emit` 到 Rust），调用窗口 `set_theme` 让标题栏亮暗一致；托盘图标按系统主题切换亮暗两套资源。
- **单实例 + 深链接**：Rust 单实例回调把 argv 中的 `ylink://` URL 用 deep-link 插件同名事件 `deep-link://new-url`（payload 为 URL 数组）转发给已有实例，前端 `onOpenUrl`（`utils/platform.ts` onDeepLink）直接收到并路由跳转；`lib.rs` 单实例回调已落地。
- **通知触发点**：前端统一封装 `utils/notify.ts`（Tauri 走 plugin-notification，Web 端 Notification API 自动降级）；触发点：订单支付成功（PaymentModal）、工单状态变为已回复（MainLayout 60s 轮询，状态快照去重）、订阅剩余 ≤3 天（窗口聚焦刷新时检测，按到期日去重）。

## 5. 自动更新

> **updater 全链路已就绪**——`tauri-plugin-updater` 已注册（`src-tauri/Cargo.toml` + `lib.rs`），`tauri.conf.json` 已配 `plugins.updater`（pubkey 已写入，endpoints = gh-proxy.com 优先 + 直连兑底），capabilities 已补 `updater:default`，Release 流水线已配置（见 §7）。**前端更新卡片已实现**：`utils/updater.ts`（checkForUpdate / downloadAndInstall，动态 import，Web 端自动降级）+ `components/app/UpdateCard.vue`（右下角浮动卡片：版本号 + 更新日志 + 下载进度 + 立即更新/稍后；App.vue 挂载，启动静默检查）+ 设置页「检查更新」入口（`views/profile/ProfileView.vue`，仅 Tauri 显示，含当前版本号）。

降级策略：更新检查失败静默忽略，不打扰用户；Web 端无更新能力，全部动态 import 保证不进 Web 产物。

签名密钥：密钥对有密码（**`TAURI_SIGNING_PRIVATE_KEY_PASSWORD` 必填**，见 `.env.production`，模板/说明见入库的 `.env.production.example`）。私钥字符串（base64）配置为 GitHub secret `TAURI_SIGNING_PRIVATE_KEY`，密码配置为 `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`（CI 读不到本地 `.env.production`，两个都必须配）；公钥已写入 `tauri.conf.json` 并与密钥验签匹配。注意：key 与 password 务必备份，丢失将无法再签名更新。

## 6. 安全要点

- CSP 收紧到 `'self'`；生产构建禁用 devtools（Cargo feature 控制）与右键菜单。
- token 仅存 store 文件，不进日志；Rust 侧不接触业务数据。
- 所有外部 URL 打开前做协议白名单校验（仅 `https:`、`mailto:`、已知的客户端 scheme）。
- 依赖审计：规划 CI 跑 `cargo audit` 与 `pnpm audit`，暂未串入流水线（CI 现状见 §7）。

## 7. 构建与发布流水线（GitHub Actions）

> `.github/workflows/ci.yml` 已含 `frontend-quality` + `frontend-e2e` + `rust`（cargo check）三个 job；Go 后端按项目决策不走 Actions。`.github/workflows/release-tauri.yml` 已配置公开产物仓库方案的 **Windows-only** Release（仅要求 Windows 打包）。

- **PR/push CI**：仅发布 tag(`v*`) 触发：`frontend-quality`（lint→typecheck→format:check→test→build:web）+ `frontend-e2e`（Playwright · Mock，失败上传报告）+ `rust`（windows-latest，先 `pnpm build:web` 生成 dist → `cargo check`——打包仅支持 Windows，Rust 检查与打包平台一致）。日常开发检查走本地 `pnpm lint/typecheck/test/format:check`。Go 后端不走 GitHub Actions。
- **Release（已配置，打 tag `v*` 或手动触发）**：`.github/workflows/release-tauri.yml`——guard（tag 格式校验）→ 单平台构建（windows-latest `pnpm tauri build --bundles nsis`，TAURI_SIGNING_PRIVATE_KEY 自动签名 .sig）→ `scripts/build-latest-json.mjs` 合并 `latest.json`（资产 url 加 `gh-proxy.com` 前缀）→ `gh release` 推送到**公开产物仓库**（`RELEASES_PAT` secret）。恢复 macOS/Linux 时往 build job 加回平台矩阵即可（文件内有注释）。
- **产物仓库**：代码仓库 `superMC5657/YLink`（private，不开源）；产物发布到公开仓库 `superMC5657/ylink-releases` 保证匿名下载 + gh-proxy 加速可行（publish job 与 `tauri.conf.json` updater endpoints 均已指向该仓库）。
- **macOS 公证**（可选二期）：Apple 证书 + notarize 步骤；无证书时文档注明用户需在「安全性与隐私」中放行。⚠ 当前只打包 Windows，此条暂不适用，恢复 macOS 打包时再启用。
- 版本号策略：语义化版本，`package.json`、`Cargo.toml`、`tauri.conf.json` 三处由脚本同步。

## 8. 与纯 Web 部署的关系

- `pnpm build` 产物即完整 Web 版，可独立部署到任意静态托管（Nginx/Caddy/对象存储 + CDN）。
- Web 版隐藏桌面专属入口（托盘/自启/更新卡片），其余功能一致；一键导入、复制等能力自动降级（见 [data-layer.md](data-layer.md)）。
- Web 部署需后端开启 CORS（允许 Web 域名），Tauri 版无此要求。

## 9. 移动端策略（Tauri 2 Mobile）

手机端体验仍由响应式 Web 完整承载（见 [design-system.md](design-system.md) 第 6 节）；同时已启用 **Tauri Android APK 打包**，把同一套 Vue SPA 包装为 Android 应用，作为 Web 的补充分发渠道（iOS 仍不打包：需要 Apple 开发者账号与审核，代理类应用上架风险高）。

### 9.1 前置环境

| 依赖 | 版本要求 | 本项目现状 |
|---|---|---|
| JDK | 17+（AGP 8 要求） | JDK 21 ✅ |
| Android SDK | platform + build-tools | `E:\envs\android_sdk`（ANDROID_HOME）✅ |
| Android NDK | 25+ | 29.0.14206865 ✅ |
| Rust Android target | `aarch64-linux-android` 等 4 个 | `rustup target add aarch64-linux-android armv7-linux-androideabi i686-linux-android x86_64-linux-android` ✅ |

> 网络受限环境的处理（本机已配置，`src-tauri/gen/android/` 内的构建脚本带注释）：`services.gradle.org` / `plugins.gradle.org` 直连会 TLS 握手失败，`gradle/wrapper/gradle-wrapper.properties` 的 `distributionUrl` 已改指腾讯云镜像，`settings.gradle`、`buildSrc/`、`build.gradle.kts` 的仓库均改为阿里云镜像优先（官方源回退）。若在无此问题的网络重建（重新 `tauri android init`），可还原为官方源。

### 9.2 工程与命令

- `tauri android init` 已执行，生成 `src-tauri/gen/android/`（自动生成物，不入库，gitignore 已含 `src-tauri/gen/`）。
- ⚠️ **图标不同步机制**：`tauri android init` 生成的 gradle 工程自带 tauri 模板默认图标（`gen/android/app/src/main/res/mipmap-*/`），**不会**从 `src-tauri/icons/android/` 拷贝。改品牌图标后必须手动同步（否则 APK 封面仍是默认图标）：
  ```bash
  for d in hdpi mdpi xhdpi xxhdpi xxxhdpi; do
    for f in ic_launcher.png ic_launcher_round.png ic_launcher_foreground.png; do
      cp "src-tauri/icons/android/mipmap-$d/$f" "src-tauri/gen/android/app/src/main/res/mipmap-$d/$f"
    done
  done
  ```
- ⚠️ **adaptive icon 背景色**：`tauri icon` 生成的 `icons/android/values/ic_launcher_background.xml` 默认是白色 `#fff`（白底白闪电会看不见），需手改为品牌主色 `#6558F5`；同时 `gen/android/app/src/main/res/` 必须存在 `mipmap-anydpi-v26/ic_launcher.xml` + `ic_launcher_round.xml`（引用 `@mipmap/ic_launcher_foreground` + `@color/ic_launcher_background`）与 `values/ic_launcher_background.xml`，并在 `AndroidManifest.xml` 声明 `android:roundIcon="@mipmap/ic_launcher_round"`。缺 anydpi 定义时 Android 8+ 按 legacy 图标处理，launcher 套 mask 缩放导致图标明显偏小。
- npm scripts（见 `package.json`）：

| 命令 | 说明 |
|---|---|
| `pnpm tauri:android:dev` | 真机/模拟器热更新调试（需 `adb` 已连接设备） |
| `pnpm tauri:android:build` | 构建 release APK（默认构建全部 4 个 ABI；`--target aarch64` 可只打 arm64） |
| `pnpm tauri:android:build:aab` | 构建 AAB（上架 Google Play 用） |

产物在 `src-tauri/gen/android/app/build/outputs/apk/`（或 `.../bundle/`）。

### 9.3 平台差异与适配

- **Rust 侧桌面专属能力已用 `#[cfg(desktop)]` 隔离**（见 `src-tauri/src/lib.rs`）：托盘/菜单、单实例、开机自启、窗口状态记忆。移动端构建自动排除，不影响桌面行为。深链接插件注册在 desktop-only 之外（桌面 + 移动端都注册，Android 经 manifest intent-filter 接收 `ylink://`）。
- **移动端入口**：`run()` 标注 `#[cfg_attr(mobile, tauri::mobile_entry_point)]`，由 `MainActivity` 经 JNI 调用；桌面入口仍在 `main.rs`。
- **前端能力适配**：`utils/platform.ts` 已抽象，`isTauri()` 分支覆盖剪贴板/打开链接/深链接；桌面专属入口（托盘/自启/更新卡片）在移动端自动隐藏。`set_window_theme` 在移动端为 no-op（无标题栏）。
- **后端 API 走 `tauri-plugin-http`**（原生栈），Android 无需 CORS；manifest 已含 `INTERNET` 权限，debug 构建允许明文流量（`usesCleartextTraffic`），release 关闭。

### 9.4 Release 签名（上架必需）

`src-tauri/gen/android/app/build.gradle.kts` 已配置签名读取逻辑，缺省时 release 构建产出未签名 APK。**本项目已配置完毕**：`src-tauri/gen/android/keystore.properties` 已存在（`keyAlias=upload`；storeFile 路径与密码只存于该文件，不入库），`pnpm tauri:android:build` 产出的 release APK 已自动签名（验证命令：`apksigner verify --print-certs app-universal-release.apk`）。

> ⚠️ 若在 CI/其他机器重建：keystore 文件与 `keystore.properties` 均在 `gen/`（不入库），需自行恢复。重新配置格式（Windows 路径**必须双反斜杠**，否则 Properties 会把 `\e` 等当转义符、路径被吞成 `E:envs...`）：
> ```
> keyAlias=<别名>
> password=<密钥库密码>
> storeFile=E:\\path\\to\\ylink.jks
> ```
> 生成命令：`keytool -genkeypair -v -keystore <path> -alias upload -keyalg RSA -keysize 2048 -validity 10000`（生成时需输入密码，勿用空密码）。

### 9.5 已知边界

- identifier 为 `com.ylink.app`（以 `.app` 结尾会触发 macOS 分发警告，与 Android 无关，暂不改动）。
- 通知插件在 Android 12+ 需运行时权限，由系统授权弹窗处理（能力降级逻辑在前端）。
- 深链接插件（`ylink://`）**已接入 Android manifest**：`tauri.conf.json` `plugins.deep-link.mobile` 配置 scheme `ylink`，AndroidManifest.xml 增加 VIEW/BROWSABLE intent-filter。注意 `src-tauri/gen/` 不入库，`tauri android init` 重建后需重新手改 manifest；iOS 需在 Info.plist 配置 `CFBundleURLTypes`（项目未打包 iOS）。
