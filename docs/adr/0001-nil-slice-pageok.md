# ADR-001 · 空列表的空值收敛位置：后端分页走 PageOK，前端 groups 保留兜底

> 状态：已接受（2026-09-15，clean-architecture 补遗轮）
> 相关：`.scratch/clean-architecture/spec.md` §2「流量重置列表直接赋值」

## 背景

Go 的 nil slice 经 `encoding/json` 序列化为 `null`，前端以 `list.length` 判空时对 `null` 抛 TypeError（列表区转圈卡死）。
历史做法是前后端各兜一层：后端 `resp.PageOK` 统一 `nil → []`，前端各调用点再 `?? []`。

## 决定

1. **分页列表**：以后端 `server/internal/pkg/resp/resp.go PageOK` 为唯一保证，前端直接赋值，不再 `?? []`（首例：`AdminTrafficImportView.vue loadLogs`）。
2. **groups 分组路径**（`GET /knowledges`、`GET /servers`）：走 `resp.OK(gin.H{"groups": data})`，不在 `PageOK` 覆盖内，且 service 可返 nil（如空库、无订阅），前端 `?? []` **保留**，定性为健壮性而非历史包袱。

## 后果

- 新增分页端点必须走 `PageOK`，否则前端裸奔；新增 groups 类端点，后端保证非 nil 之前前端不得去兜底。
- 若未来把 groups 也收敛到统一 helper，本 ADR 随之更新，前端兜底才可删。
