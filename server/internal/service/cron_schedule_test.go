package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/repo"
)

// claim ica2-3：行为变更=Cron 调度/策略/投递收敛深 Module（锁+窗口+双扇出归一）；
// 必须不变=关单/对账/确认语义、提醒双窗口、lock 占有时跳过、TG 失败不阻断；
// 最怕的失败=锁语义丢失致多实例并发、窗口改错致漏提醒、投递恐慌崩 worker。
func TestCronLockSkip(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	g := &cronGuard{rdb: rdb}

	ran := 0
	job := g.run("ut-job", func() { ran++ })
	job()
	assert.Equal(t, 1, ran)

	// 锁被占用 → 跳过不执行
	require.NoError(t, rdb.Set(context.Background(), "cron:lock:ut-job", "1", time.Minute).Err())
	job2 := g.run("ut-job", func() { ran++ })
	job2()
	assert.Equal(t, 1, ran)
}

func TestExpireWindowMatrix(t *testing.T) {
	assert.Equal(t, "3d", expireWindow(72*time.Hour))
	assert.Equal(t, "3d", expireWindow(49*time.Hour))
	assert.Equal(t, "", expireWindow(48*time.Hour)) // 边界外
	assert.Equal(t, "", expireWindow(30*time.Hour))
	assert.Equal(t, "1d", expireWindow(24*time.Hour))
	assert.Equal(t, "1d", expireWindow(time.Hour))
	assert.Equal(t, "", expireWindow(0))
	assert.Equal(t, "", expireWindow(-time.Hour))
	assert.Equal(t, "", expireWindow(73*time.Hour))
}

func TestCronPolicyDefaults(t *testing.T) {
	e := newTestEnv(t)
	p := &cronPolicy{db: e.db, repos: &repo.Repos{}}
	// settings 表无行 → 默认值（mock 返回 error）
	e.mock.ExpectQuery("SELECT .* FROM \"settings\"").WillReturnError(assert.AnError)
	assert.Equal(t, 30, p.orderExpireMinutes())
	e.mock.ExpectQuery("SELECT .* FROM \"settings\"").WillReturnError(assert.AnError)
	assert.Equal(t, 3, p.commissionConfirmDays())
	require.NoError(t, e.mock.ExpectationsWereMet())
}
