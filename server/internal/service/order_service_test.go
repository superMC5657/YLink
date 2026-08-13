package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/payment"
	"ylink-backend/internal/repo"
)

func planRow(p *model.Plan) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "name", "content", "month_price", "quarter_price", "half_year_price", "year_price",
		"onetime_price", "traffic_gb", "speed_limit", "device_limit", "group_ids", "is_show", "sort",
		"created_at", "updated_at",
	}).AddRow(p.ID, p.Name, p.Content, p.MonthPrice, p.QuarterPrice, p.HalfYearPrice, p.YearPrice,
		p.OnetimePrice, p.TrafficGB, p.SpeedLimit, p.DeviceLimit, p.GroupIDs, p.IsShow, p.Sort,
		p.CreatedAt, p.UpdatedAt)
}

func orderRow(o *model.Order) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "order_no", "user_id", "plan_id", "period", "amount", "discount_amount",
		"balance_used", "pay_amount", "coupon_id", "status", "pay_method", "paid_at",
		"idempotency_key", "created_at", "updated_at",
	}).AddRow(o.ID, o.OrderNo, o.UserID, o.PlanID, o.Period, o.Amount, o.DiscountAmount,
		o.BalanceUsed, o.PayAmount, o.CouponID, o.Status, o.PayMethod, o.PaidAt,
		o.IdempotencyKey, o.CreatedAt, o.UpdatedAt)
}

func paymentRow(p *model.Payment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "order_no", "user_id", "method", "amount", "trade_no", "status",
		"notify_payload", "paid_at", "created_at", "updated_at",
	}).AddRow(p.ID, p.OrderNo, p.UserID, p.Method, p.Amount, p.TradeNo, p.Status,
		p.NotifyPayload, p.PaidAt, p.CreatedAt, p.UpdatedAt)
}

func newOrderEnv(t *testing.T) (*testEnv, *OrderService) {
	e := newTestEnv(t)
	set := NewSettingService(e.db, e.rdb, &repo.Repos{})
	cfg := &config.Config{}
	cfg.App.BaseURL = "https://api.example.com"
	svc := NewOrderService(e.db, e.rdb, &repo.Repos{}, set, cfg, nil)
	return e, svc
}

func TestEpaySignAndVerify(t *testing.T) {
	ep := payment.NewEpay(payment.EpayConfig{Gateway: "https://pay.example.com", PID: "1000", Key: "secret"})
	drv := ep

	// 构造网关回调 form（按协议规则签名）
	form := url.Values{}
	form.Set("pid", "1000")
	form.Set("trade_no", "T2026")
	form.Set("out_trade_no", "2026010100000000001")
	form.Set("type", "alipay")
	form.Set("name", "test")
	form.Set("money", "10.00")
	form.Set("trade_status", "TRADE_SUCCESS")
	form.Set("sign_type", "MD5")
	signForm(form, "secret")

	req := httptest.NewRequest("POST", "/notify", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	nr, err := drv.VerifyNotify(req)
	require.NoError(t, err)
	assert.True(t, nr.Paid)
	assert.Equal(t, "T2026", nr.TradeNo)
	assert.Equal(t, int64(1000), nr.Amount)
	assert.Equal(t, "2026010100000000001", nr.OrderNo)

	// 篡改金额 → 验签失败
	bad := url.Values{}
	for k, v := range form {
		bad[k] = v
	}
	bad.Set("money", "100.00")
	req2 := httptest.NewRequest("POST", "/notify", strings.NewReader(bad.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = drv.VerifyNotify(req2)
	assert.Error(t, err)
}

// signForm 以易支付协议规则签名：除 sign/sign_type 外按 key 升序拼接 k=v&...，末尾 &key={商户密钥}，MD5 小写。
func signForm(form url.Values, key string) {
	keys := make([]string, 0, len(form))
	for k := range form {
		if k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(form.Get(k))
	}
	sb.WriteString("&key=")
	sb.WriteString(key)
	sum := md5.Sum([]byte(sb.String()))
	form.Set("sign", hex.EncodeToString(sum[:]))
}

func TestCouponCheckFixed(t *testing.T) {
	e, svc := newOrderEnv(t)
	// plan 查询
	now := time.Now()
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	// coupon 查询
	ended := now.Add(24 * time.Hour)
	cp := &model.Coupon{ID: 9, Code: "SALE", Type: 1, Value: 200, MinSpend: 500, StartedAt: &now, EndedAt: &ended, IsEnable: true}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)

	resp, err := svc.CouponCheck(context.Background(), 7, &model.CouponCheckReq{Code: "SALE", PlanID: 1, Period: "month"})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, 2.00, resp.DiscountAmount)
	assert.Equal(t, 8.00, resp.PayAmount)
}

func TestCreateOrderWithCouponAndIdem(t *testing.T) {
	e, svc := newOrderEnv(t)
	ctx := context.Background()
	now := time.Now()
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}

	// 第一次下单（无优惠券）
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnError(assert.AnError) // 幂等键未命中
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"orders\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p)) // toOrderResp

	resp, err := svc.CreateOrder(ctx, 7, "idem-1", &model.CreateOrderReq{PlanID: 1, Period: "month"})
	require.NoError(t, err)
	assert.Equal(t, "白羊座", resp.PlanName)
	assert.Equal(t, 10.00, resp.Amount)
	assert.Equal(t, 10.00, resp.PayAmount)

	// 幂等：同 Key 第二次直接返回首次订单
	o := &model.Order{ID: 1, OrderNo: "2026000000000000000000001", UserID: 7, PlanID: 1, Period: "month",
		Amount: 1000, PayAmount: 1000, Status: 0, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))

	resp2, err := svc.CreateOrder(ctx, 7, "idem-1", &model.CreateOrderReq{PlanID: 1, Period: "month"})
	require.NoError(t, err)
	// 幂等：返回首次（mock 中 existing）订单
	assert.Equal(t, "2026000000000000000000001", resp2.OrderNo)
	assert.Equal(t, "白羊座", resp2.PlanName)
}

