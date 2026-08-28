package service

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/mailer"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
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
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnError(assert.AnError)
	// 套餐查询
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	// 事务开始
	e.mock.ExpectBegin()
	// 优惠券查询（used_count 已达 total_limit）
	cp := &model.Coupon{ID: 9, Code: "LIMIT", Type: 1, Value: 200, MinSpend: 0, TotalLimit: 10, UsedCount: 10, IsEnable: true}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)
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

	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnError(assert.AnError)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	e.mock.ExpectBegin()
	// 优惠券查询
	cp := &model.Coupon{ID: 9, Code: "OK", Type: 1, Value: 200, MinSpend: 0, TotalLimit: 10, UsedCount: 3, IsEnable: true}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)
	// 原子占用：条件更新成功（1 行）
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"coupons\" SET \"used_count\"=used_count + 1 WHERE")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 写使用流水
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"coupon_usages\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	// 创建订单
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"orders\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()
	// toOrderResp 查套餐
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))

	resp, err := svc.CreateOrder(ctx, 7, "idem-ok", &model.CreateOrderReq{PlanID: 1, Period: "month", CouponCode: "OK"})
	require.NoError(t, err)
	assert.Equal(t, 8.00, resp.PayAmount)
	assert.Equal(t, 2.00, resp.DiscountAmount)
}

func TestAdminRefundWithCommissionRollback(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	payMethod := "epay_alipay"
	paidAt := now.Add(-time.Hour)
	planID1 := int64(1)
	future := now.Add(30 * 24 * time.Hour)
	p1 := &model.Plan{ID: 1, Name: "白羊座", TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}
	o := &model.Order{ID: 1, OrderNo: "O1", UserID: 2, PlanID: 1, Period: "month",
		Amount: 1000, PayAmount: 1000, Status: 1, PayMethod: &payMethod, PaidAt: &paidAt, CreatedAt: now, UpdatedAt: now}
	// 行锁读订单
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	// 收回订阅：锁购买人（有订阅：plan_id=1 未过期）→ 查套餐 → 清除订阅
	buyer := &model.User{ID: 2, Email: "buyer@b.com", PlanID: &planID1, ExpiredAt: &future,
		TransferEnable: 300 * 1024 * 1024 * 1024, U: 1, D: 2, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(buyer))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 订单状态 → 已退款（先 Save）
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"orders\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 佣金查询：已发放
	cl := &model.CommissionLog{ID: 1, InviteUserID: 9, FromUserID: 2, OrderNo: "O1", OrderAmount: 1000, Rate: 40, Amount: 400, Status: 1, ConfirmedAt: &now, CreatedAt: now}
	clRows := sqlmock.NewRows([]string{
		"id", "invite_user_id", "from_user_id", "order_no", "order_amount", "rate", "amount", "status", "confirmed_at", "created_at",
	}).AddRow(cl.ID, cl.InviteUserID, cl.FromUserID, cl.OrderNo, cl.OrderAmount, cl.Rate, cl.Amount, cl.Status, cl.ConfirmedAt, cl.CreatedAt)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_logs\"")).WillReturnRows(clRows)
	// 锁邀请人 → 扣回佣金
	inviter := &model.User{ID: 9, Email: "inv@b.com", CommissionBalance: 400, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(inviter))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 佣金状态 → 已撤销
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"commission_logs\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 审计日志
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	err := svc.Refund(context.Background(), 1, "O1", "测试退款", "127.0.0.1")
	require.NoError(t, err)
}

func TestAdminReviewAgentApprove(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	// 事务：读申请 → 改角色 → 更新申请 → 审计
	e.mock.ExpectBegin()
	apply := &model.AgentApply{ID: 3, UserID: 5, Status: 0, CreatedAt: now}
	applyRows := sqlmock.NewRows([]string{"id", "user_id", "status", "remark", "reviewed_at", "created_at", "updated_at"}).
		AddRow(apply.ID, apply.UserID, apply.Status, nil, nil, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"agent_applies\"")).WillReturnRows(applyRows)
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"agent_applies\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	err := svc.ReviewAgentApply(context.Background(), 1, 3, true, "", "127.0.0.1")
	require.NoError(t, err)
}

