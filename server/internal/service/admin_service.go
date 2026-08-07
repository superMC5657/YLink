package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink/internal/config"
	"ylink/internal/model"
	"ylink/internal/pkg/errs"
	"ylink/internal/pkg/sanitize"
	"ylink/internal/repo"
)

// AdminService 管理端 API（/api/v1/admin，role=admin）。
type AdminService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	cfg   *config.Config
	set   *SettingService
}

func NewAdminService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, set *SettingService) *AdminService {
	return &AdminService{db: db, rdb: rdb, repos: repos, cfg: cfg, set: set}
}

// ---- 审计 ----

// audit 写审计日志。
func (s *AdminService) audit(tx *gorm.DB, adminID int64, action, target, ip string, detail any) error {
	var detailStr *string
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			str := string(b)
			detailStr = &str
		}
	}
	ipStr := ip
	return s.repos.Audit.Create(tx, &model.AuditLog{
		AdminID: adminID, Action: action, Target: &target, Detail: detailStr, IP: &ipStr,
	})
}

// ---- 仪表盘 ----

// Overview GET /admin/stat/overview。
func (s *AdminService) Overview(ctx context.Context) (*model.AdminOverviewResp, error) {
	out := &model.AdminOverviewResp{}
	today := time.Now().Format("2006-01-02")
	type sums struct {
		Total int64
		Today int64
	}
	var usr, agent, order, completed, plan int64
	var rev sums
	if err := s.db.Model(&model.User{}).Count(&usr).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.User{}).Where("role = ?", model.RoleAgent).Count(&agent).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Order{}).Count(&order).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Order{}).Where("status = ?", model.OrderCompleted).Count(&completed).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Plan{}).Count(&plan).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Order{}).Where("status = ?", model.OrderCompleted).
		Select("COALESCE(SUM(pay_amount),0)").Scan(&rev.Total).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Order{}).Where("status = ? AND DATE(paid_at) = ?", model.OrderCompleted, today).
		Select("COALESCE(SUM(pay_amount),0)").Scan(&rev.Today).Error; err != nil {
		return nil, err
	}
	out.UserCount = usr
	out.AgentCount = agent
	out.OrderCount = order
	out.CompletedOrders = completed
	out.PlanCount = plan
	out.TotalRevenue = model.FenToYuan(rev.Total)
	out.TodayRevenue = model.FenToYuan(rev.Today)
	return out, nil
}

// ---- 用户管理 ----

// ListUsers GET /admin/users。
func (s *AdminService) ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]model.AdminUserItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.User.ListByPage(s.db, keyword, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.AdminUserItem, 0, len(list))
	for _, u := range list {
		out = append(out, model.AdminUserItem{
			ID: u.ID, Email: u.Email, Role: u.Role,
			Balance: model.FenToYuan(u.Balance), CommissionBalance: model.FenToYuan(u.CommissionBalance),
			IsBanned: u.IsBanned, InviteByID: u.InviteByID, PlanID: u.PlanID, ExpiredAt: u.ExpiredAt,
			TransferEnable: u.TransferEnable, U: u.U, D: u.D, CreatedAt: u.CreatedAt,
		})
	}
	return out, total, nil
}

// UpdateUser PUT /admin/users/{id}（封禁/角色）。
func (s *AdminService) UpdateUser(ctx context.Context, adminID, userID int64, req *model.AdminUpdateUserReq, ip string) error {
	if adminID == userID {
		return errs.ErrForbidden // 不允许操作自己（防锁死）
	}
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return errs.ErrNotFound
	}
	if req.Banned != nil && *req.Banned != user.IsBanned {
		if err := s.repos.User.SetBanned(s.db, userID, *req.Banned); err != nil {
			return err
		}
		_ = s.audit(s.db, adminID, "ban_user", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "banned": *req.Banned,
		})
	}
	if req.Role != nil && *req.Role != user.Role {
		if err := s.repos.User.UpdateRole(s.db, userID, *req.Role); err != nil {
			return err
		}
		_ = s.audit(s.db, adminID, "update_role", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "role": *req.Role,
		})
	}
	return nil
}

