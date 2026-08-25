package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/pkg/payment"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

// CronService 定时任务（worker 进程调度；所有任务由 worker 加分布式锁）。
type CronService struct {
	db     *gorm.DB
	rdb    *redis.Client
	repos  *repo.Repos
	cfg    *config.Config
	mailer *mailer.Mailer
	orders *OrderService
}

func NewCronService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, m *mailer.Mailer, orders *OrderService) *CronService {
	return &CronService{db: db, rdb: rdb, repos: repos, cfg: cfg, mailer: m, orders: orders}
}

// WithLock 分布式锁包装：仅单实例执行任务。
// 打点 cron_job_runs_total / cron_job_duration_seconds（worker /metrics 暴露，供 Grafana 看板与告警）。
func (s *CronService) WithLock(name string, fn func()) func() {
	return func() {
		ctx := context.Background()
		ok, err := s.rdb.SetNX(ctx, redispkg.Key("cron", "lock", name), "1", 15*time.Minute).Result()
		if err != nil || !ok {
			logger.L().Info("cron lock skipped", zapS("job", name))
			middleware.CronJobInc(name, "skipped")
			return
		}
		defer s.rdb.Del(ctx, redispkg.Key("cron", "lock", name))
		logger.L().Info("cron job start", zapS("job", name))
		start := time.Now()
		func() {
			defer func() {
				if r := recover(); r != nil {
					middleware.CronJobInc(name, "error")
					middleware.CronJobDuration(name, time.Since(start))
					panic(r)
				}
			}()
			fn()
			middleware.CronJobInc(name, "success")
			middleware.CronJobDuration(name, time.Since(start))
		}()
		logger.L().Info("cron job done", zapS("job", name))
	}
}

// CloseExpiredOrders 关闭超时未支付订单（每 5 分钟）。
func (s *CronService) CloseExpiredOrders(ctx context.Context) {
	minutes := 30
	type orderCfg struct {
		ExpireMinutes int `json:"expire_minutes"`
	}
	if raw, err := s.repos.Setting.Get(s.db, "order"); err == nil {
		var c orderCfg
		if json.Unmarshal([]byte(raw), &c) == nil && c.ExpireMinutes > 0 {
			minutes = c.ExpireMinutes
		}
	}
	deadline := time.Now().Add(-time.Duration(minutes) * time.Minute)
	orders, err := s.repos.Order.ListPendingBefore(s.db, deadline)
	if err != nil {
		logger.L().Error("close expired orders", zapE(err))
		return
	}
	closed := 0
	for _, o := range orders {
		err := repo.WithTx(s.db, func(tx *gorm.DB) error {
			locked, err := s.repos.Order.GetByNoForUpdate(tx, o.OrderNo)
			if err != nil {
				return err
			}
			if locked.Status != model.OrderPending {
				return nil // 已支付/已取消，跳过
			}
			if err := s.repos.Order.UpdateStatus(tx, o.OrderNo, model.OrderCanceled); err != nil {
				return err
			}
			// 同步关闭该订单残留的待支付支付单,避免查单任务反复轮询已取消订单(二期完善)
			if _, err := s.repos.Payment.ClosePendingByOrderNo(tx, o.OrderNo); err != nil {
				return err
			}
			// 回退优惠券占用（防止券与配额永久残留）
			if locked.CouponID != nil {
				if err := releaseCoupon(tx, *locked.CouponID, locked.UserID, locked.OrderNo); err != nil {
					return err
				}
			}
			return nil
		})
		if err == nil {
			closed++
		} else {
			logger.L().Error("close order fail", zapS("order_no", o.OrderNo), zapE(err))
		}
	}
	logger.L().Info("close expired orders done", zapS("closed", fmt.Sprint(closed)))
}

