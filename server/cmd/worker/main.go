// 定时任务进程入口（cron 独立部署）。
// 任务清单见 docs/backend/core-flows.md 第 9 节；全部任务带分布式锁。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"ylink-backend/internal/config"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
	"ylink-backend/internal/service"
)

// metricsAddr 为 worker 的 Prometheus 指标端点监听地址。
// 容器部署时仅 compose 内网可访问(prometheus 抓取 worker:8082/metrics),不映射宿主机端口。
const metricsAddr = ":8082"

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
	orders := service.NewOrderService(db, rdb, repos, service.NewSettingService(db, rdb, repos), cfg, mailer.New(cfg.SMTP))
	// F12：到期/流量提醒同步推送 Telegram（未配置 bot 时自动跳过，失败仅记日志）
	tg := service.NewTelegramService(db, rdb, repos, cfg)
	cronSvc := service.NewCronService(db, rdb, repos, cfg, mailer.New(cfg.SMTP), orders, tg)

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
	// 代理商月度复核（每月 1 日 03:00）
	c.AddFunc("0 0 3 1 * *", cronSvc.WithLock("agent-audit", func() {
		cronSvc.AgentAudit(ctx)
	}))

	c.Start()
	logger.L().Info("worker started with cron jobs", zap.String("tz", cfg.App.Name))

	// Prometheus 指标端点（cron 打点见 internal/service/cron_service.go WithLock）。
	metricsSrv := &http.Server{Addr: metricsAddr, Handler: promhttp.Handler()}
	go func() {
		logger.L().Info("worker metrics listening", zap.String("addr", metricsAddr))
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L().Error("worker metrics server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	_ = metricsSrv.Close()
	stopCtx := c.Stop()
	<-stopCtx.Done()
	logger.L().Info("worker exited")
}
