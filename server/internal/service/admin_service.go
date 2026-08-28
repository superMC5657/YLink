package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/mailer"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// AdminService 管理端 API（/api/v1/admin，role=admin）。
type AdminService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	cfg   *config.Config
	set   *SettingService
	ml    *mailer.Mailer
}

func NewAdminService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, set *SettingService, ml *mailer.Mailer) *AdminService {
	return &AdminService{db: db, rdb: rdb, repos: repos, cfg: cfg, set: set, ml: ml}
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
	if err := s.db.Model(&model.Order{}).Where("status = ? AND paid_at::date = ?", model.OrderCompleted, today).
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
	changed := false
	if req.Banned != nil && *req.Banned != user.IsBanned {
		if err := s.repos.User.SetBanned(s.db, userID, *req.Banned); err != nil {
			return err
		}
		changed = true
		_ = s.audit(s.db, adminID, "ban_user", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "banned": *req.Banned,
		})
	}
	if req.Role != nil && *req.Role != user.Role {
		if err := s.repos.User.UpdateRole(s.db, userID, *req.Role); err != nil {
			return err
		}
		changed = true
		_ = s.audit(s.db, adminID, "update_role", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "role": *req.Role,
		})
	}
	// 封禁/解封/角色变更 → bump 会话版本号：已签发 access token 立即失效
	if changed {
		bumpSessionVersion(ctx, s.rdb, userID)
	}
	return nil
}

