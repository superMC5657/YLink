package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ylink/internal/config"
	"ylink/internal/middleware"
	"ylink/internal/model"
	"ylink/internal/pkg/errs"
	"ylink/internal/pkg/logger"
	"ylink/internal/pkg/mailer"
	"ylink/internal/pkg/payment"
	redispkg "ylink/internal/pkg/redis"
	"ylink/internal/repo"
)

// OrderService 交易域：套餐、优惠券、下单、收银台、支付回调、开通/续期。
type OrderService struct {
	db     *gorm.DB
	rdb    *redis.Client
	repos  *repo.Repos
	set    *SettingService
	cfg    *config.Config
	mailer *mailer.Mailer
}

func NewOrderService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, set *SettingService, cfg *config.Config, mail *mailer.Mailer) *OrderService {
	return &OrderService{db: db, rdb: rdb, repos: repos, set: set, cfg: cfg, mailer: mail}
}

// ---- 套餐 ----

// Plans GET /plans 上架套餐列表（prices 只含支持的周期）。
func (s *OrderService) Plans(ctx context.Context) ([]model.PlanResp, error) {
	plans, err := s.repos.Plan.ListShown(s.db)
	if err != nil {
		return nil, err
	}
	out := make([]model.PlanResp, 0, len(plans))
	for _, p := range plans {
		out = append(out, model.PlanResp{
			ID:          p.ID,
			Name:        p.Name,
			Prices:      planPrices(&p),
			TrafficGB:   p.TrafficGB,
			SpeedLimit:  p.SpeedLimit,
			DeviceLimit: p.DeviceLimit,
			Content:     p.Content,
			Sort:        p.Sort,
		})
	}
	return out, nil
}

func planPrices(p *model.Plan) model.PlanPrices {
	pp := model.PlanPrices{}
	if p.MonthPrice != nil {
		v := model.FenToYuan(*p.MonthPrice)
		pp.Month = &v
	}
	if p.QuarterPrice != nil {
		v := model.FenToYuan(*p.QuarterPrice)
		pp.Quarter = &v
	}
	if p.HalfYearPrice != nil {
		v := model.FenToYuan(*p.HalfYearPrice)
		pp.HalfYear = &v
	}
	if p.YearPrice != nil {
		v := model.FenToYuan(*p.YearPrice)
		pp.Year = &v
	}
	if p.OnetimePrice != nil {
		v := model.FenToYuan(*p.OnetimePrice)
		pp.Onetime = &v
	}
	return pp
}

// priceOf 取周期价格；NULL=不支持。
func priceOf(p *model.Plan, period string) *int64 {
	switch period {
	case "month":
		return p.MonthPrice
	case "quarter":
		return p.QuarterPrice
	case "half_year":
		return p.HalfYearPrice
	case "year":
		return p.YearPrice
	case "onetime":
		return p.OnetimePrice
	}
	return nil
}

// ---- 优惠券 ----

// CouponCheck POST /coupons/check 纯试算。
func (s *OrderService) CouponCheck(ctx context.Context, userID int64, req *model.CouponCheckReq) (*model.CouponCheckResp, error) {
	plan, err := s.repos.Plan.GetByID(s.db, req.PlanID)
	if err != nil || !plan.IsShow {
		return nil, errs.ErrPlanNotAvailable
	}
	price := priceOf(plan, req.Period)
	if price == nil {
		return nil, errs.ErrPlanPeriod
	}
	_, discount, err := s.validateCoupon(s.db, userID, req.Code, plan, req.Period, *price)
	if err != nil {
		return nil, err
	}
	return &model.CouponCheckResp{
		Valid:          true,
		DiscountAmount: model.FenToYuan(discount),
		PayAmount:      model.FenToYuan(*price - discount),
	}, nil
}

