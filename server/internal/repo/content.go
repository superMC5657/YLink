package repo

import (
	"gorm.io/gorm"

	"nanocloud/internal/model"
)

// NoticeRepo 公告数据访问。
type NoticeRepo struct{}

// ListByPage 上架公告分页（按创建时间倒序）。
func (NoticeRepo) ListByPage(db *gorm.DB, page, pageSize int) ([]model.Notice, int64, error) {
	var list []model.Notice
	var total int64
	q := db.Model(&model.Notice{}).Where("is_show = 1")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// KnowledgeRepo 知识库数据访问。
type KnowledgeRepo struct{}

// List 按语言/关键字查询（标题模糊），返回全部（分组在 service 层）。
func (KnowledgeRepo) List(db *gorm.DB, language, keyword string) ([]model.Knowledge, error) {
	var list []model.Knowledge
	q := db.Where("is_show = 1 AND language = ?", language)
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("title LIKE ?", like)
	}
	err := q.Order("sort ASC, updated_at DESC").Find(&list).Error
	return list, err
}

// GetByID 知识详情（仅上架）。
func (KnowledgeRepo) GetByID(db *gorm.DB, id int64) (*model.Knowledge, error) {
	var k model.Knowledge
	if err := db.Where("id = ? AND is_show = 1", id).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

// PlanRepo 套餐数据访问。
type PlanRepo struct{}

// ListShown 上架套餐（按 sort）。
func (PlanRepo) ListShown(db *gorm.DB) ([]model.Plan, error) {
	var list []model.Plan
	err := db.Where("is_show = 1").Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// GetByID 任意套餐。
func (PlanRepo) GetByID(db *gorm.DB, id int64) (*model.Plan, error) {
	var p model.Plan
	if err := db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ServerRepo 节点数据访问。
type ServerRepo struct{}

// ListByGroupIDs 分组内上架节点。
func (ServerRepo) ListByGroupIDs(db *gorm.DB, groupIDs []int64) ([]model.Server, error) {
	var list []model.Server
	err := db.Where("is_show = 1 AND group_id IN ?", groupIDs).
		Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// ListGroups 全部节点分组。
func (ServerRepo) ListGroups(db *gorm.DB) ([]model.ServerGroup, error) {
	var list []model.ServerGroup
	err := db.Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}
