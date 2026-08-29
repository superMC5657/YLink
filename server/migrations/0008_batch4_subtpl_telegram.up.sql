-- 0008_batch4_subtpl_telegram.up.sql
-- 第四批(xboard-gap-fill):
--   F10 订阅模板:subscription_templates 自定义订阅模板表(按客户端类型,缺失自动回退内置生成器);
--   F12 Telegram:users.telegram_id 加部分唯一索引(防两个账号绑定同一 chat,绑定流程无需新表)。

-- F10 订阅模板:name=客户端类型(clash/sing-box/v2ray),content 为 Go text/template 全文档模板,
-- 节点列表以预渲染块变量注入(见 docs/api/README.md);自定义缺失/渲染失败自动回退内置生成器
CREATE TABLE subscription_templates (
    name       VARCHAR(32)  PRIMARY KEY,
    content    TEXT         NOT NULL,
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
COMMENT ON TABLE subscription_templates IS '自定义订阅模板(F10,按客户端类型,缺失/错误回退内置生成器)';

-- F12 Telegram 绑定:同一 chat 仅可绑定一个账号(部分唯一索引,NULL 不受约束)
CREATE UNIQUE INDEX uk_users_telegram ON users (telegram_id) WHERE telegram_id IS NOT NULL;
COMMENT ON INDEX uk_users_telegram IS 'F12 一个 Telegram chat 仅绑定一个账号';

-- F12 telegram 设置项 seed:bot_token/webhook_secret 由管理端配置,enabled 总开关
INSERT INTO settings ("key", value)
VALUES ('telegram', '{"bot_token":"","bot_username":"","webhook_secret":"","enabled":false}')
ON CONFLICT ("key") DO NOTHING;
