package repo

import (
	"time"

	"gorm.io/gorm"

	"ylink/internal/model"
)

// TicketRepo 工单数据访问。
type TicketRepo struct{}

func (TicketRepo) Create(db *gorm.DB, t *model.Ticket) error {
	// Select 强制写入 Level(0/1/2):零值 0 默认会被 GORM 跳过,
	// 落到数据库默认值 1 导致「低」变成「中」。
	return db.Select("user_id", "subject", "level", "status", "last_reply_at", "created_at").Create(t).Error
}

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

// UpdateReopen 重开工单:状态回「待回复」、reopen_count+1、刷新最近回复时间。
// 带条件原子更新(status=2 且 reopen_count=0),并发下只允许一次成功,
// 返回是否实际更新(0 行=已被并发重开/已重开过)。
func (TicketRepo) UpdateReopen(db *gorm.DB, id int64, at time.Time) (bool, error) {
	res := db.Model(&model.Ticket{}).
		Where("id = ? AND status = 2 AND reopen_count = 0", id).
		Updates(map[string]any{
			"status":        0,
			"reopen_count":  gorm.Expr("reopen_count + 1"),
			"last_reply_at": at,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (TicketRepo) UpdateLastReplyAt(db *gorm.DB, id int64, at time.Time) error {
	return db.Model(&model.Ticket{}).Where("id = ?", id).Update("last_reply_at", at).Error
}
