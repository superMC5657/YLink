package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// ---- 管理端 · 佣金提现审核（F02） ----

// ReviewWithdraw 管理端审核提现工单：approve=true 确认打款（线下打款，系统内记账）；
// false 拒绝并自动退回佣金（防资损）。两类操作均关闭工单、写 commission_logs 流水与审计。
func (s *AdminService) ReviewWithdraw(ctx context.Context, adminID, ticketID int64, approve bool, remark, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		ticket, err := s.repos.Ticket.GetByID(tx, ticketID)
		if err != nil {
			return errs.ErrNotFound
		}
		if ticket.Type != model.TicketTypeWithdraw {
			return errs.ErrConflict
		}
		w, err := s.repos.Withdraw.GetByTicketIDForUpdate(tx, ticketID)
		if err != nil {
			return errs.ErrNotFound
		}
		if w.Status != model.WithdrawPending {
			return errs.ErrWithdrawStatus
		}
		now := time.Now()
		action := "withdraw_reject"
		var sysMsg string
		if approve {
			w.Status = model.WithdrawPaid
			sysMsg = "管理员已确认打款，佣金提现完成。"
			action = "withdraw_pay"
		} else {
			// 拒绝：自动退回佣金（提交时已扣减，拒绝必须原路退回防资损）
			user, err := s.repos.User.GetByIDForUpdate(tx, w.UserID)
			if err != nil {
				return err
			}
			user.CommissionBalance += w.Amount
			if err := s.repos.User.Save(tx, user); err != nil {
				return err
			}
			w.Status = model.WithdrawRefunded
			sysMsg = "管理员已拒绝该提现申请，佣金已退回账户。"
		}
		if remark != "" {
			r := sanitize.Text(remark)
			w.ReviewRemark = &r
			sysMsg += "备注：" + r
		}
		w.ReviewedAt = &now
		if err := s.repos.Withdraw.Save(tx, w); err != nil {
			return err
		}
		// 提现流水同步三态（biz_type=1 行 status 0→1/2）
		if err := tx.Model(&model.CommissionLog{}).
			Where("order_no = ? AND biz_type = ?", fmt.Sprintf("w%d", w.ID), model.CommissionBizWithdraw).
			Updates(map[string]any{"status": w.Status, "confirmed_at": now}).Error; err != nil {
			return err
		}
		// 关闭工单 + 系统消息留痕
		if err := s.repos.Ticket.CreateMessage(tx, &model.TicketMessage{
			TicketID: ticketID, SenderType: 1, SenderID: adminID, Message: sysMsg,
		}); err != nil {
			return err
		}
		if err := s.repos.Ticket.UpdateStatus(tx, ticketID, 2); err != nil {
			return err
		}
		return s.audit(tx, adminID, action, fmt.Sprint(w.UserID), ip, map[string]any{
			"ticket_id": ticketID, "withdraw_id": w.ID,
			"amount": model.FenToYuan(w.Amount), "method": w.Method, "remark": sanitize.Text(remark),
		})
	})
}