// AdjustBalance POST /admin/users/{id}/balance（审计）。
// 强约束：调整后 balance 不允许为负（佣金回滚减的是 commission_balance，不在此约束）。
func (s *AdminService) AdjustBalance(ctx context.Context, adminID, userID int64, amountFen int64, remark, ip string) error {
	return repo.WithTx(s.db, func(tx *gorm.DB) error {
		user, err := s.repos.User.GetByIDForUpdate(tx, userID)
		if err != nil {
			return errs.ErrNotFound
		}
		if user.Balance+amountFen < 0 {
			return errs.New(40000, "调整后余额不能为负")
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

// ---- 用户管理增强（F05） ----

// ExportUsers F05 CSV 导出：分批流式回调（每批 500），行内容已联套餐名与邀请人邮箱。
// 行字段顺序与 handler 表头一致：id,email,balance,commission_balance,plan,expired_at,
// transfer_bytes,u_bytes,d_bytes,created_at,inviter_email。
func (s *AdminService) ExportUsers(ctx context.Context, keyword string, fn func(rows [][]string) error) error {
	plans, err := s.repos.Plan.ListAll(s.db)
	if err != nil {
		return err
	}
	planNames := make(map[int64]string, len(plans))
	for _, p := range plans {
		planNames[p.ID] = p.Name
	}
	return s.repos.User.StreamForExport(s.db, keyword, 500, func(batch []model.User) error {
		// 批内补齐邀请人邮箱
		invIDs := make([]int64, 0, len(batch))
		for i := range batch {
			if batch[i].InviteByID != nil {
				invIDs = append(invIDs, *batch[i].InviteByID)
			}
		}
		invEmail := make(map[int64]string, len(invIDs))
		if len(invIDs) > 0 {
			inviters, err := s.repos.User.ListByIDs(s.db, invIDs)
			if err != nil {
				return err
			}
			for _, iv := range inviters {
				invEmail[iv.ID] = iv.Email
			}
		}
		rows := make([][]string, 0, len(batch))
		for i := range batch {
			u := batch[i]
			plan := ""
			if u.PlanID != nil {
				plan = planNames[*u.PlanID]
			}
			inv := ""
			if u.InviteByID != nil {
				inv = invEmail[*u.InviteByID]
			}
			expired := ""
			if u.ExpiredAt != nil {
				expired = u.ExpiredAt.Format(time.RFC3339)
			}
			rows = append(rows, []string{
				strconv.FormatInt(u.ID, 10),
				u.Email,
				fmt.Sprintf("%.2f", model.FenToYuan(u.Balance)),
				fmt.Sprintf("%.2f", model.FenToYuan(u.CommissionBalance)),
				plan,
				expired,
				strconv.FormatInt(u.TransferEnable, 10),
				strconv.FormatInt(u.U, 10),
				strconv.FormatInt(u.D, 10),
				u.CreatedAt.Format(time.RFC3339),
				inv,
			})
		}
		return fn(rows)
	})
}

// BatchUsers F05 批量用户操作：ban/unban/adjust_balance，逐个执行并汇总成功/失败。
// 失败不打断整体：单用户失败（不存在/操作自己/余额为负等）记入 failed 列表。
func (s *AdminService) BatchUsers(ctx context.Context, adminID int64, req *model.AdminBatchUserReq, ip string) (*model.AdminBatchUserResp, error) {
	if req.Action == "adjust_balance" && req.Amount == nil {
		return nil, errs.New(40000, "参数校验失败: adjust_balance 需提供 amount")
	}
	resp := &model.AdminBatchUserResp{Failed: make([]model.AdminBatchFailedItem, 0)}
	for _, id := range req.IDs {
		var err error
		switch req.Action {
		case "ban":
			banned := true
			err = s.UpdateUser(ctx, adminID, id, &model.AdminUpdateUserReq{Banned: &banned}, ip)
		case "unban":
			banned := false
			err = s.UpdateUser(ctx, adminID, id, &model.AdminUpdateUserReq{Banned: &banned}, ip)
		case "adjust_balance":
			err = s.AdjustBalance(ctx, adminID, id, model.YuanToFen(*req.Amount), req.Remark, ip)
		}
		if err != nil {
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{
				ID: id, Reason: errs.From(err).Message,
			})
			continue
		}
		resp.Success++
	}
	return resp, nil
}

// SendMail F05 管理端发邮件：SMTP 同步逐发，结果写 mail_logs，整体操作写审计。
// 单封失败不中断其余收件人（失败原因进 mail_logs 与响应）。
func (s *AdminService) SendMail(ctx context.Context, adminID int64, req *model.AdminSendMailReq, ip string) (*model.AdminSendMailResp, error) {
	users, err := s.repos.User.ListByIDs(s.db, req.IDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]model.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	resp := &model.AdminSendMailResp{Failed: make([]model.AdminBatchFailedItem, 0)}
	subject := sanitize.Text(req.Subject)
	if s.ml == nil {
		// SMTP 未注入（未配置）：全部记为失败，留痕不丢
		for _, id := range req.IDs {
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{ID: id, Reason: "邮件服务未配置"})
		}
		_ = s.audit(s.db, adminID, "send_mail", fmt.Sprint(req.IDs), ip, map[string]any{
			"subject": subject, "sent": 0, "failed": len(resp.Failed), "error": "mailer not configured",
		})
		return resp, nil
	}
	for _, id := range req.IDs {
		u, ok := byID[id]
		if !ok {
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{ID: id, Reason: "用户不存在"})
			continue
		}
		sendErr := s.ml.Send(u.Email, subject, mailer.Template(sanitize.Markdown(req.Body)))
		log := model.MailLog{
			UserID: u.ID, Email: u.Email, Subject: subject, AdminID: adminID,
		}
		if sendErr != nil {
			log.Status = 0
			msg := sendErr.Error()
			if len(msg) > 512 {
				msg = msg[:512]
			}
			log.Error = &msg
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{ID: id, Reason: "邮件发送失败"})
		} else {
			log.Status = 1
			resp.Sent++
		}
		if err := s.repos.MailLog.Create(s.db, &log); err != nil {
			return nil, err
		}
	}
	_ = s.audit(s.db, adminID, "send_mail", fmt.Sprint(req.IDs), ip, map[string]any{
		"subject": subject, "sent": resp.Sent, "failed": len(resp.Failed),
	})
	return resp, nil
}

