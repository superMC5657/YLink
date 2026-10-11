package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// 提现域 module：审核打款/拒绝退回 + 工单附带信息。
// 本 module 为 ticket-withdraw 唯一的写 seam 与读 seam，收敛此前散落的读写点。

// ReviewWithdraw 管理端审核提现工单：approve=true 确认打款（线下打款，系统内记账）；
// false 拒绝并自动退回佣金（防资损）。两类操作均关闭工单、写 commission_logs 流水与审计。
// 状态流转收敛于生命周期深 Module，本函数只补审计（目标为单个用户 ID）。
func (m *adminWithdrawModule) ReviewWithdraw(ctx context.Context, adminID, ticketID int64, approve bool, remark, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		out, err := reviewWithdrawTx(tx, m.repos, ticketID, adminID, approve, remark)
		if err != nil {
			return err
		}
		return m.audit.write(tx, adminID, out.action, fmt.Sprint(out.userID), ip, map[string]any{
			"ticket_id": ticketID, "withdraw_id": out.withdrawID,
			"amount": model.FenToYuan(out.amountFen), "method": out.method, "remark": sanitize.Text(remark),
		})
	})
}

// withdrawDetail 供工单详情复用的读 seam：转发到生命周期深 Module。
// 查询失败降级为 nil，不阻断详情主流程。
func (m *adminWithdrawModule) withdrawDetail(t *model.Ticket) *model.TicketWithdrawInfo {
	return withdrawInfo(m.db, t)
}
