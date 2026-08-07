package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink/internal/model"
	"ylink/internal/pkg/errs"
	"ylink/internal/pkg/sanitize"
	"ylink/internal/repo"
)

// TicketService 工单域。
type TicketService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
}

func NewTicketService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos) *TicketService {
	return &TicketService{db: db, rdb: rdb, repos: repos}
}

// List GET /tickets。
func (s *TicketService) List(ctx context.Context, userID int64, page, pageSize int) ([]model.TicketListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.Ticket.ListByUser(s.db, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.TicketListItem, 0, len(list))
	for _, t := range list {
		out = append(out, model.TicketListItem{
			ID: t.ID, Subject: t.Subject, Level: t.Level, Status: t.Status,
			LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
		})
	}
	return out, total, nil
}

// Create POST /tickets（首条消息为用户发送）。
func (s *TicketService) Create(ctx context.Context, userID int64, req *model.CreateTicketReq) (*model.TicketListItem, error) {
	var ticket *model.Ticket
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		now := time.Now()
		t := &model.Ticket{UserID: userID, Subject: req.Subject, Level: req.Level, Status: 0, LastReplyAt: &now}
		if err := s.repos.Ticket.Create(tx, t); err != nil {
			return err
		}
		msg := &model.TicketMessage{TicketID: t.ID, SenderType: 0, SenderID: userID, Message: sanitize.Text(req.Message)}
		if err := s.repos.Ticket.CreateMessage(tx, msg); err != nil {
			return err
		}
		ticket = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &model.TicketListItem{
		ID: ticket.ID, Subject: ticket.Subject, Level: ticket.Level, Status: ticket.Status,
		LastReplyAt: ticket.LastReplyAt, CreatedAt: ticket.CreatedAt,
	}, nil
}

// Detail GET /tickets/{id}。
func (s *TicketService) Detail(ctx context.Context, userID int64, id int64) (*model.TicketDetailResp, error) {
	t, err := s.repos.Ticket.GetByIDAndUser(s.db, id, userID)
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
	return &model.TicketDetailResp{
		ID: t.ID, Subject: t.Subject, Level: t.Level, Status: t.Status, CreatedAt: t.CreatedAt, Messages: msgs,
	}, nil
}

// Reply POST /tickets/{id}/reply 用户回复：状态回到待回复。
func (s *TicketService) Reply(ctx context.Context, userID int64, id int64, message string) error {
	t, err := s.repos.Ticket.GetByIDAndUser(s.db, id, userID)
	if err != nil {
		return errs.ErrNotFound
	}
	if t.Status == 2 {
		return errs.ErrTicketClosed
	}
	err = repo.WithTx(s.db, func(tx *gorm.DB) error {
		now := time.Now()
		msg := &model.TicketMessage{TicketID: id, SenderType: 0, SenderID: userID, Message: sanitize.Text(message)}
		if err := s.repos.Ticket.CreateMessage(tx, msg); err != nil {
			return err
		}
		if err := s.repos.Ticket.UpdateStatus(tx, id, 0); err != nil {
			return err
		}
		return s.repos.Ticket.UpdateLastReplyAt(tx, id, now)
	})
	return err
}

// Close POST /tickets/{id}/close。
func (s *TicketService) Close(ctx context.Context, userID int64, id int64) (*model.TicketListItem, error) {
	t, err := s.repos.Ticket.GetByIDAndUser(s.db, id, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if t.Status == 2 {
		return nil, errs.ErrTicketClosed
	}
	if err := s.repos.Ticket.UpdateStatus(s.db, id, 2); err != nil {
		return nil, err
	}
	t.Status = 2
	return &model.TicketListItem{
		ID: t.ID, Subject: t.Subject, Level: t.Level, Status: t.Status,
		LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
	}, nil
}
