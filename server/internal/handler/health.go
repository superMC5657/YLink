package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink/internal/pkg/resp"
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
func (h *Health) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := dbPing(h.db); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "db": err.Error()})
		return
	}
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "redis": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Body{Code: 0, Message: "ok", Data: gin.H{"status": "ready"}})
}

func dbPing(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(context.Background())
}