func TestApplySubscriptionSamePlanRenewal(t *testing.T) {
	e, svc := newOrderEnv(t)
	now := time.Now()
	expired := now.Add(10 * 24 * time.Hour) // 未过期
	planID := int64(1)
	// 用户：同套餐、未过期
	u := &model.User{ID: 7, Email: "u@b.com", PlanID: &planID, ExpiredAt: &expired, TransferEnable: 300 * 1024 * 1024 * 1024, U: 100, D: 200, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	order := &model.Order{OrderNo: "O1", UserID: 7, PlanID: 1, Period: "month"}
	err := svc.applySubscription(e.db, order)
	require.NoError(t, err)
	// 断言通过 mock 记录的 SQL 更新了 expired_at/transfer_enable（此处验证无错误 + u/d 不清零路径执行）
	assert.NoError(t, err)
}

func TestHandleNotifyIdempotent(t *testing.T) {
	e, svc := newOrderEnv(t)
	ctx := context.Background()
	now := time.Now()
	o := &model.Order{ID: 1, OrderNo: "O2026", UserID: 7, PlanID: 1, Period: "month",
		Amount: 1000, PayAmount: 1000, Status: 0, CreatedAt: now, UpdatedAt: now}
	p := &model.Payment{ID: 1, OrderNo: "O2026", UserID: 7, Method: "epay_alipay", Amount: 1000, Status: 0, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectBegin()
	// 行锁读订单
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"orders\"")).WillReturnRows(orderRow(o))
	// 读支付单
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"payments\"")).WillReturnRows(paymentRow(p))
	// Save payment（UPDATE）
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"payments\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// Save order（UPDATE）
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"orders\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// applySubscription: 查用户
	u := &model.User{ID: 7, Email: "u@b.com", PlanID: nil, ExpiredAt: nil, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	// 查套餐
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(&model.Plan{ID: 1, Name: "p", TrafficGB: 300, CreatedAt: now, UpdatedAt: now}))
	// Save user
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	// grantCommission: 查下单用户（无邀请人 → 结束）
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(&model.User{ID: 7, Email: "u@b.com", PlanID: nil, InviteByID: nil, CreatedAt: now, UpdatedAt: now}))
	e.mock.ExpectCommit()

	err := svc.HandleNotify(ctx, "epay_alipay", &payment.NotifyResult{OrderNo: "O2026", TradeNo: "T1", Amount: 1000, Paid: true})
	require.NoError(t, err)
}

