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

// Order 交易域门面：对上层保持原有窄口，域逻辑收敛于 orderTradeModule。
//
// 术语说明：下文用 module 指代按域内聚的实现单元，用 interface/seam 指代
// module 对外暴露的窄契约，用 adapter 指代对仓储的委托封装。
// 下单计价/券校验/支付后置/履约收敛于同一 module 的同一 seam，
// 券 scope 解析与占用释放集中一处，仓储经由 seam 注入。
type OrderService struct {
	db     *gorm.DB
	rdb    *redis.Client
	repos  *repo.Repos
	set    *SettingService
	cfg    *config.Config
	mailer *mailer.Mailer
	trade  *orderTradeModule
}

func NewOrderService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, set *SettingService, cfg *config.Config, mail *mailer.Mailer) *OrderService {
	if repos == nil {
		repos = &repo.Repos{}
	}
	s := &OrderService{db: db, rdb: rdb, repos: repos, set: set, cfg: cfg, mailer: mail}
	s.trade = &orderTradeModule{db: db, repos: repos, coupons: orderCouponRepoAdapter{repos: repos}}
	return s
}

// orderTradeModule 下单域深 module：套餐取价 + 券解析/校验/占用释放 + 支付后置 + 履约内聚一处。
// 计价方法见 order_coupon.go，履约方法见 order_fulfill.go，收银台编排见 order_payment.go。
type orderTradeModule struct {
	db      *gorm.DB
	repos   *repo.Repos
	coupons orderCouponSeam
}

// orderCouponSeam 券仓储窄 seam：trade module 依赖的唯一券读写口，经由 adapter 注入。
type orderCouponSeam interface {
	GetByCode(db *gorm.DB, code string) (*model.Coupon, error)
	ListAvailable(db *gorm.DB, now time.Time) ([]model.Coupon, error)
	CountUsage(db *gorm.DB, couponID, userID int64) (int64, error)
	CountUsageLocked(db *gorm.DB, couponID, userID int64) (int64, error)
	Occupy(db *gorm.DB, couponID int64) (bool, error)
	Release(db *gorm.DB, couponID int64) error
	DeleteUsage(db *gorm.DB, couponID, userID int64, orderNo string) error
	RecordUsage(db *gorm.DB, couponID, userID int64, orderNo string) error
	CodeByID(db *gorm.DB, id int64) (string, error)
}

// orderCouponRepoAdapter seam 的 adapter：以后端 repos 装配，不直 new 仓储。
type orderCouponRepoAdapter struct {
	repos *repo.Repos
}

var _ orderCouponSeam = orderCouponRepoAdapter{}

func (a orderCouponRepoAdapter) GetByCode(db *gorm.DB, code string) (*model.Coupon, error) {
	return a.repos.Coupon.GetByCode(db, code)
}

func (a orderCouponRepoAdapter) ListAvailable(db *gorm.DB, now time.Time) ([]model.Coupon, error) {
	return a.repos.Coupon.ListAvailable(db, now)
}

func (a orderCouponRepoAdapter) CountUsage(db *gorm.DB, couponID, userID int64) (int64, error) {
	return a.repos.Coupon.CountUsage(db, couponID, userID)
}

func (a orderCouponRepoAdapter) CountUsageLocked(db *gorm.DB, couponID, userID int64) (int64, error) {
	return a.repos.Coupon.CountUsageLocked(db, couponID, userID)
}

func (a orderCouponRepoAdapter) Occupy(db *gorm.DB, couponID int64) (bool, error) {
	return a.repos.Coupon.Occupy(db, couponID)
}

func (a orderCouponRepoAdapter) Release(db *gorm.DB, couponID int64) error {
	return a.repos.Coupon.Release(db, couponID)
}

func (a orderCouponRepoAdapter) DeleteUsage(db *gorm.DB, couponID, userID int64, orderNo string) error {
	return a.repos.Coupon.DeleteUsage(db, couponID, userID, orderNo)
}

func (a orderCouponRepoAdapter) RecordUsage(db *gorm.DB, couponID, userID int64, orderNo string) error {
	return a.repos.Coupon.RecordUsage(db, couponID, userID, orderNo)
}

func (a orderCouponRepoAdapter) CodeByID(db *gorm.DB, id int64) (string, error) {
	return fetchCouponCode(db, id)
}

// orderTradeSeam trade module 对外窄 interface：下单计价/券校验/支付后置/履约收口。
// 上层仅经由 Order 门面触达本 seam，不直触仓储。
type orderTradeSeam interface {
	priceOf(p *model.Plan, period string) *int64
	planPrices(p *model.Plan) model.PlanPrices
	validate(db *gorm.DB, userID int64, code string, plan *model.Plan, period string, amount int64) (*model.Coupon, int64, error)
	available(db *gorm.DB, userID, planID int64, period string) ([]model.CouponItem, error)
	claim(tx *gorm.DB, userID int64, code string, plan *model.Plan, period string, amount int64, orderNo string) (*model.Coupon, int64, error)
	release(db *gorm.DB, couponID, userID int64, orderNo string) error
	codeByID(db *gorm.DB, id int64) (string, error)
	apply(tx *gorm.DB, order *model.Order) error
	grant(tx *gorm.DB, order *model.Order, payAmount int64) error
	isBalanceChannel(method string) bool
}

var _ orderTradeSeam = (*orderTradeModule)(nil)

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
			Prices:      s.trade.planPrices(&p),
			TrafficGB:   p.TrafficGB,
			SpeedLimit:  p.SpeedLimit,
			DeviceLimit: p.DeviceLimit,
			Content:     p.Content,
			Sort:        p.Sort,
		})
	}
	return out, nil
}

// priceOf 取周期价格；NULL=不支持。实现收敛于 trade module，本函数仅为存量薄 wrapper。
func priceOf(p *model.Plan, period string) *int64 {
	return (&orderTradeModule{}).priceOf(p, period)
}

// planPrices 套餐展示价格。实现收敛于 trade module，本函数仅为存量薄 wrapper。
func planPrices(p *model.Plan) model.PlanPrices {
	return (&orderTradeModule{}).planPrices(p)
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
	price := s.trade.priceOf(plan, req.Period)
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
			coupon, d, err := s.trade.claim(tx, userID, req.CouponCode, plan, req.Period, amount, order.OrderNo)
			if err != nil {
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
		if code, err := s.trade.codeByID(s.db, *o.CouponID); err == nil {
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
			if err := s.trade.release(tx, *o.CouponID, userID, orderNo); err != nil {
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
