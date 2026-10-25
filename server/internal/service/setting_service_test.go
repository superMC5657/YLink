package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/model"
	"ylink-backend/internal/repo"
)

// settings 缓存缝测试：热路径（佣金比例、telegram 配置）读取 60s 内只查一次库，
// 语句数不随调用次数增长（#20 Finding「缓存旁路」的回归断言）。

func settingsValueRow(raw string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"value"}).AddRow(raw)
}

// GetInt 同 key 多次读取只查一次库；失效后重新查库。
func TestSettingGetIntCached(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	ctx := context.Background()

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnRows(settingsValueRow(`{"commission_rate":30}`))

	assert.Equal(t, 30, set.GetInt(ctx, "invite", "commission_rate", 40))
	// 命中缓存：不再查库（sqlmock 未预置多余期望，多查即失败）
	assert.Equal(t, 30, set.GetInt(ctx, "invite", "commission_rate", 40))
	assert.Equal(t, 50, set.GetInt(ctx, "invite", "agent_commission_rate", 50), "同 key 第二字段也走缓存 raw")
	require.NoError(t, e.mock.ExpectationsWereMet())

	// 失效后重新查库
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnRows(settingsValueRow(`{"commission_rate":35}`))
	set.Invalidate(ctx, "invite")
	assert.Equal(t, 35, set.GetInt(ctx, "invite", "commission_rate", 40))
	require.NoError(t, e.mock.ExpectationsWereMet())
}

// commissionRateFor 每笔支付订单执行：两字段同 key 一次查库，重复调用零查库。
func TestCommissionRateForCached(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	ctx := context.Background()

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnRows(settingsValueRow(`{"commission_rate":40,"agent_commission_rate":60}`))

	assert.Equal(t, 40, commissionRateFor(ctx, set, model.RoleUser))
	assert.Equal(t, 60, commissionRateFor(ctx, set, model.RoleAgent))
	assert.Equal(t, 40, commissionRateFor(ctx, set, model.RoleUser))
	require.NoError(t, e.mock.ExpectationsWereMet())
}

// telegram settings() 经缓存：同一 env 内多次读取只查一次库。
func TestTelegramSettingsCached(t *testing.T) {
	e := newTestEnv(t)
	svc := NewTelegramService(e.db, e.rdb, &repo.Repos{}, nil, nil)

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnRows(settingsValueRow(tgSettingsJSON))

	st := svc.settings(context.Background())
	assert.True(t, st.Enabled)
	assert.Equal(t, "s3cret", st.WebhookSecret)
	st = svc.settings(context.Background())
	assert.Equal(t, "123:ABC", st.BotToken)
	require.NoError(t, e.mock.ExpectationsWereMet())
}

// jsonIntOf 默认值回退矩阵：缺失/非正/类型不符/整体坏 JSON/读取失败。
func TestSettingIntDefaults(t *testing.T) {
	assert.Equal(t, 40, jsonIntOf(`{}`, "commission_rate", 40), "字段缺失")
	assert.Equal(t, 40, jsonIntOf(`{"commission_rate":0}`, "commission_rate", 40), "非正值")
	assert.Equal(t, 40, jsonIntOf(`{"commission_rate":-1}`, "commission_rate", 40), "负值")
	assert.Equal(t, 40, jsonIntOf(`{"commission_rate":"40"}`, "commission_rate", 40), "字符串类型")
	assert.Equal(t, 40, jsonIntOf(`not-json`, "commission_rate", 40), "坏 JSON")
	assert.Equal(t, 25, jsonIntOf(`{"expire_minutes":25}`, "expire_minutes", 30), "正常值不回退")

	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT "value" FROM "settings"`)).
		WillReturnError(assert.AnError)
	assert.Equal(t, 30, set.GetInt(context.Background(), "order", "expire_minutes", 30), "读取失败回退默认")
	require.NoError(t, e.mock.ExpectationsWereMet())
}
