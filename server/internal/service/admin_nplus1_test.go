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
	"ylink-backend/internal/repo"
)

// N+1 回归缝测试：列表富化（订单/代理申请/套餐分布）与总览的 SQL 语句数
// 固定，不随页内行数增长。sqlmock 按序精确匹配：多出一条查询即失败。

func planRows(ps []*model.Plan) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"id", "name", "content", "month_price", "quarter_price", "half_year_price", "year_price",
		"onetime_price", "traffic_gb", "speed_limit", "device_limit", "group_ids", "is_show", "sort",
		"created_at", "updated_at",
	})
	for _, p := range ps {
		rows.AddRow(p.ID, p.Name, p.Content, p.MonthPrice, p.QuarterPrice, p.HalfYearPrice, p.YearPrice,
			p.OnetimePrice, p.TrafficGB, p.SpeedLimit, p.DeviceLimit, p.GroupIDs, p.IsShow, p.Sort,
			p.CreatedAt, p.UpdatedAt)
	}
	return rows
}

func orderRowsFor(os []*model.Order) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"id", "order_no", "user_id", "plan_id", "period", "amount", "discount_amount",
		"balance_used", "pay_amount", "coupon_id", "status", "pay_method", "paid_at",
		"idempotency_key", "created_at", "updated_at",
	})
	for _, o := range os {
		rows.AddRow(o.ID, o.OrderNo, o.UserID, o.PlanID, o.Period, o.Amount, o.DiscountAmount,
			o.BalanceUsed, o.PayAmount, o.CouponID, o.Status, o.PayMethod, o.PaidAt,
			o.IdempotencyKey, o.CreatedAt, o.UpdatedAt)
	}
	return rows
}

// TestAdminListOrdersStatementCountFixed 3 行订单（3 个不同套餐）→ 恰好 5 条语句：
// count + list + emails + commission + plans(IN 一条)。若逐行查套餐则为 8 条。
func TestAdminListOrdersStatementCountFixed(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminSeams(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	now := time.Now()
	pm := "epay_alipay"
	os := []*model.Order{
		{ID: 1, OrderNo: "O1", UserID: 7, PlanID: 1, Period: "month", Amount: 1000, PayAmount: 1000, Status: 1, PayMethod: &pm, CreatedAt: now, UpdatedAt: now},
		{ID: 2, OrderNo: "O2", UserID: 8, PlanID: 2, Period: "year", Amount: 2000, PayAmount: 2000, Status: 1, PayMethod: &pm, CreatedAt: now, UpdatedAt: now},
		{ID: 3, OrderNo: "O3", UserID: 9, PlanID: 3, Period: "month", Amount: 3000, PayAmount: 3000, Status: 1, PayMethod: &pm, CreatedAt: now, UpdatedAt: now},
	}

	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"orders\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(3))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).
		WillReturnRows(orderRowsFor(os))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).
			AddRow(7, "u7@b.com").AddRow(8, "u8@b.com").AddRow(9, "u9@b.com"))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_logs\" WHERE order_no IN")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "invite_user_id", "from_user_id", "order_no", "order_amount", "rate", "amount", "status", "confirmed_at", "created_at",
		}))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\" WHERE id IN ($1,$2,$3)")).
		WillReturnRows(planRows([]*model.Plan{
			{ID: 1, Name: "白羊", TrafficGB: 300, CreatedAt: now, UpdatedAt: now},
			{ID: 2, Name: "金牛", TrafficGB: 300, CreatedAt: now, UpdatedAt: now},
			{ID: 3, Name: "双子", TrafficGB: 300, CreatedAt: now, UpdatedAt: now},
		}))

	list, total, err := svc.Order.ListOrders(context.Background(), nil, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, list, 3)
	assert.Equal(t, "白羊", list[0].PlanName)
	assert.Equal(t, "u8@b.com", list[1].UserEmail)
	// 语句数断言：后续无任何期望，多出一条查询 ExpectationsWereMet 即失败
	require.NoError(t, e.mock.ExpectationsWereMet())
}

// TestListAgentAppliesStatementCountFixed 3 行申请（3 个不同申请人）→ 恰好 4 条语句：
// count + list + emails + 有效邀请数 GROUP BY 一条。若逐人 Count 则为 6 条。
func TestListAgentAppliesStatementCountFixed(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminSeams(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	now := time.Now()

	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"agent_applies\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(3))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"agent_applies\"")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "status", "remark", "reviewed_at", "created_at", "updated_at"}).
			AddRow(1, 7, 0, nil, nil, now, now).
			AddRow(2, 8, 0, nil, nil, now, now).
			AddRow(3, 9, 1, nil, &now, now, now))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).
			AddRow(7, "a7@b.com").AddRow(8, "a8@b.com").AddRow(9, "a9@b.com"))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"required_valid_invites":50}`))
	e.mock.ExpectQuery(`SELECT invite_by_id, COUNT\(\*\) AS c FROM "users" WHERE invite_by_id IN`).
		WillReturnRows(sqlmock.NewRows([]string{"invite_by_id", "c"}).AddRow(7, 60))

	list, total, err := svc.Agent.ListAgentApplies(context.Background(), -1, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, list, 3)
	assert.Equal(t, int64(60), list[0].ValidInvites)
	assert.Equal(t, int64(0), list[1].ValidInvites, "GROUP BY 未覆盖的邀请人计 0")
	require.NoError(t, e.mock.ExpectationsWereMet())
}

// TestAdminOverviewStatementCountFixed 总览恰好 8 条聚合语句（5 count + 3 sum），
// 数据量变化不增减语句。
func TestAdminOverviewStatementCountFixed(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminSeams(e.db, e.rdb, &repo.Repos{}, nil, set, nil)

	count := sqlmock.NewRows([]string{"count"}).AddRow(42)
	sum := sqlmock.NewRows([]string{"coalesce"}).AddRow(1000)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnRows(count)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnRows(count)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "orders"`)).WillReturnRows(count)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "orders"`)).WillReturnRows(count)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "plans"`)).WillReturnRows(count)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(SUM(pay_amount),0) FROM "orders"`)).WillReturnRows(sum)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(SUM(pay_amount),0) FROM "orders"`)).WillReturnRows(sum)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(SUM(balance),0) FROM "users"`)).WillReturnRows(sum)

	out, err := svc.Stats.Overview(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(42), out.UserCount)
	assert.Equal(t, 10.00, out.TotalRevenue)
	require.NoError(t, e.mock.ExpectationsWereMet())
}
