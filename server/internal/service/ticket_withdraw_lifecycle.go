package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// 提现工单生命周期深 Module：提交、审核、关闭的不变式唯一拥有者。
//
// 收敛前：不可关闭/不可重开靠注释 + 三处 Module 里的 if-guard 活着，
// 用户侧与管理侧详情各写一遍 info 助手，提交的四写只在 invite 侧。
// 收敛后：用户路径与管理路径走同一生命周期，卡住佣金的情况无法表达——
// 提交即扣佣（同事务行锁防双花）、提现工单不可由用户关闭/重开、
// 管理端只能经 pay/reject 审核闭环（直接關单被拒，被扣佣金不会长期停留）。
// 审计目标保持单个 ID（批量摘要 batch:<count> 仅批量侧，提现不混用）。
//
// Depth 所在：提交四写 + 审核三态 + 关/重开不变式的组合语义。

// guardUserCloseWithdraw 用户关闭前不变式：提现工单不可由用户关闭。
// 其生命周期由管理员 pay/reject 审核闭环（提交即扣减佣金），用户关闭后
// 管理端审核入口被禁用，佣金将长期停留在已扣减状态。
func guardUserCloseWithdraw(t *model.Ticket) error {
	if t.Type == model.TicketTypeWithdraw {
		return errs.ErrTicketWithdrawClose
	}
	return nil
}

// guardUserReopenWithdraw 用户重开前不变式：提现工单同样不可重开。
// 审核完成（pay/reject）后资金闭环，重开只会误导管理端。
func guardUserReopenWithdraw(t *model.Ticket) error {
	if t.Type == model.TicketTypeWithdraw {
		return errs.ErrTicketWithdrawClose
	}
	return nil
}

// guardAdminCloseWithdraw 管理端直接關单前不变式：提现工单必须走 pay/reject
// 审核闭环，直接關单会绕过打款/退回记账，被扣佣金无处可去。
func guardAdminCloseWithdraw(t *model.Ticket) error {
	if t.Type == model.TicketTypeWithdraw {
		return errs.ErrTicketWithdrawClose
	}
	return nil
}

// withdrawInfo 提现工单附带的结构化提现信息唯一读 seam：
// 非提现工单返回 nil，查询失败降级为 nil 不阻断详情。
func withdrawInfo(db *gorm.DB, t *model.Ticket) *model.TicketWithdrawInfo {
	if t.Type != model.TicketTypeWithdraw {
		return nil
	}
	w, err := (repo.WithdrawRepo{}).GetByTicketID(db, t.ID)
	if err != nil {
		return nil
	}
	return &model.TicketWithdrawInfo{
		ID: w.ID, UserID: w.UserID, Amount: model.FenToYuan(w.Amount),
		Method: w.Method, Account: w.Account, Status: w.Status,
		ReviewRemark: w.ReviewRemark, ReviewedAt: w.ReviewedAt, CreatedAt: w.CreatedAt,
	}
}