func TestAdminListCoupons(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	validPeriods := `["month","year"]`
	planIDs := `[1,2]`
	cp := &model.Coupon{
		ID: 1, Code: "WELCOME10", Type: 2, Value: 1000, MinSpend: 0,
		LimitPerUser: 1, TotalLimit: 100, UsedCount: 3,
		ValidPeriods: &validPeriods, PlanIDs: &planIDs,
		IsEnable: true, CreatedAt: now, UpdatedAt: now,
	}
	rows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, nil, nil, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(rows)

	out, err := svc.ListAllCoupons(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "WELCOME10", out[0].Code)
	assert.Equal(t, 2, out[0].Type)
	assert.Equal(t, 10.00, out[0].Value) // 1000 分 → 10 元
	assert.Equal(t, 0.00, out[0].MinSpend)
	assert.Equal(t, 3, out[0].UsedCount)
	assert.Equal(t, []string{"month", "year"}, out[0].ValidPeriods)
	assert.Equal(t, []int64{1, 2}, out[0].PlanIDs)
	assert.True(t, out[0].IsEnable)
}

func TestAdminListNotices(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	n := &model.Notice{ID: 1, Title: "维护公告", Content: "正文", IsShow: false, Sort: 2, CreatedAt: now}
	rows := sqlmock.NewRows([]string{"id", "title", "content", "is_show", "sort", "created_at", "updated_at"}).
		AddRow(n.ID, n.Title, n.Content, n.IsShow, n.Sort, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"notices\"")).WillReturnRows(rows)

	out, err := svc.ListAllNotices(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "维护公告", out[0].Title)
	assert.False(t, out[0].IsShow) // 含隐藏项
	assert.Equal(t, 2, out[0].Sort)
}

func TestAdminListKnowledges(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	k := &model.Knowledge{
		ID: 1, Category: "入门指南", Title: "如何导入", Body: "正文",
		Language: "zh-CN", IsShow: true, Sort: 1, UpdatedAt: now,
	}
	rows := sqlmock.NewRows([]string{
		"id", "category", "title", "body", "language", "is_show", "sort", "created_at", "updated_at",
	}).AddRow(k.ID, k.Category, k.Title, k.Body, k.Language, k.IsShow, k.Sort, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"knowledges\"")).WillReturnRows(rows)

	out, err := svc.ListAllKnowledges(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "入门指南", out[0].Category)
	assert.Equal(t, "zh-CN", out[0].Language)
	assert.True(t, out[0].IsShow)
}

