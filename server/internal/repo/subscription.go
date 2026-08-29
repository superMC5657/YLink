package repo

import (
	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// SubscriptionTemplateRepo 自定义订阅模板数据访问（F10）。
type SubscriptionTemplateRepo struct{}

// Get 按客户端类型取模板（无自定义行时返回 err，由 service 层回退内置生成器）。
func (SubscriptionTemplateRepo) Get(db *gorm.DB, name string) (*model.SubscriptionTemplate, error) {
	var st model.SubscriptionTemplate
	if err := db.Where("name = ?", name).First(&st).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// Upsert 保存自定义模板（存在则更新，不存在则写入）。
func (SubscriptionTemplateRepo) Upsert(db *gorm.DB, st *model.SubscriptionTemplate) error {
	return db.Save(st).Error
}

// Delete 删除自定义模板（恢复内置生成器）。
func (SubscriptionTemplateRepo) Delete(db *gorm.DB, name string) error {
	return db.Where("name = ?", name).Delete(&model.SubscriptionTemplate{}).Error
}
