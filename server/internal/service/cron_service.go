package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
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
	tg     *TelegramService
}

func NewCronService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, m *mailer.Mailer, orders *OrderService, tg *TelegramService) *CronService {
	return &CronService{db: db, rdb: rdb, repos: repos, cfg: cfg, mailer: m, orders: orders, tg: tg}
}

// WithLock 分布式锁包装：仅单实例执行任务。
// 薄 adapter：转发到 Cron 调度深 Module 的锁 seam；打点语义收敛于 Module 内。
// 打点 cron_job_runs_total / cron_job_duration_seconds（worker /metrics 暴露，供 Grafana 看板与告警）。
func (s *CronService) WithLock(name string, fn func()) func() {
	return (&cronGuard{rdb: s.rdb}).run(name, fn)
}

// CloseExpiredOrders 关闭超时未支付订单（每 5 分钟）。
// 关单窗口问策略 seam（order.expire_minutes，默认 30min）。
func (s *CronService) CloseExpiredOrders(ctx context.Context) {
	minutes := (&cronPolicy{db: s.db, repos: s.repos}).orderExpireMinutes()
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
// 确认窗口问策略 seam（invite.commission_confirm_days，默认 3d）。
func (s *CronService) ConfirmCommissions(ctx context.Context) {
	days := (&cronPolicy{db: s.db, repos: s.repos}).commissionConfirmDays()
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
		// 窗口分类收敛于调度深 Module 的策略 seam
		window := expireWindow(remaining)
		if window == "" {
			continue
		}
		markKey := redispkg.Key("remind", "expire", fmt.Sprint(u.ID), fmt.Sprint(u.ExpiredAt.Unix()), window)
		ok, _ := s.rdb.SetNX(ctx, markKey, "1", 30*24*time.Hour).Result()
		if !ok {
			continue
		}
		// 投递收敛于深 Module 的投递 seam（异步带 recover，不阻断主循环）
		n := &cronNotifier{db: s.db, cfg: s.cfg, mailer: s.mailer, tg: s.tg}
		u := u
		n.deliverAsync(func() { n.deliverExpire(u) })
	}
	logger.L().Info("expire remind done", zapS("count", fmt.Sprint(len(users))))
}

func (s *CronService) sendExpireMail(u model.User) {
	// 薄 adapter：转发到调度深 Module 的投递 seam。
	(&cronNotifier{db: s.db, cfg: s.cfg, mailer: s.mailer, tg: s.tg}).deliverExpire(u)
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
		// 投递收敛于深 Module 的投递 seam（mail 主通道 + TG 追加，失败只记日志）
		(&cronNotifier{db: s.db, cfg: s.cfg, mailer: s.mailer, tg: s.tg}).deliverTraffic(ctx, u, percent)
	}
	logger.L().Info("traffic remind done")
}

// TrafficDaily 流量日结转（流量模式B占位：管理端手工导入同日覆盖模式A值做校准，
// 本 job 为空跑，供二期对账；调度语义收敛于深 Module，intent 保留）。
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
				// 降级 → 经会话吊销深 Module 全量吊销：旧 access 的 role 快照立即失效
				_ = NewSessionRevoker(s.rdb, 0).RevokeAll(ctx, a.ID, RevokeReasonAgentDowngrade)
				logger.L().Info("agent downgraded", zapS("user_id", fmt.Sprint(a.ID)))
			}
		}
	}
	logger.L().Info("agent audit done", zapS("downgraded", fmt.Sprint(downgraded)))
}

// zapS / zapE 便捷字段构造。
func zapS(k, v string) zap.Field { return zap.String(k, v) }
func zapE(err error) zap.Field   { return zap.Error(err) }