// ReconcilePayments 主动查单兜底（每 10 分钟）。
func (s *CronService) ReconcilePayments(ctx context.Context) {
	payments, err := s.repos.Payment.ListPending(s.db)
	if err != nil {
		logger.L().Error("reconcile list payments", zapE(err))
		return
	}
	confirmed := 0
	for _, p := range payments {
		// 订单已非待支付（超时关闭/已取消/已退款）:支付单关闭,不再查单(二期完善)
		order, err := s.repos.Order.GetByNo(s.db, p.OrderNo)
		if err == nil && order.Status != model.OrderPending {
			if _, cerr := s.repos.Payment.ClosePendingByOrderNo(s.db, p.OrderNo); cerr != nil {
				logger.L().Error("reconcile close orphan payment", zapS("order_no", p.OrderNo), zapE(cerr))
			}
			continue
		}
		driver := payment.Get(p.Method)
		if driver == nil {
			continue
		}
		qr, err := driver.Query(ctx, p.OrderNo)
		if err != nil || !qr.Paid {
			continue
		}
		if err := s.orders.HandleNotify(ctx, p.Method, &payment.NotifyResult{
			OrderNo: p.OrderNo, TradeNo: qr.TradeNo, Amount: p.Amount, Paid: true,
		}); err == nil {
			confirmed++
		}
	}
	logger.L().Info("reconcile payments done", zapS("confirmed", fmt.Sprint(confirmed)))
}

// ConfirmCommissions 佣金确认：确认中满 N 天转已发放（每日 02:00）。
func (s *CronService) ConfirmCommissions(ctx context.Context) {
	days := 3
	type inviteCfg struct {
		CommissionConfirmDays int `json:"commission_confirm_days"`
	}
	if raw, err := s.repos.Setting.Get(s.db, "invite"); err == nil {
		var c inviteCfg
		if json.Unmarshal([]byte(raw), &c) == nil && c.CommissionConfirmDays > 0 {
			days = c.CommissionConfirmDays
		}
	}
	deadline := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	list, err := s.repos.Commission.ListPendingConfirmBefore(s.db, deadline)
	if err != nil {
		logger.L().Error("confirm commissions", zapE(err))
		return
	}
	now := time.Now()
	for _, cl := range list {
		err := repo.WithTx(s.db, func(tx *gorm.DB) error {
			// 条件更新（status=0→1）：已被退款撤销的佣金不发放
			affected, err := s.repos.Commission.UpdateStatusIfPending(tx, cl.ID)
			if err != nil {
				return err
			}
			if affected == 0 {
				return nil
			}
			inviter, err := s.repos.User.GetByIDForUpdate(tx, cl.InviteUserID)
			if err != nil {
				return err
			}
			inviter.CommissionBalance += cl.Amount
			if err := s.repos.User.Save(tx, inviter); err != nil {
				return err
			}
			confirmed := &cl
			confirmed.ConfirmedAt = &now
			return s.repos.Commission.Save(tx, confirmed)
		})
		if err != nil {
			logger.L().Error("confirm commission fail", zapS("order_no", cl.OrderNo), zapE(err))
		}
	}
	logger.L().Info("confirm commissions done", zapS("count", fmt.Sprint(len(list))))
}

// ExpireRemind 到期提醒（每日 10:00）：到期前 3 天与 1 天各发一次（按窗口去重）。
func (s *CronService) ExpireRemind(ctx context.Context) {
	var users []model.User
	now := time.Now()
	// 两个窗口：剩余 (48h, 72h] 视为「3 天」；(0, 24h] 视为「1 天」
	if err := s.db.Where("remind_expire = true AND expired_at IS NOT NULL AND is_banned = false AND "+
		"((expired_at > ? AND expired_at <= ?) OR (expired_at > ? AND expired_at <= ?))",
		now.Add(48*time.Hour), now.Add(72*time.Hour), now, now.Add(24*time.Hour)).
		Find(&users).Error; err != nil {
		logger.L().Error("expire remind query", zapE(err))
		return
	}
	for _, u := range users {
		remaining := time.Until(*u.ExpiredAt)
		var window string
		switch {
		case remaining > 48*time.Hour && remaining <= 72*time.Hour:
			window = "3d"
		case remaining > 0 && remaining <= 24*time.Hour:
			window = "1d"
		default:
			continue
		}
		markKey := redispkg.Key("remind", "expire", fmt.Sprint(u.ID), fmt.Sprint(u.ExpiredAt.Unix()), window)
		ok, _ := s.rdb.SetNX(ctx, markKey, "1", 30*24*time.Hour).Result()
		if !ok {
			continue
		}
		go s.sendExpireMail(u)
	}
	logger.L().Info("expire remind done", zapS("count", fmt.Sprint(len(users))))
}

