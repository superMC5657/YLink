package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

func newUniProxyEnv(t *testing.T) (*testEnv, *UniProxyService) {
	e := newTestEnv(t)
	svc := NewUniProxyService(e.db, e.rdb, &repo.Repos{})
	return e, svc
}

func TestUniProxyService_GetConfig(t *testing.T) {
	e, svc := newUniProxyEnv(t)
	ctx := context.Background()

	srv := &model.Server{
		ID:       1,
		GroupID:  1,
		Name:     "HK-VLESS",
		Type:     "vless",
		Host:     "hk.example.com",
		Port:     443,
		Config:   `{"network":"tcp","security":"reality","sni":"hk.example.com","dest":"www.apple.com:443"}`,
		Rate:     1.5,
		Status:   1,
		NodeKey:  "node-key-1",
	}

	// 1. 成功获取配置
	e.expectServerByID(srv)
	cfg, err := svc.GetConfig(ctx, 1, "vless")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 443, cfg.ServerPort)
	assert.Equal(t, "vless", cfg.Protocol)
	assert.Equal(t, 1, cfg.TLS)
	assert.NotNil(t, cfg.Data)
	assert.Equal(t, 443, cfg.Data.ServerPort)

	// 2. 协议兼容测试：v2ray 与 vless 兼容
	e.expectServerByID(srv)
	cfgCompat, err := svc.GetConfig(ctx, 1, "v2ray")
	require.NoError(t, err)
	assert.Equal(t, "vless", cfgCompat.Protocol)

	// 3. 协议不兼容测试：hysteria2 与 vless 不兼容
	e.expectServerByID(srv)
	_, err = svc.GetConfig(ctx, 1, "hysteria2")
	require.Error(t, err)
	var eErr *errs.Error
	require.ErrorAs(t, err, &eErr)
	assert.Equal(t, 40000, eErr.Code)

	// 4. 节点不存在
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnError(errs.ErrNotFound)
	_, err = svc.GetConfig(ctx, 999, "")
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestUniProxyService_GetUsers(t *testing.T) {
	e, svc := newUniProxyEnv(t)
	ctx := context.Background()

	srv := &model.Server{
		ID:      2,
		GroupID: 1,
		Name:    "US-Trojan",
		Type:    "trojan",
		Host:    "us.example.com",
		Port:    443,
		Rate:    1.0,
		Status:  1,
		NodeKey: "node-key-2",
	}
	e.expectServerByID(srv)

	p1 := &model.Plan{ID: 5, Name: "VIP Plan", GroupIDs: "[1]", IsShow: true}
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).WillReturnRows(planRow(p1))

	speedLimit := 100
	deviceLimit := 3
	u1 := &model.User{
		ID:          101,
		Email:       "u1@example.com",
		UUID:        "uuid-101",
		PlanID:      int64Ptr(5),
		SpeedLimit:  &speedLimit,
		DeviceLimit: &deviceLimit,
	}
	u2 := &model.User{
		ID:          102,
		Email:       "u2@example.com",
		UUID:        "uuid-102",
		PlanID:      int64Ptr(5),
	}
	e.expectActiveUsers([]model.User{*u1, *u2})

	resp, err := svc.GetUsers(ctx, 2)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, int64(101), resp.Users[0].ID)
	assert.Equal(t, "uuid-101", resp.Users[0].UUID)
	assert.Equal(t, int64(100), resp.Users[0].SpeedLimit)
	assert.Equal(t, 3, resp.Users[0].DeviceLimit)

	assert.Equal(t, int64(102), resp.Users[1].ID)
	assert.Equal(t, int64(0), resp.Users[1].SpeedLimit)
	assert.Equal(t, 0, resp.Users[1].DeviceLimit)

	assert.NotNil(t, resp.Data)
	assert.Len(t, resp.Data.Users, 2)
}

func TestUniProxyService_PushTraffic(t *testing.T) {
	e, svc := newUniProxyEnv(t)
	ctx := context.Background()

	srv := &model.Server{
		ID:      3,
		GroupID: 1,
		Rate:    1.5, // 1.5 倍率
		Status:  1,
	}
	e.expectServerByID(srv)

	p1 := &model.Plan{ID: 8, GroupIDs: "[1]"}
	future := time.Now().Add(24 * time.Hour)
	u1 := &model.User{
		ID:             201,
		UUID:           "uuid-201",
		PlanID:         int64Ptr(8),
		ExpiredAt:      &future,
		SubToken:       "token-201",
		TransferEnable: 100 << 30,
	}

	// ListByIDs
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id IN`)).
		WillReturnRows(userRows([]model.User{*u1}))

	// planIDsContainingGroup
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).
		WillReturnRows(planRow(p1))

	// Tx
	e.mock.ExpectBegin()
	// IncrTraffic: 1000 * 1.5 = 1500, 2000 * 1.5 = 3000
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WithArgs(int64(3000), int64(1500), sqlmock.AnyArg(), int64(201)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// AdditiveUpsert: traffic_logs
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "traffic_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	e.mock.ExpectCommit()

	// 预先向 redis 写入该用户的 sub:userinfo 缓存
	subKey := redispkg.Key("sub", "userinfo", "token-201")
	require.NoError(t, e.rdb.Set(ctx, subKey, "dummy", time.Minute).Err())

	items := []model.UniProxyPushItem{
		{UserID: 201, U: 1000, D: 2000},
	}
	accepted, err := svc.PushTraffic(ctx, 3, items)
	require.NoError(t, err)
	assert.Equal(t, 1, accepted)

	// 验证 sub:userinfo 缓存被清除
	val, _ := e.rdb.Get(ctx, subKey).Result()
	assert.Empty(t, val)

	// 验证节点心跳已记录
	hbKey := "node:heartbeat:3"
	hbVal, _ := e.rdb.Get(ctx, hbKey).Result()
	assert.NotEmpty(t, hbVal)
}

func TestUniProxyService_ServerByID(t *testing.T) {
	e, svc := newUniProxyEnv(t)
	ctx := context.Background()

	srv := &model.Server{
		ID:      1,
		Name:    "Test-Server",
		NodeKey: "test-key",
	}
	e.expectServerByID(srv)

	res, err := svc.ServerByID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.ID)
	assert.Equal(t, "test-key", res.NodeKey)
}