// AdjustBalance POST /admin/users/{id}/balance（审计）。
func (s *AdminService) AdjustBalance(ctx context.Context, adminID, userID int64, amountFen int64, remark, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		user, err := s.repos.User.GetByIDForUpdate(tx, userID)
		if err != nil {
			return errs.ErrNotFound
		}
		user.Balance += amountFen
		if err := s.repos.User.Save(tx, user); err != nil {
			return err
		}
		return s.audit(tx, adminID, "adjust_balance", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "amount": amountFen, "remark": sanitize.Text(remark),
		})
	})
}

// ---- 订单与退款 ----

// ListOrders GET /admin/orders。
func (s *AdminService) ListOrders(ctx context.Context, status *int, page, pageSize int) ([]model.AdminOrderItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.Order.ListByPage(s.db, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	emails := s.emailsOf(userIDsOf(list))
	planNames := map[int64]string{}
	out := make([]model.AdminOrderItem, 0, len(list))
	for _, o := range list {
		name, ok := planNames[o.PlanID]
		if !ok {
			if p, err := s.repos.Plan.GetByID(s.db, o.PlanID); err == nil {
				name = p.Name
				planNames[o.PlanID] = name
			}
		}
		out = append(out, model.AdminOrderItem{
			OrderNo: o.OrderNo, UserID: o.UserID, UserEmail: emails[o.UserID], PlanName: name,
			Period: o.Period, Amount: model.FenToYuan(o.Amount), DiscountAmount: model.FenToYuan(o.DiscountAmount),
			BalanceUsed: model.FenToYuan(o.BalanceUsed), PayAmount: model.FenToYuan(o.PayAmount),
			Status: o.Status, PayMethod: o.PayMethod, PaidAt: o.PaidAt, CreatedAt: o.CreatedAt,
		})
	}
	return out, total, nil
}

// CloseOrder POST /admin/orders/{no}/close：管理员关闭待支付订单（回退优惠券占用 + 审计）。
func (s *AdminService) CloseOrder(ctx context.Context, adminID int64, orderNo, remark, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		order, err := s.repos.Order.GetByNoForUpdate(tx, orderNo)
		if err != nil {
			return errs.ErrNotFound
		}
		if order.Status != model.OrderPending {
			return errs.ErrOrderStatus
		}
		// 条件更新（防与支付回调竞态）：影响行数为 0 说明已被并发完成/取消
		affected, err := s.repos.Order.UpdateStatusIfPending(tx, orderNo, model.OrderCanceled)
		if err != nil {
			return err
		}
		if affected == 0 {
			return errs.ErrOrderStatus
		}
		if order.CouponID != nil {
			if err := releaseCoupon(tx, *order.CouponID, order.UserID, orderNo); err != nil {
				return err
			}
		}
		return s.audit(tx, adminID, "close_order", orderNo, ip, map[string]any{
			"remark": sanitize.Text(remark),
		})
	})
}

// Refund POST /admin/orders/{no}/refund：退款 + 佣金回滚 + 优惠券回退 + 审计。
func (s *AdminService) Refund(ctx context.Context, adminID int64, orderNo, remark, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		order, err := s.repos.Order.GetByNoForUpdate(tx, orderNo)
		if err != nil {
			return errs.ErrNotFound
		}
		if order.Status != model.OrderCompleted {
			return errs.ErrOrderStatus
		}
		// 余额支付退回余额
		if order.BalanceUsed > 0 {
			user, err := s.repos.User.GetByIDForUpdate(tx, order.UserID)
			if err != nil {
				return err
			}
			user.Balance += order.BalanceUsed
			if err := s.repos.User.Save(tx, user); err != nil {
				return err
			}
		}
		// 优惠券占用回退
		if order.CouponID != nil {
			if err := releaseCoupon(tx, *order.CouponID, order.UserID, order.OrderNo); err != nil {
				return err
			}
		}
		order.Status = model.OrderRefunded
		if err := s.repos.Order.Save(tx, order); err != nil {
			return err
		}
		// 佣金回滚（core-flows 第 4 节）
		if cl, err := s.repos.Commission.GetByOrderNo(tx, order.OrderNo); err == nil {
			switch cl.Status {
			case model.CommissionPending:
				cl.Status = model.CommissionRevoked
				if err := s.repos.Commission.Save(tx, cl); err != nil {
					return err
				}
			case model.CommissionGranted:
				inviter, err := s.repos.User.GetByIDForUpdate(tx, cl.InviteUserID)
				if err != nil {
					return err
				}
				inviter.CommissionBalance -= cl.Amount // 可为负，记录审计
				if err := s.repos.User.Save(tx, inviter); err != nil {
					return err
				}
				cl.Status = model.CommissionRevoked
				if err := s.repos.Commission.Save(tx, cl); err != nil {
					return err
				}
			}
		}
		return s.audit(tx, adminID, "refund", orderNo, ip, map[string]any{
			"pay_amount": order.PayAmount, "remark": sanitize.Text(remark),
		})
	})
}

