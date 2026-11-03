# [P1] NodeInstallDialog：数据层违规与 apiBase 漂移

Status: resolved
Type: task

## Finding

`src/components/admin/NodeInstallDialog.vue`：

1. `:38-42` 本地 `apiBase` 重实现 `http.ts:43 resolveApiBase()`，但忽略持久化的运行时 apiBase 覆盖（http.ts:38-41 文档化的 P1 陷阱）——设置页改过后端地址后，弹窗渲染的安装命令指向错误端点。
2. `:10,63,88` 组件直调 `apiAdmin.settings()/saveSetting`——data-layer.md §2 的 admin 豁免仅限 `views/admin/*`，组件应保持纯展示；兄弟 8 个 dialog 均经 useAdminNodes。
3. `:110` 复制失败 toast 硬编码中文 `'复制失败，请手动选择复制'`（§7 约定错误文案走 i18n）；`:143,169,216` 另有 3 处未翻译文案（未生成 / 💡 Ubuntu… / 可用占位符）。

## Plan

1. `resolveApiBase` 从 http.ts 导出，弹窗直接复用（含运行时覆盖优先级）；删除本地实现。
2. 设置读写动作下沉 `useAdminNodes`（loadSettings/saveSettings + 状态），弹窗经 props/emits 纯展示。
3. 4 处文案补 `locales/{zh,en}` 键并替换。

## Acceptance

- 运行时改 apiBase 后打开弹窗，安装命令端点随之变化（测试模拟 localStorage 覆盖）。
- 组件内不再 import `@/api/admin`；grep 校验。
- 中英文 locale 均无硬编码中文残留；`pnpm test` 全绿。