// validateCoupon 校验优惠券并返回优惠金额（分）。db 由调用方传入（下单传事务）。
func (s *OrderService) validateCoupon(db *gorm.DB, userID int64, code string, plan *model.Plan, period string, amount int64) (*model.Coupon, int64, error) {
	coupon, err := s.repos.Coupon.GetByCode(db, code)
	if err != nil {
		return nil, 0, errs.ErrCoupon
	}
	now := time.Now()
	if coupon.StartedAt != nil && now.Before(*coupon.StartedAt) {
		return nil, 0, errs.ErrCoupon
	}
	if coupon.EndedAt != nil && now.After(*coupon.EndedAt) {
		return nil, 0, errs.ErrCoupon
	}
	// 适用套餐
	if coupon.PlanIDs != nil {
		var ids []int64
		if json.Unmarshal([]byte(*coupon.PlanIDs), &ids) == nil && !containsInt64(ids, plan.ID) {
			return nil, 0, errs.ErrCoupon
		}
	}
	// 适用周期
	if coupon.ValidPeriods != nil {
		var periods []string
		if json.Unmarshal([]byte(*coupon.ValidPeriods), &periods) == nil && !containsStr(periods, period) {
			return nil, 0, errs.ErrCoupon
		}
	}
	if amount < coupon.MinSpend {
		return nil, 0, errs.ErrCoupon
	}
	if coupon.TotalLimit > 0 && coupon.UsedCount >= coupon.TotalLimit {
		return nil, 0, errs.ErrCoupon
	}
	if coupon.LimitPerUser > 0 {
		n, err := s.repos.Coupon.CountUsage(db, coupon.ID, userID)
		if err == nil && n >= int64(coupon.LimitPerUser) {
			return nil, 0, errs.ErrCoupon
		}
	}
	// 计算优惠
	var discount int64
	if coupon.Type == 1 { // 固定金额
		discount = min64(coupon.Value, amount)
	} else { // 百分比
		discount = amount * coupon.Value / 100
		if discount > amount {
			discount = amount
		}
	}
	return coupon, discount, nil
}

// ---- 下单 ----

// CreateOrder POST /orders（幂等键防重复建单；事务内原子占用优惠券防超发）。
func (s *OrderService) CreateOrder(ctx context.Context, userID int64, idemKey string, req *model.CreateOrderReq) (*model.OrderResp, error) {
	if idemKey != "" {
		if existing, err := s.repos.Order.GetByIdempotencyKey(s.db, idemKey, userID); err == nil {
			return s.toOrderResp(s.db, existing)
		}
	}
	plan, err := s.repos.Plan.GetByID(s.db, req.PlanID)
	if err != nil || !plan.IsShow {
		return nil, errs.ErrPlanNotAvailable
	}
	price := priceOf(plan, req.Period)
	if price == nil {
		return nil, errs.ErrPlanPeriod
	}
	amount := *price

	order := &model.Order{
		OrderNo: genOrderNo(),
		UserID:  userID,
		PlanID:  plan.ID,
		Period:  req.Period,
		Amount:  amount,
		Status:  model.OrderPending,
	}
	if idemKey != "" {
		order.IdempotencyKey = &idemKey
	}

	err = repo.WithTx(s.db, func(tx *gorm.DB) error {
		var discount int64
		if req.CouponCode != "" {
			coupon, d, err := s.validateCoupon(tx, userID, req.CouponCode, plan, req.Period, amount)
			if err != nil {
				return err
			}
			// 原子占用总限量（条件更新），失败即超发 → 12001
			ok, err := s.repos.Coupon.Occupy(tx, coupon.ID)
			if err != nil {
				return err
			}
			if !ok {
				return errs.ErrCoupon
			}
			if err := s.repos.Coupon.RecordUsage(tx, coupon.ID, userID, order.OrderNo); err != nil {
				return err
			}
			discount = d
			order.CouponID = &coupon.ID
		}
		order.DiscountAmount = discount
		order.PayAmount = amount - discount
		return s.repos.Order.Create(tx, order)
	})
	if err != nil {
		return nil, err
	}
	return s.toOrderResp(s.db, order)
}

// toOrderResp 组装订单响应（附带套餐名）。
func (s *OrderService) toOrderResp(db *gorm.DB, o *model.Order) (*model.OrderResp, error) {
	plan, err := s.repos.Plan.GetByID(db, o.PlanID)
	if err != nil {
		return nil, err
	}
	return &model.OrderResp{
		OrderNo:        o.OrderNo,
		PlanName:       plan.Name,
		Period:         o.Period,
		Amount:         model.FenToYuan(o.Amount),
		DiscountAmount: model.FenToYuan(o.DiscountAmount),
		PayAmount:      model.FenToYuan(o.PayAmount),
		Status:         o.Status,
		CreatedAt:      o.CreatedAt,
	}, nil
}

