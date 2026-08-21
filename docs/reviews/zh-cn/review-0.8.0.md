# 代码评审 — YLink v0.8.0（移动端分享面板 & 工单已回复本地通知增强）

- **版本：** 0.8.0
- **应用版本：** v0.4.1(git 标签)。评审文档按评审周期编号(0.2.0 → 0.8.0),与应用发布版本相互独立,两套版本号无对应关系。
- **日期：** 2026-08-14
- **范围：** 新增移动端优先分享面板（`SharePanel.vue`）并接入邀请页；「工单已回复」桌面端本地通知增强（窗口聚焦/可见时立即检查）。
- **方法：** 评审模型审查 diff；`pnpm typecheck`、`pnpm lint`、`pnpm format:check`、Vitest（54/54 全绿，新增 3 个 SharePanel 用例）均通过。
- **状态：** 第一~七轮发现已全部解决(第七轮由第八轮修复,2026-08-22)。

## 摘要

二期待办中的「移动端分享面板」已实现：~~可复用底部弹层分享面板（`SharePanel.vue`，`n-drawer placement="bottom"`，移动端优先）~~ —— 渲染二维码（复用既有 `qrcode` 依赖，颜色取 CSS 变量与 `PaymentModal` 同款）、展示可分享链接、复制链接 + ~~系统分享（`navigator.share` 能力检测，Tauri 下自动隐藏）~~（第六轮起改为浮动 panel + 下载图片，见下文）。邀请页新增「分享」按钮，以注册链接（前缀 + 首个邀请码）打开面板。「工单已回复」本地通知（此前已由 `useLocalNotifications.checkTickets` + MainLayout 60s 轮询实现）增强为 `onFocus`/`onVisibility` 时也执行 `checkTickets()`，用户切回窗口不再需要最多等 60s。文档已同步（前后端 progress、候补项改为完成）。

## 变更

- `src/components/business/SharePanel.vue`（新增）：props `show` / `title` / `text` / 可选 `desc`，v-model `update:show`；二维码生成监听 `show`/`text`/重试计数；失败态带重试；复制走 `copyText`；~~系统分享仅 `navigator.share` 存在时显示~~（第六轮起改为下载图片，见下文）。
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
- `src/components/business/SharePanel.vue`:重构为品牌邀请卡片(品牌渐变 + 站点名 + 邀请码 + 白块二维码,二维码固定深色码保证暗色主题下可扫码);~~系统分享优先把二维码作为 `File` 分享(`navigator.canShare({ files })` 能力检测,不支持退回文本分享)~~(第六轮起改为下载图片,见下文)。
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

---

## 第五轮(2026-08-14):`register_url_prefix` 仅返回路径后缀

评审反馈:后端字段此前仍返回完整 URL(`http://localhost:8081/register?code=…`,由 `cfg.App.BaseURL` 拼接 —— 是 API 地址而非前端站点地址)。既然前端已按自身 origin 拼前缀,完整 URL 只会误导,字段应只携带路径后缀。

### 变更

- `server/internal/service/invite_service.go`:`register_url_prefix` 改为返回常量 `/#/register?code=`(仅路径后缀、不含域名);注释说明 API 地址 ≠ 前端站点地址,完整链接由前端 `effectiveRegisterUrlPrefix` 拼接。
- `server/internal/model/dto_invite.go`:字段注明为契约占位(仅后缀,前端不消费其值)。
- `src/stores/__tests__/invite.spec.ts`:删除误导性的死赋值 `registerUrlPrefix = 'http://localhost:8081/register?code='`(getter 从不读该字段);测试改为直接断言 jsdom origin 拼接;fetchCodes mock 更新为后缀形式。
- `mock/business.ts`:`register_url_prefix` mock 更新为 `/#/register?code=`。
- `docs/api/README.md`:契约示例改为 `/#/register?code=`,注明完整前缀由前端拼接。

### 验证(第五轮)

- 后端:`go test ./internal/service/ ./internal/model/ ./internal/middleware/` 通过。
- 前端:`pnpm test` invite + SharePanel 用例全绿;`pnpm typecheck` 通过。

---

## 第六轮(2026-08-14):分享面板改为浮动 panel,系统分享改为下载图片

使用反馈:分享面板不要从底部弹出(bottom sheet),改为浮动在网站窗口之上的 panel;「系统分享」改名为「下载图片」,把紫色邀请卡片(站点名 + 邀请码 + 二维码)生成图片供用户下载,纯前端实现。

### 变更

