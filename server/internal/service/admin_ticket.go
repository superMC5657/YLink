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

// ---- 工单（管理端） ----

// ListTickets GET /admin/tickets。
func (s *AdminService) ListTickets(ctx context.Context, page, pageSize int) ([]model.AdminTicketItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.Ticket.ListByPage(s.db, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(list))
	for _, t := range list {
		ids = append(ids, t.UserID)
	}
	emails := s.emailsOf(ids)
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
func (s *AdminService) GetTicketDetail(ctx context.Context, id int64) (*model.TicketDetailResp, error) {
	t, err := s.repos.Ticket.GetByID(s.db, id)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	messages, err := s.repos.Ticket.MessagesByTicket(s.db, id)
	if err != nil {
		return nil, err
	}
	msgs := make([]model.TicketMsgResp, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, model.TicketMsgResp{ID: m.ID, SenderType: m.SenderType, Message: m.Message, CreatedAt: m.CreatedAt})
	}
	return &model.TicketDetailResp{ID: t.ID, Subject: t.Subject, Level: t.Level, Type: t.Type,
		Status: t.Status, CreatedAt: t.CreatedAt, Messages: msgs, Withdraw: ticketWithdrawInfo(s.db, t),
	}, nil
}

// ReplyTicket POST /admin/tickets/{id}/reply（客服回复 → 状态已回复）。
func (s *AdminService) ReplyTicket(ctx context.Context, adminID, id int64, message string) error {
	t, err := s.repos.Ticket.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	if t.Status == 2 {
		return errs.ErrTicketClosed
	}
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		now := time.Now()
		if err := s.repos.Ticket.CreateMessage(tx, &model.TicketMessage{
			TicketID: id, SenderType: 1, SenderID: adminID, Message: sanitize.Text(message),
		}); err != nil {
			return err
		}
		if err := s.repos.Ticket.UpdateStatus(tx, id, 1); err != nil {
			return err
		}
		return s.repos.Ticket.UpdateLastReplyAt(tx, id, now)
	})
}

// CloseTicket POST /admin/tickets/{id}/close。
func (s *AdminService) CloseTicket(ctx context.Context, id int64) error {
	t, err := s.repos.Ticket.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	if t.Status == 2 {
		return errs.ErrTicketClosed
	}
	return s.repos.Ticket.UpdateStatus(s.db, id, 2)
}
