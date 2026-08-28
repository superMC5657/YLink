# [P3] Sessions without metadata must return null created_at

Status: resolved
Type: task

## Finding

升级前 refresh 白名单值是字符串 `"1"`；`ListSessions` 元数据解析失败时 `CreatedAt` 保持 Go 零值 `time.Time` 并序列化为 `0001-01-01T00:00:00Z`。前端 `s.created_at ? formatTime(...) : '--'` 视其为有效，显示 0001/1/1。

## Resolution

`UserSessionItem.CreatedAt` 改为 `*time.Time`，无元数据保持 nil → JSON `null`；前端 `api.d.ts` 同步为 `string | null`（ProfileView 的 truthiness 判断天然兼容，显示「--」）。

## Comments

2026-08-28: 已修复；`TestListAndRevokeSessions` 补断言——历史会话（白名单值 "1"）`CreatedAt` 必须为 nil，有元数据会话非 nil。
