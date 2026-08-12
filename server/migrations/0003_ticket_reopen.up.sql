-- 0003_ticket_reopen.up.sql
-- 工单「重开一次」:tickets 增加 reopen_count(该工单累计重开次数)。
-- 语义(见 docs/backend/core-flows.md §7):关闭后的工单用户最多重开一次,
-- 重开后状态回到「待回复」,reopen_count 置 1,此后不可再重开。

ALTER TABLE tickets
    ADD COLUMN reopen_count TINYINT NOT NULL DEFAULT 0 COMMENT '已重开次数(0/1,最多一次)';