func (s *CronService) sendExpireMail(u model.User) {
	if u.ExpiredAt == nil {
		return
	}
	date := u.ExpiredAt.Format("2006-01-02")
	body := mailer.Template(fmt.Sprintf("您的订阅将于 <b>%s</b> 到期，请及时续费以免影响使用。", date))
	rendered, err := mailer.Render(body, s.cfg.App.Name, nil)
	if err != nil {
		return
	}
	subject := fmt.Sprintf("[%s] 订阅到期提醒", s.cfg.App.Name)
	if err := s.mailer.Send(u.Email, subject, rendered); err != nil {
		logger.L().Error("send expire mail failed", zapS("email", u.Email), zapE(err))
	}
}

// TrafficRemind 流量提醒（每日 10:00）：用量 ≥80% 且开启提醒。
func (s *CronService) TrafficRemind(ctx context.Context) {
	var users []model.User
	if err := s.db.Where("remind_traffic = true AND transfer_enable > 0 AND is_banned = false").Find(&users).Error; err != nil {
		logger.L().Error("traffic remind query", zapE(err))
		return
	}
	for _, u := range users {
		if u.U+u.D < u.TransferEnable*8/10 {
			continue
		}
		markKey := redispkg.Key("remind", "traffic", fmt.Sprint(u.ID))
		ok, _ := s.rdb.SetNX(ctx, markKey, "1", 30*24*time.Hour).Result()
		if !ok {
			continue
		}
		percent := (u.U + u.D) * 100 / u.TransferEnable
		body := mailer.Template(fmt.Sprintf("您的流量已使用 <b>%d%%</b>，请注意剩余流量。", percent))
		rendered, _ := mailer.Render(body, s.cfg.App.Name, nil)
		subject := fmt.Sprintf("[%s] 流量使用提醒", s.cfg.App.Name)
		if err := s.mailer.Send(u.Email, subject, rendered); err != nil {
			logger.L().Error("send traffic mail failed", zapS("email", u.Email), zapE(err))
		}
	}
	logger.L().Info("traffic remind done")
}

// TrafficDaily 流量日结转（模式 B 空跑，供二期对账）。
func (s *CronService) TrafficDaily(ctx context.Context) {
	logger.L().Info("traffic daily (mode B: no-op)")
}

// AgentAudit 代理商月度复核（core-flows 第 5 节）：有效邀请人数不满足阈值 → 降级 role=0。
func (s *CronService) AgentAudit(ctx context.Context) {
	required, validDays := agentPolicy(s.db)
	var agents []model.User
	if err := s.db.Where("role = ?", model.RoleAgent).Find(&agents).Error; err != nil {
		logger.L().Error("agent audit query", zapE(err))
		return
	}
	downgraded := 0
	for _, a := range agents {
		valid, err := s.repos.User.CountValidInvited(s.db, a.ID, validDays)
		if err != nil {
			continue
		}
		if valid < int64(required) {
			if err := s.repos.User.UpdateRole(s.db, a.ID, model.RoleUser); err == nil {
				downgraded++
				// 降级 → bump 会话版本号：旧 access token 的 role 快照立即失效
				bumpSessionVersion(ctx, s.rdb, a.ID)
				logger.L().Info("agent downgraded", zapS("user_id", fmt.Sprint(a.ID)))
			}
		}
	}
	logger.L().Info("agent audit done", zapS("downgraded", fmt.Sprint(downgraded)))
}

// zapS / zapE 便捷字段构造。
func zapS(k, v string) zap.Field { return zap.String(k, v) }
func zapE(err error) zap.Field   { return zap.Error(err) }
