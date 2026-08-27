-- 0006_admin_batch_stats.up.sql
-- 第二批(xboard-gap-fill):
--   F16 流量重置管理:traffic_reset_logs 重置记录表;
--   F04 统计报表:时间范围聚合查询补索引(traffic_logs.date / orders.paid_at / users.created_at)。
--   F09 节点批量操作不引入新表(审计走既有 audit_logs)。

CREATE TABLE traffic_reset_logs (
    id                     BIGSERIAL    PRIMARY KEY,
    user_id                BIGINT       NOT NULL,
    admin_id               BIGINT       NOT NULL,
    mode                   VARCHAR(16)  NOT NULL, -- clear_usage=清零用量 reset_quota=重新给量
    before_u               BIGINT       NOT NULL DEFAULT 0,
    before_d               BIGINT       NOT NULL DEFAULT 0,
    before_transfer_enable BIGINT       NOT NULL DEFAULT 0,
    after_transfer_enable  BIGINT       NOT NULL DEFAULT 0,
    created_at             TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX idx_traffic_reset_logs_user ON traffic_reset_logs (user_id, created_at DESC);
COMMENT ON TABLE traffic_reset_logs IS '管理端流量重置记录(F16)';
COMMENT ON COLUMN traffic_reset_logs.mode IS 'clear_usage=清零用量 reset_quota=重新给量(按当前套餐额度)';

-- F04 报表聚合走时间范围 GROUP BY,补日期索引避免全表扫描
CREATE INDEX idx_traffic_logs_date ON traffic_logs (date);
CREATE INDEX idx_orders_paid_at ON orders (paid_at) WHERE paid_at IS NOT NULL;
CREATE INDEX idx_users_created_at ON users (created_at);