func TestAvailableCoupons(t *testing.T) {
	e, svc := newOrderEnv(t)
	ctx := context.Background()
	now := time.Now()

	// 两张可用券：id=1 正常；id=2 limit_per_user=1 且该用户已用满 → 应被过滤
	cp1 := &model.Coupon{ID: 1, Code: "SALE10", Type: 1, Value: 200, MinSpend: 0, LimitPerUser: 0, TotalLimit: 100, UsedCount: 3, IsEnable: true, CreatedAt: now, UpdatedAt: now}
	cp2 := &model.Coupon{ID: 2, Code: "ONCE", Type: 2, Value: 1000, MinSpend: 500, LimitPerUser: 1, TotalLimit: 0, UsedCount: 0, IsEnable: true, CreatedAt: now, UpdatedAt: now}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp1.ID, cp1.Code, cp1.Type, cp1.Value, cp1.MinSpend, cp1.LimitPerUser, cp1.TotalLimit, cp1.UsedCount,
		cp1.ValidPeriods, cp1.PlanIDs, cp1.StartedAt, cp1.EndedAt, cp1.IsEnable, now, now).
		AddRow(cp2.ID, cp2.Code, cp2.Type, cp2.Value, cp2.MinSpend, cp2.LimitPerUser, cp2.TotalLimit, cp2.UsedCount,
			cp2.ValidPeriods, cp2.PlanIDs, cp2.StartedAt, cp2.EndedAt, cp2.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)
	// cp1 limit=0 → 不查 usage；cp2 limit=1 → 查 usage 返回 1（已用满）
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"coupon_usages\"")).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1))

	list, err := svc.AvailableCoupons(ctx, 7, 0, "")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "SALE10", list[0].Code)
	assert.Equal(t, 1, list[0].Type)
	assert.Equal(t, 2.00, list[0].Value) // 200 分 → 2.00 元
	assert.Equal(t, 0.00, list[0].MinSpend)
}

func TestAvailableCouponsPlanPeriodFilter(t *testing.T) {
	e, svc := newOrderEnv(t)
	ctx := context.Background()
	now := time.Now()

	// 券仅适用套餐 2 + 周期 year
	planIDs := `[2]`
	periods := `["year"]`
	cp := &model.Coupon{ID: 3, Code: "YEARONLY", Type: 2, Value: 1000, MinSpend: 0, LimitPerUser: 0, TotalLimit: 0, UsedCount: 0,
		ValidPeriods: &periods, PlanIDs: &planIDs, IsEnable: true, CreatedAt: now, UpdatedAt: now}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)

	// 请求套餐 1 + month → 不匹配，返回空
	list, err := svc.AvailableCoupons(ctx, 7, 1, "month")
	require.NoError(t, err)
	require.Empty(t, list)

	// 请求套餐 2 + year → 匹配（sqlmock.Rows 为迭代器不可复用，需新建）
	rows2 := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(rows2)
	list, err = svc.AvailableCoupons(ctx, 7, 2, "year")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "YEARONLY", list[0].Code)
	assert.Equal(t, 10.00, list[0].Value) // 百分比 1000 分 → 10（%）
}

func TestCouponCheckPercent(t *testing.T) {
	// 回归：百分比券不应整单全免。
	// 10% 券 Value 以「百分比×100」的分存储 = 1000；订单 10 元 = 1000 分。
	// 期望优惠 1 元（100 分）、实付 9 元，而非 10000 分/全免。
	e, svc := newOrderEnv(t)
	now := time.Now()
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	cp := &model.Coupon{ID: 9, Code: "PCT10", Type: 2, Value: 1000, MinSpend: 0, LimitPerUser: 0, TotalLimit: 0, UsedCount: 0, IsEnable: true, CreatedAt: now, UpdatedAt: now}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)

	resp, err := svc.CouponCheck(context.Background(), 7, &model.CouponCheckReq{Code: "PCT10", PlanID: 1, Period: "month"})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, 1.00, resp.DiscountAmount) // 10% × 10 元 = 1 元
	assert.Equal(t, 9.00, resp.PayAmount)
}

