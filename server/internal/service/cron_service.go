package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"nanocloud/internal/config"
	"nanocloud/internal/model"
	"nanocloud/internal/pkg/logger"
	"nanocloud/internal/pkg/mailer"
	"nanocloud/internal/pkg/payment"
	redispkg "nanocloud/internal/pkg/redis"
	"nanocloud/internal/repo"
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
func (s *CronService) WithLock(name string, fn func()) func() {
	return func() {
		ctx := context.Background()
		ok, err := s.rdb.SetNX(ctx, redispkg.Key("cron", "lock", name), "1", 15*time.Minute).Result()
		if err != nil || !ok {
			logger.L().Info("cron lock skipped", zapS("job", name))
			return
		}
		defer s.rdb.Del(ctx, redispkg.Key("cron", "lock", name))
		logger.L().Info("cron job start", zapS("job", name))
		fn()
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
		// 条件更新（status=0），已支付/已完成订单不会被误关
		if err := s.repos.Order.UpdateStatusIfPending(s.db, o.OrderNo, model.OrderCanceled); err == nil {
			closed++
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
			inviter, err := s.repos.User.GetByIDForUpdate(tx, cl.InviteUserID)
			if err != nil {
				return err
			}
			inviter.CommissionBalance += cl.Amount
			if err := s.repos.User.Save(tx, inviter); err != nil {
				return err
			}
			cl.Status = model.CommissionGranted
			cl.ConfirmedAt = &now
			return s.repos.Commission.Save(tx, &cl)
		})
		if err != nil {
			logger.L().Error("confirm commission fail", zapS("order_no", cl.OrderNo), zapE(err))
		}
	}
	logger.L().Info("confirm commissions done", zapS("count", fmt.Sprint(len(list))))
}

// ExpireRemind 到期提醒（每日 10:00）：到期前 3 天内且开启提醒。
func (s *CronService) ExpireRemind(ctx context.Context) {
	var users []model.User
	from := time.Now()
	to := from.Add(72 * time.Hour)
	if err := s.db.Where("remind_expire = 1 AND expired_at IS NOT NULL AND expired_at > ? AND expired_at <= ? AND is_banned = 0", from, to).Find(&users).Error; err != nil {
		logger.L().Error("expire remind query", zapE(err))
		return
	}
	for _, u := range users {
		markKey := redispkg.Key("remind", "expire", fmt.Sprint(u.ID), fmt.Sprint(u.ExpiredAt.Unix()))
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
	if err := s.db.Where("remind_traffic = 1 AND transfer_enable > 0 AND is_banned = 0").Find(&users).Error; err != nil {
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

// zapS / zapE 便捷字段构造。
func zapS(k, v string) zap.Field { return zap.String(k, v) }
func zapE(err error) zap.Field   { return zap.Error(err) }
