package handler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ylink-backend/internal/pkg/logger"
)

func init() { logger.Nop() }

// 缝测试：/readyz 的 DB 探测必须受请求 ctx 超时约束（DB 挂起时 2s 内返回），
// 且失败响应不含内部错误细节（端点公网可达）。最怕的失败=探针无限挂起拖垮
// 编排健康检查、或把 DSN/主机拓扑回显给公网调用方。

// hangConn 模拟挂起的 DB 连接：真实探测（ctx 可取消）时阻塞到 ctx 取消；
// gorm.Open 握手阶段的 Ping 走 Background ctx，放行以免初始化挂死。
type hangConn struct{}

func (c *hangConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *hangConn) Close() error                        { return nil }
func (c *hangConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }
func (c *hangConn) Ping(ctx context.Context) error {
	if ctx.Done() == nil {
		return nil
	}
	<-ctx.Done()
	return ctx.Err()
}

type hangDriver struct{}

func (hangDriver) Open(string) (driver.Conn, error) { return &hangConn{}, nil }

type hangConnector struct{}

func (hangConnector) Connect(context.Context) (driver.Conn, error) { return &hangConn{}, nil }
func (hangConnector) Driver() driver.Driver                        { return hangDriver{} }

func readinessContext(t *testing.T, h *Health, deadline time.Duration) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/readyz", nil)
	if deadline > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), deadline)
		defer cancel()
		req = req.WithContext(ctx)
	}
	c.Request = req
	start := time.Now()
	h.Readiness(c)
	t.Logf("readiness elapsed: %s", time.Since(start))
	return w
}

// DB 挂起：/readyz 在请求 ctx 截止时（远早于 2s 派生上限）返回 503，响应体只有通用文案。
func TestReadinessDBHangReturnsWithinDeadline(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sql.OpenDB(hangConnector{}), PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	h := NewHealth(db, redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	w := readinessContext(t, h, 100*time.Millisecond)
	assert.Equal(t, 503, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unavailable", body["status"])
	assert.Len(t, body, 1, "失败响应不得携带 db/redis 错误细节")
}

// Redis 不可达：503 + 通用文案，连接错误原文（含地址）不回显。
func TestReadinessRedisDownHidesErrorDetail(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	// 127.0.0.1:1 立即拒绝连接，错误原文含地址——用于断言脱敏
	h := NewHealth(db, redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))

	w := readinessContext(t, h, 0)
	assert.Equal(t, 503, w.Code)
	assert.JSONEq(t, `{"status":"unavailable"}`, w.Body.String())
	assert.NotContains(t, w.Body.String(), "127.0.0.1")
	assert.NotContains(t, w.Body.String(), "dial")
}

// DB/Redis 均连通：200 + ready。
func TestReadinessHealthy(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	h := NewHealth(db, redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	w := readinessContext(t, h, 0)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}
