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

// ---- F09 节点批量操作 / 复制 / 排序 ----

func newAdminSvc(t *testing.T) (*testEnv, *AdminService) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	return e, svc
}

func TestBatchServersDeleteWithSummary(t *testing.T) {
	e, svc := newAdminSvc(t)

	// 整批单事务：id=1 删除成功（1 行），id=2 不存在（0 行 → failed）
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "servers" WHERE id = $1`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "servers" WHERE id = $1`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	e.mock.ExpectCommit()
	// 整体审计（默认事务包裹）
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.BatchServers(context.Background(), 1, &model.AdminBatchServerReq{
		Action: "delete", IDs: []int64{1, 2},
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, resp.Success)
	require.Len(t, resp.Failed, 1)
	assert.Equal(t, int64(2), resp.Failed[0].ID)
	assert.Equal(t, "节点不存在", resp.Failed[0].Reason)
}

func TestBatchServersUpdateRequiresFields(t *testing.T) {
	_, svc := newAdminSvc(t)

	// update 未提供任何字段 → 400
	_, err := svc.BatchServers(context.Background(), 1, &model.AdminBatchServerReq{
		Action: "update", IDs: []int64{1},
	}, "127.0.0.1")
	assert.Equal(t, 40000, codeOf(err))

	// rate 非正数 → 400
	rate := 0.0
	_, err = svc.BatchServers(context.Background(), 1, &model.AdminBatchServerReq{
		Action: "update", IDs: []int64{1}, Rate: &rate,
	}, "127.0.0.1")
	assert.Equal(t, 40000, codeOf(err))
}

func TestBatchServersUpdateCommonFields(t *testing.T) {
	e, svc := newAdminSvc(t)

	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "servers" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	isShow := false
	resp, err := svc.BatchServers(context.Background(), 1, &model.AdminBatchServerReq{
		Action: "update", IDs: []int64{1}, IsShow: &isShow,
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, resp.Success)
	assert.Empty(t, resp.Failed)
}

func TestCopyServerGeneratesNewNodeKey(t *testing.T) {
	e, svc := newAdminSvc(t)

	src := &model.Server{ID: 5, GroupID: 1, Name: "香港 01", Type: "shadowsocks",
		Host: "hk1.example.com", Port: 443, Config: "{}", Rate: 1.5,
		Status: 1, IsShow: true, Sort: 2, NodeKey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).
		WillReturnRows(serverRow(src))
	// 默认事务包裹的单条 INSERT
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "servers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(6))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	view, err := svc.CopyServer(context.Background(), 1, 5, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "香港 01-copy", view.Name)
	assert.Equal(t, int64(6), view.ID)
	assert.NotEqual(t, src.NodeKey, view.NodeKey, "复制节点必须生成新 node_key")
	assert.Len(t, view.NodeKey, 32)
	assert.Equal(t, src.Config, view.Config)
	assert.InDelta(t, src.Rate, view.Rate, 1e-9)
}

func TestSortServersSingleTx(t *testing.T) {
	e, svc := newAdminSvc(t)

	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "servers" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "servers" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	err := svc.SortServers(context.Background(), 1, []model.AdminSortItem{
		{ID: 1, Sort: 0}, {ID: 2, Sort: 1},
	}, "127.0.0.1")
	require.NoError(t, err)
}

// ---- F16 流量重置 ----

func TestResetTrafficClearUsageKeepsSnapshot(t *testing.T) {
	e, svc := newAdminSvc(t)

	u := &model.User{ID: 7, Email: "u7@y.link", TransferEnable: 100, U: 30, D: 70, SubToken: "t7", UUID: "uuid-7"}
	// 单用户单事务：行锁读 → 清零 u/d（不动 transfer_enable）→ 写重置记录
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(u))
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "traffic_reset_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()
	// 汇总审计
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.ResetTraffic(context.Background(), 1, &model.AdminTrafficResetReq{
		UserIDs: []int64{7}, Mode: "clear_usage",
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, resp.Success)
	assert.Empty(t, resp.Failed)
	// 关键约束：重置不得删除 node_user_stats 快照（快照差分继续，防止全量重复计费），
	// 故整个用例的期望序列中不存在对 node_user_stats 的 DELETE。
}

func TestResetTrafficResetQuotaWithoutPlanFails(t *testing.T) {
	e, svc := newAdminSvc(t)

	u := &model.User{ID: 8, Email: "u8@y.link", U: 1, D: 2, SubToken: "t8", UUID: "uuid-8"}
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(u))
	e.mock.ExpectRollback()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.ResetTraffic(context.Background(), 1, &model.AdminTrafficResetReq{
		UserIDs: []int64{8}, Mode: "reset_quota",
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 0, resp.Success)
	require.Len(t, resp.Failed, 1)
	assert.Equal(t, int64(8), resp.Failed[0].ID)
	assert.Contains(t, resp.Failed[0].Reason, "无生效套餐")
}

// ---- F04 统计报表 ----

func TestStatOrdersFillsZeroDays(t *testing.T) {
	e, svc := newAdminSvc(t)

	today := time.Now().Format("2006-01-02")
	// 每日创建订单数（仅今天有 2 单）
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_char(created_at::date`)).
		WillReturnRows(sqlmock.NewRows([]string{"d", "c"}).AddRow(today, 2))
	// 每日实收（1 单 1000 分）
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_char(paid_at::date`)).
		WillReturnRows(sqlmock.NewRows([]string{"d", "c", "a"}).AddRow(today, 1, 1000))
	// 每日退款（500 分）
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_char(updated_at::date`)).
		WillReturnRows(sqlmock.NewRows([]string{"d", "c", "a"}).AddRow(today, 0, 500))

	resp, err := svc.StatOrders(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Days)
	require.Len(t, resp.Items, 3, "无数据日补零")
	last := resp.Items[2]
	assert.Equal(t, today, last.Date)
	assert.EqualValues(t, 2, last.OrderCount)
	assert.EqualValues(t, 1, last.CompletedCount)
	assert.InDelta(t, 10.0, last.Revenue, 1e-9, "分→元")
	assert.InDelta(t, 5.0, last.Refunded, 1e-9)
	assert.EqualValues(t, 0, resp.Items[0].OrderCount)
}

func TestStatTrafficTopN(t *testing.T) {
	e, svc := newAdminSvc(t)

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT traffic_logs.user_id, users.email`)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "total"}).
			AddRow(7, "u7@y.link", 1073741824))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT node_user_stats.server_id, servers.name`)).
		WillReturnRows(sqlmock.NewRows([]string{"server_id", "name", "total"}).
			AddRow(5, "香港 01", 2147483648))

	resp, err := svc.StatTraffic(context.Background(), 30)
	require.NoError(t, err)
	require.Len(t, resp.UserTop, 1)
	assert.Equal(t, "u7@y.link", resp.UserTop[0].Email)
	assert.EqualValues(t, 1073741824, resp.UserTop[0].TotalBytes)
	require.Len(t, resp.NodeTop, 1)
	assert.Equal(t, "香港 01", resp.NodeTop[0].Name)
	assert.EqualValues(t, 2147483648, resp.NodeTop[0].Bytes)
}
