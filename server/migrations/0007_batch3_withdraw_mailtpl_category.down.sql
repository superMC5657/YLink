-- 0007_batch3_withdraw_mailtpl_category.down.sql(逆序回滚 0007)
-- 前置条件:回滚前提现业务数据须已结清——提现流水行(order_no='w<提现单ID>')在
-- 提交/完成/退回三态间复用同一 order_no,回滚到 0006 前的全表 UNIQUE(order_no)
-- 无法容纳重复值,故先按 biz_type=1 清除提现流水行再重建全量唯一约束。

ALTER TABLE knowledges DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS knowledge_categories;

DROP TABLE IF EXISTS mail_templates;

-- biz_type 列尚存,按其精确清除提现流水行(须先于 DROP COLUMN)
DELETE FROM commission_logs WHERE biz_type = 1;
DROP INDEX IF EXISTS idx_commission_biz;
DROP INDEX IF EXISTS uk_commission_order;
ALTER TABLE commission_logs DROP CONSTRAINT IF EXISTS uk_commission_order;
ALTER TABLE commission_logs ADD CONSTRAINT uk_commission_order UNIQUE (order_no);
ALTER TABLE commission_logs DROP COLUMN IF EXISTS biz_type;

DROP TABLE IF EXISTS commission_withdraws;

ALTER TABLE tickets DROP COLUMN IF EXISTS type;
