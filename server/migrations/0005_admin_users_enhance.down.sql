-- 0005_admin_users_enhance.down.sql
-- 回滚:删除 mail_logs 表(发送日志一并丢弃)。

DROP TABLE IF EXISTS mail_logs;
