-- 0007_batch3_withdraw_mailtpl_category.up.sql
-- 第三批(xboard-gap-fill):
--   F02 佣金提现:tickets.type 工单类型区分 + commission_withdraws 提现单 + commission_logs.biz_type 账本分类;
--   F11 邮件模板:mail_templates 自定义模板表(缺失自动回退内置文案);
--   F15 内容管理:knowledge_categories 知识库分类表 + knowledges.category_id 归属(存量数据兜底回填)。
--   F14 会话管理不引入新表(Redis refresh 白名单扩展元数据);F19/F20 走配置,无表结构变更。

-- F02 工单类型:0=普通 1=佣金提现(与 level 优先级语义区分,见 spec F02)
ALTER TABLE tickets ADD COLUMN type SMALLINT NOT NULL DEFAULT 0;
COMMENT ON COLUMN tickets.type IS '0=普通 1=佣金提现(F02)';

-- F02 提现单:提交即扣减 commission_balance(行锁防双花),拒绝时自动退回并退流水
CREATE TABLE commission_withdraws (
    id            BIGSERIAL    PRIMARY KEY,
    user_id       BIGINT       NOT NULL,
    ticket_id     BIGINT       NOT NULL,
    amount        BIGINT       NOT NULL, -- 分
    method        VARCHAR(32)  NOT NULL,
    account       VARCHAR(255) NOT NULL,
    status        SMALLINT     NOT NULL DEFAULT 0, -- 0=处理中 1=已发放 2=已退回
    review_remark VARCHAR(255) NULL,
    reviewed_at   TIMESTAMP(3) NULL,
    created_at    TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE UNIQUE INDEX uk_withdraw_ticket ON commission_withdraws (ticket_id);
CREATE INDEX idx_withdraw_user ON commission_withdraws (user_id, created_at DESC);
COMMENT ON TABLE commission_withdraws IS '佣金提现单(F02,仅代理商,工单提现管理员手动发放)';
COMMENT ON COLUMN commission_withdraws.status IS '0=处理中 1=已发放 2=已退回';

-- F02 佣金账本分类:0=订单佣金 1=提现流水(提交/完成/退回复用 status 0/1/2 三态)
ALTER TABLE commission_logs ADD COLUMN biz_type SMALLINT NOT NULL DEFAULT 0;
-- order_no 全表唯一改为仅约束订单佣金(biz_type=0);提现流水 order_no='w<提现单ID>' 标记
ALTER TABLE commission_logs DROP CONSTRAINT uk_commission_order;
CREATE UNIQUE INDEX uk_commission_order ON commission_logs (order_no) WHERE biz_type = 0;
CREATE INDEX idx_commission_biz ON commission_logs (biz_type);
COMMENT ON COLUMN commission_logs.biz_type IS '0=订单佣金 1=提现流水(F02)';

-- F11 邮件模板:管理端自定义,验证码/到期提醒/流量提醒三类(占位符见 docs/api/README.md)
CREATE TABLE mail_templates (
    name       VARCHAR(64)  PRIMARY KEY,
    subject    VARCHAR(255) NOT NULL,
    body       TEXT         NOT NULL,
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
COMMENT ON TABLE mail_templates IS '自定义邮件模板(F11),缺失自动回退内置文案';

-- F15 知识库分类:按语言隔离,sort 决定用户端分组展示顺序
CREATE TABLE knowledge_categories (
    id         BIGSERIAL    PRIMARY KEY,
    language   VARCHAR(10)  NOT NULL DEFAULT 'zh-CN',
    name       VARCHAR(64)  NOT NULL,
    sort       INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE UNIQUE INDEX uk_knowledge_categories ON knowledge_categories (language, name);
COMMENT ON TABLE knowledge_categories IS '知识库分类(F15,存量数据按 language+category 去重回填)';

ALTER TABLE knowledges ADD COLUMN category_id BIGINT NULL REFERENCES knowledge_categories (id) ON DELETE SET NULL;
CREATE INDEX idx_knowledges_category ON knowledges (category_id);
-- 存量数据兜底:已有 (language, category) 组合全部建分类并回填归属
INSERT INTO knowledge_categories (language, name, sort)
SELECT DISTINCT language, category, 0 FROM knowledges ORDER BY language, category;
UPDATE knowledges k SET category_id = kc.id
    FROM knowledge_categories kc
    WHERE kc.name = k.category AND kc.language = k.language;
