package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/resp"
)

// Health 提供健康检查端点。
type Health struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHealth(db *gorm.DB, rdb *redis.Client) *Health {
	return &Health{db: db, rdb: rdb}
}

// Liveness GET /healthz —— 进程存活。
func (h *Health) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness GET /readyz —— DB/Redis 连通。
// DB 探测同样走请求 ctx 派生的 2s 超时；失败只回通用文案（端点在 SafeMode/api
// 分组之外、公网可达，不泄漏 DSN/主机等内部拓扑），具体错误进日志。
func (h *Health) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := dbPing(h.db, ctx); err != nil {
		h.unavailable(c, "db", err)
		return
	}
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		h.unavailable(c, "redis", err)
		return
	}
	c.JSON(http.StatusOK, resp.Body{Code: 0, Message: "ok", Data: gin.H{"status": "ready"}})
}

// unavailable 就绪失败统一出口：对外 503 + 通用文案，细节只进日志。
func (h *Health) unavailable(c *gin.Context, dep string, err error) {
	logger.L().Error("readyz probe failed", zap.String("dep", dep), zap.Error(err))
	c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
}

func dbPing(db *gorm.DB, ctx context.Context) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
