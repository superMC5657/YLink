# CONTEXT.md · YLink 领域上下文（单上下文仓）

> 面向 Agent 与新成员的领域词汇与文档地图。词汇以此处为准，勿用同义词漂移；跨轮架构决策见 `docs/adr/`。

## 文档地图（事实单点来源）

| 事实 | 唯一来源 | 其余位置只放链接 |
|---|---|---|
| 接口契约（端点/错误码/字段语义） | `docs/api/README.md` | mock、`types/api.d.ts`、Swagger 与之对齐 |
| 表结构与 Redis Key | `docs/backend/data-model.md` | progress 只提能力，不复述结构 |
| 业务状态机与核心流程 | `docs/backend/core-flows.md` | — |
| 需求状态台账 | `.scratch/<feature>/spec.md`（✅=完成，删除线=不做） | spec 正文不堆实施细节 |
| 面向访客的功能清单 | 根 `README.md` | docs 内不重复 |
| 当前能力/未完成/前置条件 | `docs/frontend/progress.md`、`docs/backend/progress.md` | 只记当前态，不记数、不记流水 |
| 冻结历史 | `docs/reviews/`（评审快照）、git log | Original docs 不保留修改记录 |

## 核心词汇

| 词 | 含义 | 勿用 |
|---|---|---|
| 白名单（refresh 白名单） | Redis `refresh:{uid}:{jti}`，值为 `{ip,ua,ts}` JSON，TTL 14d；会话列表与踢下线的数据维度 | token 黑名单 |
| 会话版本号 SV | Redis `auth:ver:{uid}`，封禁/改密等 bump 后已签发 access 立即失效 | — |
| 订阅凭证 | 每用户独立 `users.uuid`，各协议统一按下发，节点据此归因流量 | 共享密码 |
| 审计 target 摘要 | 批量动作 target 只存 `batch:<count>`，完整 ID 列表在 detail JSON | ID 列表原文 |
| 流量模式 A / 模式 B | A=节点累计值上报差分计费；B=管理端手工导入（同日覆盖 A 值做校准） | 实时/离线流量 |
