package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/model"
)

// claim ica2-6：行为变更=节点心跳读写收敛在线状态深 Module（Key+TTL+富化归一）；
// 必须不变=上报写心跳与快照、列表富化在线/负载、TTL 语义、mock 与契约同形；
// 最怕的失败=Key 改一处而另一处未动致在线状态全灭、形状漂移致前端气泡卡白屏。
func TestPresenceRoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	// 写端只报原始心跳
	sys := &model.NodeSystemStats{CPUPercent: 18.5, MemoryPercent: 37.5, DiskPercent: 40, NetSpeedIn: 1 << 20, NetSpeedOut: 2 << 20, Connections: 42, Uptime: 864000}
	presenceReport(ctx, rdb, 7, sys)

	// 读端问在线否、负载如何
	views := []model.AdminServerView{{ID: 7, Name: "hk1"}, {ID: 8, Name: "us1"}}
	presenceEnrich(ctx, rdb, views)
	require.True(t, views[0].IsOnline)
	require.NotNil(t, views[0].LastHeartbeatAt)
	require.NotNil(t, views[0].System)
	assert.Equal(t, 18.5, views[0].System.CPUPercent)
	assert.Equal(t, 42, views[0].System.Connections)
	// 无心跳 → 离线，负载 nil
	assert.False(t, views[1].IsOnline)
	assert.Nil(t, views[1].LastHeartbeatAt)
	assert.Nil(t, views[1].System)

	// TTL 语义：快照快过期于心跳
	hbTTL := rdb.TTL(ctx, presenceHeartbeatKey(7)).Val()
	stTTL := rdb.TTL(ctx, presenceStatsKey(7)).Val()
	assert.InDelta(t, PresenceHeartbeatTTL.Seconds(), hbTTL.Seconds(), 5)
	assert.InDelta(t, PresenceStatsTTL.Seconds(), stTTL.Seconds(), 5)

	// 心跳过期 → 离线（快进 TTL）
	mr.FastForward(PresenceHeartbeatTTL + time.Second)
	views2 := []model.AdminServerView{{ID: 7}}
	presenceEnrich(ctx, rdb, views2)
	assert.False(t, views2[0].IsOnline)

	// 坏快照降级 nil，不阻断在线判定
	require.NoError(t, rdb.Set(ctx, presenceHeartbeatKey(9), "123", time.Minute).Err())
	require.NoError(t, rdb.Set(ctx, presenceStatsKey(9), "{bad", time.Minute).Err())
	views3 := []model.AdminServerView{{ID: 9}}
	presenceEnrich(ctx, rdb, views3)
	assert.True(t, views3[0].IsOnline)
	assert.Nil(t, views3[0].System)

	// nil rdb / 空集 no-op 不恐慌
	presenceReport(ctx, nil, 1, nil)
	presenceEnrich(ctx, nil, nil)
	presenceEnrich(ctx, rdb, nil)
}
