package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/repo"
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
			// 每用户限次：Occupy 已锁 coupon 行串行化并发下单，再用锁定读校验
			if coupon.LimitPerUser > 0 {
				n, err := s.repos.Coupon.CountUsageLocked(tx, coupon.ID, userID)
				if err != nil {
					return err
				}
				if n >= int64(coupon.LimitPerUser) {
					return errs.ErrCoupon
				}
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
		// 幂等键并发竞争：唯一索引兜底，命中后返回首次创建的订单而非 500
		if idemKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			if existing, e2 := s.repos.Order.GetByIdempotencyKey(s.db, idemKey, userID); e2 == nil {
				return s.toOrderResp(s.db, existing)
			}
		}
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
	// 套餐可能已被删除：历史订单仍可查看，套餐名回退占位
	planName := "已删除套餐"
	if plan, err := s.repos.Plan.GetByID(s.db, o.PlanID); err == nil {
		planName = plan.Name
	}
	resp := &model.OrderDetailResp{
		OrderNo:        o.OrderNo,
		PlanName:       planName,
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
		if code, err := couponCode(s.db, *o.CouponID); err == nil {
			resp.CouponCode = &code
		}
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
			if err := releaseCoupon(tx, *o.CouponID, userID, orderNo); err != nil {
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

// ---- 支付回调 ----

// ---- 工具 ----

// genOrderNo 生成订单号：YYYYMMDDHHMMSS + 13 位随机数字。
func genOrderNo() string {
	ts := time.Now().Format("20060102150405")
	n, _ := rand.Int(rand.Reader, big.NewInt(1e13))
	return fmt.Sprintf("%s%013d", ts, n.Int64())
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
