-- 0005_admin_users_enhance.up.sql
-- F05 用户管理增强:
--   mail_logs:管理端向用户发送邮件的发送日志(站内通知/营销触达留痕)。
-- 批量操作 / CSV 导出 / 重置订阅密钥不引入新表(审计走既有 audit_logs)。

CREATE TABLE mail_logs (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL,
    email      VARCHAR(190) NOT NULL,
    subject    VARCHAR(255) NOT NULL,
    status     SMALLINT     NOT NULL DEFAULT 0, -- 0=发送失败 1=发送成功
    error      VARCHAR(512),
    admin_id   BIGINT       NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);
CREATE INDEX idx_mail_logs_user ON mail_logs (user_id, created_at DESC);
COMMENT ON TABLE mail_logs IS '管理端邮件发送日志(F05)';
COMMENT ON COLUMN mail_logs.status IS '0=发送失败 1=发送成功';