func TestAdminAdjustBalanceNegativeRejected(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	// 事务：行锁读用户（余额 0）→ 调 -100 → 服务层拒绝，回滚
	e.mock.ExpectBegin()
	u := &model.User{ID: 5, Email: "u@b.com", Balance: 0, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	e.mock.ExpectRollback()

	err := svc.AdjustBalance(context.Background(), 1, 5, -100, "扣款", "127.0.0.1")
	require.Error(t, err)
	assert.Equal(t, 40000, codeOf(err))
}

func TestAdminUpdateUserBanBumpsSessionVersion(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	u := &model.User{ID: 5, Email: "u@b.com", Role: 0, IsBanned: false, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	// GORM 默认单写操作包事务：SetBanned 与审计各一个 BEGIN/COMMIT
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	banned := true
	err := svc.UpdateUser(context.Background(), 1, 5, &model.AdminUpdateUserReq{Banned: &banned}, "127.0.0.1")
	require.NoError(t, err)

	// 会话版本号已 bump → 该用户旧 access token 立即失效
	n, err := e.rdb.Get(context.Background(), redispkg.SessionVersionKey(5)).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestAdminUpdateUserRoleBumpsSessionVersion(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	u := &model.User{ID: 6, Email: "r@b.com", Role: 0, IsBanned: false, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	// GORM 默认单写操作包事务：UpdateRole 与审计各一个 BEGIN/COMMIT
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	role := 2
	err := svc.UpdateUser(context.Background(), 1, 6, &model.AdminUpdateUserReq{Role: &role}, "127.0.0.1")
	require.NoError(t, err)

	n, err := e.rdb.Get(context.Background(), redispkg.SessionVersionKey(6)).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestAdminRefundRevokesOnetimeTraffic(t *testing.T) {
	// 退款一次性流量包：仅扣回本次流量，不清除订阅结构
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	payMethod := "epay_alipay"
	paidAt := now.Add(-time.Hour)
	p1 := &model.Plan{ID: 1, Name: "白羊座", TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}
	o := &model.Order{ID: 2, OrderNo: "O2", UserID: 3, PlanID: 1, Period: "onetime",
		Amount: 1000, PayAmount: 1000, Status: 1, PayMethod: &payMethod, PaidAt: &paidAt, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	// onetime：锁用户（当前 transfer_enable=200G）→ 查套餐(300G) → 扣回 300G → 下限 0
	buyer := &model.User{ID: 3, Email: "o@b.com", TransferEnable: 200 * 1024 * 1024 * 1024, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(buyer))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"orders\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 无佣金
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_logs\"")).WillReturnError(assert.AnError)
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	err := svc.Refund(context.Background(), 1, "O2", "退流量", "127.0.0.1")
	require.NoError(t, err)
}

func TestAdminRefundKeepsOtherPlanSubscription(t *testing.T) {
	// 退款订单的套餐 ≠ 用户当前订阅套餐：不清除订阅（用户当前用的是别的套餐）
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)

	now := time.Now()
	payMethod := "epay_alipay"
	paidAt := now.Add(-time.Hour)
	future := now.Add(30 * 24 * time.Hour)
	p1 := &model.Plan{ID: 1, Name: "白羊座", TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}
	o := &model.Order{ID: 3, OrderNo: "O3", UserID: 4, PlanID: 1, Period: "month",
		Amount: 1000, PayAmount: 1000, Status: 1, PayMethod: &payMethod, PaidAt: &paidAt, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	// 锁用户：当前订阅是套餐 2（≠ 订单套餐 1）→ 不清除 → 无 UPDATE users
	otherPlan := int64(2)
	buyer := &model.User{ID: 4, Email: "p@b.com", PlanID: &otherPlan, ExpiredAt: &future, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(buyer))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"orders\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// 无佣金
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_logs\"")).WillReturnError(assert.AnError)
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	err := svc.Refund(context.Background(), 1, "O3", "退其他套餐单", "127.0.0.1")
	require.NoError(t, err)
}

// TestAdminListOrdersCommission:管理端订单列表返回佣金金额(批量查 commission_logs 映射)。
func TestAdminListOrdersCommission(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, &config.Config{}, set, nil)
	now := time.Now()
	pm := "epay_alipay"
	o := &model.Order{ID: 1, OrderNo: "O2026", UserID: 7, PlanID: 1, Period: "month",
		Amount: 1000, PayAmount: 1000, Status: 1, PayMethod: &pm, PaidAt: &now, CreatedAt: now, UpdatedAt: now}

	// 分页计数
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"orders\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))
	// 分页列表
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_no", "user_id", "plan_id", "period", "amount", "discount_amount",
			"balance_used", "pay_amount", "coupon_id", "status", "pay_method", "paid_at",
			"idempotency_key", "created_at", "updated_at",
		}).AddRow(o.ID, o.OrderNo, o.UserID, o.PlanID, o.Period, o.Amount, o.DiscountAmount,
			o.BalanceUsed, o.PayAmount, o.CouponID, o.Status, o.PayMethod, o.PaidAt,
			o.IdempotencyKey, o.CreatedAt, o.UpdatedAt))
	// emailsOf:按 user_id 查邮箱
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(7, "u7@b.com"))
	// ListByOrderNos:批量查佣金
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_logs\" WHERE order_no IN")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "invite_user_id", "from_user_id", "order_no", "order_amount", "rate", "amount", "status", "confirmed_at", "created_at",
		}).AddRow(1, 9, 7, "O2026", 1000, 40, 400, 0, nil, now))
	// plan 名(GetByID 按主键查)
	p := &model.Plan{ID: 1, Name: "白羊座", TrafficGB: 300, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\" WHERE \"plans\".\"id\" = $1 ORDER BY \"plans\".\"id\" LIMIT $2")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "content", "month_price", "quarter_price", "half_year_price", "year_price",
			"onetime_price", "traffic_gb", "speed_limit", "device_limit", "group_ids", "is_show", "sort",
			"created_at", "updated_at",
		}).AddRow(p.ID, p.Name, p.Content, p.MonthPrice, p.QuarterPrice, p.HalfYearPrice, p.YearPrice,
			p.OnetimePrice, p.TrafficGB, p.SpeedLimit, p.DeviceLimit, p.GroupIDs, p.IsShow, p.Sort,
			p.CreatedAt, p.UpdatedAt))

	status := 1
	list, total, err := svc.ListOrders(context.Background(), &status, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].CommissionAmount)
	assert.Equal(t, 4.00, *list[0].CommissionAmount) // 400 分 → 4 元
}

