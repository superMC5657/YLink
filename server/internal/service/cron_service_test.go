package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/repo"
)

// TestCronCloseExpiredOrdersClosesPayments:超时关单时同步关闭该订单残留的待支付支付单,
// 避免查单任务反复轮询已取消订单(二期完善)。
func TestCronCloseExpiredOrdersClosesPayments(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	ordersSvc := &OrderService{}
	cron := NewCronService(e.db, e.rdb, repos, &config.Config{}, nil, ordersSvc)
	now := time.Now()
	o := &model.Order{OrderNo: "O1", UserID: 7, Status: model.OrderPending, CreatedAt: now.Add(-time.Hour)}

	// settings 读取 order 配置(失败→默认 30 分钟)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT `value` FROM `settings` WHERE `key` = ?")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"expire_minutes":30}`))
	// ListPendingBefore
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `orders` WHERE status = ? AND created_at < ?")).
		WillReturnRows(orderRow(o))
	// 事务:GetByNoForUpdate → UpdateStatus → ClosePendingByOrderNo → Commit
	e.mock.ExpectBegin()
	e.mock.ExpectQuery("SELECT .* FROM `orders` WHERE order_no = \\? ORDER BY .* FOR UPDATE").
		WillReturnRows(orderRow(o))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `orders`")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 关闭支付单:UPDATE payments SET status=2 WHERE order_no=? AND status=0
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `payments`")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	cron.CloseExpiredOrders(context.Background())
	require.NoError(t, e.mock.ExpectationsWereMet())
}

// TestCronReconcileSkipsOrphanPayment:查单任务发现订单已非待支付时关闭支付单并跳过查单。
func TestCronReconcileSkipsOrphanPayment(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	ordersSvc := &OrderService{}
	cron := NewCronService(e.db, e.rdb, repos, &config.Config{}, nil, ordersSvc)
	now := time.Now()

	// ListPending:一条支付单,订单已取消
	e.mock.ExpectQuery("SELECT .* FROM `payments`").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_no", "user_id", "method", "amount", "trade_no", "status",
			"notify_payload", "paid_at", "created_at", "updated_at",
		}).AddRow(1, "O1", 7, "epay_alipay", 1000, "", model.PayPending, "", nil, now, now))
	// 订单已取消
	o := &model.Order{OrderNo: "O1", UserID: 7, Status: model.OrderCanceled, CreatedAt: now}
	e.mock.ExpectQuery("SELECT .* FROM `orders`").
		WillReturnRows(orderRow(o))
	// 关闭支付单(GORM Updates 隐式事务:Begin → UPDATE → Commit)
	e.mock.ExpectBegin()
	e.mock.ExpectExec("UPDATE .*`payments` SET .*").WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	cron.ReconcilePayments(context.Background())
	require.NoError(t, e.mock.ExpectationsWereMet())
}