// ResetUserSubToken F05 管理端重置用户订阅密钥：无需用户密码，旧链接立即失效，写审计。
func (s *AdminService) ResetUserSubToken(ctx context.Context, adminID, userID int64, ip string) (string, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return "", errs.ErrNotFound
	}
	oldToken := user.SubToken
	user.SubToken = uuid.NewString()
	if err := s.repos.User.Update(s.db, user); err != nil {
		return "", err
	}
	// 清除旧 token 缓存（userinfo / 限流计数），与用户端自助重置一致
	s.rdb.Del(ctx, redispkg.Key("sub", "userinfo", oldToken))
	s.rdb.Del(ctx, redispkg.Key("sub", "rl", oldToken))
	_ = s.audit(s.db, adminID, "reset_sub_token", fmt.Sprint(userID), ip, map[string]any{
		"email": user.Email,
	})
	return s.cfg.App.BaseURL + "/api/v1/" + s.cfg.Security.SubscribePathOrDefault() + "/subscribe/" + user.SubToken, nil
}

// ---- 审计日志（F08） ----

// AuditLogFilter 审计日志筛选参数（handler 解析后的 query）。
type AuditLogFilter struct {
	AdminID *int64
	Action  string
	Target  string
	From    string // YYYY-MM-DD，可空
	To      string // YYYY-MM-DD，可空（含当天）
}