// TestAdminListOrdersCommissionQueryError:佣金批量查询失败时 ListOrders 必须上抛错误,
// 不得静默返回 commission_amount 全 null 的成功响应(review-0.5.0 P2 回归)。
func TestAdminListOrdersCommissionQueryError(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, &config.Config{}, set, nil)
	now := time.Now()
	pm := "epay_alipay"
	o := &model.Order{ID: 1, OrderNo: "O2026", UserID: 7, PlanID: 1, Period: "month",
		Amount: 1000, PayAmount: 1000, Status: 1, PayMethod: &pm, PaidAt: &now, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"orders\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_no", "user_id", "plan_id", "period", "amount", "discount_amount",
			"balance_used", "pay_amount", "coupon_id", "status", "pay_method", "paid_at",
			"idempotency_key", "created_at", "updated_at",
		}).AddRow(o.ID, o.OrderNo, o.UserID, o.PlanID, o.Period, o.Amount, o.DiscountAmount,
			o.BalanceUsed, o.PayAmount, o.CouponID, o.Status, o.PayMethod, o.PaidAt,
			o.IdempotencyKey, o.CreatedAt, o.UpdatedAt))
	// emailsOf
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(7, "u7@b.com"))
	// 佣金查询失败
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_logs\" WHERE order_no IN")).
		WillReturnError(assert.AnError)

	status := 1
	_, _, err := svc.ListOrders(context.Background(), &status, 1, 10)
	require.Error(t, err, "佣金查询失败时 ListOrders 必须返回错误,不得静默成功")
}