// ---- 代理审批 ----

// ListAgentApplies GET /admin/agent/applies。
func (s *AdminService) ListAgentApplies(ctx context.Context, status int, page, pageSize int) ([]model.AdminAgentApplyItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.AgentApply.ListByStatus(s.db, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.UserID)
	}
	emails := s.emailsOf(ids)
	_, validDays := agentPolicy(s.db)
	validCounts := map[int64]int64{}
	for _, uid := range ids {
		if n, err := s.repos.User.CountValidInvited(s.db, uid, validDays); err == nil {
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
func (s *AdminService) ReviewAgentApply(ctx context.Context, adminID, applyID int64, approve bool, remark, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		apply, err := s.repos.AgentApply.GetByIDForUpdate(tx, applyID)
		if err != nil {
			return errs.ErrNotFound
		}
		if apply.Status != 0 {
			return errs.ErrConflict
		}
		status := 2
		if approve {
			status = 1
			if err := s.repos.User.UpdateRole(tx, apply.UserID, model.RoleAgent); err != nil {
				return err
			}
		}
		now := time.Now()
		apply.Status = status
		if remark != "" {
			r := sanitize.Text(remark)
			apply.Remark = &r
		}
		apply.ReviewedAt = &now
		if err := s.repos.AgentApply.Save(tx, apply); err != nil {
			return err
		}
		action := "agent_reject"
		if approve {
			action = "agent_approve"
		}
		return s.audit(tx, adminID, action, fmt.Sprint(apply.UserID), ip, map[string]any{"apply_id": apply.ID})
	})
}

// ---- 佣金日志 ----

// ListCommissions GET /admin/commission-logs。
func (s *AdminService) ListCommissions(ctx context.Context, status *int, page, pageSize int) ([]model.AdminCommissionItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.Commission.ListByPage(s.db, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(list)*2)
	for _, cl := range list {
		ids = append(ids, cl.InviteUserID, cl.FromUserID)
	}
	emails := s.emailsOf(ids)
	out := make([]model.AdminCommissionItem, 0, len(list))
	for _, cl := range list {
		out = append(out, model.AdminCommissionItem{
			ID: cl.ID, InviteUserID: cl.InviteUserID, InviteEmail: emails[cl.InviteUserID],
			FromUserID: cl.FromUserID, FromEmail: emails[cl.FromUserID], OrderNo: cl.OrderNo,
			OrderAmount: model.FenToYuan(cl.OrderAmount), Rate: cl.Rate, Amount: model.FenToYuan(cl.Amount),
			Status: cl.Status, ConfirmedAt: cl.ConfirmedAt, CreatedAt: cl.CreatedAt,
		})
	}
	return out, total, nil
}

// ---- 流量导入 ----

// ImportTraffic POST /admin/traffic/import（模式 B 手工导入）。
func (s *AdminService) ImportTraffic(ctx context.Context, adminID int64, req *model.TrafficImportReq, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if err := s.repos.Traffic.Upsert(tx, &model.TrafficLog{UserID: item.UserID, Date: item.Date, U: item.U, D: item.D}); err != nil {
				return err
			}
		}
		return s.audit(tx, adminID, "traffic_import", "", ip, map[string]any{"count": len(req.Items)})
	})
}

