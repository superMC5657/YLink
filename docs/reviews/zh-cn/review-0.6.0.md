# 代码评审 — proxy-seller-web v0.6.0（Tauri 存储初始化与迁移）

- **版本：** 0.6.0
- **日期：** 2026-08-13
- **范围：** Tauri `plugin-store` 存储迁移及持久化 API 地址初始化。
- **方法：** 评审模型对存储后端迁移的审查。
- **状态：** 全部问题已解决（P1、P2 于 2026-08-13 修复）。

## 摘要

本次将存储后端改为通过同步内存 facade 使用 Tauri `plugin-store`。但 API 客户端模块级的 `API_BASE` 在 `bootstrap()` 等待 `initStorage()` 之前就已解析，导致本次会话不会使用已持久化的 API 地址。与此同时，现有 WebView `localStorage` 数据没有迁移到 `app-settings.json`，升级后可能导致用户退出登录并丢失已保存设置。

## 已完成

~~引入 Tauri `plugin-store` 后端，通过同步内存 facade 和异步落盘实现存储适配。~~

~~修复 P1：`API_BASE` 改为每次请求时惰性解析（`src/utils/http.ts` 的 `resolveApiBase()`），不再于模块导入时求值，`bootstrap()` 完成 `initStorage()` 后持久化的自定义 `apiBase` 即可生效。~~

~~修复 P2：`initStorage()`（`src/utils/storage.ts`）预载 `app-settings.json` 后执行一次性迁移，将 WebView `localStorage` 中残留的 `app:` 前缀键导入 plugin-store（已有同键不覆盖，`app:_legacy:migrated:v1` 标记保证只迁移一次）。~~

## 发现

### [P1] 在解析 API 地址前初始化存储 — src/utils/storage.ts:35-39

**状态：** 已修复（2026-08-13）。`API_BASE` 替换为惰性求值的 `resolveApiBase()`，在请求时（`initStorage()` 执行后）读取持久化的 `apiBase`；主请求路径与 401 静默刷新路径（`/auth/refresh`）均使用它。由 `src/utils/__tests__/http.spec.ts` 覆盖（持久化自定义 apiBase 用于请求与刷新）。

### [P2] 将已有 WebView 存储迁移到 plugin-store — src/utils/storage.ts:38-43

迁移前，Tauri 用户的 token 和设置保存在 WebView `localStorage` 中。新的初始化逻辑只读取 `app-settings.json`，没有导入这些已有 key。现有用户首次升级后可能丢失登录会话和持久化设置。

**状态：** 已修复（2026-08-13）。`initStorage()` 预载 `app-settings.json` 后调用 `migrateLegacyLocalStorage()`：WebView `localStorage` 中残留的每个 `app:` 前缀键都会导入 plugin-store（经异步 facade 落盘），除非该键已存在；`app:_legacy:migrated:v1` 标记保证迁移一次性。由 `src/utils/__tests__/storage.spec.ts` 覆盖。

## 验证

- 评审结果：发现 1 项 P1、1 项 P2。
- 本记录未包含代码修复。
- **2026-08-13：** 两项发现均已修复；`npm run typecheck` / `npm run lint` 全绿，vitest 51/51 通过（含新增 `http.spec.ts` 惰性 apiBase 用例与 `storage.spec.ts` 迁移用例）。
