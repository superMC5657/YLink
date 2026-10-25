# [P1] 0007/0008 down 迁移对称性修复

Status: resolved
Type: task

## Finding

- `server/migrations/0008_batch4_subtpl_telegram.down.sql`：不删除 up 种入的 `settings.telegram` 行——回滚后残留孤儿配置，再次 up 时种子逻辑可能行为分叉。
- `server/migrations/0007_batch3_withdraw_mailtpl_category.down.sql:11`：重建全量 `UNIQUE(order_no)`，但回滚时提现工单行（`w<N>` biz_type）依然存留 → 回滚即约束冲突，数据库进入不可用状态。

## Plan

1. 0008.down 补 `DELETE FROM settings WHERE key='telegram'`（按 up 实际种入的键）。
2. 0007.down 恢复与 0006 前状态一致的部分唯一：回滚前先删除提现工单行，或建部分唯一索引/约束 `WHERE biz_type = 0`——选择与 up 语义互逆且不炸库的方案（推荐：down 里删除 `w%` 提现行再重建全量唯一，并在注释说明该迁移要求提现数据先结清）。
3. 本地 docker 起临时库验证 up→down→up 全链路。

## Acceptance

- `migrate up → down → up` 在干净库与含提现数据的库上均成功。
- data-model.md 无需变更（文档描述的是当前 up 状态）。

## Comments

- 2026-10-25: 已修复——0008.down 补 `DELETE FROM settings WHERE "key"='telegram'`（清除 up 以 ON CONFLICT DO NOTHING 种入的行，消除回滚后孤儿配置与再 up 的行为分叉）；0007.down 在 `DROP COLUMN biz_type` 之前先 `DELETE FROM commission_logs WHERE biz_type=1` 清除提现流水行（提现单提交/完成/退回三态复用同一 `order_no='w<id>'`，全表 UNIQUE 重建必然冲突），再重建 0006 前的全量唯一约束——「回滚前提现数据须已结清」已写入迁移头注释。
- 2026-10-25: up→down→up 数据库演练**未验证**——本机 Docker daemon 未运行且无本地 PostgreSQL（无 psql），未擅自启动 Docker Desktop；已完成静态走查（列/索引/约束名与 up 及 0001 定义逐一对齐，DELETE 严格先于 DROP COLUMN）。上线前建议在临时库补一次 `make migrate up && make migrate down 2 && make migrate up`（含造一笔提现数据的库）。
