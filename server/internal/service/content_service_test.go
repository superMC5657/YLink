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

func newContentEnv(t *testing.T) (*testEnv, *ContentService) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewContentService(e.db, e.rdb, &repo.Repos{}, set)
	return e, svc
}

func settingsRow(value string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"value"}).AddRow(value)
}

func TestSiteConfig(t *testing.T) {
	e, svc := newContentEnv(t)

	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT `value` FROM `settings`")).
		WillReturnRows(settingsRow(`{"site_name":"TestCloud","site_logo":"","site_description":"desc","register_enabled":true,"invite_code_required":true,"app_downloads":{},"telegram":{},"customer_service_url":"https://t.me/x","free_traffic_tips":"tip","languages":["zh-CN"]}`))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT `value` FROM `settings`")).
		WillReturnRows(settingsRow(`{"methods":[{"code":"balance","name":"余额支付","icon":"wallet","enabled":true}]}`))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT `value` FROM `settings`")).
		WillReturnRows(settingsRow(`{"required_valid_invites":50,"benefits":["b1"],"notes":["n1"]}`))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT `value` FROM `settings`")).
		WillReturnRows(settingsRow(`{"commission_rate":40}`))

	cfg, err := svc.SiteConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "TestCloud", cfg.SiteName)
	assert.True(t, cfg.RegisterEnabled)
	assert.True(t, cfg.InviteCodeRequired)
	assert.Equal(t, "https://t.me/x", cfg.CustomerServiceURL)
	assert.Equal(t, 50, cfg.AgentPolicy.RequiredValidInvites)
	assert.Equal(t, 40, cfg.AgentPolicy.CommissionRate)
	assert.Len(t, cfg.PaymentMethods, 1)
	assert.Equal(t, []string{"zh-CN"}, cfg.Languages)
}

func TestKnowledgesGroups(t *testing.T) {
	e, svc := newContentEnv(t)

	rows := sqlmock.NewRows([]string{"id", "category", "title", "body", "language", "is_show", "sort", "created_at", "updated_at"}).
		AddRow(31, "安卓配置教程", "Nano (推荐使用)", "", "zh-CN", 1, 1, nowTime(), nowTime()).
		AddRow(32, "安卓配置教程", "Clash Meta (备用)", "", "zh-CN", 1, 2, nowTime(), nowTime()).
		AddRow(40, "防失联", "TG 群组", "", "zh-CN", 1, 1, nowTime(), nowTime())
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `knowledges`")).WillReturnRows(rows)

	groups, err := svc.Knowledges(context.Background(), "zh-CN", "")
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "安卓配置教程", groups[0].Category)
	assert.Len(t, groups[0].Items, 2)
	assert.Equal(t, "防失联", groups[1].Category)
}

func TestKnowledgeDetailNotFound(t *testing.T) {
	e, _ := newContentEnv(t)
	svc := NewContentService(e.db, e.rdb, &repo.Repos{}, NewSettingService(e.db, e.rdb, &repo.Repos{}))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `knowledges`")).WillReturnError(assert.AnError)
	_, err := svc.KnowledgeDetail(context.Background(), 99)
	assert.Equal(t, 40400, codeOf(err))
}

func TestServerListWithoutPlan(t *testing.T) {
	e, _ := newContentEnv(t)
	// 无订阅用户
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users`")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password_hash", "role", "balance", "commission_balance", "invite_by_id",
			"is_banned", "remind_expire", "remind_traffic", "telegram_id", "plan_id", "expired_at",
			"transfer_enable", "u", "d", "speed_limit", "device_limit", "sub_token", "created_at", "updated_at",
		}).AddRow(1, "a@b.com", "h", 0, 0, 0, nil, false, true, false, nil, nil, nil, 0, 0, 0, nil, nil, "t", nowTime(), nowTime()))

	srvSvc := NewServerService(e.db, e.rdb, &repo.Repos{})
	groups, err := srvSvc.List(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, groups)
}

func nowTime() any { return model.Time{} }
