package repo

import (
	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// NoticeRepo 公告数据访问。
type NoticeRepo struct{}

// ListByPage 上架公告分页（sort 升序优先，同序值按创建时间倒序；F15 排序即时生效）。
func (NoticeRepo) ListByPage(db *gorm.DB, page, pageSize int) ([]model.Notice, int64, error) {
	var list []model.Notice
	var total int64
	q := db.Model(&model.Notice{}).Where("is_show = true")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("sort ASC, created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// KnowledgeRepo 知识库数据访问。
type KnowledgeRepo struct{}

// List 按语言/关键字查询（标题模糊），返回全部（分组在 service 层）。
func (KnowledgeRepo) List(db *gorm.DB, language, keyword string) ([]model.Knowledge, error) {
	var list []model.Knowledge
	q := db.Where("is_show = true AND language = ?", language)
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
	if err := db.Where("id = ? AND is_show = true", id).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

// KnowledgeCategoryRepo 知识库分类数据访问（F15）。
type KnowledgeCategoryRepo struct{}

// ListByLanguage 按语言取分类（sort 升序，同序值按 id）。
func (KnowledgeCategoryRepo) ListByLanguage(db *gorm.DB, language string) ([]model.KnowledgeCategory, error) {
	var list []model.KnowledgeCategory
	err := db.Where("language = ?", language).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// ListAll 全部分类（管理端）。
func (KnowledgeCategoryRepo) ListAll(db *gorm.DB) ([]model.KnowledgeCategory, error) {
	var list []model.KnowledgeCategory
	err := db.Order("language ASC, sort ASC, id ASC").Find(&list).Error
	return list, err
}

// GetByID 单个分类。
func (KnowledgeCategoryRepo) GetByID(db *gorm.DB, id int64) (*model.KnowledgeCategory, error) {
	var kc model.KnowledgeCategory
	if err := db.First(&kc, id).Error; err != nil {
		return nil, err
	}
	return &kc, nil
}

// GetByLanguageAndName 按语言 + 名称取分类（知识保存时归类）。
func (KnowledgeCategoryRepo) GetByLanguageAndName(db *gorm.DB, language, name string) (*model.KnowledgeCategory, error) {
	var kc model.KnowledgeCategory
	if err := db.Where("language = ? AND name = ?", language, name).First(&kc).Error; err != nil {
		return nil, err
	}
	return &kc, nil
}

// Create 新建分类。
func (KnowledgeCategoryRepo) Create(db *gorm.DB, kc *model.KnowledgeCategory) error {
	return db.Select("language", "name", "sort", "created_at", "updated_at").Create(kc).Error
}

// UpdateMap 更新分类字段。
func (KnowledgeCategoryRepo) UpdateMap(db *gorm.DB, id int64, updates map[string]any) error {
	return db.Model(&model.KnowledgeCategory{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除分类（仅当无知识文档引用时由 service 层调用）。
func (KnowledgeCategoryRepo) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&model.KnowledgeCategory{}, id).Error
}

// CountKnowledges 分类下知识文档数量（删除前校验 + 管理端列表展示）。
func (KnowledgeCategoryRepo) CountKnowledges(db *gorm.DB, categoryID int64) (int64, error) {
	var n int64
	err := db.Model(&model.Knowledge{}).Where("category_id = ?", categoryID).Count(&n).Error
	return n, err
}

// CountByCategoryIDs 批量取各分类的知识文档数量。
func (KnowledgeCategoryRepo) CountByCategoryIDs(db *gorm.DB, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		CategoryID int64
		Cnt        int64
	}
	err := db.Model(&model.Knowledge{}).
		Select("category_id, COUNT(*) AS cnt").
		Where("category_id IN ?", ids).
		Group("category_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.CategoryID] = r.Cnt
	}
	return out, nil
}

// PlanRepo 套餐数据访问。
type PlanRepo struct{}

// ListShown 上架套餐（按 sort）。
func (PlanRepo) ListShown(db *gorm.DB) ([]model.Plan, error) {
	var list []model.Plan
	err := db.Where("is_show = true").Order("sort ASC, id ASC").Find(&list).Error
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
	err := db.Where("is_show = true AND group_id IN ?", groupIDs).
		Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// ListGroups 全部节点分组。
func (ServerRepo) ListGroups(db *gorm.DB) ([]model.ServerGroup, error) {
	var list []model.ServerGroup
	err := db.Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}
