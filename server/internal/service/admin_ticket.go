package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// 工单域 module：列表/详情/回复/关闭。
// 提现附带信息经由 withdraw 缝合读取，收敛 ticket-withdraw 散落。

// ListTickets GET /admin/tickets。
func (m *adminTicketModule) ListTickets(ctx context.Context, page, pageSize int) ([]model.AdminTicketItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := m.repos.Ticket.ListByPage(m.db, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(list))
	for _, t := range list {
		ids = append(ids, t.UserID)
	}
	emails := m.withdraw.audit.emailsOf(ids)
	out := make([]model.AdminTicketItem, 0, len(list))
	for _, t := range list {
		out = append(out, model.AdminTicketItem{
			ID: t.ID, UserID: t.UserID, UserEmail: emails[t.UserID], Subject: t.Subject,
			Level: t.Level, Type: t.Type, Status: t.Status, ReopenCount: t.ReopenCount, LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
		})
	}
	return out, total, nil
}

// GetTicketDetail GET /admin/tickets/{id}。
func (m *adminTicketModule) GetTicketDetail(ctx context.Context, id int64) (*model.TicketDetailResp, error) {
	t, err := m.repos.Ticket.GetByID(m.db, id)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	messages, err := m.repos.Ticket.MessagesByTicket(m.db, id)
	if err != nil {
		return nil, err
	}
	msgs := make([]model.TicketMsgResp, 0, len(messages))
	for _, msg := range messages {
		msgs = append(msgs, model.TicketMsgResp{ID: msg.ID, SenderType: msg.SenderType, Message: msg.Message, CreatedAt: msg.CreatedAt})
	}
	return &model.TicketDetailResp{ID: t.ID, Subject: t.Subject, Level: t.Level, Type: t.Type,
		Status: t.Status, CreatedAt: t.CreatedAt, Messages: msgs, Withdraw: m.withdraw.withdrawDetail(t),
	}, nil
}

// ReplyTicket POST /admin/tickets/{id}/reply（客服回复 → 状态已回复）。
func (m *adminTicketModule) ReplyTicket(ctx context.Context, adminID, id int64, message string) error {
	t, err := m.repos.Ticket.GetByID(m.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	if t.Status == 2 {
		return errs.ErrTicketClosed
	}
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		now := time.Now()
		if err := m.repos.Ticket.CreateMessage(tx, &model.TicketMessage{
			TicketID: id, SenderType: 1, SenderID: adminID, Message: sanitize.Text(message),
		}); err != nil {
			return err
		}
		if err := m.repos.Ticket.UpdateStatus(tx, id, 1); err != nil {
			return err
		}
		return m.repos.Ticket.UpdateLastReplyAt(tx, id, now)
	})
}

// CloseTicket POST /admin/tickets/{id}/close。
func (m *adminTicketModule) CloseTicket(ctx context.Context, id int64) error {
	t, err := m.repos.Ticket.GetByID(m.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	if t.Status == 2 {
		return errs.ErrTicketClosed
	}
	return m.repos.Ticket.UpdateStatus(m.db, id, 2)
}
