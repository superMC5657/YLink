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

- 无明文长效 token 落盘；订阅不再全文裸露；登录/登出/重置回归通过。

## Comments
