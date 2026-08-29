# 管理后台总览 · 全体用户余额统计

> 日期: 2026-08-30

## 1. 需求

管理后台总览页（`/admin/overview`）增加一张「用户余额总额」统计卡片，展示全体用户 `users.balance`（不含佣金余额）的合计金额（元）。

## 2. 方案

- **后端**：`GET /admin/stat/overview` 响应（`AdminOverviewResp`）新增 `total_balance`（float64，元，分→元经 `model.FenToYuan`）。`AdminService.Overview` 内联查询 `SELECT COALESCE(SUM(balance),0) FROM users`（与该 service 现有 Count/SUM 直连 `s.db` 的模式一致）。
- **前端**：
  - `AdminOverviewResp`（`src/types/api.d.ts`）新增 `total_balance: number`；
  - `AdminOverviewView.vue` 在「今日收入」后新增 `StatNumber` 卡片（icon=wallet，色 `--c-blue`，单位元）；
  - i18n 新增 `adminOverview.totalBalance`（zh: 用户余额总额 / en: Total User Balance）。

## 3. 范围说明

- 仅统计 `balance`，不包含 `commission_balance`（佣金余额另有口径，代理商佣金在佣金日志/提现工单中体现）。
- 只读统计，不新增接口、不改权限。

## 4. 验收

- [x] 后端编译通过（`go build ./...`）
- [x] 前端类型检查通过（`vue-tsc --noEmit`）
- [x] 契约文档同步（`docs/api/README.md` §16 仪表盘行标注 `total_balance`）

## 5. 实施记录（2026-08-30）

- ✅ 全部完成：`server/internal/model/dto_admin.go`、`server/internal/service/admin_service.go`、`src/types/api.d.ts`、`src/views/admin/AdminOverviewView.vue`、`src/locales/zh-CN.ts`、`src/locales/en-US.ts`、`docs/api/README.md`。
