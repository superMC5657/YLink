package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/repo"
)

func newTicketEnv(t *testing.T) (*testEnv, *TicketService) {
	e := newTestEnv(t)
	svc := NewTicketService(e.db, e.rdb, &repo.Repos{})
	return e, svc
}

func ticketRow(t *model.Ticket) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "user_id", "subject", "level", "type", "status", "reopen_count", "last_reply_at", "created_at",
	}).AddRow(t.ID, t.UserID, t.Subject, t.Level, t.Type, t.Status, t.ReopenCount, t.LastReplyAt, t.CreatedAt)
}

func TestTicketCreate(t *testing.T) {
	e, svc := newTicketEnv(t)
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"tickets\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"ticket_messages\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.Create(context.Background(), 1, &model.CreateTicketReq{Subject: "无法连接节点", Level: 1, Message: "详细描述"})
	require.NoError(t, err)
	assert.Equal(t, int64(7), resp.ID)
	assert.Equal(t, 0, resp.Status) // 待回复
}

// TestTicketCloseWithdrawRejected 提现工单（type=1）不可由用户关闭（14003）：
// 提交即扣减佣金，生命周期依赖管理员 pay/reject，用户关闭会阻塞管理端审核入口。
func TestTicketCloseWithdrawRejected(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "佣金提现申请", Level: 0, Type: model.TicketTypeWithdraw, Status: 0, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))

	_, err := svc.Close(context.Background(), 1, 7)
	assert.Equal(t, 14003, codeOf(err))
}

// TestTicketReopenWithdrawRejected 提现工单审核完成（管理员关闭）后同样不可重开（14003）。
func TestTicketReopenWithdrawRejected(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "佣金提现申请", Level: 0, Type: model.TicketTypeWithdraw, Status: 2, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))

	_, err := svc.Reopen(context.Background(), 1, 7)
	assert.Equal(t, 14003, codeOf(err))
}

// TestTicketCloseNormalStillAllowed 普通工单（type=0）关闭行为不受影响。
func TestTicketCloseNormalStillAllowed(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "s", Level: 1, Type: 0, Status: 1, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	resp, err := svc.Close(context.Background(), 1, 7)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Status)
}

func TestTicketReplyClosed(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	closed := 2
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "s", Level: 1, Status: closed, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))

	err := svc.Reply(context.Background(), 1, 7, "再问")
	assert.Equal(t, 14001, codeOf(err))
}

func TestTicketReply(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	lastReply := now.Add(-time.Hour)
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "s", Level: 1, Status: 1, LastReplyAt: &lastReply, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"ticket_messages\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	err := svc.Reply(context.Background(), 1, 7, "补充说明")
	require.NoError(t, err)
}

func TestTicketReopen(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "s", Level: 1, Status: 2, ReopenCount: 0, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	resp, err := svc.Reopen(context.Background(), 1, 7)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Status) // 重开后回「待回复」
	assert.Equal(t, 1, resp.ReopenCount)
}

func TestTicketReopenNotClosed(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "s", Level: 1, Status: 0, ReopenCount: 0, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))

	_, err := svc.Reopen(context.Background(), 1, 7)
	assert.Equal(t, 40900, codeOf(err)) // 未关闭不可重开
}

func TestTicketReopenLimit(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "s", Level: 1, Status: 2, ReopenCount: 1, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))

	_, err := svc.Reopen(context.Background(), 1, 7)
	assert.Equal(t, 14002, codeOf(err)) // 仅可重开一次
}

func TestTicketReopenConcurrentUpdateZero(t *testing.T) {
	e, svc := newTicketEnv(t)
	now := time.Now()
	// 读到的快照 reopen_count=0,但条件更新命中 0 行(并发下已被重开)→ 仍按 14002 拒绝
	tk := &model.Ticket{ID: 7, UserID: 1, Subject: "s", Level: 1, Status: 2, ReopenCount: 0, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(ticketRow(tk))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 0))
	e.mock.ExpectCommit()

	_, err := svc.Reopen(context.Background(), 1, 7)
	assert.Equal(t, 14002, codeOf(err))
}

