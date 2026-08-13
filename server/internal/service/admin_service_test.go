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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, nil, set)

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
	svc := NewAdminService(e.db, e.rdb, repos, &config.Config{}, set)
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
	svc := NewAdminService(e.db, e.rdb, repos, &config.Config{}, set)
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