func TestResetNodeKey(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)

	// 节点存在 → 更新密钥(默认事务包裹) + 审计
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(
		&model.Server{ID: 5, GroupID: 1, Name: "HK-01", Type: "trojan", Host: "hk.example.com", Port: 443,
			Config: "{}", Rate: 1.0, NodeKey: "old-key-0000000000000000"}))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "servers" SET`)).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	key, err := svc.ResetNodeKey(context.Background(), 1, 5, "127.0.0.1")
	require.NoError(t, err)
	assert.Len(t, key, 32, "新密钥为 32 位十六进制")
	assert.NotEqual(t, "old-key-0000000000000000", key)

	// 节点不存在 → 40400
	e2 := newTestEnv(t)
	svc2 := NewAdminService(e2.db, e2.rdb, &repo.Repos{}, nil, set, nil)
	e2.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnError(gorm.ErrRecordNotFound)
	_, err = svc2.ResetNodeKey(context.Background(), 1, 99, "127.0.0.1")
	assert.Equal(t, 40400, codeOf(err))
}

func auditLogRows(items []model.AdminAuditLogItem) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"id", "admin_id", "admin_email", "action", "target", "detail", "ip", "created_at",
	})
	for _, it := range items {
		rows.AddRow(it.ID, it.AdminID, it.AdminEmail, it.Action, it.Target, it.Detail, it.IP, it.CreatedAt)
	}
	return rows
}

func TestListAuditLogs(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	ctx := context.Background()

	// 无筛选：count + 分页 + 目标可读化（users 反查）+ 动作列表
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "audit_logs" JOIN users ON users.id = audit_logs.admin_id`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(auditLogRows([]model.AdminAuditLogItem{
		{ID: 2, AdminID: 1, AdminEmail: "admin@y.link", Action: "adjust_balance", Target: strPtr("7"), CreatedAt: time.Now()},
		{ID: 1, AdminID: 1, AdminEmail: "admin@y.link", Action: "ban_user", Target: strPtr("8"), CreatedAt: time.Now()},
	}))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email FROM "users" WHERE id IN`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(7, "u7@y.link").AddRow(8, "u8@y.link"))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT "action" FROM "audit_logs"`)).WillReturnRows(
		sqlmock.NewRows([]string{"action"}).AddRow("ban_user").AddRow("adjust_balance"))

	list, total, actions, err := svc.ListAuditLogs(ctx, AuditLogFilter{}, 1, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, list, 2)
	assert.Equal(t, "admin@y.link", list[0].AdminEmail, "联表取操作人邮箱")
	assert.Equal(t, []string{"ban_user", "adjust_balance"}, actions)
	// 目标可读化：user 动作 → 反查邮箱
	if assert.NotNil(t, list[0].TargetKind) && assert.NotNil(t, list[0].TargetDisplay) {
		assert.Equal(t, "user", *list[0].TargetKind)
		assert.Equal(t, "u7@y.link", *list[0].TargetDisplay)
	}

	// 非法日期 → 40000
	_, _, _, err = svc.ListAuditLogs(ctx, AuditLogFilter{From: "2026/08/28"}, 1, 20)
	assert.Equal(t, 40000, codeOf(err))
	_, _, _, err = svc.ListAuditLogs(ctx, AuditLogFilter{To: "bad-date"}, 1, 20)
	assert.Equal(t, 40000, codeOf(err))
}

func strPtr(s string) *string { return &s }