// ---- 设置 ----

// ListSettings GET /admin/settings。
func (s *AdminService) ListSettings(ctx context.Context) ([]model.AdminSettingsResp, error) {
	var rows []model.Setting
	if err := s.db.Order("`key` ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]model.AdminSettingsResp, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.AdminSettingsResp{Key: r.Key, Value: r.Value})
	}
	return out, nil
}

// SaveSetting PUT /admin/settings（写后失效 Redis 缓存）。
func (s *AdminService) SaveSetting(ctx context.Context, key, value string) error {
	if err := s.repos.Setting.Set(s.db, key, value); err != nil {
		return err
	}
	s.set.Invalidate(ctx, key)
	return nil
}

// ---- 工单（管理端） ----

// ListTickets GET /admin/tickets。
func (s *AdminService) ListTickets(ctx context.Context, page, pageSize int) ([]model.AdminTicketItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.Ticket.ListByPage(s.db, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(list))
	for _, t := range list {
		ids = append(ids, t.UserID)
	}
	emails := s.emailsOf(ids)
	out := make([]model.AdminTicketItem, 0, len(list))
	for _, t := range list {
		out = append(out, model.AdminTicketItem{
			ID: t.ID, UserID: t.UserID, UserEmail: emails[t.UserID], Subject: t.Subject,
			Level: t.Level, Status: t.Status, LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
		})
	}
	return out, total, nil
}

// GetTicketDetail GET /admin/tickets/{id}。
func (s *AdminService) GetTicketDetail(ctx context.Context, id int64) (*model.TicketDetailResp, error) {
	t, err := s.repos.Ticket.GetByID(s.db, id)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	messages, err := s.repos.Ticket.MessagesByTicket(s.db, id)
	if err != nil {
		return nil, err
	}
	msgs := make([]model.TicketMsgResp, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, model.TicketMsgResp{ID: m.ID, SenderType: m.SenderType, Message: m.Message, CreatedAt: m.CreatedAt})
	}
	return &model.TicketDetailResp{ID: t.ID, Subject: t.Subject, Level: t.Level, Status: t.Status, CreatedAt: t.CreatedAt, Messages: msgs}, nil
}

// ReplyTicket POST /admin/tickets/{id}/reply（客服回复 → 状态已回复）。
func (s *AdminService) ReplyTicket(ctx context.Context, adminID, id int64, message string) error {
	t, err := s.repos.Ticket.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	if t.Status == 2 {
		return errs.ErrTicketClosed
	}
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		now := time.Now()
		if err := s.repos.Ticket.CreateMessage(tx, &model.TicketMessage{
			TicketID: id, SenderType: 1, SenderID: adminID, Message: sanitize.Text(message),
		}); err != nil {
			return err
		}
		if err := s.repos.Ticket.UpdateStatus(tx, id, 1); err != nil {
			return err
		}
		return s.repos.Ticket.UpdateLastReplyAt(tx, id, now)
	})
}

// CloseTicket POST /admin/tickets/{id}/close。
func (s *AdminService) CloseTicket(ctx context.Context, id int64) error {
	t, err := s.repos.Ticket.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	if t.Status == 2 {
		return errs.ErrTicketClosed
	}
	return s.repos.Ticket.UpdateStatus(s.db, id, 2)
}

// ---- 辅助 ----

// emailsOf 批量取用户邮箱。
func (s *AdminService) emailsOf(ids []int64) map[int64]string {
	out := map[int64]string{}
	if len(ids) == 0 {
		return out
	}
	var users []model.User
	if err := s.db.Select("id, email").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return out
	}
	for _, u := range users {
		out[u.ID] = u.Email
	}
	return out
}

func userIDsOf(list []model.Order) []int64 {
	ids := make([]int64, 0, len(list))
	for _, o := range list {
		ids = append(ids, o.UserID)
	}
	return ids
}