// ListOrders GET /orders 订单列表。
func (s *OrderService) ListOrders(ctx context.Context, userID int64, status *int, page, pageSize int) ([]model.OrderResp, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	orders, total, err := s.repos.Order.ListByUser(s.db, userID, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	// 批量取套餐名
	planNames := map[int64]string{}
	out := make([]model.OrderResp, 0, len(orders))
	for _, o := range orders {
		name, ok := planNames[o.PlanID]
		if !ok {
			if p, err := s.repos.Plan.GetByID(s.db, o.PlanID); err == nil {
				name = p.Name
				planNames[o.PlanID] = name
			}
		}
		out = append(out, model.OrderResp{
			OrderNo:        o.OrderNo,
			PlanName:       name,
			Period:         o.Period,
			Amount:         model.FenToYuan(o.Amount),
			DiscountAmount: model.FenToYuan(o.DiscountAmount),
			PayAmount:      model.FenToYuan(o.PayAmount),
			Status:         o.Status,
			CreatedAt:      o.CreatedAt,
		})
	}
	return out, total, nil
}

// GetOrder GET /orders/{order_no} 详情（仅本人）。
func (s *OrderService) GetOrder(ctx context.Context, userID int64, orderNo string) (*model.OrderDetailResp, error) {
	o, err := s.repos.Order.GetByNoAndUser(s.db, orderNo, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	plan, _ := s.repos.Plan.GetByID(s.db, o.PlanID)
	resp := &model.OrderDetailResp{
		OrderNo:        o.OrderNo,
		PlanName:       plan.Name,
		Period:         o.Period,
		Amount:         model.FenToYuan(o.Amount),
		DiscountAmount: model.FenToYuan(o.DiscountAmount),
		BalanceUsed:    model.FenToYuan(o.BalanceUsed),
		PayAmount:      model.FenToYuan(o.PayAmount),
		Status:         o.Status,
		PayMethod:      o.PayMethod,
		PaidAt:         o.PaidAt,
		CreatedAt:      o.CreatedAt,
	}
	if o.CouponID != nil {
		code := couponCode(s.db, *o.CouponID)
		resp.CouponCode = &code
	}
	return resp, nil
}

// CancelOrder POST /orders/{no}/cancel 仅待支付可取消；取消时回退优惠券占用。
func (s *OrderService) CancelOrder(ctx context.Context, userID int64, orderNo string) (*model.OrderResp, error) {
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		o, err := s.repos.Order.GetByNoAndUser(tx, orderNo, userID)
		if err != nil {
			return errs.ErrNotFound
		}
		if o.Status != model.OrderPending {
			return errs.ErrOrderStatus
		}
		// 条件更新（防与支付回调竞态）：影响行数为 0 说明已被并发完成/取消
		affected, err := s.repos.Order.UpdateStatusIfPending(tx, orderNo, model.OrderCanceled)
		if err != nil {
			return err
		}
		if affected == 0 {
			return errs.ErrOrderStatus
		}
		if o.CouponID != nil {
			if err := s.repos.Coupon.Release(tx, *o.CouponID); err != nil {
				return err
			}
			if err := s.repos.Coupon.DeleteUsage(tx, *o.CouponID, userID, orderNo); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 读回更新后状态
	o, err := s.repos.Order.GetByNoAndUser(s.db, orderNo, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	return s.toOrderResp(s.db, o)
}

// ---- 收银台 ----

// Checkout POST /orders/{no}/checkout 拉起支付或余额直付。
func (s *OrderService) Checkout(ctx context.Context, userID int64, orderNo, method string, r *http.Request) (*model.CheckoutResp, error) {
	order, err := s.repos.Order.GetByNoAndUser(s.db, orderNo, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if order.Status != model.OrderPending {
		return nil, errs.ErrOrderStatus
	}

	if method == "balance" {
		return s.checkoutBalance(ctx, order)
	}

	driver := payment.Get(method)
	if driver == nil {
		return nil, errs.ErrPayMethod
	}
	// 30 分钟内重复 checkout 返回原支付单
	cacheKey := redispkg.Key("order", "paying", orderNo)
	if cached, err := s.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
		var res model.CheckoutResp
		if json.Unmarshal(cached, &res) == nil {
			return &res, nil
		}
	}

	paymentRec := &model.Payment{
		OrderNo: order.OrderNo,
		UserID:  userID,
		Method:  method,
		Amount:  order.PayAmount,
		Status:  model.PayPending,
	}
	if err := s.repos.Payment.Create(s.db, paymentRec); err != nil {
		return nil, err
	}
	notifyURL := s.cfg.App.BaseURL + "/api/v1/payment/notify/" + method
	result, err := driver.CreatePayment(ctx, &payment.Payment{
		OrderNo:   order.OrderNo,
		Method:    method,
		Amount:    order.PayAmount,
		Subject:   "YLink 订阅",
		NotifyURL: notifyURL,
		ReturnURL: s.cfg.App.BaseURL,
	})
	if err != nil {
		return nil, errs.ErrInternal
	}
	res := &model.CheckoutResp{Type: result.Type, Content: result.Content, ExpireIn: result.ExpireIn}
	if b, err := json.Marshal(res); err == nil {
		s.rdb.Set(ctx, cacheKey, b, 30*time.Minute)
	}
	return res, nil
}

// checkoutBalance 余额支付：扣减余额 → 走与在线支付相同的后置逻辑。
func (s *OrderService) checkoutBalance(ctx context.Context, order *model.Order) (*model.CheckoutResp, error) {
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		locked, err := s.repos.Order.GetByNoForUpdate(tx, order.OrderNo)
		if err != nil {
			return err
		}
		if locked.Status != model.OrderPending {
			return errs.ErrOrderStatus
		}
		user, err := s.repos.User.GetByIDForUpdate(tx, locked.UserID)
		if err != nil {
			return err
		}
		if user.Balance < locked.PayAmount {
			return errs.ErrBalanceInsufficient
		}
		user.Balance -= locked.PayAmount
		if err := s.repos.User.Save(tx, user); err != nil {
			return err
		}
		locked.BalanceUsed = locked.PayAmount
		locked.PayAmount = 0
		payMethod := "balance"
		now := time.Now()
		locked.Status = model.OrderCompleted
		locked.PayMethod = &payMethod
		locked.PaidAt = &now
		if err := s.repos.Order.Save(tx, locked); err != nil {
			return err
		}
		if err := s.applySubscription(tx, locked); err != nil {
			return err
		}
		return s.grantCommission(tx, locked)
	})
	if err != nil {
		return nil, err
	}
	s.sendReceiptMailAsync(order.OrderNo)
	return &model.CheckoutResp{Type: "paid", Content: "", ExpireIn: 0}, nil
}

// ---- 支付回调 ----

// HandleNotify 处理异步通知：验签在 handler 层完成，这里做幂等与账务。
func (s *OrderService) HandleNotify(ctx context.Context, method string, nr *payment.NotifyResult) error {
	if nr == nil || !nr.Paid || nr.TradeNo == "" {
		return errors.New("notify not paid")
	}
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		order, err := s.repos.Order.GetByNoForUpdate(tx, nr.OrderNo)
		if err != nil {
			return err
		}
		if order.Status != model.OrderPending {
			return nil // 已处理（幂等）
		}
		if order.PayAmount != nr.Amount {
			return errors.New("amount mismatch")
		}
		// 支付单落成功（trade_no 唯一约束兜底并发）
		paymentRec, err := s.repos.Payment.GetByOrderAndMethod(tx, nr.OrderNo, method)
		if err != nil {
			return err
		}
		if paymentRec.Status == model.PaySuccess {
			return nil
		}
		now := time.Now()
		paymentRec.TradeNo = &nr.TradeNo
		paymentRec.Status = model.PaySuccess
		paymentRec.PaidAt = &now
		if err := s.repos.Payment.Save(tx, paymentRec); err != nil {
			return err
		}
		order.Status = model.OrderCompleted
		order.PayMethod = &method
		order.PaidAt = &now
		if err := s.repos.Order.Save(tx, order); err != nil {
			return err
		}
		if err := s.applySubscription(tx, order); err != nil {
			return err
		}
		return s.grantCommission(tx, order)
	})
	if err == nil {
		middleware.PaySuccessInc(method)
		s.sendReceiptMailAsync(nr.OrderNo)
	}
	return err
}

