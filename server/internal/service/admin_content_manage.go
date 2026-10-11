package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// 内容域 module：公告/知识/分类/排序（F15 收敛一处）。
// 公告与知识的增删改查、排序、分类归并均落于此 module。

// SortNotices F15 公告排序：items 按 sort 值更新（前端按展示顺序生成），单事务 + 审计。
func (m *adminContentModule) SortNotices(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		for _, it := range items {
			if err := m.repos.Notice.UpdateMap(tx, it.ID, map[string]any{"sort": it.Sort}); err != nil {
				return err
			}
		}
		return m.audit.write(tx, adminID, "sort_notice", "", ip, map[string]any{"count": len(items)})
	})
}

// SortKnowledges F15 知识库排序：items 按 sort 值更新，单事务 + 审计。
func (m *adminContentModule) SortKnowledges(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		for _, it := range items {
			if err := m.repos.Knowledge.UpdateMap(tx, it.ID, map[string]any{"sort": it.Sort}); err != nil {
				return err
			}
		}
		return m.audit.write(tx, adminID, "sort_knowledge", "", ip, map[string]any{"count": len(items)})
	})
}

// ListKnowledgeCategories F15 分类列表：language 为空返回全部，附各分类知识文档计数。
func (m *adminContentModule) ListKnowledgeCategories(ctx context.Context, language string) ([]model.AdminKnowledgeCategoryItem, error) {
	var cats []model.KnowledgeCategory
	var err error
	if language == "" {
		cats, err = m.repos.KnowledgeCat.ListAll(m.db)
	} else {
		cats, err = m.repos.KnowledgeCat.ListByLanguage(m.db, language)
	}
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(cats))
	for _, c := range cats {
		ids = append(ids, c.ID)
	}
	counts, err := m.repos.KnowledgeCat.CountByCategoryIDs(m.db, ids)
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
func (m *adminContentModule) CreateKnowledgeCategory(ctx context.Context, adminID int64, req *model.AdminKnowledgeCategoryReq, ip string) (*model.KnowledgeCategory, error) {
	name := sanitize.Text(req.Name)
	if name == "" {
		return nil, errs.New(40000, "参数校验失败: 分类名称不能为空")
	}
	if _, err := m.repos.KnowledgeCat.GetByLanguageAndName(m.db, req.Language, name); err == nil {
		return nil, errs.New(40000, "该语言下已存在同名分类")
	}
	kc := &model.KnowledgeCategory{Language: req.Language, Name: name}
	if req.Sort != nil {
		kc.Sort = *req.Sort
	}
	if err := m.repos.KnowledgeCat.Create(m.db, kc); err != nil {
		return nil, err
	}
	_ = m.audit.write(m.db, adminID, "create_knowledge_category", fmt.Sprint(kc.ID), ip, map[string]any{
		"language": kc.Language, "name": kc.Name,
	})
	return kc, nil
}

// UpdateKnowledgeCategory F15 更新分类（改名级联同步知识文档的展示分类字符串）。
func (m *adminContentModule) UpdateKnowledgeCategory(ctx context.Context, adminID, id int64, req *model.AdminKnowledgeCategoryUpdateReq, ip string) error {
	kc, err := m.repos.KnowledgeCat.GetByID(m.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	name := sanitize.Text(req.Name)
	if name == "" {
		return errs.New(40000, "参数校验失败: 分类名称不能为空")
	}
	if dup, err := m.repos.KnowledgeCat.GetByLanguageAndName(m.db, kc.Language, name); err == nil && dup.ID != id {
		return errs.New(40000, "该语言下已存在同名分类")
	}
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		updates := map[string]any{"name": name}
		if req.Sort != nil {
			updates["sort"] = *req.Sort
		}
		if err := m.repos.KnowledgeCat.UpdateMap(tx, id, updates); err != nil {
			return err
		}
		// 改名级联：归属该分类的知识文档展示分类字符串同步更新（用户端按字符串分组）
		if name != kc.Name {
			if err := tx.Model(&model.Knowledge{}).Where("category_id = ?", id).
				Update("category", name).Error; err != nil {
				return err
			}
		}
		return m.audit.write(tx, adminID, "update_knowledge_category", fmt.Sprint(id), ip, map[string]any{
			"language": kc.Language, "name": name,
		})
	})
}