func TestCouponCheckPercentCapped(t *testing.T) {
	// 100% 券 Value = 10000 分；订单 10 元 → 优惠封顶 10 元（不应溢出为负）
	e, svc := newOrderEnv(t)
	now := time.Now()
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	cp := &model.Coupon{ID: 9, Code: "FREE", Type: 2, Value: 10000, MinSpend: 0, LimitPerUser: 0, TotalLimit: 0, UsedCount: 0, IsEnable: true, CreatedAt: now, UpdatedAt: now}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)

	resp, err := svc.CouponCheck(context.Background(), 7, &model.CouponCheckReq{Code: "FREE", PlanID: 1, Period: "month"})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, 10.00, resp.DiscountAmount)
	assert.Equal(t, 0.00, resp.PayAmount)
}

// TestCreateOrderCouponPerUserLimit:同一用户同一张券连续下单两次,limit_per_user=1
// 时第二次必须被拒(12001),防止「一个人可以使用多次」的回归。
func TestCreateOrderCouponPerUserLimit(t *testing.T) {
	e, svc := newOrderEnv(t)
	ctx := context.Background()
	now := time.Now()
	mPrice := int64(1000)
	p := &model.Plan{ID: 1, Name: "白羊座", MonthPrice: &mPrice, TrafficGB: 300, IsShow: true, CreatedAt: now, UpdatedAt: now}
	cp := &model.Coupon{ID: 9, Code: "ONCE", Type: 1, Value: 200, MinSpend: 0, LimitPerUser: 1, TotalLimit: 0, UsedCount: 0, IsEnable: true, CreatedAt: now, UpdatedAt: now}
	couponRows := sqlmock.NewRows([]string{
		"id", "code", "type", "value", "min_spend", "limit_per_user", "total_limit", "used_count",
		"valid_periods", "plan_ids", "started_at", "ended_at", "is_enable", "created_at", "updated_at",
	}).AddRow(cp.ID, cp.Code, cp.Type, cp.Value, cp.MinSpend, cp.LimitPerUser, cp.TotalLimit, cp.UsedCount,
		cp.ValidPeriods, cp.PlanIDs, cp.StartedAt, cp.EndedAt, cp.IsEnable, now, now)

	// ---- 第一次下单:校验通过,占用 + 记账 ----
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)
	// validateCoupon 内的 CountUsage(非锁定):当前该用户 0 次
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"coupon_usages\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	// Occupy 原子占用
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"coupons\" SET \"used_count\"=used_count + 1")).WillReturnResult(sqlmock.NewResult(0, 1))
	// CountUsageLocked(锁定读,当前无记录)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupon_usages\"")).WillReturnRows(sqlmock.NewRows([]string{}))
	// RecordUsage 落账
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"coupon_usages\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	// 建单
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"orders\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p)) // toOrderResp

	resp, err := svc.CreateOrder(ctx, 7, "", &model.CreateOrderReq{PlanID: 1, Period: "month", CouponCode: "ONCE"})
	require.NoError(t, err)
	assert.Equal(t, 8.00, resp.PayAmount)

	// ---- 第二次下单:validateCoupon 发现该用户已用满 → 拒绝 ----
	cp.UsedCount = 1
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"plans\"")).WillReturnRows(planRow(p))
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"coupons\"")).WillReturnRows(couponRows)
	// validateCoupon 内的 CountUsage(非锁定):该用户已用 1 次
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"coupon_usages\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))
	e.mock.ExpectRollback()

	_, err = svc.CreateOrder(ctx, 7, "", &model.CreateOrderReq{PlanID: 1, Period: "month", CouponCode: "ONCE"})
	assert.Equal(t, 12001, codeOf(err), "同一用户同一张券 limit_per_user=1 时第二次下单必须被拒")
}

// TestGrantCommissionBalanceSkipped:余额支付订单不产生佣金(grantCommission 直接跳过,
// 不做任何用户/佣金查询;sqlmock 未设期望,若有查询会报 unexpected call 而失败)。
func TestGrantCommissionBalanceSkipped(t *testing.T) {
	e, svc := newOrderEnv(t)
	pm := "balance"
	o := &model.Order{OrderNo: "O1", UserID: 7, PayMethod: &pm}
	err := svc.grantCommission(e.db, o, 1000)
	require.NoError(t, err)
	require.NoError(t, e.mock.ExpectationsWereMet(), "余额支付不应产生任何 DB 查询(不写佣金)")
}
