# ADR-0005 · ica-A facade 退役：管理端装配直取域 seam

> 状态：已接受（2026-10-12，dev-ica3）
> 相关：`server/internal/service/admin_modules.go`（`NewAdminSeams`）、
> `server/internal/handler/admin/admin.go`（`NewAdmin`）

## 背景

ica-A 建 `AdminService` facade 作迁移暂存点（70 个纯转发方法），注释明示
“便于存量调用方逐步迁移到各域 seam”。暂存点完成使命：全部调用方
（handler 装配 + 全量单测）已可直取域，facade 只剩转发；另有
`adminTelegramAdapter` 单调用者假设缝（late 注入本可直注）。

## 决定

1. **删 `admin_service.go`（facade）与 `admin_telegram.go`（adapter）**：
   装配唯一入口为 `NewAdminSeams`（签名与退役构造一致），返回 13 域 seam 总装；
   `admin.NewAdmin(seams, tgSvc)` 直取各域，telegram 由 router 直注
   `*service.TelegramService`（恒非空，原 nil 守卫随 adapter 同删，不可达）。
2. **域 seam（13 个）保留**：handler 各域文件 + 单测双重消费，非单调用者；
   仅 telegram 单调用缝折叠进属主（`TelegramService` 直调）。
3. **单测平移**：构造换 `NewAdminSeams`，调用按域点取（`svc.User.X` 形）；
   断言零改动——丢的只是转发层重测，域覆盖不少一行。

## 后果

- 新增管理端域能力：module + seam + `AdminSeams` 装配一行 + handler 取用，
  不再经 facade 中转；`AdminService`/`SetTelegram` 等名彻底消失（grep 为证）。
- 本 ADR 关闭 ica-A 迁移暂存点；后继不得以“过渡”为名重引入转发层。
