-- 0002_balance_check.up.sql
-- 余额负值强约束（PostgreSQL CHECK 约束）：
-- 用户钱包 balance 不允许为负。所有余额路径均已做服务层保护：
--   划转（commission_balance → balance）与余额支付在行锁内校验充足性；
--   管理员调余额拒绝调整后为负（service 层 40000）；
--   退款只做「加」余额。
-- 注意：佣金回滚减的是 commission_balance（已发放佣金池），允许为负并记账审计，
-- 因此只对 balance 加约束、不对 commission_balance 加。

ALTER TABLE users
    ADD CONSTRAINT chk_users_balance_nonneg CHECK (balance >= 0);