// sendReceiptMailAsync 支付成功回执邮件（异步，不阻塞主流程）。
func (s *OrderService) sendReceiptMailAsync(orderNo string) {
	if s.mailer == nil || s.cfg.App.Name == "" {
		return
	}
	go func() {
		o, err := s.repos.Order.GetByNo(s.db, orderNo)
		if err != nil {
			return
		}
		user, err := s.repos.User.GetByID(s.db, o.UserID)
		if err != nil {
			return
		}
		plan, err := s.repos.Plan.GetByID(s.db, o.PlanID)
		if err != nil {
			return
		}
		body := fmt.Sprintf("您的订单 <b>%s</b> 已支付成功。<br>套餐：%s（%s）<br>实付：¥%.2f",
			o.OrderNo, plan.Name, o.Period, model.FenToYuan(o.PayAmount))
		rendered, err := mailer.Render(mailer.Template(body), s.cfg.App.Name, nil)
		if err != nil {
			return
		}
		subject := fmt.Sprintf("[%s] 支付成功", s.cfg.App.Name)
		if err := s.mailer.Send(user.Email, subject, rendered); err != nil {
			logger.L().Error("send receipt mail failed", zap.String("order_no", orderNo), zap.Error(err))
		}
	}()
}

// applySubscription 开通/续期规则（core-flows 2.1）。
func (s *OrderService) applySubscription(tx *gorm.DB, order *model.Order) error {
	user, err := s.repos.User.GetByID(tx, order.UserID)
	if err != nil {
		return err
	}
	plan, err := s.repos.Plan.GetByID(tx, order.PlanID)
	if err != nil {
		return err
	}
	trafficBytes := int64(plan.TrafficGB) * 1024 * 1024 * 1024
	now := time.Now()

	if order.Period == "onetime" {
		// 一次性：只叠加流量，不改到期时间
		user.TransferEnable += trafficBytes
	} else {
		dur := model.PeriodDuration(order.Period)
		samePlan := user.PlanID != nil && *user.PlanID == plan.ID &&
			user.ExpiredAt != nil && user.ExpiredAt.After(now)
		if samePlan {
			// 同套餐续费（未过期）：expired_at += 周期；transfer_enable += 流量；u/d 不清零
			exp := user.ExpiredAt.Add(dur)
			user.ExpiredAt = &exp
			user.TransferEnable += trafficBytes
		} else {
			// 无订阅/已过期/不同套餐：替换；expired_at = max(now, 原expired_at) + 周期；u/d 清零
			base := now
			if user.ExpiredAt != nil && user.ExpiredAt.After(now) {
				base = *user.ExpiredAt
			}
			exp := base.Add(dur)
			user.ExpiredAt = &exp
			user.PlanID = &plan.ID
			user.TransferEnable = trafficBytes
			user.U = 0
			user.D = 0
		}
	}
	user.SpeedLimit = plan.SpeedLimit
	user.DeviceLimit = plan.DeviceLimit
	return s.repos.User.Save(tx, user)
}

