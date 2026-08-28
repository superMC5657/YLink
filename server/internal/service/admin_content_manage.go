package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// ---- 管理端 · 公告 / 知识库排序与分类（F15） ----

// SortNotices F15 公告排序：items 按 sort 值更新（前端按展示顺序生成），单事务 + 审计。
func (s *AdminService) SortNotices(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		for _, it := range items {
			if err := s.repos.Notice.UpdateMap(tx, it.ID, map[string]any{"sort": it.Sort}); err != nil {
				return err
			}
		}
		return s.audit(tx, adminID, "sort_notice", "", ip, map[string]any{"count": len(items)})
	})
}

// SortKnowledges F15 知识库排序：items 按 sort 值更新，单事务 + 审计。
func (s *AdminService) SortKnowledges(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		for _, it := range items {
			if err := s.repos.Knowledge.UpdateMap(tx, it.ID, map[string]any{"sort": it.Sort}); err != nil {
				return err
			}
		}
		return s.audit(tx, adminID, "sort_knowledge", "", ip, map[string]any{"count": len(items)})
	})
}

// ListKnowledgeCategories F15 分类列表：language 为空返回全部，附各分类知识文档计数。
func (s *AdminService) ListKnowledgeCategories(ctx context.Context, language string) ([]model.AdminKnowledgeCategoryItem, error) {
	var cats []model.KnowledgeCategory
	var err error
	if language == "" {
		cats, err = s.repos.KnowledgeCat.ListAll(s.db)
	} else {
		cats, err = s.repos.KnowledgeCat.ListByLanguage(s.db, language)
	}
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(cats))
	for _, c := range cats {
		ids = append(ids, c.ID)
	}
	counts, err := s.repos.KnowledgeCat.CountByCategoryIDs(s.db, ids)
	if err != nil {
		return nil, err
	}
	out := make([]model.AdminKnowledgeCategoryItem, 0, len(cats))
	for _, c := range cats {
		out = append(out, model.AdminKnowledgeCategoryItem{
			ID: c.ID, Language: c.Language, Name: c.Name, Sort: c.Sort,
			KnowledgeCount: counts[c.ID],
		})
	}
	return out, nil
}

// CreateKnowledgeCategory F15 新建分类（同语言同名冲突返回参数错误）。
func (s *AdminService) CreateKnowledgeCategory(ctx context.Context, adminID int64, req *model.AdminKnowledgeCategoryReq, ip string) (*model.KnowledgeCategory, error) {
	name := sanitize.Text(req.Name)
	if name == "" {
		return nil, errs.New(40000, "参数校验失败: 分类名称不能为空")
	}
	if _, err := s.repos.KnowledgeCat.GetByLanguageAndName(s.db, req.Language, name); err == nil {
		return nil, errs.New(40000, "该语言下已存在同名分类")
	}
	kc := &model.KnowledgeCategory{Language: req.Language, Name: name}
	if req.Sort != nil {
		kc.Sort = *req.Sort
	}
	if err := s.repos.KnowledgeCat.Create(s.db, kc); err != nil {
		return nil, err
	}
	_ = s.audit(s.db, adminID, "create_knowledge_category", fmt.Sprint(kc.ID), ip, map[string]any{
		"language": kc.Language, "name": kc.Name,
	})
	return kc, nil
}

// UpdateKnowledgeCategory F15 更新分类（改名级联同步知识文档的展示分类字符串）。
func (s *AdminService) UpdateKnowledgeCategory(ctx context.Context, adminID, id int64, req *model.AdminKnowledgeCategoryUpdateReq, ip string) error {
	kc, err := s.repos.KnowledgeCat.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	name := sanitize.Text(req.Name)
	if name == "" {
		return errs.New(40000, "参数校验失败: 分类名称不能为空")
	}
	if dup, err := s.repos.KnowledgeCat.GetByLanguageAndName(s.db, kc.Language, name); err == nil && dup.ID != id {
		return errs.New(40000, "该语言下已存在同名分类")
	}
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		updates := map[string]any{"name": name}
		if req.Sort != nil {
			updates["sort"] = *req.Sort
		}
		if err := s.repos.KnowledgeCat.UpdateMap(tx, id, updates); err != nil {
			return err
		}
		// 改名级联：归属该分类的知识文档展示分类字符串同步更新（用户端按字符串分组）
		if name != kc.Name {
			if err := tx.Model(&model.Knowledge{}).Where("category_id = ?", id).
				Update("category", name).Error; err != nil {
				return err
			}
		}
		return s.audit(tx, adminID, "update_knowledge_category", fmt.Sprint(id), ip, map[string]any{
			"language": kc.Language, "name": name,
		})
	})
}

// DeleteKnowledgeCategory F15 删除分类：分类下仍有知识文档时拒绝（防文档归类悬空）。
func (s *AdminService) DeleteKnowledgeCategory(ctx context.Context, adminID, id int64, ip string) error {
	kc, err := s.repos.KnowledgeCat.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	n, err := s.repos.KnowledgeCat.CountKnowledges(s.db, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.New(40000, "该分类下仍有知识文档，请先移动文档后再删除")
	}
	if err := s.repos.KnowledgeCat.Delete(s.db, id); err != nil {
		return err
	}
	_ = s.audit(s.db, adminID, "delete_knowledge_category", fmt.Sprint(id), ip, map[string]any{
		"language": kc.Language, "name": kc.Name,
	})
	return nil
}
