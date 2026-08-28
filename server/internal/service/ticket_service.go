package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
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
			ID: t.ID, Subject: t.Subject, Level: t.Level, Type: t.Type, Status: t.Status, ReopenCount: t.ReopenCount,
			LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
		})
	}
	return out, total, nil
}

// ticketWithdrawInfo 提现工单附带的结构化提现信息（非提现工单返回 nil，查询失败降级为 nil 不阻断详情）。
func ticketWithdrawInfo(db *gorm.DB, t *model.Ticket) *model.TicketWithdrawInfo {
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
		ID: ticket.ID, Subject: ticket.Subject, Level: ticket.Level, Type: ticket.Type,
		Status: ticket.Status, ReopenCount: ticket.ReopenCount,
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
		ID: t.ID, Subject: t.Subject, Level: t.Level, Type: t.Type, Status: t.Status, ReopenCount: t.ReopenCount,
		CreatedAt: t.CreatedAt, Messages: msgs, Withdraw: ticketWithdrawInfo(s.db, t),
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
// 提现工单（type=1）不可由用户关闭：其生命周期由管理员 pay/reject 审核闭环
// （提交即扣减佣金），用户关闭后管理端审核入口被禁用，佣金将长期停留在已扣减状态。
func (s *TicketService) Close(ctx context.Context, userID int64, id int64) (*model.TicketListItem, error) {
	t, err := s.repos.Ticket.GetByIDAndUser(s.db, id, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if t.Type == model.TicketTypeWithdraw {
		return nil, errs.ErrTicketWithdrawClose
	}
	if t.Status == 2 {
		return nil, errs.ErrTicketClosed
	}
	if err := s.repos.Ticket.UpdateStatus(s.db, id, 2); err != nil {
		return nil, err
	}
	t.Status = 2
	return &model.TicketListItem{
		ID: t.ID, Subject: t.Subject, Level: t.Level, Type: t.Type, Status: t.Status, ReopenCount: t.ReopenCount,
		LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
	}, nil
}

// Reopen POST /tickets/{id}/reopen:已关闭的工单可重开一次(core-flows §7)。
// 重开后状态回「待回复」,reopen_count+1;已重开过或未关闭则拒绝。
// 提现工单（type=1）同样不可重开:审核完成（pay/reject）后资金闭环,重开只会误导管理端。
func (s *TicketService) Reopen(ctx context.Context, userID int64, id int64) (*model.TicketListItem, error) {
	t, err := s.repos.Ticket.GetByIDAndUser(s.db, id, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if t.Type == model.TicketTypeWithdraw {
		return nil, errs.ErrTicketWithdrawClose
	}
	if t.Status != 2 {
		return nil, errs.ErrConflict
	}
	if t.ReopenCount >= 1 {
		return nil, errs.ErrTicketReopenLimit
	}
	now := time.Now()
	updated, err := s.repos.Ticket.UpdateReopen(s.db, id, now)
	if err != nil {
		return nil, err
	}
	// 0 行:并发下已被重开(读到的旧快照),按已重开处理
	if !updated {
		return nil, errs.ErrTicketReopenLimit
	}
	t.Status = 0
	t.ReopenCount = 1
	t.LastReplyAt = &now
	return &model.TicketListItem{
		ID: t.ID, Subject: t.Subject, Level: t.Level, Type: t.Type, Status: t.Status, ReopenCount: t.ReopenCount,
		LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
	}, nil
}