// TestAuditTargetDisplay 审计日志目标可读化：users 列表 / 节点 / 知识分类 / 订单 / 空 target / 未知动作。
func TestAuditTargetDisplay(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	ctx := context.Background()

	// count + 分页（7 条：含用户已删除但 detail 留痕邮箱）+ users/servers/knowledge_categories 批量反查 + 动作列表
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "audit_logs" JOIN users ON users.id = audit_logs.admin_id`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(auditLogRows([]model.AdminAuditLogItem{
		{ID: 7, AdminID: 1, AdminEmail: "admin@y.link", Action: "adjust_balance", Target: strPtr("99"),
			Detail: strPtr(`{"email":"deleted@y.link","amount":100}`), CreatedAt: time.Now()}, // 用户 99 已删除
		{ID: 6, AdminID: 1, AdminEmail: "admin@y.link", Action: "send_mail", Target: strPtr("[7 8 9]"), CreatedAt: time.Now()},
		{ID: 5, AdminID: 1, AdminEmail: "admin@y.link", Action: "copy_server", Target: strPtr("5"), CreatedAt: time.Now()},
		{ID: 4, AdminID: 1, AdminEmail: "admin@y.link", Action: "create_knowledge_category", Target: strPtr("3"), CreatedAt: time.Now()},
		{ID: 3, AdminID: 1, AdminEmail: "admin@y.link", Action: "refund", Target: strPtr("ORD20260828001"), CreatedAt: time.Now()},
		{ID: 2, AdminID: 1, AdminEmail: "admin@y.link", Action: "batch_server_delete", Target: strPtr(""), CreatedAt: time.Now()},
		{ID: 1, AdminID: 1, AdminEmail: "admin@y.link", Action: "unknown_action", Target: strPtr("42"), CreatedAt: time.Now()},
	}))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email FROM "users" WHERE id IN`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(7, "u7@y.link").AddRow(8, "u8@y.link"))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name FROM "servers" WHERE id IN`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(5, "HK-01"))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name FROM "knowledge_categories" WHERE id IN`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(3, "使用教程"))
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT "action" FROM "audit_logs"`)).WillReturnRows(
		sqlmock.NewRows([]string{"action"}).AddRow("send_mail"))

	list, _, _, err := svc.ListAuditLogs(ctx, AuditLogFilter{}, 1, 20)
	require.NoError(t, err)
	require.Len(t, list, 7)

	// 用户已删除：users 表查不到 → detail 里留痕的 email 兜底
	if assert.NotNil(t, list[0].TargetKind) && assert.NotNil(t, list[0].TargetDisplay) {
		assert.Equal(t, "user", *list[0].TargetKind)
		assert.Equal(t, "deleted@y.link", *list[0].TargetDisplay)
	}
	// users 列表 "[7 8 9]"：9 已不存在，只展示查到的邮箱
	if assert.NotNil(t, list[1].TargetKind) && assert.NotNil(t, list[1].TargetDisplay) {
		assert.Equal(t, "users", *list[1].TargetKind)
		assert.Equal(t, "u7@y.link, u8@y.link", *list[1].TargetDisplay)
	}
	// 节点 → 节点名
	if assert.NotNil(t, list[2].TargetKind) && assert.NotNil(t, list[2].TargetDisplay) {
		assert.Equal(t, "server", *list[2].TargetKind)
		assert.Equal(t, "HK-01", *list[2].TargetDisplay)
	}
	// 知识分类 → 分类名
	if assert.NotNil(t, list[3].TargetKind) && assert.NotNil(t, list[3].TargetDisplay) {
		assert.Equal(t, "knowledge_category", *list[3].TargetKind)
		assert.Equal(t, "使用教程", *list[3].TargetDisplay)
	}
	// 订单号本身可读：kind 标注 + 原样透出
	if assert.NotNil(t, list[4].TargetKind) && assert.NotNil(t, list[4].TargetDisplay) {
		assert.Equal(t, "order", *list[4].TargetKind)
		assert.Equal(t, "ORD20260828001", *list[4].TargetDisplay)
	}
	// 空 target（批量/排序类动作）与未收录动作：不做增强，前端回退
	assert.Nil(t, list[5].TargetKind)
	assert.Nil(t, list[5].TargetDisplay)
	assert.Nil(t, list[6].TargetKind)
	assert.Nil(t, list[6].TargetDisplay)
}

// ---- F05 用户管理增强 ----

func TestBatchUsersBanAndMissing(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	ctx := context.Background()

	u7 := &model.User{ID: 7, Email: "u7@y.link", SubToken: "t7", UUID: "uuid-7"}
	u8 := &model.User{ID: 8, Email: "u8@y.link", SubToken: "t8", UUID: "uuid-8"}
	_ = u8

	// 用户 7：查询 + 封禁更新（gorm 单写包事务）+ 审计（同样包事务）
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(u7))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()
	// 用户 8：不存在
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnError(gorm.ErrRecordNotFound)

	resp, err := svc.BatchUsers(ctx, 1, &model.AdminBatchUserReq{Action: "ban", IDs: []int64{7, 8}}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, resp.Success, "用户 7 封禁成功")
	require.Len(t, resp.Failed, 1)
	assert.EqualValues(t, 8, resp.Failed[0].ID)
	assert.Equal(t, "资源不存在", resp.Failed[0].Reason)
}

func TestBatchUsersAdjustBalanceNegativeGuard(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	ctx := context.Background()

	u := &model.User{ID: 7, Email: "u7@y.link", Balance: 100, SubToken: "t7", UUID: "uuid-7"}
	// WithTx：Begin → FOR UPDATE 查询 → 校验失败 Rollback
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(u))
	e.mock.ExpectRollback()

	amount := -99.0 // 调整后 100 分 - 9900 分 < 0
	resp, err := svc.BatchUsers(ctx, 1, &model.AdminBatchUserReq{
		Action: "adjust_balance", IDs: []int64{7}, Amount: &amount,
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 0, resp.Success)
	require.Len(t, resp.Failed, 1)
	assert.Equal(t, "调整后余额不能为负", resp.Failed[0].Reason)
}

func TestBatchUsersRequiresAmount(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	_, err := svc.BatchUsers(context.Background(), 1, &model.AdminBatchUserReq{
		Action: "adjust_balance", IDs: []int64{7},
	}, "127.0.0.1")
	assert.Equal(t, 40000, codeOf(err))
}

func TestSendMailLogsFailures(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	ml := mailer.New(config.SMTPConfig{Host: "invalid.invalid", Port: 1}) // 不可达 SMTP
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, ml)
	ctx := context.Background()

	users := []model.User{
		{ID: 7, Email: "u7@y.link", SubToken: "t7", UUID: "uuid-7"},
	}
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id IN`)).
		WillReturnRows(userRows(users))
	// SMTP 不可达 → 发送失败 → mail_logs 记录失败（gorm 单写包事务）+ 审计（包事务）
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "mail_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.SendMail(ctx, 1, &model.AdminSendMailReq{
		IDs: []int64{7}, Subject: "公告", Body: "<b>hello</b><script>alert(1)</script>",
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 0, resp.Sent, "SMTP 不可达时发送失败")
	require.Len(t, resp.Failed, 1)
}