- `src/components/business/SharePanel.vue`:`n-drawer placement="bottom"` 底部弹层 → `n-modal` preset=card 居中浮动 panel(悬浮于窗口之上,宽度 `min(92vw, 30rem)` 自适应);移除 `navigator.share` 系统分享(含 `canShare({ files })` 能力检测),新增「下载图片」按钮 —— 用 canvas 合成紫色邀请卡片 PNG(720×940:渐变背景 + 站点名 + 邀请码 + 白色圆角二维码块 + 提示语 + 注册链接;圆角用手写 `roundRectPath` 兼容旧环境,文本超宽自动降字号),`canvas.toBlob` + `<a download="ylink-invite-{code}.png">` 触发下载,无后端依赖。
- `src/locales/{zh-CN,en-US}.ts`:`share.systemShare` → `share.downloadImage`(下载图片 / Download image),新增 `share.downloadFailed`(图片生成失败,请重试)。
- `src/components/business/__tests__/SharePanel.spec.ts`:stub 由 `n-drawer` 改为 `n-modal`;mock `qrcode` 返回假 dataURL(jsdom 无 canvas);新增下载图片用例(按钮渲染 + 画布不可用提示失败 + 画布可用合成 PNG 触发下载、文件名含邀请码、revokeObjectURL 延迟 1s 回收 + 超宽链接画布截断加省略号)。

### 发现

### ✅ [P3] jsdom 无 canvas 2D 上下文,下载分支需可测 — SharePanel.spec.ts
jsdom 的 `HTMLCanvasElement.getContext` 返回 null(未实现),`qrcode.toDataURL` 亦依赖 canvas。已处理:测试 mock `qrcode` 返回假 dataURL 让二维码分支可达;失败分支 `spyOn(getContext).mockReturnValue(null)` 断言 `downloadFailed` 提示;成功分支 stub `Image`/`getContext`/`toBlob`/`URL.createObjectURL` 断言触发下载且文件名含邀请码;超宽用例 stub `measureText` 按字符数计宽,断言 `fillText` 收到以省略号结尾的截断文本。

### ✅ [P2] canvas 居中文本未设置 textAlign,整体偏右 — SharePanel.vue
`drawCenteredText` 未设 `ctx.textAlign`,默认 `start` 导致 `fillText(IMG_W/2, y)` 实际从画布中线左对齐绘制,全部文本偏右、长文本越界。已修复:`textAlign='center'` 绘制后还原为原值;最小字号仍超宽时按可容纳宽度截断并追加省略号(最差退化为仅省略号),避免溢出画布。

### ✅ [P3] 下载后立即 revokeObjectURL,Safari/iOS 会中止下载 — SharePanel.vue
`a.click()` 后立即 `URL.revokeObjectURL(url)` 在 Safari/iOS 已知会中止进行中的下载。已修复:延迟 1000ms 回收,并在 `a.click()` 前 `appendChild` 到 DOM、点击后 `remove`(旧版 Safari 要求 anchor 挂载在 DOM 中才能触发下载)。

### 验证(第六轮)

- `pnpm test`:58/58 全绿(9 个文件;SharePanel 6 用例)。
- `pnpm typecheck` / `pnpm lint` / `pnpm build` 通过;Playwright e2e(desktop/mobile)14/14 通过。

---

## 第七轮(2026-08-22):`1ccacb8...HEAD` 全范围双轴评审

对 `99016a7..88b8656` 六个提交(移动端分享面板与前缀自适应、`register_url_prefix` 仅返回后缀、浮动 panel + 下载图片、文档同步、Swagger 网关兜底、版本号)做整体评审,以两个并行评审轴进行:**标准轴**(仓库成文规范 —— `docs/frontend/design-system.md`、`docs/frontend/README.md` §8、`docs/README.md` 架构约定、API 契约 —— 叠加 Fowler 代码坏味基线)与**规格轴**(本文档第一~六轮、`docs/frontend/progress.md` / `docs/backend/progress.md`、提交意图)。两轴分开汇报,互不遮蔽。以下发现已在第八轮全部修复(见下)。工作区未提交的 `src-tauri/tauri.conf.json` 改动不在本次评审范围内。

### 标准轴发现(第八轮已修复)

