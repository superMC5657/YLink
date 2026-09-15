# ✅ 架构精简 · 清理过时设计与兼容性兜底

> 状态: ✅ 已完成
> 日期: 2026-09-14
> 范围: 订阅凭证每用户直发 / 移除存量共享密码兜底 / 移除老用户 UUID 懒生成 / 移除桌面端本地存储一次性迁移垫片 / 审计 target 单 ID 解析 / 会话列表严格解析 / 令牌 savedAt 必填 / 流量重置列表直接赋值

## 1. 目标

贯彻「架构简洁干净、不为过时设计做妥协与兼容兜底」的工程原则，彻底移除系统演进过程中引入的历史存量兼容分支与过渡垫片，保证代码与文档纯粹表达系统当前态。

## 2. 清理项

### ✅ 订阅节点凭证单一归一化
- 移除 `server/internal/service/subscribe_service.go` 中 `toNode` 对 `per_user_credentials` 的开关判断与 `conf.Password/UUID/ID` 兜底回退。
- 所有节点统一且强制下发每用户独立凭证 `users.uuid`，节点按凭证进行流量归因。
- 订阅测试去除存量共享凭证回退用例，测试基准锁定为每用户凭证。

### ✅ 移除老用户 UUID 懒生成补丁
- 移除 `SubscribeService.Generate` 中的 `if user.UUID == ""` 懒生成与写库逻辑。
- 系统新架构在注册及初始化环节确保 UUID 必定存在，下发链路不再保留防御性历史包袱。

### ✅ 移除桌面端 LocalStorage 迁移垫片
- 移除 `src/utils/storage.ts` 中的 `_legacy:migrated:v1` 标记、`migrateLegacyLocalStorage()` 与 `persistToStore()`。
- Tauri 环境直接依赖 `tauri-plugin-store` 预载与持久化，去除对浏览器 `window.localStorage` 的历史迁移扫描。
- 单元测试重构为对 Store 内存缓存、防抖落盘与降级行为的纯粹校验。

### ✅ 审计 target 单 ID 解析
- `server/internal/service/admin_audit.go` 中 `auditTargetIDs` 仅解析单个 ID（当前格式为 `fmt.Sprint(userID)`），非 ID 内容返回空。
- `send_mail` / `traffic_reset` 的 target 为摘要 `batch:<count>`，完整 ID 列表在 detail；不再解析 `[7 8 9]`（fmt.Sprint 切片）与 `7,8,9` 历史格式。

### ✅ 会话列表严格解析
- `server/internal/service/user_service.go` 中 `ListSessions` 要求白名单值为 `issueSession` 写入的 JSON `{ip,ua,ts}`；Redis 缺失或解析失败的条目直接跳过。
- 不再对历史版本白名单值（`"1"`）做未知设备降级展示；`UserSessionItem.CreatedAt` 恒有值（`server/internal/model/dto.go` 契约同步）。

### ✅ 令牌 savedAt 必填
- `src/utils/tokens.ts` 中 `restorePersisted` 将缺失 `savedAt` 视为过期脏数据：丢弃持久化并强制重新登录。
- 不再兼容无 `savedAt` 的旧格式数据恢复；`applyTokens` 写入恒带 `savedAt`。

### ✅ 流量重置列表直接赋值
- `src/views/admin/AdminTrafficImportView.vue` 中 `loadLogs` 直接赋值 `res.list`；空列表由后端 `resp.PageOK` 统一兜底为空数组 `[]`，nil 切片不透出 null。
- 知识库 / 节点分组走 `resp.OK` 的 groups 路径仍可为 null，其 `?? []` 作为健壮性保留，不在此列。

## 3. 验收标准

- [x] 后端测试 `go test ./...` 全绿
- [x] 前端测试 `pnpm test` 全绿
- [x] 前端代码检查 `pnpm lint` 与 `pnpm typecheck` 全绿
- [x] 相关文档同步为当前态，删除过时兼容描述
