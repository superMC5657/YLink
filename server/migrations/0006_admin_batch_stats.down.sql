-- 0006_admin_batch_stats.down.sql
-- 回滚:删除 traffic_reset_logs 与 F04 报表聚合索引。

DROP TABLE IF EXISTS traffic_reset_logs;
DROP INDEX IF EXISTS idx_traffic_logs_date;
DROP INDEX IF EXISTS idx_orders_paid_at;
DROP INDEX IF EXISTS idx_users_created_at;