func TestConfirmCommissions(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	orders := NewOrderService(e.db, e.rdb, repos, set, &config.Config{}, nil)
	cronSvc := NewCronService(e.db, e.rdb, repos, &config.Config{}, mailer.New(config.SMTPConfig{}), orders, nil)
	now := time.Now()
	// settings invite（确认期 3 天）
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"commission_confirm_days":3}`))
	// 待确认佣金列表
	cl := &model.CommissionLog{ID: 1, InviteUserID: 9, FromUserID: 2, OrderNo: "O1", OrderAmount: 1000, Rate: 40, Amount: 400, Status: 0, CreatedAt: now}
	clRows := sqlmock.NewRows([]string{
		"id", "invite_user_id", "from_user_id", "order_no", "order_amount", "rate", "amount", "status", "confirmed_at", "created_at",
	}).AddRow(cl.ID, cl.InviteUserID, cl.FromUserID, cl.OrderNo, cl.OrderAmount, cl.Rate, cl.Amount, cl.Status, cl.ConfirmedAt, cl.CreatedAt)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_logs\"")).WillReturnRows(clRows)

	// 事务：条件更新佣金(status=0→1) → 锁邀请人 → 更新余额 → 更新佣金 confirmed_at → 提交
	inviter := &model.User{ID: 9, Email: "inv@b.com", CommissionBalance: 0, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"commission_logs\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(inviter))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"commission_logs\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	cronSvc.ConfirmCommissions(context.Background())
}

func TestCloseExpiredOrders(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	orders := NewOrderService(e.db, e.rdb, repos, set, &config.Config{}, nil)
	cronSvc := NewCronService(e.db, e.rdb, repos, &config.Config{}, mailer.New(config.SMTPConfig{}), orders, nil)

	now := time.Now()
	old := now.Add(-time.Hour)
	// settings order
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"expire_minutes":30}`))
	// 超时待支付订单（列表）
	o := &model.Order{ID: 1, OrderNo: "O1", UserID: 2, PlanID: 1, Period: "month", Amount: 1000, PayAmount: 1000, Status: 0, CreatedAt: old, UpdatedAt: old}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	// 事务：行锁读订单 → 状态仍待支付 → 关单 → 提交
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"orders\" SET \"status\"")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	cronSvc.CloseExpiredOrders(context.Background())
}

func TestCloseExpiredOrdersReleasesCoupon(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	orders := NewOrderService(e.db, e.rdb, repos, set, &config.Config{}, nil)
	cronSvc := NewCronService(e.db, e.rdb, repos, &config.Config{}, mailer.New(config.SMTPConfig{}), orders, nil)

	now := time.Now()
	old := now.Add(-time.Hour)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"expire_minutes":30}`))
	couponID := int64(9)
	o := &model.Order{ID: 1, OrderNo: "O2", UserID: 2, PlanID: 1, Period: "month", Amount: 1000, PayAmount: 800, CouponID: &couponID, Status: 0, CreatedAt: old, UpdatedAt: old}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	// 事务：行锁读 → 关单 → 回退优惠券 used_count → 删使用流水 → 提交
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"orders\" SET \"status\"")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"coupons\" SET \"used_count\"=used_count - 1 WHERE")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("DELETE FROM \"coupon_usages\"")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	cronSvc.CloseExpiredOrders(context.Background())
}

func TestCancelOrderConcurrentPaid(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewOrderService(e.db, e.rdb, &repo.Repos{}, set, &config.Config{}, nil)

	now := time.Now()
	couponID := int64(9)
	o := &model.Order{ID: 1, OrderNo: "O1", UserID: 7, PlanID: 1, Period: "month", Amount: 1000, PayAmount: 800, CouponID: &couponID, Status: 0, CreatedAt: now, UpdatedAt: now}
	// 事务：读订单 → 条件更新影响行数 0（并发已被支付完成）→ 回滚，不释放优惠券
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"orders\" SET \"status\"")).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 行：状态已非待支付
	e.mock.ExpectRollback()

	_, err := svc.CancelOrder(context.Background(), 7, "O1")
	assert.Equal(t, 11003, codeOf(err))
}
