-- 0008_batch4_subtpl_telegram.down.sql(逆序回滚 0008)
DROP INDEX IF EXISTS uk_users_telegram;
DROP TABLE IF EXISTS subscription_templates;
-- 清除 up 种入的 settings.telegram 行(ON CONFLICT DO NOTHING 种子),
-- 避免回滚后残留孤儿配置、再次 up 时种子逻辑行为分叉
DELETE FROM settings WHERE "key" = 'telegram';