// submitWithdrawTx 生命周期·提交：行锁查用户 → 代理身份 → 余额 → 提交即扣减 →
// 四写（工单/首条结构化消息/提现单/提现流水）。调用方须先经 money 深 Module
// checked 金额（非法金额在此即不可表达），本函数内不再二次解析元。
func submitWithdrawTx(tx *gorm.DB, repos *repo.Repos, userID, amountFen int64, method, account string) (*model.WithdrawItem, error) {
	user, err := repos.User.GetByIDForUpdate(tx, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if user.Role != model.RoleAgent {
		return nil, errs.ErrWithdrawForbidden
	}
	if user.CommissionBalance < amountFen {
		return nil, errs.ErrCommissionInsufficient
	}
	// 提交即扣减：重复提交相同金额受余额拦截（防双花）
	user.CommissionBalance -= amountFen
	if err := repos.User.Save(tx, user); err != nil {
		return nil, err
	}
	now := time.Now()
	ticket := &model.Ticket{
		UserID: userID, Subject: "佣金提现申请", Level: 0, Type: model.TicketTypeWithdraw,
		Status: 0, LastReplyAt: &now,
	}
	if err := repos.Ticket.Create(tx, ticket); err != nil {
		return nil, err
	}
	method = sanitize.Text(method)
	account = sanitize.Text(account)
	msg := fmt.Sprintf("【佣金提现申请】\n提现方式：%s\n收款账号：%s\n金额：%.2f 元",
		method, account, model.FenToYuan(amountFen))
	if err := repos.Ticket.CreateMessage(tx, &model.TicketMessage{
		TicketID: ticket.ID, SenderType: 0, SenderID: userID, Message: msg,
	}); err != nil {
		return nil, err
	}
	w := &model.CommissionWithdraw{
		UserID: userID, TicketID: ticket.ID, Amount: amountFen,
		Method: method, Account: account, Status: model.WithdrawPending,
	}
	if err := repos.Withdraw.Create(tx, w); err != nil {
		return nil, err
	}
	// 提现流水入账（order_no='w<提现单ID>'；仅 biz_type=0 受订单号唯一约束）
	if err := repos.Commission.Create(tx, &model.CommissionLog{
		InviteUserID: userID, FromUserID: userID,
		OrderNo: fmt.Sprintf("w%d", w.ID), Amount: amountFen,
		BizType: model.CommissionBizWithdraw, Status: model.WithdrawPending,
	}); err != nil {
		return nil, err
	}
	return toWithdrawItem(w), nil
}

// withdrawReviewOut 生命周期·审核的域输出：调用方（admin）据此写审计，
// 目标保持单个用户 ID（提现不混用批量摘要 batch:<count>）。
type withdrawReviewOut struct {
	action    string // withdraw_pay / withdraw_reject
	userID    int64
	withdrawID int64
	amountFen int64
	method    string
}

// reviewWithdrawTx 生命周期·审核：取工单 → 取提现单（行锁）→ pay 确认打款 /
// reject 自动退回佣金（提交时已扣减，拒绝必须原路退回防资损）→ 流水同步三态 →
// 關单 + 系统消息留痕。DB 操作顺序与历史一致（mock 顺序敏感）。
func reviewWithdrawTx(tx *gorm.DB, repos *repo.Repos, ticketID, adminID int64, approve bool, remark string) (*withdrawReviewOut, error) {
	ticket, err := repos.Ticket.GetByID(tx, ticketID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if ticket.Type != model.TicketTypeWithdraw {
		return nil, errs.ErrConflict
	}
	w, err := repos.Withdraw.GetByTicketIDForUpdate(tx, ticketID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if w.Status != model.WithdrawPending {
		return nil, errs.ErrWithdrawStatus
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
		user, err := repos.User.GetByIDForUpdate(tx, w.UserID)
		if err != nil {
			return nil, err
		}
		user.CommissionBalance += w.Amount
		if err := repos.User.Save(tx, user); err != nil {
			return nil, err
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
	if err := repos.Withdraw.Save(tx, w); err != nil {
		return nil, err
	}
	// 提现流水同步三态（biz_type=1 行 status 0→1/2）
	if err := tx.Model(&model.CommissionLog{}).
		Where("order_no = ? AND biz_type = ?", fmt.Sprintf("w%d", w.ID), model.CommissionBizWithdraw).
		Updates(map[string]any{"status": w.Status, "confirmed_at": now}).Error; err != nil {
		return nil, err
	}
	// 关闭工单 + 系统消息留痕
	if err := repos.Ticket.CreateMessage(tx, &model.TicketMessage{
		TicketID: ticketID, SenderType: 1, SenderID: adminID, Message: sysMsg,
	}); err != nil {
		return nil, err
	}
	if err := repos.Ticket.UpdateStatus(tx, ticketID, 2); err != nil {
		return nil, err
	}
	return &withdrawReviewOut{
		action: action, userID: w.UserID, withdrawID: w.ID,
		amountFen: w.Amount, method: w.Method,
	}, nil
}
