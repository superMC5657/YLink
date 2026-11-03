package repo

import (
	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// TrafficLogRepo 流量日明细数据访问。
type TrafficLogRepo struct{}

// ListByRange 日期范围明细（升序）。空结果保证非 nil：
// Go nil slice 经 JSON 序列化为 null，前端 list.length 判空会崩（ADR-0001，空值保证收口在后端）。
func (TrafficLogRepo) ListByRange(db *gorm.DB, userID int64, from, to string) ([]model.TrafficLog, error) {
	list := []model.TrafficLog{}
	err := db.Where("user_id = ? AND date >= ? AND date <= ?", userID, from, to).
		Order("date ASC").Find(&list).Error
	return list, err
}