- **✅ [P2] 业务代码硬编码品牌渐变 — SharePanel.vue。** 模板 `style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"` 违反 design-system.md(「所有视觉决策以 CSS 变量令牌落地,业务代码不写死任何颜色」)及 frontend/README.md §8。design-system §2.3 虽认可该渐变,但同时规定了暗色变体(`#7C72FF → #A78BFA`)—— 字面量无法切换,暗色模式仍显示浅色卡片。
- **✅ [P3] canvas 颜色字面量**(`#6558f5`、`#8b5cf6`、`#FFFFFF`、`QR_DARK = '#1F2430'`):判断性 —— 导出 PNG 本就刻意与主题无关,且 canvas 不经 `getComputedStyle` 读不到 CSS 变量。
- **✅ [P3] SharePanel 头部注释**未标注对应截图页面/契约接口(frontend/README.md §8)。判断性 —— 新通用组件。
- **✅ [P3] 重复代码** —— 渐变色值同时出现在模板 style 与 canvas `addColorStop`;应抽共享常量。
- **✅ [P3] 重复代码** —— `InviteView.vue` 三处拼接 `prefix + codes[0]`(`registerLinkText` computed、`copyRegisterLink`、模板插值);应复用 computed。
- **✅ [P3] 死守卫 / 疑似过度设计** —— `openShare` 与 `copyRegisterLink` 检查 `!effectiveRegisterUrlPrefix`,但 getter 永不返回假值(末尾 `return '/#/register?code='`);应删除该判断。
- **✅ [P3] 重复代码** —— `server/docker-compose.dev.yml` 中 api 与 worker 的 `environment` 块完全相同(可用 YAML anchor 去重);前缀 getter 内 `'/#/register?code='` 字面量出现 3 次。
- **✅ [P3] 重复代码(既有问题,本次扩散)** —— 第一轮给 `MainLayout.vue` 的 `onFocus`/`onVisibility` 都加上 `checkTickets()` 后,两函数体完全相同;可抽一个 `onVisible()`。

### 规格轴发现(第八轮已修复)

- **✅ [P2] Tauri 兜底在 Windows 上永不生效 — invite.ts:37。** `if (origin && !origin.startsWith('tauri:'))` 假设 Tauri origin 以 `tauri:` 开头,但 Windows Tauri(唯一打包平台,见 progress.md)origin 为 `http://tauri.localhost`,能通过该判断。未配置 `VITE_WEB_BASE_URL` 的打包版会生成不可用的 `http://tauri.localhost/#/register?code=` 链接,而非相对路径兜底。
- **✅ [P2] `.env.production` 说明不在仓库内。** 第二/三轮声称「`.env.production`:已补充 `VITE_WEB_BASE_URL` 说明」,但该文件被 gitignore(`.gitignore:32`)且不在 diff 中 —— 打包要求只存在于本地未跟踪文件,新克隆什么都拿不到。应把指引落进受跟踪文档(deploy.md / 前端 README)或提供 `.env.production.example`。
- **✅ [P3] Swagger 兜底漏掉不带尾斜杠的 `/swagger`。** Caddyfile `@swagger path /swagger/*` 不匹配 `/swagger`,该路径仍被代理。轻微:deploy.md 验证 curl 用的是 `/swagger/index.html`,已覆盖。
- **✅ [P3] 评审记录不完整。** backend/progress.md 记录了 dev-docker.sh 重写、MSYS 路径转换修复、`docker-compose.dev.yml`,但本文档没有对应轮次。
- **✅ [P3] 超范围。** dev-docker.sh 全面重写(`require_env`、REDIS/APP_REDIS 一致性检查、MSYS 守卫)+ `server/.env.example` 新增 DEMO_EMAIL/DEMO_PASSWORD,超出 99016a7 标题所述范围与本文档记载;仅在 progress.md 事后补记。
- **✅ [P3] 版本号体系不一致。** 88b8656 将 package.json/Cargo.toml/tauri.conf.json 升到 0.4.1(各文件互相一致),但本周期文档标题均为 v0.8.0 —— 代码与文档版本体系不一致。
- **✅ [P3] 死守卫** —— 与标准轴同一条:`openShare` 的 `!effectiveRegisterUrlPrefix` 恒为假。

### 验证通过项

Caddyfile 拒绝 `/swagger/*` + deploy.md 404 curl 步骤;MainLayout 聚焦/可见时调用 `checkTickets()`;CORS https 白名单 + 5 用例 `cors_test.go`;`register_url_prefix` = `/#/register?code=` 在 service/dto/mock/api README 一致;SharePanel 浮动 `n-modal` + canvas PNG 下载含 Safari/iOS revoke 处理;新增文案全部走 i18n(中英);测试数量与文档声称一致(前端 58、SharePanel 6、后端 71)。

### 小结