// DeleteKnowledgeCategory F15 删除分类：分类下仍有知识文档时拒绝（防文档归类悬空）。
func (m *adminContentModule) DeleteKnowledgeCategory(ctx context.Context, adminID, id int64, ip string) error {
	kc, err := m.repos.KnowledgeCat.GetByID(m.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	n, err := m.repos.KnowledgeCat.CountKnowledges(m.db, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.New(40000, "该分类下仍有知识文档，请先移动文档后再删除")
	}
	if err := m.repos.KnowledgeCat.Delete(m.db, id); err != nil {
		return err
	}
	_ = m.audit.write(m.db, adminID, "delete_knowledge_category", fmt.Sprint(id), ip, map[string]any{
		"language": kc.Language, "name": kc.Name,
	})
	return nil
}

func (m *adminContentModule) CreateNotice(ctx context.Context, req *model.AdminNoticeReq) (*model.Notice, error) {
	n := &model.Notice{
		Title: sanitize.Text(req.Title), Content: sanitize.Markdown(req.Content), IsShow: true, Sort: req.Sort,
	}
	if req.IsShow != nil {
		n.IsShow = *req.IsShow
	}
	if err := m.repos.Notice.Create(m.db, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (m *adminContentModule) UpdateNotice(ctx context.Context, id int64, req *model.AdminNoticeReq) error {
	if _, err := m.repos.Notice.GetByID(m.db, id); err != nil {
		return errs.ErrNotFound
	}
	updates := map[string]any{
		"title":   sanitize.Text(req.Title),
		"content": sanitize.Markdown(req.Content),
		"sort":    req.Sort,
	}
	if req.IsShow != nil {
		updates["is_show"] = *req.IsShow
	}
	return m.repos.Notice.UpdateMap(m.db, id, updates)
}

func (m *adminContentModule) DeleteNotice(ctx context.Context, id int64) error {
	return m.repos.Notice.Delete(m.db, id)
}

// resolveKnowledgeCategory F15：按 (language, name) 找到或创建分类，返回分类 ID 与展示名。
// 显式 categoryID 优先；无匹配 ID 时按名称归并（首次出现的新分类自动建行，存量数据兜底）。
func (m *adminContentModule) resolveKnowledgeCategory(db *gorm.DB, language, name string, categoryID *int64) (catID *int64, displayName string, err error) {
	displayName = name
	if categoryID != nil && *categoryID > 0 {
		if kc, e := m.repos.KnowledgeCat.GetByID(db, *categoryID); e == nil {
			return &kc.ID, kc.Name, nil
		}
	}
	if name == "" {
		return nil, "", nil
	}
	if kc, e := m.repos.KnowledgeCat.GetByLanguageAndName(db, language, name); e == nil {
		return &kc.ID, kc.Name, nil
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, "", e
	}
	kc := &model.KnowledgeCategory{Language: language, Name: name}
	if err := m.repos.KnowledgeCat.Create(db, kc); err != nil {
		return nil, "", err
	}
	return &kc.ID, kc.Name, nil
}

func (m *adminContentModule) CreateKnowledge(ctx context.Context, req *model.AdminKnowledgeReq) (*model.Knowledge, error) {
	lang := req.Language
	if lang == "" {
		lang = "zh-CN"
	}
	catName := sanitize.Text(req.Category)
	catID, displayName, err := m.resolveKnowledgeCategory(m.db, lang, catName, req.CategoryID)
	if err != nil {
		return nil, err
	}
	k := &model.Knowledge{
		Category: displayName, CategoryID: catID,
		Title: sanitize.Text(req.Title),
		Body:  sanitize.Markdown(req.Body), Language: lang, IsShow: true, Sort: req.Sort,
	}
	if req.IsShow != nil {
		k.IsShow = *req.IsShow
	}
	if err := m.repos.Knowledge.Create(m.db, k); err != nil {
		return nil, err
	}
	return k, nil
}

func (m *adminContentModule) UpdateKnowledge(ctx context.Context, id int64, req *model.AdminKnowledgeReq) error {
	k, err := m.repos.Knowledge.GetByID(m.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	lang := req.Language
	if lang == "" {
		lang = k.Language
	}
	catName := sanitize.Text(req.Category)
	// 未显式传 category_id 时沿用原归属（仅当分类名未变），避免改标题等字段丢失归类
	var catIDArg *int64
	if req.CategoryID != nil {
		catIDArg = req.CategoryID
	} else if catName == k.Category && k.CategoryID != nil {
		catIDArg = k.CategoryID
	}
	catID, displayName, err := m.resolveKnowledgeCategory(m.db, lang, catName, catIDArg)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"category":    displayName,
		"category_id": catID,
		"title":       sanitize.Text(req.Title),
		"body":        sanitize.Markdown(req.Body),
		"language":    lang,
		"sort":        req.Sort,
	}
	if req.IsShow != nil {
		updates["is_show"] = *req.IsShow
	}
	return m.repos.Knowledge.UpdateMap(m.db, id, updates)
}

func (m *adminContentModule) DeleteKnowledge(ctx context.Context, id int64) error {
	return m.repos.Knowledge.Delete(m.db, id)
}

// ListAllNotices GET /admin/notices（含隐藏,创建时间倒序）。
func (m *adminContentModule) ListAllNotices(ctx context.Context) ([]model.AdminNoticeItem, error) {
	var list []model.Notice
	if err := m.db.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]model.AdminNoticeItem, 0, len(list))
	for _, n := range list {
		out = append(out, model.AdminNoticeItem{
			ID: n.ID, Title: n.Title, Content: n.Content, IsShow: n.IsShow, Sort: n.Sort, CreatedAt: n.CreatedAt,
		})
	}
	return out, nil
}

// ListAllKnowledges GET /admin/knowledges（含隐藏）。
func (m *adminContentModule) ListAllKnowledges(ctx context.Context) ([]model.AdminKnowledgeItem, error) {
	var list []model.Knowledge
	if err := m.db.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]model.AdminKnowledgeItem, 0, len(list))
	for _, k := range list {
		out = append(out, model.AdminKnowledgeItem{
			ID: k.ID, Category: k.Category, Title: k.Title, Body: k.Body,
			Language: k.Language, IsShow: k.IsShow, Sort: k.Sort, UpdatedAt: k.UpdatedAt,
		})
	}
	return out, nil
}
