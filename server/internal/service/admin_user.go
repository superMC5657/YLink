package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/mailer"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// 用户域 module：查询/封禁/余额/批量/邮件/订阅密钥。
// 审计 target 摘要保持 batch:<count>。

// ListUsers GET /admin/users。
func (m *adminUserModule) ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]model.AdminUserItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := m.repos.User.ListByPage(m.db, keyword, page, pageSize)
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
func (m *adminUserModule) UpdateUser(ctx context.Context, adminID, userID int64, req *model.AdminUpdateUserReq, ip string) error {
	if adminID == userID {
		return errs.ErrForbidden // 不允许操作自己（防锁死）
	}
	user, err := m.repos.User.GetByID(m.db, userID)
	if err != nil {
		return errs.ErrNotFound
	}
	changed := false
	if req.Banned != nil && *req.Banned != user.IsBanned {
		if err := m.repos.User.SetBanned(m.db, userID, *req.Banned); err != nil {
			return err
		}
		changed = true
		_ = m.audit.write(m.db, adminID, "ban_user", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "banned": *req.Banned,
		})
	}
	if req.Role != nil && *req.Role != user.Role {
		if err := m.repos.User.UpdateRole(m.db, userID, *req.Role); err != nil {
			return err
		}
		changed = true
		_ = m.audit.write(m.db, adminID, "update_role", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "role": *req.Role,
		})
	}
	// 封禁/解封/角色变更 → bump 会话版本号：已签发 access token 立即失效
	if changed {
		bumpSessionVersion(ctx, m.rdb, userID)
	}
	return nil
}

// AdjustBalance POST /admin/users/{id}/balance（审计）。
// 强约束：调整后 balance 不允许为负（佣金回滚减的是 commission_balance，不在此约束）。
func (m *adminUserModule) AdjustBalance(ctx context.Context, adminID, userID int64, amountFen int64, remark, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		user, err := m.repos.User.GetByIDForUpdate(tx, userID)
		if err != nil {
			return errs.ErrNotFound
		}
		if user.Balance+amountFen < 0 {
			return errs.New(40000, "调整后余额不能为负")
		}
		user.Balance += amountFen
		if err := m.repos.User.Save(tx, user); err != nil {
			return err
		}
		return m.audit.write(tx, adminID, "adjust_balance", fmt.Sprint(userID), ip, map[string]any{
			"email": user.Email, "amount": amountFen, "remark": sanitize.Text(remark),
		})
	})
}

// ExportUsers F05 CSV 导出：分批流式回调（每批 500），行内容已联套餐名与邀请人邮箱。
// 行字段顺序与 handler 表头一致：id,email,balance,commission_balance,plan,expired_at,
// transfer_bytes,u_bytes,d_bytes,created_at,inviter_email。
func (m *adminUserModule) ExportUsers(ctx context.Context, keyword string, fn func(rows [][]string) error) error {
	plans, err := m.repos.Plan.ListAll(m.db)
	if err != nil {
		return err
	}
	planNames := make(map[int64]string, len(plans))
	for _, p := range plans {
		planNames[p.ID] = p.Name
	}
	return m.repos.User.StreamForExport(m.db, keyword, 500, func(batch []model.User) error {
		// 批内补齐邀请人邮箱
		invIDs := make([]int64, 0, len(batch))
		for i := range batch {
			if batch[i].InviteByID != nil {
				invIDs = append(invIDs, *batch[i].InviteByID)
			}
		}
		invEmail := make(map[int64]string, len(invIDs))
		if len(invIDs) > 0 {
			inviters, err := m.repos.User.ListByIDs(m.db, invIDs)
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
func (m *adminUserModule) BatchUsers(ctx context.Context, adminID int64, req *model.AdminBatchUserReq, ip string) (*model.AdminBatchUserResp, error) {
	if req.Action == "adjust_balance" && req.Amount == nil {
		return nil, errs.New(40000, "参数校验失败: adjust_balance 需提供 amount")
	}
	resp := &model.AdminBatchUserResp{Failed: make([]model.AdminBatchFailedItem, 0)}
	for _, id := range req.IDs {
		var err error
		switch req.Action {
		case "ban":
			banned := true
			err = m.UpdateUser(ctx, adminID, id, &model.AdminUpdateUserReq{Banned: &banned}, ip)
		case "unban":
			banned := false
			err = m.UpdateUser(ctx, adminID, id, &model.AdminUpdateUserReq{Banned: &banned}, ip)
		case "adjust_balance":
			err = m.AdjustBalance(ctx, adminID, id, model.YuanToFen(*req.Amount), req.Remark, ip)
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
func (m *adminUserModule) SendMail(ctx context.Context, adminID int64, req *model.AdminSendMailReq, ip string) (*model.AdminSendMailResp, error) {
	users, err := m.repos.User.ListByIDs(m.db, req.IDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]model.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	resp := &model.AdminSendMailResp{Failed: make([]model.AdminBatchFailedItem, 0)}
	subject := sanitize.Text(req.Subject)
	if m.ml == nil {
		// SMTP 未注入（未配置）：全部记为失败，留痕不丢
		for _, id := range req.IDs {
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{ID: id, Reason: "邮件服务未配置"})
		}
		_ = m.audit.write(m.db, adminID, "send_mail", batchAuditTarget(len(req.IDs)), ip, map[string]any{
			"subject": subject, "sent": 0, "failed": len(resp.Failed), "error": "mailer not configured", "ids": req.IDs,
		})
		return resp, nil
	}
	for _, id := range req.IDs {
		u, ok := byID[id]
		if !ok {
			resp.Failed = append(resp.Failed, model.AdminBatchFailedItem{ID: id, Reason: "用户不存在"})
			continue
		}
		sendErr := m.ml.Send(u.Email, subject, mailer.Template(sanitize.Markdown(req.Body)))
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
		if err := m.repos.MailLog.Create(m.db, &log); err != nil {
			return nil, err
		}
	}
	_ = m.audit.write(m.db, adminID, "send_mail", batchAuditTarget(len(req.IDs)), ip, map[string]any{
		"subject": subject, "sent": resp.Sent, "failed": len(resp.Failed), "ids": req.IDs,
	})
	return resp, nil
}

// ResetUserSubToken F05 管理端重置用户订阅密钥：无需用户密码，旧链接立即失效，写审计。
func (m *adminUserModule) ResetUserSubToken(ctx context.Context, adminID, userID int64, ip string) (string, error) {
	user, err := m.repos.User.GetByID(m.db, userID)
	if err != nil {
		return "", errs.ErrNotFound
	}
	oldToken := user.SubToken
	user.SubToken = uuid.NewString()
	if err := m.repos.User.Update(m.db, user); err != nil {
		return "", err
	}
	// 清除旧 token 缓存（userinfo / 限流计数），与用户端自助重置一致
	m.rdb.Del(ctx, redispkg.Key("sub", "userinfo", oldToken))
	m.rdb.Del(ctx, redispkg.Key("sub", "rl", oldToken))
	_ = m.audit.write(m.db, adminID, "reset_sub_token", fmt.Sprint(userID), ip, map[string]any{
		"email": user.Email,
	})
	return m.cfg.App.BaseURL + "/api/v1/" + m.cfg.Security.SubscribePathOrDefault() + "/subscribe/" + user.SubToken, nil
}
