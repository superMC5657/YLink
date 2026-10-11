package service

import (
	"context"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// 流量域 module：模式 B 手工导入 + 按用户批量重置 + 重置记录。
// 流量模式语义不变：clear_usage=清零用量；reset_quota=按当前套餐额度重新给量。
// 与节点上报差分幂等兼容：重置时保留 node_user_stats 快照，见 resetOneUserTraffic。

// ImportTraffic POST /admin/traffic/import（模式 B 手工导入）。
func (m *adminTrafficModule) ImportTraffic(ctx context.Context, adminID int64, req *model.TrafficImportReq, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if err := m.repos.Traffic.Upsert(tx, &model.TrafficLog{UserID: item.UserID, Date: item.Date, U: item.U, D: item.D}); err != nil {
				return err
			}
		}
		return m.audit.write(tx, adminID, "traffic_import", "", ip, map[string]any{"count": len(req.Items)})
	})
}

// ResetTraffic F16 按用户批量重置流量，逐用户单事务执行并写入 traffic_reset_logs。
//
// 与节点上报差分幂等兼容（验收要点）：重置时**保留 node_user_stats 快照**——
// 快照存的是节点侧累计值，下次上报按「当前累计 − 快照」差分，只有重置之后的新流量
// 会计入 users.u/d；若清空快照，下一次上报会把节点全周期累计值整体重算一遍，造成重复计费。
func (m *adminTrafficModule) ResetTraffic(ctx context.Context, adminID int64, req *model.AdminTrafficResetReq, ip string) (*model.AdminTrafficResetResp, error) {
	resp := &model.AdminTrafficResetResp{Failed: make([]model.AdminBatchFailedItem, 0)}
	for _, uid := range req.UserIDs {
		if err := m.resetOneUserTraffic(adminID, uid, req.Mode); err != nil {
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{
				ID: uid, Reason: errs.From(err).Message,
			})
			continue
		}
		resp.Success++
	}
	_ = m.audit.write(m.db, adminID, "traffic_reset", batchAuditTarget(len(req.UserIDs)), ip, map[string]any{
		"mode": req.Mode, "success": resp.Success, "failed": len(resp.Failed), "user_ids": req.UserIDs,
	})
	return resp, nil
}

// resetOneUserTraffic 单用户重置：清零 u/d；reset_quota 另将 transfer_enable 重置为当前套餐流量额度。
func (m *adminTrafficModule) resetOneUserTraffic(adminID, userID int64, mode string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		u, err := m.repos.User.GetByIDForUpdate(tx, userID)
		if err != nil {
			return errs.ErrNotFound
		}
		updates := map[string]any{"u": int64(0), "d": int64(0)}
		afterTransfer := u.TransferEnable
		if mode == "reset_quota" {
			if u.PlanID == nil {
				return errs.New(40000, "用户无生效套餐，无法重新给量")
			}
			plan, err := m.repos.Plan.GetByID(tx, *u.PlanID)
			if err != nil {
				return errs.ErrNotFound
			}
			afterTransfer = int64(plan.TrafficGB) * 1024 * 1024 * 1024
			updates["transfer_enable"] = afterTransfer
		}
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}
		return m.repos.TrafficReset.Create(tx, &model.TrafficResetLog{
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
func (m *adminTrafficModule) ListTrafficResets(ctx context.Context, userID *int64, page, pageSize int) ([]model.AdminTrafficResetLogItem, int64, error) {
	return m.repos.TrafficReset.ListByPage(m.db, repo.TrafficResetQuery{UserID: userID}, page, pageSize)
}
