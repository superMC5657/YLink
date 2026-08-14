# 代码评审 — YLink v0.8.0（移动端分享面板 & 工单已回复本地通知增强）

- **版本：** 0.8.0
- **日期：** 2026-08-14
- **范围：** 新增移动端优先分享面板（`SharePanel.vue`）并接入邀请页；「工单已回复」桌面端本地通知增强（窗口聚焦/可见时立即检查）。
- **方法：** 评审模型审查 diff；`pnpm typecheck`、`pnpm lint`、`pnpm format:check`、Vitest（54/54 全绿，新增 3 个 SharePanel 用例）均通过。
- **状态：** 全部发现已解决（2026-08-14）。

## 摘要

二期待办中的「移动端分享面板」已实现：可复用底部弹层分享面板（`SharePanel.vue`，`n-drawer placement="bottom"`，移动端优先）——渲染二维码（复用既有 `qrcode` 依赖，颜色取 CSS 变量与 `PaymentModal` 同款）、展示可分享链接、复制链接 + 系统分享（`navigator.share` 能力检测，Tauri 下自动隐藏）。邀请页新增「分享」按钮，以注册链接（前缀 + 首个邀请码）打开面板。「工单已回复」本地通知（此前已由 `useLocalNotifications.checkTickets` + MainLayout 60s 轮询实现）增强为 `onFocus`/`onVisibility` 时也执行 `checkTickets()`，用户切回窗口不再需要最多等 60s。文档已同步（前后端 progress、候补项改为完成）。

## 变更

- `src/components/business/SharePanel.vue`（新增）：props `show` / `title` / `text` / 可选 `desc`，v-model `update:show`；二维码生成监听 `show`/`text`/重试计数；失败态带重试；复制走 `copyText`；系统分享仅 `navigator.share` 存在时显示。
- `src/views/invite/InviteView.vue`：注册链接行新增「分享」按钮 → 打开 `SharePanel`；无邀请码守卫改用 i18n key `invite.needCode`。
- `src/components/ui/AppIcon.vue`：新增 `share-2`（Lucide share-2 路径）。
- `src/locales/{zh-CN,en-US}.ts`：新增 `share.*` 段 + `invite.shareLink/shareDesc/needCode`。
- `src/layouts/MainLayout.vue`：`onFocus`/`onVisibility` 补调 `checkTickets()`。
- `src/components/business/__tests__/SharePanel.spec.ts`（新增）：3 个用例（标题/链接/操作渲染、复制 → `copyText` + 成功提示、关闭 → `update:show(false)`）。

## 发现

### ✅ [P3] 二维码失败态与「加载中」共用占位 — SharePanel.vue
`QRCode.toDataURL` 抛错（如无 canvas）时面板会一直显示「加载中」且无法恢复。已修复：拆出独立 `qrFailed` 失败态 + 「重试」按钮，经重试计数重新触发生成。

### ✅ [P3] 面板打开期间 `text` 变化不会重建二维码 — SharePanel.vue
watch 只监听 `show`，分享文本变化时会残留旧二维码。已修复：watch `[show, text, retryTick]`，空文本时清空。

### ✅ [P3] 邀请页硬编码中文提示 — InviteView.vue
`message.warning('请先生成邀请码')` 绕过 i18n。已修复：改用 `t('invite.needCode')`（中英文均已补充）。

## 验证

- `pnpm test`：54/54 全绿（9 个文件；既有 51 + 新增 3 个 SharePanel 用例）。
- `pnpm typecheck`：通过；`pnpm lint`：0 错误；`pnpm format:check`：全部 Prettier 合规。

---

## 第二轮(2026-08-14):注册链接前缀修复、卡片式面板、系统分享二维码图片

使用反馈:注册链接生成成了后端 API 地址(`http://localhost:8081/register?code=…`),因为服务端用 `cfg.App.BaseURL`(API 地址)拼 `register_url_prefix`,而非前端页面 origin。同时要求:二维码展示做成卡片式、系统分享改为分享二维码图片。

### 变更