// grantCommission 下单支付成功后写佣金（确认中）。
func (s *OrderService) grantCommission(tx *gorm.DB, order *model.Order) error {
	user, err := s.repos.User.GetByID(tx, order.UserID)
	if err != nil {
		return err
	}
	if user.InviteByID == nil {
		return nil
	}
	inviter, err := s.repos.User.GetByID(tx, *user.InviteByID)
	if err != nil {
		return nil // 邀请人异常不阻塞支付
	}
	rate := s.commissionRate(tx, inviter.Role)
	amount := order.PayAmount * int64(rate) / 100
	if amount <= 0 {
		return nil
	}
	return s.repos.Commission.Create(tx, &model.CommissionLog{
		InviteUserID: *user.InviteByID,
		FromUserID:   user.ID,
		OrderNo:      order.OrderNo,
		OrderAmount:  order.PayAmount,
		Rate:         rate,
		Amount:       amount,
		Status:       model.CommissionPending,
	})
}

// commissionRate 取佣金比例（代理商取 agent 比例）。
func (s *OrderService) commissionRate(tx *gorm.DB, inviterRole int) int {
	type inviteCfg struct {
		CommissionRate      int `json:"commission_rate"`
		AgentCommissionRate int `json:"agent_commission_rate"`
	}
	var cfg inviteCfg
	if raw, err := s.repos.Setting.Get(tx, "invite"); err == nil {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.CommissionRate <= 0 {
		cfg.CommissionRate = 40
	}
	if cfg.AgentCommissionRate <= 0 {
		cfg.AgentCommissionRate = 50
	}
	if inviterRole == model.RoleAgent {
		return cfg.AgentCommissionRate
	}
	return cfg.CommissionRate
}

// ---- 工具 ----

// genOrderNo 生成订单号：YYYYMMDDHHMMSS + 13 位随机数字。
func genOrderNo() string {
	ts := time.Now().Format("20060102150405")
	n, _ := rand.Int(rand.Reader, big.NewInt(1e13))
	return fmt.Sprintf("%s%013d", ts, n.Int64())
}

func couponCode(db *gorm.DB, id int64) string {
	var c struct {
		Code string
	}
	db.Model(&model.Coupon{}).Select("code").Where("id = ?", id).Scan(&c)
	return c.Code
}

func containsInt64(list []int64, v int64) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
