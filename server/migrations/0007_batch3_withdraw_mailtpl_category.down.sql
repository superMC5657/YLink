-- 0007_batch3_withdraw_mailtpl_category.down.sql(逆序回滚 0007)

ALTER TABLE knowledges DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS knowledge_categories;

DROP TABLE IF EXISTS mail_templates;

DROP INDEX IF EXISTS idx_commission_biz;
DROP INDEX IF EXISTS uk_commission_order;
ALTER TABLE commission_logs DROP CONSTRAINT IF EXISTS uk_commission_order;
ALTER TABLE commission_logs ADD CONSTRAINT uk_commission_order UNIQUE (order_no);
ALTER TABLE commission_logs DROP COLUMN IF EXISTS biz_type;

DROP TABLE IF EXISTS commission_withdraws;

ALTER TABLE tickets DROP COLUMN IF EXISTS type;
