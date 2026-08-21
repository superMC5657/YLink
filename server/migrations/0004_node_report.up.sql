-- 0004_node_report.up.sql
-- 流量模式 A(节点上报,见 docs/backend/core-flows.md §8):
--   1) users.uuid:每用户订阅凭证(vmess/vless/tuic 的 uuid、ss/trojan/hysteria2 的密码),
--      存量用户用 gen_random_uuid() 回填;
--   2) servers.node_key:每节点上报密钥(X-Node-Key 鉴权),存量节点回填随机 md5;
--   3) node_user_stats:节点上报累计值快照,差分得增量(重复上报幂等)。

ALTER TABLE users
    ADD COLUMN uuid CHAR(36);
UPDATE users SET uuid = gen_random_uuid()::text WHERE uuid IS NULL;
ALTER TABLE users
    ALTER COLUMN uuid SET NOT NULL,
    ADD CONSTRAINT uq_users_uuid UNIQUE (uuid);
COMMENT ON COLUMN users.uuid IS '用户订阅凭证(节点上报归因)';

ALTER TABLE servers
    ADD COLUMN node_key CHAR(32);
UPDATE servers SET node_key = md5(random()::text || clock_timestamp()::text) WHERE node_key IS NULL;
ALTER TABLE servers
    ALTER COLUMN node_key SET NOT NULL,
    ADD CONSTRAINT uq_servers_node_key UNIQUE (node_key);
COMMENT ON COLUMN servers.node_key IS '节点上报密钥(X-Node-Key)';

CREATE TABLE node_user_stats (
    id         BIGSERIAL    PRIMARY KEY,
    server_id  BIGINT       NOT NULL,
    user_id    BIGINT       NOT NULL,
    last_u     BIGINT       NOT NULL DEFAULT 0,
    last_d     BIGINT       NOT NULL DEFAULT 0,
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE UNIQUE INDEX uq_node_user_stats ON node_user_stats (server_id, user_id);
COMMENT ON TABLE node_user_stats IS '节点上报累计值快照(模式 A 幂等差分)';
COMMENT ON COLUMN node_user_stats.last_u IS '上次上报累计上行(字节,未乘倍率)';
COMMENT ON COLUMN node_user_stats.last_d IS '上次上报累计下行(字节,未乘倍率)';
