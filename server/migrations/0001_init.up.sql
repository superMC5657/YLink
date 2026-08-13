-- 0001_init.up.sql
-- 代理售卖系统 · 初始表结构与初始化数据（PostgreSQL）
-- 金额一律 BIGINT 存「分」；流量一律 BIGINT 存「字节」；JSON 列一律 JSONB。

-- ---------------------------------------------------------------
-- 用户
-- ---------------------------------------------------------------
CREATE TABLE users (
    id                 BIGSERIAL PRIMARY KEY,
    email              VARCHAR(190) NOT NULL,
    password_hash      VARCHAR(255) NOT NULL,
    role               SMALLINT     NOT NULL DEFAULT 0,
    balance            BIGINT       NOT NULL DEFAULT 0,
    commission_balance BIGINT       NOT NULL DEFAULT 0,
    invite_by_id       BIGINT       NULL,
    is_banned          BOOLEAN      NOT NULL DEFAULT FALSE,
    remind_expire      BOOLEAN      NOT NULL DEFAULT TRUE,
    remind_traffic     BOOLEAN      NOT NULL DEFAULT FALSE,
    telegram_id        BIGINT       NULL,
    plan_id            BIGINT       NULL,
    expired_at         TIMESTAMP(3) NULL,
    transfer_enable    BIGINT       NOT NULL DEFAULT 0,
    u                  BIGINT       NOT NULL DEFAULT 0,
    d                  BIGINT       NOT NULL DEFAULT 0,
    speed_limit        INT          NULL,
    device_limit       INT          NULL,
    sub_token          CHAR(36)     NOT NULL,
    created_at         TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at         TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_users_email UNIQUE (email),
    CONSTRAINT uk_users_sub_token UNIQUE (sub_token)
);
CREATE INDEX idx_users_plan_expire ON users (plan_id, expired_at);
CREATE INDEX idx_users_invite_by ON users (invite_by_id);
COMMENT ON TABLE users IS '用户';
COMMENT ON COLUMN users.role IS '0=用户 1=管理员 2=代理商';
COMMENT ON COLUMN users.balance IS '钱包余额（分）';
COMMENT ON COLUMN users.commission_balance IS '可划转佣金（分）';
COMMENT ON COLUMN users.invite_by_id IS '邀请人 user id';
COMMENT ON COLUMN users.plan_id IS '当前订阅套餐';
COMMENT ON COLUMN users.expired_at IS '订阅到期时间';
COMMENT ON COLUMN users.transfer_enable IS '套餐总流量（字节）';
COMMENT ON COLUMN users.u IS '已用上行（字节）';
COMMENT ON COLUMN users.d IS '已用下行（字节）';
COMMENT ON COLUMN users.speed_limit IS '套餐限速 Mbps 快照';
COMMENT ON COLUMN users.device_limit IS '同时在线设备数快照';
COMMENT ON COLUMN users.sub_token IS '订阅 token（UUID，可重置）';

