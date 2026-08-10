-- 0001_init.up.sql
-- 代理售卖系统 · 初始表结构与初始化数据
-- 金额一律 BIGINT 存「分」；流量一律 BIGINT 存「字节」；utf8mb4

SET NAMES utf8mb4;

-- ---------------------------------------------------------------
-- 用户
-- ---------------------------------------------------------------
CREATE TABLE users (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    email              VARCHAR(190)    NOT NULL,
    password_hash      VARCHAR(255)    NOT NULL,
    role               TINYINT         NOT NULL DEFAULT 0 COMMENT '0=用户 1=管理员 2=代理商',
    balance            BIGINT          NOT NULL DEFAULT 0 COMMENT '钱包余额（分）',
    commission_balance BIGINT          NOT NULL DEFAULT 0 COMMENT '可划转佣金（分）',
    invite_by_id       BIGINT UNSIGNED NULL COMMENT '邀请人 user id',
    is_banned          TINYINT(1)      NOT NULL DEFAULT 0,
    remind_expire      TINYINT(1)      NOT NULL DEFAULT 1,
    remind_traffic     TINYINT(1)      NOT NULL DEFAULT 0,
    telegram_id        BIGINT UNSIGNED NULL,
    plan_id            BIGINT UNSIGNED NULL COMMENT '当前订阅套餐',
    expired_at         DATETIME(3)     NULL COMMENT '订阅到期时间',
    transfer_enable    BIGINT          NOT NULL DEFAULT 0 COMMENT '套餐总流量（字节）',
    u                  BIGINT          NOT NULL DEFAULT 0 COMMENT '已用上行（字节）',
    d                  BIGINT          NOT NULL DEFAULT 0 COMMENT '已用下行（字节）',
    speed_limit        INT             NULL COMMENT '套餐限速 Mbps 快照',
    device_limit       INT             NULL COMMENT '同时在线设备数快照',
    sub_token          CHAR(36)        NOT NULL COMMENT '订阅 token（UUID，可重置）',
    created_at         DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at         DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_email (email),
    UNIQUE KEY uk_users_sub_token (sub_token),
    KEY idx_users_plan_expire (plan_id, expired_at),
    KEY idx_users_invite_by (invite_by_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户';

-- ---------------------------------------------------------------
-- 套餐
-- ---------------------------------------------------------------
CREATE TABLE plans (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name           VARCHAR(64)     NOT NULL,
    content        TEXT            NULL COMMENT '描述 Markdown',
    month_price    BIGINT          NULL COMMENT '月付价（分），NULL=不支持',
    quarter_price  BIGINT          NULL,
    half_year_price BIGINT         NULL,
    year_price     BIGINT          NULL,
    onetime_price  BIGINT          NULL COMMENT '一次性',
    traffic_gb     INT             NOT NULL COMMENT '每周期流量 GB',
    speed_limit    INT             NULL COMMENT '限速 Mbps，NULL=不限制',
    device_limit   INT             NULL COMMENT '同时在线设备数，NULL=不限制',
    group_ids      JSON            NOT NULL COMMENT '可用节点分组 id 数组',
    is_show        TINYINT(1)      NOT NULL DEFAULT 1,
    sort           INT             NOT NULL DEFAULT 0,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='套餐';

-- ---------------------------------------------------------------
-- 订单
-- ---------------------------------------------------------------
CREATE TABLE orders (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_no        VARCHAR(32)     NOT NULL,
    user_id         BIGINT UNSIGNED NOT NULL,
    plan_id         BIGINT UNSIGNED NOT NULL,
    period          VARCHAR(16)     NOT NULL COMMENT 'month/quarter/half_year/year/onetime',
    amount          BIGINT          NOT NULL COMMENT '套餐原价（分）',
    discount_amount BIGINT          NOT NULL DEFAULT 0,
    balance_used    BIGINT          NOT NULL DEFAULT 0,
    pay_amount      BIGINT          NOT NULL COMMENT '应付 = amount - discount - balance_used',
    coupon_id       BIGINT UNSIGNED NULL,
    status          TINYINT         NOT NULL DEFAULT 0 COMMENT '0=待支付 1=已完成 2=已取消 3=已退款',
    pay_method      VARCHAR(32)     NULL,
    paid_at         DATETIME(3)     NULL,
    idempotency_key VARCHAR(64)     NULL,
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_orders_no (order_no),
    UNIQUE KEY uk_orders_idem (idempotency_key),
    KEY idx_orders_user_status (user_id, status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单';

-- ---------------------------------------------------------------
-- 支付单
-- ---------------------------------------------------------------
CREATE TABLE payments (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_no       VARCHAR(32)     NOT NULL,
    user_id        BIGINT UNSIGNED NOT NULL,
    method         VARCHAR(32)     NOT NULL,
    amount         BIGINT          NOT NULL COMMENT '实收（分）',
    trade_no       VARCHAR(64)     NULL COMMENT '网关流水号（回调幂等约束）',
    status         TINYINT         NOT NULL DEFAULT 0 COMMENT '0=待支付 1=成功 2=失败/关闭',
    notify_payload TEXT            NULL COMMENT '回调原文',
    paid_at        DATETIME(3)     NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_payments_trade_no (trade_no),
    KEY idx_payments_order (order_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付单';

-- ---------------------------------------------------------------
-- 优惠券
-- ---------------------------------------------------------------
CREATE TABLE coupons (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code           VARCHAR(64)     NOT NULL,
    type           TINYINT         NOT NULL COMMENT '1=固定金额 2=百分比',
    value          BIGINT          NOT NULL COMMENT '金额（分）或百分比整数',
    min_spend      BIGINT          NOT NULL DEFAULT 0 COMMENT '门槛（分）',
    limit_per_user INT             NOT NULL DEFAULT 0 COMMENT '0=不限',
    total_limit    INT             NOT NULL DEFAULT 0 COMMENT '总限量，0=不限',
    used_count     INT             NOT NULL DEFAULT 0,
    valid_periods  JSON            NULL COMMENT '限定周期数组，NULL=全场',
    plan_ids       JSON            NULL COMMENT '限定套餐数组，NULL=全场',
    started_at     DATETIME(3)     NULL,
    ended_at       DATETIME(3)     NULL,
    is_enable      TINYINT(1)      NOT NULL DEFAULT 1,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_coupons_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='优惠券';

CREATE TABLE coupon_usages (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    coupon_id  BIGINT UNSIGNED NOT NULL,
    user_id    BIGINT UNSIGNED NOT NULL,
    order_no   VARCHAR(32)     NOT NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_coupon_usage (coupon_id, user_id, order_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='优惠券使用记录';

-- ---------------------------------------------------------------
-- 邀请码 / 佣金
-- ---------------------------------------------------------------
CREATE TABLE invite_codes (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id    BIGINT UNSIGNED NOT NULL,
    code       VARCHAR(32)     NOT NULL,
    status     TINYINT         NOT NULL DEFAULT 1 COMMENT '1=有效 0=停用',
    used_count INT             NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_invite_codes (code),
    KEY idx_invite_codes_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='邀请码';

CREATE TABLE commission_logs (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    invite_user_id BIGINT UNSIGNED NOT NULL COMMENT '获得佣金的邀请人',
    from_user_id   BIGINT UNSIGNED NOT NULL COMMENT '下单用户',
    order_no       VARCHAR(32)     NOT NULL,
    order_amount   BIGINT          NOT NULL COMMENT '订单实付（分）',
    rate           INT             NOT NULL COMMENT '佣金比例 %（快照）',
    amount         BIGINT          NOT NULL COMMENT '佣金（分）',
    status         TINYINT         NOT NULL DEFAULT 0 COMMENT '0=确认中 1=已发放 2=已撤销',
    confirmed_at   DATETIME(3)     NULL,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_commission_order (order_no),
    KEY idx_commission_invite (invite_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='佣金记录';

-- ---------------------------------------------------------------
-- 节点
-- ---------------------------------------------------------------
CREATE TABLE server_groups (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name       VARCHAR(64)     NOT NULL,
    sort       INT             NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点分组';

CREATE TABLE servers (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    group_id   BIGINT UNSIGNED NOT NULL,
    name       VARCHAR(64)     NOT NULL,
    type       VARCHAR(32)     NOT NULL COMMENT 'shadowsocks/vmess/vless/trojan/hysteria2/tuic',
    host       VARCHAR(255)    NOT NULL,
    port       INT             NOT NULL,
    config     JSON            NOT NULL COMMENT '协议私有参数',
    rate       DECIMAL(3,1)    NOT NULL DEFAULT 1.0 COMMENT '流量倍率',
    tags       JSON            NULL,
    status     TINYINT         NOT NULL DEFAULT 1 COMMENT '1=正常 2=拥挤 3=维护',
    is_show    TINYINT(1)      NOT NULL DEFAULT 1,
    sort       INT             NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_servers_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点';

-- ---------------------------------------------------------------
-- 公告 / 知识库
-- ---------------------------------------------------------------
CREATE TABLE notices (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title      VARCHAR(128)    NOT NULL,
    content    MEDIUMTEXT      NOT NULL COMMENT 'Markdown（已清洗）',
    is_show    TINYINT(1)      NOT NULL DEFAULT 1,
    sort       INT             NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='公告';

CREATE TABLE knowledges (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    category   VARCHAR(64)     NOT NULL,
    title      VARCHAR(128)    NOT NULL,
    body       MEDIUMTEXT      NOT NULL COMMENT 'Markdown（已清洗）',
    language   VARCHAR(10)     NOT NULL DEFAULT 'zh-CN',
    is_show    TINYINT(1)      NOT NULL DEFAULT 1,
    sort       INT             NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_knowledges_lang (language, category, is_show)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='知识库';

-- ---------------------------------------------------------------
-- 工单
-- ---------------------------------------------------------------
CREATE TABLE tickets (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id       BIGINT UNSIGNED NOT NULL,
    subject       VARCHAR(128)    NOT NULL,
    level         TINYINT         NOT NULL DEFAULT 1 COMMENT '0=低 1=中 2=高',
    status        TINYINT         NOT NULL DEFAULT 0 COMMENT '0=待回复 1=已回复 2=已关闭',
    last_reply_at DATETIME(3)     NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_tickets_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工单';

CREATE TABLE ticket_messages (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_id   BIGINT UNSIGNED NOT NULL,
    sender_type TINYINT         NOT NULL COMMENT '0=用户 1=客服',
    sender_id   BIGINT UNSIGNED NOT NULL,
    message     TEXT            NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_ticket_msgs (ticket_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工单消息';

-- ---------------------------------------------------------------
-- 流量日明细
-- ---------------------------------------------------------------
CREATE TABLE traffic_logs (
    id      BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    date    DATE            NOT NULL,
    u       BIGINT          NOT NULL DEFAULT 0,
    d       BIGINT          NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_traffic_user_date (user_id, date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='流量日明细';

-- ---------------------------------------------------------------
-- 站点配置 / 审计
-- ---------------------------------------------------------------
CREATE TABLE settings (
    `key`  VARCHAR(64) NOT NULL,
    value  JSON        NOT NULL,
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='站点配置';

CREATE TABLE audit_logs (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    admin_id   BIGINT UNSIGNED NOT NULL,
    action     VARCHAR(64)     NOT NULL COMMENT 'adjust_balance/refund/ban_user...',
    target     VARCHAR(128)    NULL,
    detail     JSON            NULL,
    ip         VARCHAR(64)     NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='审计日志';

-- ---------------------------------------------------------------
-- 代理商申请
-- ---------------------------------------------------------------
CREATE TABLE agent_applies (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id     BIGINT UNSIGNED NOT NULL,
    status      TINYINT         NOT NULL DEFAULT 0 COMMENT '0=待审核 1=通过 2=拒绝',
    remark      VARCHAR(255)    NULL,
    reviewed_at DATETIME(3)     NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_agent_apply_user (user_id),
    KEY idx_agent_apply_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代理商申请';

-- ===============================================================
-- 初始化数据
-- ===============================================================

INSERT INTO server_groups (id, name, sort) VALUES
  (1, '香港', 1), (2, '台湾', 2), (3, '日本', 3), (4, '新加坡', 4), (5, '美国', 5);

INSERT INTO plans (id, name, content, month_price, quarter_price, half_year_price, year_price, onetime_price, traffic_gb, speed_limit, device_limit, group_ids, is_show, sort) VALUES
  (1, '白羊座', '购买套餐后可能需要等待5分钟左右才能连接\n支持 **5 台**设备同时在线', 1000, 2700, NULL, 9600, NULL, 300, 300, 5, JSON_ARRAY(1,2,3,4,5), 1, 1),
  (2, '金牛座', '高速节点不限速畅享', 1500, 4000, NULL, 14400, NULL, 500, 500, 8, JSON_ARRAY(1,2,3,4,5), 1, 2),
  (3, '射手座', '不限速旗舰套餐', 2000, NULL, NULL, NULL, NULL, 650, NULL, 10, JSON_ARRAY(1,2,3,4,5), 1, 3),
  (4, '猎户座', '旗舰不限速不限设备', 3000, 8100, NULL, 28800, NULL, 1024, NULL, NULL, JSON_ARRAY(1,2,3,4,5), 1, 4);

INSERT INTO settings (`key`, value) VALUES
  ('site', JSON_OBJECT(
    'site_name', 'YLink', 'site_logo', '', 'site_description', '高速稳定的网络加速服务',
    'register_enabled', TRUE, 'invite_code_required', FALSE,
    'app_downloads', JSON_OBJECT(), 'telegram', JSON_OBJECT(),
    'customer_service_url', '', 'free_traffic_tips', '绑定 TG 机器人每天领取免费流量',
    'languages', JSON_ARRAY('zh-CN','en-US'))),
  ('payment', JSON_OBJECT(
    'methods', JSON_ARRAY(
      JSON_OBJECT('code','balance','name','余额支付','icon','wallet','enabled',TRUE),
      JSON_OBJECT('code','epay_alipay','name','支付宝','icon','alipay','enabled',TRUE),
      JSON_OBJECT('code','epay_wxpay','name','微信支付','icon','wechat','enabled',TRUE)))),
  ('invite', JSON_OBJECT(
    'commission_rate', 40, 'agent_commission_rate', 50,
    'commission_confirm_days', 3, 'invite_code_limit', 5)),
  ('agent', JSON_OBJECT(
    'required_valid_invites', 50, 'audit_months', 12,
    'benefits', JSON_ARRAY('佣金比例：40%（循环）', '套餐福利：赠送免费的年付订阅套餐', '订单推送：享受 bot 订单实时推送', '审验周期：12个月'),
    'notes', JSON_ARRAY('点击按钮申请代理权限，审核通过后将获得以上特权。'))),
  ('order', JSON_OBJECT('expire_minutes', 30)),
  ('templates', JSON_OBJECT(
    'captcha', '您的验证码是 <b>{code}</b>，10 分钟内有效。',
    'welcome', '欢迎注册 {site_name}！',
    'expire', '您的订阅将于 {expired_at} 到期，请及时续费。',
    'traffic', '您的流量已使用 {percent}%，请注意剩余流量。'));
