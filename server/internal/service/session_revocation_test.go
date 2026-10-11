package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	redispkg "ylink-backend/internal/pkg/redis"
)

// claim ica2-1：行为变更=吊销意图收敛到 SessionRevoker（白名单/SV/kill 三键归一）；
// 必须不变=旧 access 在 bump 后 401、单踢只影响目标会话、Refresh 旋转语义不变；
// 最怕的失败=漏 bump 导致旧权限静默留存、kill 误删其余会话。
func TestSessionRevokerMatrix(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	v := NewSessionRevoker(rdb, time.Hour)

	// 全量吊销：白名单清 + SV bump
	require.NoError(t, rdb.Set(ctx, redispkg.RefreshKey(42, "a"), `{"ip":"1","ua":"u","ts":"2026-01-01T00:00:00Z"}`, time.Hour).Err())
	require.NoError(t, rdb.Set(ctx, redispkg.RefreshKey(42, "b"), `{"ip":"1","ua":"u","ts":"2026-01-01T00:00:00Z"}`, time.Hour).Err())
	require.NoError(t, v.RevokeAll(ctx, 42, RevokeReasonBan))
	assert.Equal(t, int64(1), v.CurrentVersion(ctx, 42))
	assert.False(t, v.Allowed(ctx, 42, "a", 0)) // 旧 sv 快照失效
	assert.True(t, v.Allowed(ctx, 42, "a", 1))
	exists, _ := rdb.Exists(ctx, redispkg.RefreshKey(42, "a")).Result()
	assert.Equal(t, int64(0), exists)

	// 保留式吊销：keepJTI 留存
	require.NoError(t, rdb.Set(ctx, redispkg.RefreshKey(43, "keep"), "{}", time.Hour).Err())
	require.NoError(t, rdb.Set(ctx, redispkg.RefreshKey(43, "drop"), "{}", time.Hour).Err())
	require.NoError(t, v.RevokeOthers(ctx, 43, "keep"))
	keepExists, _ := rdb.Exists(ctx, redispkg.RefreshKey(43, "keep")).Result()
	dropExists, _ := rdb.Exists(ctx, redispkg.RefreshKey(43, "drop")).Result()
	assert.Equal(t, int64(1), keepExists)
	assert.Equal(t, int64(0), dropExists)

	// 单踢：只影响目标会话
	require.NoError(t, rdb.Set(ctx, redispkg.RefreshKey(44, "victim"), "{}", time.Hour).Err())
	require.NoError(t, rdb.Set(ctx, redispkg.RefreshKey(44, "other"), "{}", time.Hour).Err())
	require.NoError(t, v.RevokeOne(ctx, 44, "victim", "other"))
	assert.False(t, v.Allowed(ctx, 44, "victim", 0))
	assert.True(t, v.Allowed(ctx, 44, "other", 0))

	// 自踢拒绝
	require.Error(t, v.RevokeOne(ctx, 44, "other", "other"))

	// 列表形状：空集非 nil，元数据解析
	list, err := v.List(ctx, 44, "other")
	require.NoError(t, err)
	require.NotNil(t, list)
	empty, err := v.List(ctx, 99999, "")
	require.NoError(t, err)
	assert.NotNil(t, empty)
	assert.Empty(t, empty)
}