-- ---------------------------------------------------------------
-- 套餐
-- ---------------------------------------------------------------
CREATE TABLE plans (
    id              BIGSERIAL    PRIMARY KEY,
    name            VARCHAR(64)  NOT NULL,
    content         TEXT         NULL,
    month_price     BIGINT       NULL,
    quarter_price   BIGINT       NULL,
    half_year_price BIGINT       NULL,
    year_price      BIGINT       NULL,
    onetime_price   BIGINT       NULL,
    traffic_gb      INT          NOT NULL,
    speed_limit     INT          NULL,
    device_limit    INT          NULL,
    group_ids       JSONB        NOT NULL,
    is_show         BOOLEAN      NOT NULL DEFAULT TRUE,
    sort            INT          NOT NULL DEFAULT 0,
    created_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
COMMENT ON TABLE plans IS '套餐';
COMMENT ON COLUMN plans.content IS '描述 Markdown';
COMMENT ON COLUMN plans.month_price IS '月付价（分），NULL=不支持';
COMMENT ON COLUMN plans.onetime_price IS '一次性';
COMMENT ON COLUMN plans.traffic_gb IS '每周期流量 GB';
COMMENT ON COLUMN plans.speed_limit IS '限速 Mbps，NULL=不限制';
COMMENT ON COLUMN plans.device_limit IS '同时在线设备数，NULL=不限制';
COMMENT ON COLUMN plans.group_ids IS '可用节点分组 id 数组';

-- ---------------------------------------------------------------
-- 订单
-- ---------------------------------------------------------------
CREATE TABLE orders (
    id              BIGSERIAL    PRIMARY KEY,
    order_no        VARCHAR(32)  NOT NULL,
    user_id         BIGINT       NOT NULL,
    plan_id         BIGINT       NOT NULL,
    period          VARCHAR(16)  NOT NULL,
    amount          BIGINT       NOT NULL,
    discount_amount BIGINT       NOT NULL DEFAULT 0,
    balance_used    BIGINT       NOT NULL DEFAULT 0,
    pay_amount      BIGINT       NOT NULL,
    coupon_id       BIGINT       NULL,
    status          SMALLINT     NOT NULL DEFAULT 0,
    pay_method      VARCHAR(32)  NULL,
    paid_at         TIMESTAMP(3) NULL,
    idempotency_key VARCHAR(64)  NULL,
    created_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_orders_no UNIQUE (order_no),
    CONSTRAINT uk_orders_idem UNIQUE (idempotency_key)
);
CREATE INDEX idx_orders_user_status ON orders (user_id, status, created_at);
COMMENT ON TABLE orders IS '订单';
COMMENT ON COLUMN orders.period IS 'month/quarter/half_year/year/onetime';
COMMENT ON COLUMN orders.amount IS '套餐原价（分）';
COMMENT ON COLUMN orders.pay_amount IS '应付 = amount - discount - balance_used';
COMMENT ON COLUMN orders.status IS '0=待支付 1=已完成 2=已取消 3=已退款';

-- ---------------------------------------------------------------
-- 支付单
-- ---------------------------------------------------------------
CREATE TABLE payments (
    id             BIGSERIAL    PRIMARY KEY,
    order_no       VARCHAR(32)  NOT NULL,
    user_id        BIGINT       NOT NULL,
    method         VARCHAR(32)  NOT NULL,
    amount         BIGINT       NOT NULL,
    trade_no       VARCHAR(64)  NULL,
    status         SMALLINT     NOT NULL DEFAULT 0,
    notify_payload TEXT         NULL,
    paid_at        TIMESTAMP(3) NULL,
    created_at     TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_payments_trade_no UNIQUE (trade_no)
);
CREATE INDEX idx_payments_order ON payments (order_no);
COMMENT ON TABLE payments IS '支付单';
COMMENT ON COLUMN payments.amount IS '实收（分）';
COMMENT ON COLUMN payments.trade_no IS '网关流水号（回调幂等约束）';
COMMENT ON COLUMN payments.status IS '0=待支付 1=成功 2=失败/关闭';
COMMENT ON COLUMN payments.notify_payload IS '回调原文';

-- ---------------------------------------------------------------
-- 优惠券
-- ---------------------------------------------------------------
CREATE TABLE coupons (
    id             BIGSERIAL    PRIMARY KEY,
    code           VARCHAR(64)  NOT NULL,
    type           SMALLINT     NOT NULL,
    value          BIGINT       NOT NULL,
    min_spend      BIGINT       NOT NULL DEFAULT 0,
    limit_per_user INT          NOT NULL DEFAULT 0,
    total_limit    INT          NOT NULL DEFAULT 0,
    used_count     INT          NOT NULL DEFAULT 0,
    valid_periods  JSONB        NULL,
    plan_ids       JSONB        NULL,
    started_at     TIMESTAMP(3) NULL,
    ended_at       TIMESTAMP(3) NULL,
    is_enable      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_coupons_code UNIQUE (code)
);
COMMENT ON TABLE coupons IS '优惠券';
COMMENT ON COLUMN coupons.type IS '1=固定金额 2=百分比';
COMMENT ON COLUMN coupons.value IS '金额（分）或百分比整数';
COMMENT ON COLUMN coupons.min_spend IS '门槛（分）';
COMMENT ON COLUMN coupons.limit_per_user IS '0=不限';
COMMENT ON COLUMN coupons.total_limit IS '总限量，0=不限';
COMMENT ON COLUMN coupons.valid_periods IS '限定周期数组，NULL=全场';
COMMENT ON COLUMN coupons.plan_ids IS '限定套餐数组，NULL=全场';

CREATE TABLE coupon_usages (
    id         BIGSERIAL    PRIMARY KEY,
    coupon_id  BIGINT       NOT NULL,
    user_id    BIGINT       NOT NULL,
    order_no   VARCHAR(32)  NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_coupon_usage UNIQUE (coupon_id, user_id, order_no)
);
COMMENT ON TABLE coupon_usages IS '优惠券使用记录';

-- ---------------------------------------------------------------
-- 邀请码 / 佣金
-- ---------------------------------------------------------------
CREATE TABLE invite_codes (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL,
    code       VARCHAR(32)  NOT NULL,
    status     SMALLINT     NOT NULL DEFAULT 1,
    used_count INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_invite_codes UNIQUE (code)
);
CREATE INDEX idx_invite_codes_user ON invite_codes (user_id);
COMMENT ON TABLE invite_codes IS '邀请码';
COMMENT ON COLUMN invite_codes.status IS '1=有效 0=停用';

CREATE TABLE commission_logs (
    id             BIGSERIAL    PRIMARY KEY,
    invite_user_id BIGINT       NOT NULL,
    from_user_id   BIGINT       NOT NULL,
    order_no       VARCHAR(32)  NOT NULL,
    order_amount   BIGINT       NOT NULL,
    rate           INT          NOT NULL,
    amount         BIGINT       NOT NULL,
    status         SMALLINT     NOT NULL DEFAULT 0,
    confirmed_at   TIMESTAMP(3) NULL,
    created_at     TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_commission_order UNIQUE (order_no)
);
CREATE INDEX idx_commission_invite ON commission_logs (invite_user_id);
COMMENT ON TABLE commission_logs IS '佣金记录';
COMMENT ON COLUMN commission_logs.invite_user_id IS '获得佣金的邀请人';
COMMENT ON COLUMN commission_logs.from_user_id IS '下单用户';
COMMENT ON COLUMN commission_logs.order_amount IS '订单实付（分）';
COMMENT ON COLUMN commission_logs.rate IS '佣金比例 %（快照）';
COMMENT ON COLUMN commission_logs.amount IS '佣金（分）';
COMMENT ON COLUMN commission_logs.status IS '0=确认中 1=已发放 2=已撤销';

-- ---------------------------------------------------------------
-- 节点
-- ---------------------------------------------------------------
CREATE TABLE server_groups (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(64)  NOT NULL,
    sort       INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
COMMENT ON TABLE server_groups IS '节点分组';

CREATE TABLE servers (
    id         BIGSERIAL    PRIMARY KEY,
    group_id   BIGINT       NOT NULL,
    name       VARCHAR(64)  NOT NULL,
    type       VARCHAR(32)  NOT NULL,
    host       VARCHAR(255) NOT NULL,
    port       INT          NOT NULL,
    config     JSONB        NOT NULL,
    rate       DECIMAL(3,1) NOT NULL DEFAULT 1.0,
    tags       JSONB        NULL,
    status     SMALLINT     NOT NULL DEFAULT 1,
    is_show    BOOLEAN      NOT NULL DEFAULT TRUE,
    sort       INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX idx_servers_group ON servers (group_id);
COMMENT ON TABLE servers IS '节点';
COMMENT ON COLUMN servers.type IS 'shadowsocks/vmess/vless/trojan/hysteria2/tuic';
COMMENT ON COLUMN servers.config IS '协议私有参数';
COMMENT ON COLUMN servers.rate IS '流量倍率';
COMMENT ON COLUMN servers.status IS '1=正常 2=拥挤 3=维护';

-- ---------------------------------------------------------------
-- 公告 / 知识库
-- ---------------------------------------------------------------
CREATE TABLE notices (
    id         BIGSERIAL    PRIMARY KEY,
    title      VARCHAR(128) NOT NULL,
    content    TEXT         NOT NULL,
    is_show    BOOLEAN      NOT NULL DEFAULT TRUE,
    sort       INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
COMMENT ON TABLE notices IS '公告';
COMMENT ON COLUMN notices.content IS 'Markdown（已清洗）';

CREATE TABLE knowledges (
    id         BIGSERIAL    PRIMARY KEY,
    category   VARCHAR(64)  NOT NULL,
    title      VARCHAR(128) NOT NULL,
    body       TEXT         NOT NULL,
    language   VARCHAR(10)  NOT NULL DEFAULT 'zh-CN',
    is_show    BOOLEAN      NOT NULL DEFAULT TRUE,
    sort       INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX idx_knowledges_lang ON knowledges (language, category, is_show);
COMMENT ON TABLE knowledges IS '知识库';
COMMENT ON COLUMN knowledges.body IS 'Markdown（已清洗）';

-- ---------------------------------------------------------------
-- 工单
-- ---------------------------------------------------------------
CREATE TABLE tickets (
    id            BIGSERIAL    PRIMARY KEY,
    user_id       BIGINT       NOT NULL,
    subject       VARCHAR(128) NOT NULL,
    level         SMALLINT     NOT NULL DEFAULT 1,
    status        SMALLINT     NOT NULL DEFAULT 0,
    last_reply_at TIMESTAMP(3) NULL,
    created_at    TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX idx_tickets_user ON tickets (user_id);
COMMENT ON TABLE tickets IS '工单';
COMMENT ON COLUMN tickets.level IS '0=低 1=中 2=高';
COMMENT ON COLUMN tickets.status IS '0=待回复 1=已回复 2=已关闭';

CREATE TABLE ticket_messages (
    id          BIGSERIAL    PRIMARY KEY,
    ticket_id   BIGINT       NOT NULL,
    sender_type SMALLINT     NOT NULL,
    sender_id   BIGINT       NOT NULL,
    message     TEXT         NOT NULL,
    created_at  TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX idx_ticket_msgs ON ticket_messages (ticket_id);
COMMENT ON TABLE ticket_messages IS '工单消息';
COMMENT ON COLUMN ticket_messages.sender_type IS '0=用户 1=客服';

-- ---------------------------------------------------------------
-- 流量日明细
-- ---------------------------------------------------------------
CREATE TABLE traffic_logs (
    id      BIGSERIAL PRIMARY KEY,
    user_id BIGINT    NOT NULL,
    date    DATE      NOT NULL,
    u       BIGINT    NOT NULL DEFAULT 0,
    d       BIGINT    NOT NULL DEFAULT 0,
    CONSTRAINT uk_traffic_user_date UNIQUE (user_id, date)
);
COMMENT ON TABLE traffic_logs IS '流量日明细';

-- ---------------------------------------------------------------
-- 站点配置 / 审计
-- ---------------------------------------------------------------
CREATE TABLE settings (
    "key"      VARCHAR(64)  NOT NULL PRIMARY KEY,
    value      JSONB        NOT NULL,
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
COMMENT ON TABLE settings IS '站点配置';

CREATE TABLE audit_logs (
    id         BIGSERIAL    PRIMARY KEY,
    admin_id   BIGINT       NOT NULL,
    action     VARCHAR(64)  NOT NULL,
    target     VARCHAR(128) NULL,
    detail     JSONB        NULL,
    ip         VARCHAR(64)  NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
COMMENT ON TABLE audit_logs IS '审计日志';
COMMENT ON COLUMN audit_logs.action IS 'adjust_balance/refund/ban_user...';

-- ---------------------------------------------------------------
-- 代理商申请
-- ---------------------------------------------------------------
CREATE TABLE agent_applies (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    status      SMALLINT     NOT NULL DEFAULT 0,
    remark      VARCHAR(255) NULL,
    reviewed_at TIMESTAMP(3) NULL,
    created_at  TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT uk_agent_apply_user UNIQUE (user_id)
);
CREATE INDEX idx_agent_apply_status ON agent_applies (status);
COMMENT ON TABLE agent_applies IS '代理商申请';
COMMENT ON COLUMN agent_applies.status IS '0=待审核 1=通过 2=拒绝';

-- ===============================================================
-- 初始化数据
-- ===============================================================

INSERT INTO server_groups (id, name, sort) VALUES
  (1, '香港', 1), (2, '台湾', 2), (3, '日本', 3), (4, '新加坡', 4), (5, '美国', 5);
-- 显式插入 id 后推进序列，避免后续插入主键冲突
SELECT setval(pg_get_serial_sequence('server_groups', 'id'), (SELECT MAX(id) FROM server_groups));

INSERT INTO plans (id, name, content, month_price, quarter_price, half_year_price, year_price, onetime_price, traffic_gb, speed_limit, device_limit, group_ids, is_show, sort) VALUES
  (1, '白羊座', '购买套餐后可能需要等待5分钟左右才能连接\n支持 **5 台**设备同时在线', 1000, 2700, NULL, 9600, NULL, 300, 300, 5, '[1,2,3,4,5]'::jsonb, true, 1),
  (2, '金牛座', '高速节点不限速畅享', 1500, 4000, NULL, 14400, NULL, 500, 500, 8, '[1,2,3,4,5]'::jsonb, true, 2),
  (3, '射手座', '不限速旗舰套餐', 2000, NULL, NULL, NULL, NULL, 650, NULL, 10, '[1,2,3,4,5]'::jsonb, true, 3),
  (4, '猎户座', '旗舰不限速不限设备', 3000, 8100, NULL, 28800, NULL, 1024, NULL, NULL, '[1,2,3,4,5]'::jsonb, true, 4);
SELECT setval(pg_get_serial_sequence('plans', 'id'), (SELECT MAX(id) FROM plans));

INSERT INTO settings ("key", value) VALUES
  ('site', '{"site_name":"YLink","site_logo":"","site_description":"高速稳定的网络加速服务","register_enabled":true,"invite_code_required":false,"app_downloads":{},"telegram":{},"customer_service_url":"","free_traffic_tips":"绑定 TG 机器人每天领取免费流量","languages":["zh-CN","en-US"]}'::jsonb),
  ('payment', '{"methods":[{"code":"balance","name":"余额支付","icon":"wallet","enabled":true},{"code":"epay_alipay","name":"支付宝","icon":"alipay","enabled":true},{"code":"epay_wxpay","name":"微信支付","icon":"wechat","enabled":true}]}'::jsonb),
  ('invite', '{"commission_rate":40,"agent_commission_rate":50,"commission_confirm_days":3,"invite_code_limit":5}'::jsonb),
  ('agent', '{"required_valid_invites":50,"audit_months":12,"benefits":["佣金比例：40%（循环）","套餐福利：赠送免费的年付订阅套餐","订单推送：享受 bot 订单实时推送","审验周期：12个月"],"notes":["点击按钮申请代理权限，审核通过后将获得以上特权。"]}'::jsonb),
  ('order', '{"expire_minutes":30}'::jsonb),
  ('templates', '{"captcha":"您的验证码是 <b>{code}</b>，10 分钟内有效。","welcome":"欢迎注册 {site_name}！","expire":"您的订阅将于 {expired_at} 到期，请及时续费。","traffic":"您的流量已使用 {percent}%，请注意剩余流量。"}'::jsonb);
