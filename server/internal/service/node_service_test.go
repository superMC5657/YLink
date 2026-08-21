package service

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/repo"
)

func int64Ptr(i int64) *int64 { return &i }

func serverRow(s *model.Server) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "group_id", "name", "type", "host", "port", "config", "rate", "tags", "status", "is_show", "sort", "node_key",
	}).AddRow(s.ID, s.GroupID, s.Name, s.Type, s.Host, s.Port, s.Config, s.Rate, s.Tags, s.Status, s.IsShow, s.Sort, s.NodeKey)
}

func nodeStatRow(st *model.NodeUserStat) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "server_id", "user_id", "last_u", "last_d", "updated_at"}).
		AddRow(st.ID, st.ServerID, st.UserID, st.LastU, st.LastD, st.UpdatedAt)
}

func newNodeEnv(t *testing.T) (*testEnv, *NodeService) {
	e := newTestEnv(t)
	svc := NewNodeService(e.db, e.rdb, &repo.Repos{})
	return e, svc
}

func activeUser(id int64, uid string) *model.User {
	future := time.Now().Add(24 * time.Hour)
	return &model.User{ID: id, Email: "u@b.com", UUID: uid, PlanID: int64Ptr(9), ExpiredAt: &future, TransferEnable: 1 << 40}
}

// expectServerByID 节点查询(rate 可调,验证倍率乘算)。
func (e *testEnv) expectServerByID(s *model.Server) {
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(s))
}

// expectReportUsers 上报用户批量查询。
func (e *testEnv) expectReportUsers(users []model.User) {
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE uuid IN`)).WillReturnRows(userRowsFor(users))
}

// expectActiveUsers 用户同步(套餐内有效订阅)查询。
func (e *testEnv) expectActiveUsers(users []model.User) {
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE plan_id IN`)).WillReturnRows(userRowsFor(users))
}

func userRowsFor(users []model.User) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "role", "balance", "commission_balance", "invite_by_id",
		"is_banned", "remind_expire", "remind_traffic", "telegram_id", "plan_id", "expired_at",
		"transfer_enable", "u", "d", "speed_limit", "device_limit", "sub_token", "uuid", "created_at", "updated_at",
	})
	for i := range users {
		u := users[i]
		rows.AddRow(u.ID, u.Email, u.PasswordHash, u.Role, u.Balance, u.CommissionBalance, u.InviteByID,
			u.IsBanned, u.RemindExpire, u.RemindTraffic, u.TelegramID, u.PlanID, u.ExpiredAt,
			u.TransferEnable, u.U, u.D, u.SpeedLimit, u.DeviceLimit, u.SubToken, u.UUID, u.CreatedAt, u.UpdatedAt)
	}
	return rows
}

func TestNodeUsers(t *testing.T) {
	e, svc := newNodeEnv(t)
	ctx := context.Background()

	srv := &model.Server{ID: 5, GroupID: 1, Name: "HK-01", Type: "trojan", Host: "hk.example.com", Port: 443,
		Config: `{"password":"legacy"}`, Rate: 1.5, NodeKey: "k1"}
	e.expectServerByID(srv)

	// 套餐:plan 9 含分组 1,plan 10 含分组 2(应被过滤)
	p1 := &model.Plan{ID: 9, Name: "白羊", GroupIDs: "[1]", IsShow: true}
	p2 := &model.Plan{ID: 10, Name: "金牛", GroupIDs: "[2]", IsShow: true}
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).WillReturnRows(planRow(p1).AddRow(
		p2.ID, p2.Name, p2.Content, p2.MonthPrice, p2.QuarterPrice, p2.HalfYearPrice, p2.YearPrice,
		p2.OnetimePrice, p2.TrafficGB, p2.SpeedLimit, p2.DeviceLimit, p2.GroupIDs, p2.IsShow, p2.Sort,
		p2.CreatedAt, p2.UpdatedAt))

	u1 := activeUser(1, "uuid-1")
	u2 := activeUser(2, "uuid-2") // onetime:expired_at 为空
	u2.ExpiredAt = nil
	e.expectActiveUsers([]model.User{*u1, *u2})

	resp, err := svc.Users(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, 1.5, resp.Rate)
	require.Len(t, resp.Users, 2)
	assert.Equal(t, "uuid-1", resp.Users[0].UUID)
	assert.Equal(t, int64(1<<40), resp.Users[0].TransferEnable)
	require.NotNil(t, resp.Users[0].ExpiredAt)
	assert.Nil(t, resp.Users[1].ExpiredAt, "onetime 无到期时间为 nil")
}

