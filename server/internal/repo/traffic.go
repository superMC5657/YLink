package repo

import (
	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// TrafficLogRepo 流量日明细数据访问。
type TrafficLogRepo struct{}

// ListByRange 日期范围明细（升序）。
func (TrafficLogRepo) ListByRange(db *gorm.DB, userID int64, from, to string) ([]model.TrafficLog, error) {
	var list []model.TrafficLog
	err := db.Where("user_id = ? AND date >= ? AND date <= ?", userID, from, to).
		Order("date ASC").Find(&list).Error
	return list, err
}
