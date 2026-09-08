# [P0] 07 Token持久面+订阅掩码

Status: ready-for-agent
Type: task

## Finding

- `stores/auth.ts:67-70 persist pick:[accessToken,refreshToken,user]` 经 `storage.ts:130` 明文落 localStorage/app-settings.json；XSS 即长效接管。
- `CopyText.vue:title=text` 全文进 DOM；订阅 URL 在 ImportClientSheet/Profile/AdminUsers 全文展示。

## Task

1. 缩小持久面：内存存 access，仅持久最小 refresh（或后续 httpOnly），加过期清理；登出/reset 后 `flushStorage+清剪贴板提示`。
2. `CopyText` 去 title 全文，改点击复制/显后4位；订阅卡默认掩码+显式显示原文；保留“重置即旧链失效”用户教育。
3. `config` 加 persist+TTL+SWR，`storage.ts saveQueue` 加 100ms 防抖。

## Done

- [x] 新 `utils/tokens.ts` 单一写入方收口 'app:auth' 键:access 仅内存不落盘,持久化只含 {refreshToken,user,savedAt};启动恢复仅取回 refresh(access 由首请求 401→静默刷新补回);savedAt 超 30d 启动时丢弃(过期清理);auth store 去 pinia persist(消除双写覆盖)。
- [x] `isLoggedIn = !!(accessToken || refreshToken)`:刷新后守卫据持久化 refresh 放行,等首个请求补回 access。
- [x] 登出:`auth.logout` 末尾 `await flushStorage()`(防抖窗口写落盘,防 token 残留)+ 尽力清剪贴板。
- [x] `CopyText` 去 `:title="text"` 全文 tooltip;订阅链接三处(ImportClientSheet/Profile 重置弹窗/AdminUsers 重置成功弹窗)默认 `maskUrl` 掩码(协议+域名+••••+末4位)+ 显式「显示/隐藏」切换;AdminUsers 弹窗补上此前缺失的真实复制动作;Profile 保留「重置即旧链失效」教育文案。
- [x] `config` store 改 SWR:有缓存先落地,过期后台再验证(失败保旧值),TTL 60s 不变;`storage.ts` Tauri 写入改 100ms 防抖批量落盘(失败批次回滚重试),flushStorage 语义保留。
- [x] http 层会话过期判定兼容两种形态:HTTP 401 或 HTTP 200+业务码 40100 均走静默刷新(mock 以 200 包裹 40100,此前依赖 fixture 预置 access 而未暴露)。
- [x] 门禁:typecheck/lint + vitest 79(新增 tokens 5 例/maskUrl 2 例)+ E2E 双 project 56/56。

## Comments

- e2e fixture 同步改为新持久化格式 {refreshToken,user};authed 会话现真实走一遍 401→静默刷新链路,顺带覆盖刷新路径。
- 已知取舍:access 仅内存意味着每次冷启动多一次 refresh 往返;后续可迁移 httpOnly cookie 方案(服务端改造,超出本批次)。