func TestNodeReportFirstReport(t *testing.T) {
	e, svc := newNodeEnv(t)
	ctx := context.Background()

	e.expectServerByID(&model.Server{ID: 5, GroupID: 1, Name: "HK-01", Type: "trojan", Rate: 1.0, NodeKey: "k1"})
	u := activeUser(1, "uuid-1")
	u.SubToken = "tok-1"
	e.expectReportUsers([]model.User{*u})

	e.mock.ExpectBegin()
	// 首次上报:无快照(gorm.ErrRecordNotFound)→ 增量 = 累计值全量
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "node_user_stats" WHERE`)).WillReturnError(gorm.ErrRecordNotFound)
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "traffic_logs"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "node_user_stats"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.Report(ctx, 5, &NodeReportReq{Data: []NodeReportItem{{UUID: "uuid-1", U: 1000, D: 2000}}})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Accepted)
	assert.Empty(t, resp.Skipped)
}

func TestNodeReportIdempotentRetry(t *testing.T) {
	e, svc := newNodeEnv(t)
	ctx := context.Background()

	e.expectServerByID(&model.Server{ID: 5, GroupID: 1, Name: "HK-01", Type: "trojan", Rate: 1.0, NodeKey: "k1"})
	u := activeUser(1, "uuid-1")
	e.expectReportUsers([]model.User{*u})

	e.mock.ExpectBegin()
	// 快照与上报值相同 → 差分 0,不写 users/traffic_logs,仅刷新快照
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "node_user_stats" WHERE`)).
		WillReturnRows(nodeStatRow(&model.NodeUserStat{ID: 7, ServerID: 5, UserID: 1, LastU: 1000, LastD: 2000}))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "node_user_stats"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.Report(ctx, 5, &NodeReportReq{Data: []NodeReportItem{{UUID: "uuid-1", U: 1000, D: 2000}}})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Accepted)
}

func TestNodeReportCounterReset(t *testing.T) {
	e, svc := newNodeEnv(t)
	ctx := context.Background()

	e.expectServerByID(&model.Server{ID: 5, GroupID: 1, Name: "HK-01", Type: "trojan", Rate: 1.0, NodeKey: "k1"})
	u := activeUser(1, "uuid-1")
	e.expectReportUsers([]model.User{*u})

	e.mock.ExpectBegin()
	// 计数器回退(u: 500 → 100 视为节点重启,增量取当前值 100;d 正常差分 200-50=150)
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "node_user_stats" WHERE`)).
		WillReturnRows(nodeStatRow(&model.NodeUserStat{ID: 7, ServerID: 5, UserID: 1, LastU: 500, LastD: 50}))
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "traffic_logs"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "node_user_stats"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.Report(ctx, 5, &NodeReportReq{Data: []NodeReportItem{{UUID: "uuid-1", U: 100, D: 200}}})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Accepted)
}

func TestNodeReportRateScale(t *testing.T) {
	e, svc := newNodeEnv(t)
	ctx := context.Background()

	// 倍率 0.5:增量 100/200 → 计费 50/100
	e.expectServerByID(&model.Server{ID: 5, GroupID: 1, Name: "HK-01", Type: "trojan", Rate: 0.5, NodeKey: "k1"})
	u := activeUser(1, "uuid-1")
	u.SubToken = "tok-1"
	e.expectReportUsers([]model.User{*u})

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "node_user_stats" WHERE`)).WillReturnError(gorm.ErrRecordNotFound)
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "traffic_logs"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "node_user_stats"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.Report(ctx, 5, &NodeReportReq{Data: []NodeReportItem{{UUID: "uuid-1", U: 100, D: 200}}})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Accepted)
}

func TestNodeReportSkips(t *testing.T) {
	e, svc := newNodeEnv(t)
	ctx := context.Background()

	e.expectServerByID(&model.Server{ID: 5, GroupID: 1, Name: "HK-01", Type: "trojan", Rate: 1.0, NodeKey: "k1"})
	// uuid-2 无订阅(plan_id nil);uuid-3 不存在(unknown_user)
	u := activeUser(1, "uuid-1")
	u.PlanID = nil
	e.expectReportUsers([]model.User{*u})

	e.mock.ExpectBegin()
	e.mock.ExpectCommit()

	resp, err := svc.Report(ctx, 5, &NodeReportReq{Data: []NodeReportItem{
		{UUID: "uuid-1", U: 10, D: 20},
		{UUID: "uuid-3", U: 10, D: 20},
	}})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Accepted)
	require.Len(t, resp.Skipped, 2)
	reasons := map[string]string{resp.Skipped[0].UUID: resp.Skipped[0].Reason, resp.Skipped[1].UUID: resp.Skipped[1].Reason}
	assert.Equal(t, "not_subscribed", reasons["uuid-1"])
	assert.Equal(t, "unknown_user", reasons["uuid-3"])
}

func TestNodeReportServerError(t *testing.T) {
	e, svc := newNodeEnv(t)
	ctx := context.Background()

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnError(errors.New("db down"))
	_, err := svc.Report(ctx, 99, &NodeReportReq{Data: []NodeReportItem{{UUID: "x", U: 1, D: 1}}})
	assert.Error(t, err)
}

// cumDelta/scaleRate 单元:差分、回退、倍率取整边界。
func TestCumDeltaAndScale(t *testing.T) {
	assert.Equal(t, int64(300), cumDelta(500, 200))
	assert.Equal(t, int64(0), cumDelta(200, 200))
	assert.Equal(t, int64(100), cumDelta(100, 500), "回退视为重启,增量取当前值")
	assert.Equal(t, int64(50), scaleRate(100, 0.5))
	assert.Equal(t, int64(150), scaleRate(100, 1.5))
	assert.Equal(t, int64(101), scaleRate(101, 1.0))
	assert.Equal(t, int64(0), scaleRate(0, 1.5))
}
