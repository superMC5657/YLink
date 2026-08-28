# ✅ NSIS 默认安装位置改为 AppData\Local\Programs

- **需求**：Tauri 2（YLink）安装器的默认安装位置由 `C:\Users\<user>\AppData\Local\YLink` 改为 `C:\Users\<user>\AppData\Local\Programs\YLink`。
- **状态**：✅ 已完成（2026-02）
- **影响面**：仅 NSIS 安装器（`bundle.targets = ["nsis"]`），`installMode: "currentUser"` 分支的**全新安装**默认路径。

## 背景与约束

- Tauri 的 NSIS 配置没有"默认安装目录"选项；默认路径硬编码在 tauri-bundler 内置模板 `installer.nsi` 的 `.onInit` 中：
  `StrCpy $INSTDIR "$LOCALAPPDATA\${PRODUCTNAME}"`（tauri-cli v2.11.4 第 514 行）。
- `bundle.windows.nsis.installerHooks` 的四个 hook（PRE/POSTINSTALL、PRE/POSTUNINSTALL）都插入在 `.onInit` 之后的 Section 里，**无法**在目录页显示前覆盖 `$INSTDIR`，所以 hook 方案不可行。
- 官方支持的途径是 `bundle.windows.nsis.template`：自定义模板，占位符必须与内置模板一致。

## 方案

1. 复制 tauri-cli **v2.11.4** 官方模板 → `src-tauri/nsis/installer.nsi`。
2. 唯一逻辑改动（currentUser 分支）：
   `StrCpy $INSTDIR "$LOCALAPPDATA\${PRODUCTNAME}"` → `StrCpy $INSTDIR "$LOCALAPPDATA\Programs\${PRODUCTNAME}"`。
3. `tauri.conf.json` 的 `bundle.windows.nsis` 增加 `"template": "nsis/installer.nsi"`。
4. 模板头部保留注释：来源版本、改动点、升级 tauri-cli 时需重新 diff 同步。

## 过程记录

- 从 GitHub（`tauri-cli-v2.11.4` tag，经 gh-proxy 镜像）拉取官方 `installer.nsi` 作基准。
- 与官方模板逐行 diff 确认：仅头部注释 + 一行路径改动，无其他差异。
- `pnpm tauri build` 全量打包验证（见下方验证结果），updater 产物（`.exe` + `.sig`）不受影响。

## 验证结果

- `pnpm tauri build` 全量构建成功：`YLink_0.6.0_x64-setup.exe` + updater 签名（`.sig`）正常生成，自定义模板占位符渲染与 makensis 编译通过。
- 一手实证：临时以 `--config` 覆盖 `bundle.windows.nsis.compression = "none"` 重新打包（solid lzma 会压缩字符串表导致不可搜索），对产物做 UTF-16LE 字节搜索——
  - 新路径字面量 `Programs\YLink` 命中 1 处，位于 `.onInit` 字符串区（紧邻 `EarlyChecks`，与模板结构吻合），即 `StrCpy $INSTDIR "$LOCALAPPDATA\Programs\YLink"` 的编译产物；
  - 旧路径字面量 `$LOCALAPPDATA + "\YLink"` 不再作为默认值出现。
- 验证后已用默认 lzma 压缩重新打包，`bundle/nsis` 下的正式产物与仓库配置一致；临时 config 与探针脚本均已清理。

## 行为说明（重要）

- **全新安装**：默认装到 `%LOCALAPPDATA%\Programs\YLink`。
- **已有安装升级**：模板逻辑 `RestorePreviousInstallLocation` 会从注册表 `HKCU\Software\com.ylink.app\YLink` 恢复旧安装路径，升级包继续装在旧位置（如 `AppData\Local\YLink`）。这是刻意的保守行为：避免强制搬家产生孤儿目录/卸载残留。如需强制迁移旧安装，另行设计（清注册表记录 + 迁移旧目录），不在本需求范围。

## 维护注意

- 升级 `@tauri-apps/cli` 时，内置模板可能变化：重新从对应 tag 拉取官方 `installer.nsi`，与本文件 diff 后重新套用"currentUser 分支路径"改动，保持其余部分与官方一致。
