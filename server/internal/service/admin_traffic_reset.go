package service

import (
	"context"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// ---- 管理端 · 流量重置（F16） ----

// ResetTraffic F16 按用户批量重置流量，逐用户单事务执行并写入 traffic_reset_logs。
//
// 与节点上报差分幂等兼容（验收要点）：重置时**保留 node_user_stats 快照**——
// 快照存的是节点侧累计值，下次上报按「当前累计 − 快照」差分，只有重置之后的新流量
// 会计入 users.u/d；若清空快照，下一次上报会把节点全周期累计值整体重算一遍，造成重复计费。
func (s *AdminService) ResetTraffic(ctx context.Context, adminID int64, req *model.AdminTrafficResetReq, ip string) (*model.AdminTrafficResetResp, error) {
	resp := &model.AdminTrafficResetResp{Failed: make([]model.AdminBatchFailedItem, 0)}
	for _, uid := range req.UserIDs {
		if err := s.resetOneUserTraffic(adminID, uid, req.Mode); err != nil {
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{
				ID: uid, Reason: errs.From(err).Message,
			})
			continue
		}
		resp.Success++
	}
	_ = s.audit(s.db, adminID, "traffic_reset", batchAuditTarget(len(req.UserIDs)), ip, map[string]any{
		"mode": req.Mode, "success": resp.Success, "failed": len(resp.Failed), "user_ids": req.UserIDs,
	})
	return resp, nil
}

// resetOneUserTraffic 单用户重置：清零 u/d；reset_quota 另将 transfer_enable 重置为当前套餐流量额度。
func (s *AdminService) resetOneUserTraffic(adminID, userID int64, mode string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		u, err := s.repos.User.GetByIDForUpdate(tx, userID)
		if err != nil {
			return errs.ErrNotFound
		}
		updates := map[string]any{"u": int64(0), "d": int64(0)}
		afterTransfer := u.TransferEnable
		if mode == "reset_quota" {
			if u.PlanID == nil {
				return errs.New(40000, "用户无生效套餐，无法重新给量")
			}
			plan, err := s.repos.Plan.GetByID(tx, *u.PlanID)
			if err != nil {
				return errs.ErrNotFound
			}
			afterTransfer = int64(plan.TrafficGB) * 1024 * 1024 * 1024
			updates["transfer_enable"] = afterTransfer
		}
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}
		return s.repos.TrafficReset.Create(tx, &model.TrafficResetLog{
			UserID:               userID,
			AdminID:              adminID,
			Mode:                 mode,
			BeforeU:              u.U,
			BeforeD:              u.D,
			BeforeTransferEnable: u.TransferEnable,
			AfterTransferEnable:  afterTransfer,
		})
	})
}

// ListTrafficResets GET /admin/traffic/resets：重置记录分页（可按用户筛选）。
func (s *AdminService) ListTrafficResets(ctx context.Context, userID *int64, page, pageSize int) ([]model.AdminTrafficResetLogItem, int64, error) {
	return s.repos.TrafficReset.ListByPage(s.db, repo.TrafficResetQuery{UserID: userID}, page, pageSize)
}
