-- 0004_node_report.down.sql
-- 回滚:删除 node_user_stats 表与 users.uuid / servers.node_key 列(快照数据一并丢弃)。

DROP TABLE IF EXISTS node_user_stats;
ALTER TABLE servers
    DROP CONSTRAINT IF EXISTS uq_servers_node_key,
    DROP COLUMN IF EXISTS node_key;
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS uq_users_uuid,
    DROP COLUMN IF EXISTS uuid;
