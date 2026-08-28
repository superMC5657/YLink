package repo

import (
	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// MailTemplateRepo 自定义邮件模板数据访问（F11）。
type MailTemplateRepo struct{}

// Get 按名称取模板（无自定义行时返回 err，由 service 层回退内置文案）。
func (MailTemplateRepo) Get(db *gorm.DB, name string) (*model.MailTemplate, error) {
	var mt model.MailTemplate
	if err := db.Where("name = ?", name).First(&mt).Error; err != nil {
		return nil, err
	}
	return &mt, nil
}

// ListAll 全部自定义模板。
func (MailTemplateRepo) ListAll(db *gorm.DB) ([]model.MailTemplate, error) {
	var list []model.MailTemplate
	err := db.Order("name ASC").Find(&list).Error
	return list, err
}

// Upsert 保存自定义模板（存在则更新，不存在则写入）。
func (MailTemplateRepo) Upsert(db *gorm.DB, mt *model.MailTemplate) error {
	return db.Save(mt).Error
}

// Delete 删除自定义模板（恢复默认）。
func (MailTemplateRepo) Delete(db *gorm.DB, name string) error {
	return db.Where("name = ?", name).Delete(&model.MailTemplate{}).Error
}
