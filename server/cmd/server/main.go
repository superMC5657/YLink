// API 服务入口。
//
//	@title        YLink API
//	@version      1.0
//	@description  代理订阅售卖系统后端 API。统一信封 {code,message,data}；鉴权 Bearer <access_token>。
//	@host         api.example.com
//	@BasePath     /api/v1
//	@securityDefinitions.apikey BearerAuth
//	@in            header
//	@name          Authorization
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"ylink-backend/internal/config"
	jwtpkg "ylink-backend/internal/pkg/jwt"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/pkg/payment"
	"ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
	"ylink-backend/internal/router"
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

	// 首个管理员初始化（环境变量注入，幂等）
	if err := repo.EnsureAdmin(db, os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")); err != nil {
		logger.L().Warn("ensure admin skipped", zap.Error(err))
	}
	// 演示账号初始化（环境变量注入，幂等；仅本地联调用，生产可不设置）
	if err := repo.EnsureDemoUser(db, os.Getenv("DEMO_EMAIL"), os.Getenv("DEMO_PASSWORD")); err != nil {
		logger.L().Warn("ensure demo user skipped", zap.Error(err))
	}

	jwtMgr := jwtpkg.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	mail := mailer.New(cfg.SMTP)

	// 注册支付驱动（易支付：alipay/wxpay 渠道）
	registerPaymentDrivers(cfg)

	engine := router.New(router.Deps{DB: db, Redis: rdb, JWT: jwtMgr, Mailer: mail, Cfg: cfg})

	srv := &http.Server{
		Addr:         cfg.App.Addr,
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		logger.L().Info("server started", zap.String("addr", cfg.App.Addr), zap.String("env", cfg.App.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.L().Fatal("server listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L().Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.L().Error("shutdown error", zap.Error(err))
	}
	logger.L().Info("server exited")
}

// registerPaymentDrivers 按配置注册易支付渠道（epay_alipay / epay_wxpay ...）。
func registerPaymentDrivers(cfg *config.Config) {
	ec := cfg.Payment.Epay
	if ec.Gateway == "" || ec.PID == "" {
		logger.L().Warn("epay not configured, online payment disabled")
		return
	}
	codes := make([]string, 0, len(ec.Methods))
	for _, m := range ec.Methods {
		codes = append(codes, "epay_"+m)
	}
	payment.Register(payment.NewEpay(payment.EpayConfig{
		Gateway: ec.Gateway,
		PID:     ec.PID,
		Key:     ec.Key,
	}), codes...)
	logger.L().Info("epay driver registered", zap.Strings("methods", codes))
}
