package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// ---- 管理端 · 节点批量操作 / 复制 / 排序（F09） ----

// BatchServers F09 节点批量操作：delete=删除；update=批量更新公共字段（status/is_show/group_id/rate）。
// 整批单事务执行，逐节点记录成功/失败（不存在或更新失败记入 failed，不打断其余节点），整体写审计。
func (s *AdminService) BatchServers(ctx context.Context, adminID int64, req *model.AdminBatchServerReq, ip string) (*model.AdminBatchServerResp, error) {
	updates := map[string]any{}
	if req.Action == "update" {
		if req.Status != nil {
			updates["status"] = *req.Status
		}
		if req.IsShow != nil {
			updates["is_show"] = *req.IsShow
		}
		if req.GroupID != nil {
			updates["group_id"] = *req.GroupID
		}
		if req.Rate != nil {
			if *req.Rate <= 0 {
				return nil, errs.New(40000, "参数校验失败: rate 须为正数")
			}
			updates["rate"] = *req.Rate
		}
		if len(updates) == 0 {
			return nil, errs.New(40000, "参数校验失败: update 至少提供一项待更新字段")
		}
	}
	resp := &model.AdminBatchServerResp{Failed: make([]model.AdminBatchFailedItem, 0)}
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		for _, id := range req.IDs {
			var res *gorm.DB
			if req.Action == "delete" {
				res = tx.Where("id = ?", id).Delete(&model.Server{})
			} else {
				res = tx.Model(&model.Server{}).Where("id = ?", id).Updates(updates)
			}
			if res.Error != nil {
				resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{ID: id, Reason: "操作失败"})
				continue
			}
			if res.RowsAffected == 0 {
				resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{ID: id, Reason: "节点不存在"})
				continue
			}
			resp.Success++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	action := "batch_server_update"
	if req.Action == "delete" {
		action = "batch_server_delete"
	}
	_ = s.audit(s.db, adminID, action, "", ip, map[string]any{
		"ids": req.IDs, "success": resp.Success, "failed": len(resp.Failed), "updates": updates,
	})
	return resp, nil
}

// CopyServer F09 复制节点：除名称追加 -copy 外全字段复制，重新生成 node_key（不与源节点共享）。
func (s *AdminService) CopyServer(ctx context.Context, adminID, serverID int64, ip string) (*model.AdminServerView, error) {
	src, err := s.repos.Server.GetByID(s.db, serverID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	cp := *src
	cp.ID = 0
	cp.Name = truncateServerName(src.Name + "-copy")
	cp.NodeKey = newNodeKey()
	if err := s.repos.Server.Create(s.db, &cp); err != nil {
		return nil, err
	}
	_ = s.audit(s.db, adminID, "copy_server", fmt.Sprint(serverID), ip, map[string]any{
		"new_id": cp.ID, "name": cp.Name,
	})
	view := toServerView(&cp)
	return &view, nil
}

// truncateServerName 复制节点名称截断到列宽（VARCHAR(64)）。
func truncateServerName(name string) string {
	runes := []rune(name)
	if len(runes) <= 64 {
		return name
	}
	return string(runes[:64])
}

// SortServers F09 节点排序：按传入 items 更新 sort（前端按展示顺序生成），单事务 + 审计。
func (s *AdminService) SortServers(ctx context.Context, adminID int64, items []model.AdminSortItem, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		for _, it := range items {
			if err := s.repos.Server.UpdateMap(tx, it.ID, map[string]any{"sort": it.Sort}); err != nil {
				return err
			}
		}
		return s.audit(tx, adminID, "sort_server", "", ip, map[string]any{"count": len(items)})
	})
}