标准轴 8 项(1 项硬性违规、7 项判断性)—— 最重:硬编码渐变违反令牌规则,暗色模式拿不到暗色变体。规格轴 7 项(3 项缺失/部分、1 项超范围、3 项实现可疑)—— 最重:Windows 打包版注册链接兜底失效,唯一打包平台受影响。

---

## 第八轮(2026-08-22):第七轮发现全部修复;补记 dev-docker 重写

第七轮 15 项发现全部修复(2×P2、13×P3;「死守卫」两轴各计一次)。同时在此补记此前缺少轮次记录的 dev-docker.sh 重写。

### 变更

- `src/styles/tokens.css`:新增 `--c-primary-grad-end` 令牌(浅色 `#8b5cf6` / 暗色 `#a78bfa`,design-system §2.3)。SharePanel 卡片渐变改为 `linear-gradient(135deg, var(--c-primary), var(--c-primary-grad-end))`,暗色模式终于拿到 §2.3 的暗色变体;下载图片的 canvas 绘制时经 `cssColor()`(getComputedStyle)读同一令牌(jsdom 下回退浅色值)。design-system.md §2.3 已注明令牌化写法。后续跟进(超出第七轮范围,本轮未做):同样的渐变字面量还存在于 6 个既有组件(AppSidebar、CustomerServiceFab、OrderCardList、OrderTable、SubscribeCard、UpdateCard),可择机迁移到令牌。
- `src/stores/invite.ts`:Tauri 兜底改用 `isTauri()`(`__TAURI_INTERNALS__`,utils/platform.ts)判定,不再匹配 `tauri:` origin 前缀 —— Windows 打包版(origin `http://tauri.localhost`)现在正确走相对路径 `/#/register?code=` 兜底,而不是生成不可访问的链接。路径后缀抽为 `REGISTER_URL_SUFFIX` 常量(原先 3 处内联)。+1 测试(mock isTauri 为 true → 相对兜底)。
- `src/views/invite/InviteView.vue`:展示改用新增 `registerLinkDisplay` computed(保留 `------` 占位);`copyRegisterLink` 复用 `registerLinkText`;删除 `openShare`/`copyRegisterLink` 里恒假的 `!effectiveRegisterUrlPrefix` 死守卫。
- `src/layouts/MainLayout.vue`:函数体完全相同的 `onFocus`/`onVisibility` 合并为单一 `onWindowActive`,两个事件共用。
- `src/components/business/SharePanel.vue`:头部注释补注调用方(邀请页,截图4)与「无契约接口(纯展示组件)」。
- `server/deploy/Caddyfile` + `docs/backend/deploy.md`:swagger 匹配改为 `path /swagger /swagger/*`,裸路径同样拒绝;部署验证步骤补充裸路径 curl。
- `server/docker-compose.dev.yml`:api/worker 的 `environment` 用 YAML 锚点 `x-api-worker-env` 去重(锚字解析已验证)。
- `.env.production.example`(新增,入库):说明 `VITE_WEB_BASE_URL`(含 Tauri Windows origin 陷阱)与 Tauri 签名变量;`docs/frontend/README.md` 目录树、`desktop-tauri.md`、`progress.md` 改为引用它,不再只指向被 gitignore 的 `.env.production`。
- 版本体系已在文档头部注明:评审文档按评审周期编号,应用发布走 git 标签(当前 v0.4.1)。代码 0.4.1 是正确的 —— 本轮不改版本号。

### 补记:dev-docker.sh 重写(commit 99016a7,此前本文档无记录)

该提交还重写了 `scripts/dev-docker.sh`,超出提交标题所述:`.env.dev` 作为唯一 env 来源(`require_env`,不内置默认值)、REDIS/APP_REDIS 一致性检查、Windows Git Bash 的 MSYS 路径转换守卫(cygpath 导出 `DEV_CADDYFILE`/`DEV_DIST`),并在 `server/.env.example` 新增 `DEMO_EMAIL`/`DEMO_PASSWORD`(演示账号)。动机:旧脚本内置默认值会与 dev.sh 的 env 处理悄然分叉。该工作保留;本记录即关闭第七轮规格轴的「评审记录不完整」与「超范围」两项发现。

### 验证(第八轮)

- `pnpm test`:59/59 全绿(9 个文件;`invite.spec.ts` +1 Tauri 兜底用例)。
- `pnpm typecheck` / `pnpm lint` / `pnpm format:check`:通过。
- `server/docker-compose.dev.yml`:YAML 解析通过,锚点解析后 api/worker 环境完全一致(已验证)。
- 本轮未改服务端 Go 代码(仅 Caddyfile/compose/文档),故未重跑 `go test`。