// ---- 审计写入参数断言辅助 ----

// argEq 断言 SQL 参数为指定字符串（audit_logs.target / ip 等）。
type argEq string

func (w argEq) Match(v driver.Value) bool {
	s, ok := v.(string)
	return ok && s == string(w)
}

// argContains 断言 SQL 参数（字符串）包含指定子串（用于 detail JSON）。
type argContains string

func (w argContains) Match(v driver.Value) bool {
	s, ok := v.(string)
	return ok && strings.Contains(s, string(w))
}

// TestSendMailAuditTargetSummary 批量邮件审计 target 必须是 batch:<count> 摘要
// （target 列 VARCHAR(128)，完整 ID 列表超长会导致审计写入失败静默丢失），
// 完整 ID 列表留痕在 detail JSON 的 ids 字段。
func TestSendMailAuditTargetSummary(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil) // 无 mailer

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id IN`)).
		WillReturnRows(userRows(nil))
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WithArgs(int64(1), "send_mail", argEq("batch:2"), argContains(`"ids":[7,8]`),
			argEq("127.0.0.1"), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.SendMail(context.Background(), 1, &model.AdminSendMailReq{
		IDs: []int64{7, 8}, Subject: "hi", Body: "hi",
	}, "127.0.0.1")
	require.NoError(t, err)
	require.Len(t, resp.Failed, 2, "SMTP 未配置时全部失败且留痕")
}

func TestSendMailMailerNotConfigured(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil) // 无 mailer

	// 仍先查用户，再进入“未配置”分支：全部失败且留痕
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id IN`)).
		WillReturnRows(userRows(nil))
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.SendMail(context.Background(), 1, &model.AdminSendMailReq{
		IDs: []int64{7, 8}, Subject: "hi", Body: "hi",
	}, "127.0.0.1")
	require.NoError(t, err)
	assert.EqualValues(t, 0, resp.Sent)
	require.Len(t, resp.Failed, 2, "SMTP 未配置时全部失败且留痕")
}

