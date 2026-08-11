-- 0002_balance_check.down.sql
-- 回滚：删除 balance 非负约束

ALTER TABLE users
    DROP CONSTRAINT chk_users_balance_nonneg;