// ListAuditLogs GET /admin/audit-logs：审计日志分页查询（只读，数据源为 audit_logs 表）。
func (s *AdminService) ListAuditLogs(ctx context.Context, f AuditLogFilter, page, pageSize int) ([]model.AdminAuditLogItem, int64, []string, error) {
	q := repo.AuditLogQuery{AdminID: f.AdminID, Action: f.Action, Target: f.Target}
	parseDay := func(s string) (time.Time, error) {
		return time.ParseInLocation("2006-01-02", s, time.Local)
	}
	if f.From != "" {
		t, err := parseDay(f.From)
		if err != nil {
			return nil, 0, nil, errs.New(40000, "参数校验失败: from 需为 YYYY-MM-DD")
		}
		q.From = &t
	}
	if f.To != "" {
		t, err := parseDay(f.To)
		if err != nil {
			return nil, 0, nil, errs.New(40000, "参数校验失败: to 需为 YYYY-MM-DD")
		}
		t = t.AddDate(0, 0, 1) // 含 to 当天
		q.To = &t
	}
	list, total, err := s.repos.Audit.ListByPage(s.db, q, page, pageSize)
	if err != nil {
		return nil, 0, nil, err
	}
	actions, err := s.repos.Audit.ListActions(s.db)
	if err != nil {
		return list, total, nil, nil // 动作列表失败不阻断主查询
	}
	return list, total, actions, nil
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
	// 批量查佣金:order_no → amount(一个订单至多一条佣金记录,uniqueIndex)。
	// 查询失败必须上抛,否则接口会以成功响应返回 commission_amount 全 null,
	// 把数据缺失误表现为「无佣金」(review-0.5.0 P2)。
	commissionByOrder := map[string]float64{}
	comms, err := s.repos.Commission.ListByOrderNos(s.db, orderNosOf(list))
	if err != nil {
		return nil, 0, err
	}
	for _, cl := range comms {
		commissionByOrder[cl.OrderNo] = model.FenToYuan(cl.Amount)
	}
	out := make([]model.AdminOrderItem, 0, len(list))
	for _, o := range list {
		name, ok := planNames[o.PlanID]
		if !ok {
			if p, err := s.repos.Plan.GetByID(s.db, o.PlanID); err == nil {
				name = p.Name
				planNames[o.PlanID] = name
			}
		}
		var commission *float64
		if v, has := commissionByOrder[o.OrderNo]; has {
			commission = &v
		}
		out = append(out, model.AdminOrderItem{
			OrderNo: o.OrderNo, UserID: o.UserID, UserEmail: emails[o.UserID], PlanName: name,
			Period: o.Period, Amount: model.FenToYuan(o.Amount), DiscountAmount: model.FenToYuan(o.DiscountAmount),
			BalanceUsed: model.FenToYuan(o.BalanceUsed), PayAmount: model.FenToYuan(o.PayAmount),
			CommissionAmount: commission,
			Status:           o.Status, PayMethod: o.PayMethod, PaidAt: o.PaidAt, CreatedAt: o.CreatedAt,
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

// Refund POST /admin/orders/{no}/refund：退款 + 收回订阅 + 佣金回滚 + 优惠券回退 + 审计。
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
		// 收回订阅：退款后用户不应继续享有该订单对应的订阅服务
		if err := s.revokeSubscriptionOnRefund(tx, order); err != nil {
			return err
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

// revokeSubscriptionOnRefund 退款收回订阅（core-flows 2.1 的逆操作）：
//   - onetime：仅扣回本次叠加的流量（下限 0），不动到期时间
//   - 周期套餐：若用户当前生效订阅正是该订单套餐（plan_id 相同且未过期），清除订阅
//     （plan_id/expired_at 置空、流量清零、限速/设备数清空）
//
// 说明：系统未保存订单开通前的订阅快照，因此「同套餐续期叠加」场景退款后会整体清除订阅
// （用户原有未过期订阅一并收回）；管理员对续期叠加订单退款时需知悉此行为。
func (s *AdminService) revokeSubscriptionOnRefund(tx *gorm.DB, order *model.Order) error {
	user, err := s.repos.User.GetByIDForUpdate(tx, order.UserID)
	if err != nil {
		return err
	}
	plan, err := s.repos.Plan.GetByID(tx, order.PlanID)
	if err != nil {
		return err
	}
	trafficBytes := int64(plan.TrafficGB) * 1024 * 1024 * 1024

	if order.Period == "onetime" {
		// 一次性流量包：扣回本次流量
		user.TransferEnable -= trafficBytes
		if user.TransferEnable < 0 {
			user.TransferEnable = 0
		}
		return s.repos.User.Save(tx, user)
	}
	// 周期订阅：仅当当前生效订阅正是该订单套餐时清除
	if user.PlanID != nil && *user.PlanID == order.PlanID &&
		user.ExpiredAt != nil && user.ExpiredAt.After(time.Now()) {
		user.PlanID = nil
		user.ExpiredAt = nil
		user.TransferEnable = 0
		user.U = 0
		user.D = 0
		user.SpeedLimit = nil
		user.DeviceLimit = nil
		return s.repos.User.Save(tx, user)
	}
	return nil
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
			// 升级为代理商 → bump 会话版本号：旧 access token 的 role 快照立即失效
			bumpSessionVersion(ctx, s.rdb, apply.UserID)
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
			Type: cl.BizType, Status: cl.Status, ConfirmedAt: cl.ConfirmedAt, CreatedAt: cl.CreatedAt,
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
	if err := s.db.Order("key ASC").Find(&rows).Error; err != nil {
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
			Level: t.Level, Type: t.Type, Status: t.Status, ReopenCount: t.ReopenCount, LastReplyAt: t.LastReplyAt, CreatedAt: t.CreatedAt,
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
	return &model.TicketDetailResp{ID: t.ID, Subject: t.Subject, Level: t.Level, Type: t.Type,
		Status: t.Status, CreatedAt: t.CreatedAt, Messages: msgs, Withdraw: ticketWithdrawInfo(s.db, t),
	}, nil
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

func orderNosOf(list []model.Order) []string {
	nos := make([]string, 0, len(list))
	for _, o := range list {
		nos = append(nos, o.OrderNo)
	}
	return nos
}
