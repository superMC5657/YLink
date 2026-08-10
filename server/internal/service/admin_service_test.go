package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink/internal/model"
	"ylink/internal/pkg/sanitize"
	"ylink/internal/repo"
)

func TestSanitize(t *testing.T) {
	// 富文本：脚本/事件被剥离，基础标签保留
	html := `<p onclick="alert(1)">hello <b>bold</b><script>alert(2)</script><img src=x onerror=alert(3)></p>`
	out := sanitize.Markdown(html)
	assert.NotContains(t, out, "script")
	assert.NotContains(t, out, "onclick")
	assert.NotContains(t, out, "onerror")
	assert.Contains(t, out, "<b>bold</b>")

	// 纯文本：全部标签剥离
	assert.Equal(t, "hello", sanitize.Text("<b>hello</b>"))
}

func TestCreateOrderCouponLimitExceeded(t *testing.T) {
	e, svc := newOrderEnv(t)
	ctx := context.Background()
	now := time.Now()
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}

	// 幂等键未命中
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `orders`")).WillReturnError(assert.AnError)
	// 套餐查询
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `plans`")).WillReturnRows(planRow(p))
	// 事务开始
	e.mock.ExpectBegin()
	// 优惠券查询（used_count 已达 total_limit）
	cp := &model.Coupon{ID: 9, Code: "LIMIT", Type: 1, Value: 200, MinSpend: 0, TotalLimit: 10, UsedCount: 10, IsEnable: true}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `coupons`")).WillReturnRows(couponRows)
	// validateCoupon 检查 used_count >= total_limit → 直接 ErrCoupon（未到 Occupy）
	e.mock.ExpectRollback()

	_, err := svc.CreateOrder(ctx, 7, "idem-9", &model.CreateOrderReq{PlanID: 1, Period: "month", CouponCode: "LIMIT"})
	assert.Equal(t, 12001, codeOf(err))
}

func TestCreateOrderCouponOccupied(t *testing.T) {
	e, svc := newOrderEnv(t)
	ctx := context.Background()
	now := time.Now()
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `orders`")).WillReturnError(assert.AnError)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `plans`")).WillReturnRows(planRow(p))
	e.mock.ExpectBegin()
	// 优惠券查询
	cp := &model.Coupon{ID: 9, Code: "OK", Type: 1, Value: 200, MinSpend: 0, TotalLimit: 10, UsedCount: 3, IsEnable: true}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `coupons`")).WillReturnRows(couponRows)
	// 原子占用：条件更新成功（1 行）
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `coupons` SET `used_count`=used_count + 1 WHERE")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 写使用流水
	e.mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `coupon_usages`")).WillReturnResult(sqlmock.NewResult(1, 1))
	// 创建订单
	e.mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `orders`")).WillReturnResult(sqlmock.NewResult(1, 1))
	e.mock.ExpectCommit()
	// toOrderResp 查套餐
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `plans`")).WillReturnRows(planRow(p))

	resp, err := svc.CreateOrder(ctx, 7, "idem-ok", &model.CreateOrderReq{PlanID: 1, Period: "month", CouponCode: "OK"})
	require.NoError(t, err)
	assert.Equal(t, 8.00, resp.PayAmount)
	assert.Equal(t, 2.00, resp.DiscountAmount)
}

func TestAdminRefundWithCommissionRollback(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

	now := time.Now()
	payMethod := "epay_alipay"
	paidAt := now.Add(-time.Hour)
	o := &model.Order{ID: 1, OrderNo: "O1", UserID: 2, PlanID: 1, Period: "month",
		Amount: 1000, PayAmount: 1000, Status: 1, PayMethod: &payMethod, PaidAt: &paidAt, CreatedAt: now, UpdatedAt: now}
	// 行锁读订单
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `orders`")).WillReturnRows(orderRow(o))
	// 订单状态 → 已退款（先 Save）
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `orders`")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 佣金查询：已发放
	cl := &model.CommissionLog{ID: 1, InviteUserID: 9, FromUserID: 2, OrderNo: "O1", OrderAmount: 1000, Rate: 40, Amount: 400, Status: 1, ConfirmedAt: &now, CreatedAt: now}
	clRows := sqlmock.NewRows([]string{
		"id", "invite_user_id", "from_user_id", "order_no", "order_amount", "rate", "amount", "status", "confirmed_at", "created_at",
	}).AddRow(cl.ID, cl.InviteUserID, cl.FromUserID, cl.OrderNo, cl.OrderAmount, cl.Rate, cl.Amount, cl.Status, cl.ConfirmedAt, cl.CreatedAt)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `commission_logs`")).WillReturnRows(clRows)
	// 锁邀请人 → 扣回佣金
	inviter := &model.User{ID: 9, Email: "inv@b.com", CommissionBalance: 400, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users`")).WillReturnRows(userRow(inviter))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `users`")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 佣金状态 → 已撤销
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `commission_logs`")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 审计日志
	e.mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `audit_logs`")).WillReturnResult(sqlmock.NewResult(1, 1))
	e.mock.ExpectCommit()

	err := svc.Refund(context.Background(), 1, "O1", "测试退款", "127.0.0.1")
	require.NoError(t, err)
}

func TestAdminReviewAgentApprove(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

	now := time.Now()
	// 事务：读申请 → 改角色 → 更新申请 → 审计
	e.mock.ExpectBegin()
	apply := &model.AgentApply{ID: 3, UserID: 5, Status: 0, CreatedAt: now}
	applyRows := sqlmock.NewRows([]string{"id", "user_id", "status", "remark", "reviewed_at", "created_at", "updated_at"}).
		AddRow(apply.ID, apply.UserID, apply.Status, nil, nil, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agent_applies`")).WillReturnRows(applyRows)
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `users`")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE `agent_applies`")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `audit_logs`")).WillReturnResult(sqlmock.NewResult(1, 1))
	e.mock.ExpectCommit()

	err := svc.ReviewAgentApply(context.Background(), 1, 3, true, "", "127.0.0.1")
	require.NoError(t, err)
}
