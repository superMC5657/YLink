// 定时任务进程入口（cron 独立部署）。
// 任务清单见 docs/backend/core-flows.md 第 9 节；全部任务带分布式锁。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"nanocloud/internal/config"
	"nanocloud/internal/pkg/logger"
	"nanocloud/internal/pkg/mailer"
	"nanocloud/internal/pkg/redis"
	"nanocloud/internal/repo"
	"nanocloud/internal/service"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := logger.Init(cfg.Log); err != nil {
		log.Fatalf("init logger: %v", err)
	}

	ctx := context.Background()
	db, err := repo.NewDB(cfg.Database)
	if err != nil {
		logger.L().Fatal("connect database", zap.Error(err))
	}
	rdb, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		logger.L().Fatal("connect redis", zap.Error(err))
	}

	repos := &repo.Repos{}
	orders := service.NewOrderService(db, rdb, repos, service.NewSettingService(db, rdb, repos), cfg)
	cronSvc := service.NewCronService(db, rdb, repos, cfg, mailer.New(cfg.SMTP), orders)

	c := cron.New(cron.WithSeconds())

	// 关闭 30 分钟未支付订单（每 5 分钟）
	c.AddFunc("0 */5 * * * *", cronSvc.WithLock("close-expired-orders", func() {
		cronSvc.CloseExpiredOrders(ctx)
	}))
	// 主动查单兜底（每 10 分钟）
	c.AddFunc("0 */10 * * * *", cronSvc.WithLock("reconcile-payments", func() {
		cronSvc.ReconcilePayments(ctx)
	}))
	// 佣金确认（每日 02:00）
	c.AddFunc("0 0 2 * * *", cronSvc.WithLock("confirm-commissions", func() {
		cronSvc.ConfirmCommissions(ctx)
	}))
	// 到期提醒（每日 10:00）
	c.AddFunc("0 0 10 * * *", cronSvc.WithLock("expire-remind", func() {
		cronSvc.ExpireRemind(ctx)
	}))
	// 流量提醒（每日 10:30）
	c.AddFunc("0 30 10 * * *", cronSvc.WithLock("traffic-remind", func() {
		cronSvc.TrafficRemind(ctx)
	}))
	// 流量日结转（每日 01:00，模式 B 空跑）
	c.AddFunc("0 0 1 * * *", cronSvc.WithLock("traffic-daily", func() {
		cronSvc.TrafficDaily(ctx)
	}))

	c.Start()
	logger.L().Info("worker started with cron jobs", zap.String("tz", cfg.App.Name))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	stopCtx := c.Stop()
	<-stopCtx.Done()
	logger.L().Info("worker exited")
}