func TestSendMailMissingUser(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	ml := mailer.New(config.SMTPConfig{Host: "invalid.invalid", Port: 1})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, ml)

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id IN`)).
		WillReturnRows(userRows(nil))
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	resp, err := svc.SendMail(context.Background(), 1, &model.AdminSendMailReq{
		IDs: []int64{42}, Subject: "hi", Body: "hi",
	}, "127.0.0.1")
	require.NoError(t, err)
	require.Len(t, resp.Failed, 1)
	assert.Equal(t, "用户不存在", resp.Failed[0].Reason)
}

func TestResetUserSubTokenByAdmin(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.BaseURL = "https://api.example.com"
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, cfg, set, nil)

	u := &model.User{ID: 7, Email: "u7@y.link", SubToken: "old-token", UUID: "uuid-7"}
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(userRow(u))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	url, err := svc.ResetUserSubToken(context.Background(), 1, 7, "127.0.0.1")
	require.NoError(t, err)
	assert.Contains(t, url, "https://api.example.com/api/v1/client/subscribe/")
	assert.NotContains(t, url, "old-token", "新链接不得包含旧 token")
	// 旧 token 缓存已清除
	assert.False(t, e.mr.Exists("sub:userinfo:old-token"))

	// 用户不存在 → 40400
	e2 := newTestEnv(t)
	svc2 := NewAdminService(e2.db, e2.rdb, &repo.Repos{}, cfg, set, nil)
	e2.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnError(gorm.ErrRecordNotFound)
	_, err = svc2.ResetUserSubToken(context.Background(), 1, 99, "127.0.0.1")
	assert.Equal(t, 40400, codeOf(err))
}

func userRows(users []model.User) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "role", "balance", "commission_balance", "invite_by_id",
		"is_banned", "remind_expire", "remind_traffic", "telegram_id", "plan_id", "expired_at",
		"transfer_enable", "u", "d", "speed_limit", "device_limit", "sub_token", "uuid", "created_at", "updated_at",
	})
	for _, u := range users {
		rows.AddRow(u.ID, u.Email, u.PasswordHash, u.Role, u.Balance, u.CommissionBalance, u.InviteByID,
			u.IsBanned, u.RemindExpire, u.RemindTraffic, u.TelegramID, u.PlanID, u.ExpiredAt,
			u.TransferEnable, u.U, u.D, u.SpeedLimit, u.DeviceLimit, u.SubToken, u.UUID, u.CreatedAt, u.UpdatedAt)
	}
	return rows
}

func TestExportUsersStreamsBatches(t *testing.T) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, set, nil)
	ctx := context.Background()

	mPrice := int64(1000)
	plan := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice}
	inviter := model.User{ID: 3, Email: "inviter@y.link", SubToken: "t3", UUID: "uuid-3"}
	u := model.User{ID: 7, Email: "u7@y.link", Balance: 1000, CommissionBalance: 500,
		InviteByID: &inviter.ID, PlanID: &plan.ID, TransferEnable: 100, SubToken: "t7", UUID: "uuid-7"}

	// 1) 套餐名称映射
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).
		WillReturnRows(planRow(plan))
	// 2) 第一批（含 1 个用户）
	e.mock.ExpectQuery(`SELECT \* FROM "users" WHERE id >`).
		WillReturnRows(userRows([]model.User{u}))
	// 3) 批内邀请人补齐
	e.mock.ExpectQuery(`SELECT \* FROM "users" WHERE id IN`).
		WillReturnRows(userRows([]model.User{inviter}))
	// 4) 第二批为空 → 结束
	e.mock.ExpectQuery(`SELECT \* FROM "users" WHERE id >`).
		WillReturnRows(userRows(nil))

	var got [][]string
	err := svc.ExportUsers(ctx, "", func(rows [][]string) error {
		got = append(got, rows...)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "7", got[0][0])
	assert.Equal(t, "u7@y.link", got[0][1])
	assert.Equal(t, "10.00", got[0][2], "余额分→元")
	assert.Equal(t, "5.00", got[0][3])
	assert.Equal(t, "白羊座", got[0][4])
	assert.Equal(t, "100", got[0][6], "流量导出为字节")
	assert.Equal(t, "inviter@y.link", got[0][10])
}
