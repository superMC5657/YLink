package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// 代理与佣金域 module：申请审批 + 佣金列表。

// ListAgentApplies GET /admin/agent/applies。
func (m *adminAgentModule) ListAgentApplies(ctx context.Context, status int, page, pageSize int) ([]model.AdminAgentApplyItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := m.repos.AgentApply.ListByStatus(m.db, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.UserID)
	}
	emails := m.audit.emailsOf(ids)
	_, validDays := agentPolicy(m.db)
	validCounts := map[int64]int64{}
	for _, uid := range ids {
		if n, err := m.repos.User.CountValidInvited(m.db, uid, validDays); err == nil {
			validCounts[uid] = n
		}
	}
	out := make([]model.AdminAgentApplyItem, 0, len(list))
	for _, a := range list {
		out = append(out, model.AdminAgentApplyItem{
			ID: a.ID, UserID: a.UserID, UserEmail: emails[a.UserID],
			ValidInvites: validCounts[a.UserID], Status: a.Status, CreatedAt: a.CreatedAt,
		})
	}
	return out, total, nil
}

// ReviewAgentApply POST /admin/agent/applies/{id}/approve|reject。
func (m *adminAgentModule) ReviewAgentApply(ctx context.Context, adminID, applyID int64, approve bool, remark, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		apply, err := m.repos.AgentApply.GetByIDForUpdate(tx, applyID)
		if err != nil {
			return errs.ErrNotFound
		}
		if apply.Status != 0 {
			return errs.ErrConflict
		}
		status := 2
		if approve {
			status = 1
			if err := m.repos.User.UpdateRole(tx, apply.UserID, model.RoleAgent); err != nil {
				return err
			}
			// 升级为代理商 → bump 会话版本号：旧 access token 的 role 快照立即失效
			bumpSessionVersion(ctx, m.rdb, apply.UserID)
		}
		now := time.Now()
		apply.Status = status
		if remark != "" {
			r := sanitize.Text(remark)
			apply.Remark = &r
		}
		apply.ReviewedAt = &now
		if err := m.repos.AgentApply.Save(tx, apply); err != nil {
			return err
		}
		action := "agent_reject"
		if approve {
			action = "agent_approve"
		}
		return m.audit.write(tx, adminID, action, fmt.Sprint(apply.UserID), ip, map[string]any{"apply_id": apply.ID})
	})
}

// ListCommissions GET /admin/commission-logs。
func (m *adminAgentModule) ListCommissions(ctx context.Context, status *int, page, pageSize int) ([]model.AdminCommissionItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := m.repos.Commission.ListByPage(m.db, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(list)*2)
	for _, cl := range list {
		ids = append(ids, cl.InviteUserID, cl.FromUserID)
	}
	emails := m.audit.emailsOf(ids)
	out := make([]model.AdminCommissionItem, 0, len(list))
	for _, cl := range list {
		out = append(out, model.AdminCommissionItem{
			ID: cl.ID, InviteUserID: cl.InviteUserID, InviteEmail: emails[cl.InviteUserID],
			FromUserID: cl.FromUserID, FromEmail: emails[cl.FromUserID], OrderNo: cl.OrderNo,
			OrderAmount: model.FenToYuan(cl.OrderAmount), Rate: cl.Rate, Amount: model.FenToYuan(cl.Amount),
			Type: cl.BizType, Status: cl.Status, ConfirmedAt: cl.ConfirmedAt, CreatedAt: cl.CreatedAt,
		})
	}
	return out, total, nil
}
