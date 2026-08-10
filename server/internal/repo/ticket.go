package repo

import (
	"time"

	"gorm.io/gorm"

	"ylink/internal/model"
)

// TicketRepo 工单数据访问。
type TicketRepo struct{}

func (TicketRepo) Create(db *gorm.DB, t *model.Ticket) error { return db.Create(t).Error }

func (TicketRepo) CreateMessage(db *gorm.DB, m *model.TicketMessage) error {
	return db.Create(m).Error
}

func (TicketRepo) ListByUser(db *gorm.DB, userID int64, page, pageSize int) ([]model.Ticket, int64, error) {
	var list []model.Ticket
	var total int64
	q := db.Model(&model.Ticket{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (TicketRepo) GetByIDAndUser(db *gorm.DB, id, userID int64) (*model.Ticket, error) {
	var t model.Ticket
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (TicketRepo) GetByID(db *gorm.DB, id int64) (*model.Ticket, error) {
	var t model.Ticket
	if err := db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (TicketRepo) MessagesByTicket(db *gorm.DB, ticketID int64) ([]model.TicketMessage, error) {
	var list []model.TicketMessage
	err := db.Where("ticket_id = ?", ticketID).Order("id ASC").Find(&list).Error
	return list, err
}

func (TicketRepo) UpdateStatus(db *gorm.DB, id int64, status int) error {
	return db.Model(&model.Ticket{}).Where("id = ?", id).Update("status", status).Error
}

func (TicketRepo) UpdateLastReplyAt(db *gorm.DB, id int64, at time.Time) error {
	return db.Model(&model.Ticket{}).Where("id = ?", id).Update("last_reply_at", at).Error
}
