-- 0003_ticket_reopen.down.sql
-- 回滚:删除 reopen_count 列

ALTER TABLE tickets
    DROP COLUMN reopen_count;
