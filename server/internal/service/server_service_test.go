package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/repo"
)

func TestServerListWithoutPlan(t *testing.T) {
	e, _ := newContentEnv(t)
	// 无订阅用户
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "role", "balance", "commission_balance", "invite_by_id",
			"is_banned", "remind_expire", "remind_traffic", "telegram_id", "plan_id", "expired_at",
			"transfer_enable", "u", "d", "speed_limit", "device_limit", "sub_token", "created_at", "updated_at",
		}).AddRow(1, "a@b.com", "h", 0, 0, 0, nil, false, true, false, nil, nil, nil, 0, 0, 0, nil, nil, "t", nowTime(), nowTime()))

	srvSvc := NewServerService(e.db, &repo.Repos{})
	groups, err := srvSvc.List(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, groups)
}
