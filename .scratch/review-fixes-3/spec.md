# Review Fixes 3

> 状态: 修复中——P0 完成 4/4；P1 完成 10/10（05-13）；P2 完成 2/5（14、15），余 P2×3 / P3×3
> 日期: 2026-10-25
> 范围: 全量代码评审（review-round-09）产出的修复项——安全与启动稳定、并发/正确性、分层与契约合规、令牌化与文档对齐、死代码清理
> 关联: 评审快照 `docs/reviews/review-round-09.md`；分段执行手册 `execution.md`（一次贴一段）；前两轮见 `.scratch/review-fixes/`、`.scratch/review-fixes-2/`

## Requirements

### [P0] 01 生产配置不得静默使用开发密钥 ✅

config.yaml 开发密钥改为 `${ENV}` 占位；config.Load 在生产模式对已知的开发默认值（JWT secret/DB/Redis 密码）fail-fast 拒绝启动；worker 日志键勘误。→ issues/01

### [P0] 02 种子账号必须携带订阅凭证 UUID ✅

EnsureAdmin/EnsureDemoUser 填充 `users.uuid`，消除唯一约束冲突与模式 A 归因缺失。→ issues/02

### [P0] 03 探针与 Webhook 加固 ✅

`/readyz` 的 DB ping 纳入超时上下文、不向公网泄漏内部错误细节；Telegram Webhook 先校验 secret 再 bind body（403 语义）。→ issues/03

### [P0] 04 cron 分布式锁修复 ✅

SetNX 错误与锁竞争分路径处理（Redis 故障不再静默跳过）；锁删除带属主 token 防误删继任者。→ issues/04

### [P1] 05 改密/重置订阅不得覆盖并发写 ✅

user_service/subscribe_service 的全结构体陈旧快照更新改为按列条件更新（沿用 telegram.go 范式）。→ issues/05

### [P1] 06 工单回复与流量导入的并发竞态 ✅

ReplyTicket 条件更新守卫（reopen-once 状态机）；TrafficLogRepo.Upsert 改 OnConflict；UpdateUser 包事务。→ issues/06

### [P1] 07 异步投递与密码学卫生 ✅

验证码 GetDel、issueSession 不忽略 Set 错误、rand.Int 错误传播、异步邮件 recover、mailer 超时、epay 签名常量时间比较。→ issues/07

### [P1] 08 迁移回滚对称性 ✅

0007/0008 down 迁移修复（提现行残留、settings.telegram 种子行清理）。→ issues/08

### [P1] 09 审计分页档位钳制正确 ✅

clampPageSize 支持 Standard/Audit 双档（ADR-0004），审计页选 100/页真实生效。→ issues/09

### [P1] 10 管理端变更后刷新不得产生未处理拒绝 ✅

`void load()` 全部改为非抛出的 reload 语义。→ issues/10

### [P1] 11 前端正确性批修 ✅

mock 安装命令模板对齐 0009 规范、cssVar 回落值对齐令牌、流量导入日期统一 localDateKey、支付方式配置化、缺失 icon key。→ issues/11

### [P1] 12 NodeInstallDialog 合规与 apiBase 复用 ✅

组件内 apiAdmin 直调下沉 useAdminNodes；apiBase 复用 resolveApiBase（尊重运行时覆盖）；文案全部 i18n 化。→ issues/12

### [P1] 13 Tauri 深链与更新链路可达 ✅

desktop schemes 注册（ylink:// 可用）、latest-cn.json 接入 updater endpoint、capabilities 最小授权、Cargo 冗余依赖清理。→ issues/13

### [P2] 14 品牌渐变全量令牌化 ✅

约 30 处写死渐变 + uno btn-primary + theme.ts base 色迁移至 CSS 变量；check-tokens-sync MAPPING 补键堵住漂移。→ issues/14

### [P2] 15 后端契约与规范合规批 ✅

批量审计 target 统一 `batch:<count>`（5 处）、IDParam 收敛（6 处）、models.go tag 对齐 data-model、agent_applies/used_count 补文档、metrics label 基数治理。→ issues/15

### [P2] 16 前端数据层合规批

ProfileView/useLocalNotifications 下沉 store、OrderConfirmModal silent:true、api seam 保留现状补 ADR-0006 并更新 data-layer.md、AppIcon 文档对齐内联 Lucide、api.d.ts 会话契约同步。→ issues/16

### [P2] 17 UI 规范批

useMediaQuery 替换冻结宽度、侧边栏宽度以代码 200px/72px 为准更新 pages.md、ForgotView 表单对齐、AdminNodesView 调色板类令牌化、提现状态接入 statusMeta、小组件规范项。→ issues/17

### [P2] 18 文档对齐批

README §3/launch-checklist/deploy/data-model/desktop-tauri 等全部陈旧描述改为当前态；Makefile swag 输出目录；compose prod caddy !reset；邀请分成默认值单源化。→ issues/18

### [P3] 19 后端死代码清理

零引用函数/字段/空壳结构体/pkg validate 整包删除（清单见 issue）。→ issues/19

### [P3] 20 后端重复形状收敛

repo paginate helper、stdlib contains/min、settings 解析单源、sort helper 三合一、N+1 批查、service GORM 绕过回 repo、缓存旁路、审计失败记日志、魔法数字常量化。→ issues/20

### [P3] 21 前端重复形状与死代码收敛

CRUD 脚手架工厂、排序弹窗/ErrorPage/useEmailCode/formatDateTime 单源化、mock envelope、i18n 键齐平测试、死代码删除（清单见 issue）。→ issues/21