- `src/stores/invite.ts`:新增 getter `effectiveRegisterUrlPrefix` —— ① 构建时注入 `VITE_WEB_BASE_URL`(生产 / Tauri 打包显式配置,见 `.env.production`);② 否则取 `window.location.origin`(自动区分本地 Vite dev `:5174`、Caddy `:80`、生产 HTTPS `:443`);③ 兜底相对路径 `/register?code=`(仅 Tauri 未注入时走到,不再泄露 API `8081` 地址)。
- `src/views/invite/InviteView.vue`:展示/复制/分享/守卫全部改用 effective 前缀;向面板传入 `:code`。
- `src/components/business/SharePanel.vue`:重构为品牌邀请卡片(品牌渐变 + 站点名 + 邀请码 + 白块二维码,二维码固定深色码保证暗色主题下可扫码);系统分享优先把二维码作为 `File` 分享(`navigator.canShare({ files })` 能力检测,不支持退回文本分享)。
- `.env.production`:补充 `VITE_WEB_BASE_URL` 说明。
- 测试:`invite.spec.ts` +1(effective 前缀 = jsdom 页面 origin);`SharePanel.spec.ts` 更新(pinia + 邀请码断言)。

### 发现

### ✅ [P3] 二维码颜色跟随主题,暗色下反色 — SharePanel.vue
暗色模式下 `--c-text` 解析为浅色,白块内出现浅色二维码,扫码可靠性下降。已修复:二维码 `dark`/`light` 固定(`#1F2430` / `#FFFFFF`),白块背景恒为白色。

### ✅ [P3] Tauri 兜底仍泄露 API `8081` 前缀 — invite.ts
未配置 `VITE_WEB_BASE_URL` 时,桌面打包版回退到后端拼接的前缀(`http://localhost:8081/register?code=`),原 bug 在桌面端残留。已修复:兜底改为相对路径 `/register?code=`;打包版必须配置 `VITE_WEB_BASE_URL`(见 `.env.production` 注释)。

### 验证(第二轮)

- `pnpm test`:55/55 全绿(9 个文件;+1 invite store 用例)。
- `pnpm typecheck` / `pnpm lint` / `pnpm format:check`:全部通过。

---

## 第三轮(2026-08-14):注册链接补 hash 路由段(`#/register`)

生成的分享链接缺少 hash 段 —— 应用使用 `createWebHashHistory()`,注册页实际 URL 为 `…/#/register?code=…`,而非 `…/register?code=…`。

### 变更

- `src/stores/invite.ts`:`effectiveRegisterUrlPrefix` 三条分支(注入 base / 页面 origin / 相对兜底)全部输出 `…/#/register?code=`。
- `server/internal/service/invite_service.go`:占位字段 `register_url_prefix` 同步为 `…/#/register?code=`(契约一致性;前端已不再消费该字段)。
- `docs/api/README.md`:契约示例改为 `https://panel.example.com/#/register?code=`,并注明前缀由前端按自身 origin 拼接。
- `.env.production`:注释补充 hash 路由链接形态。
- 测试:`invite.spec.ts` 期望值更新(`http://localhost:3000/#/register?code=`);`SharePanel.spec.ts` LINK 更新。

### 验证(第三轮)

- `pnpm test`:55/55 全绿;`pnpm typecheck` / `pnpm lint` / `pnpm format:check`:通过。
- 后端:`go test ./internal/service/` 通过。

---

## 第四轮(2026-08-14):CORS 支持 https 来源

CORS 白名单此前只含 `http` 变体,https 前端来源(Vite https / Caddy 本地 TLS / 生产 `https://panel.example.com`)会被浏览器预检拦截。

### 变更

- `server/configs/config.yaml`:白名单补充 `https://localhost` / `https://127.0.0.1` / `https://localhost:5174` / `https://localhost:1420`(注释说明本地 https 用法;生产用 `https://panel.example.com`,见 deploy.md)。
- `server/internal/middleware/cors_test.go`(新增):5 个用例 —— https 来源放行、http 来源放行、非白名单来源拒绝(不返回 `Access-Control-Allow-Origin`)、OPTIONS 预检返回 204 且放行头完整、无 Origin 请求不受影响。
- `docs/backend/deploy.md`:生产示例标注 https。

### 验证(第四轮)

- 后端:`go test ./internal/middleware/ -run TestCORS` 5/5 通过;`go test ./...` 全部通过。
