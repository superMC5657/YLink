package service

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// ---- 优惠券域 module：试算 / 可用列表 / 校验与占用释放 ----
// 计价 scope（plan_ids / valid_periods）解析与优惠计算收敛于本 module，
// 券×周期×渠道的 N×M 分支不再散落于各调用方。

// priceOf 取周期价格；NULL=不支持。trade module 内唯一的周期取价口。
func (m *orderTradeModule) priceOf(p *model.Plan, period string) *int64 {
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

// planPrices 套餐展示价格。trade module 内唯一的展示取价口。
func (m *orderTradeModule) planPrices(p *model.Plan) model.PlanPrices {
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

// scopeOf 券适用 scope 唯一解析口：plan_ids / valid_periods JSON 各解析一次。
// 空（nil / 解析失败 / 空数组）= 不限，与展示文案「空=全部」一致。
func (m *orderTradeModule) scopeOf(c *model.Coupon) (planIDs []int64, periods []string) {
	if c.PlanIDs != nil {
		_ = json.Unmarshal([]byte(*c.PlanIDs), &planIDs)
	}
	if c.ValidPeriods != nil {
		_ = json.Unmarshal([]byte(*c.ValidPeriods), &periods)
	}
	return planIDs, periods
}

// planAllowed 套餐是否在券 scope 内（空=全部）。
func (m *orderTradeModule) planAllowed(planIDs []int64, planID int64) bool {
	return len(planIDs) == 0 || containsInt64(planIDs, planID)
}

// periodAllowed 周期是否在券 scope 内（空=不限）。
func (m *orderTradeModule) periodAllowed(periods []string, period string) bool {
	return len(periods) == 0 || containsStr(periods, period)
}

// discountFor 券优惠唯一计算口：固定金额与百分比收敛一处并封顶。
func (m *orderTradeModule) discountFor(c *model.Coupon, amount int64) int64 {
	var discount int64
	if c.Type == 1 { // 固定金额（Value 为分，直接与订单金额分比较）
		discount = min64(c.Value, amount)
	} else { // 百分比（Value 为「百分比×100」的分存储：10% 券存 1000 分 → /10000 还原百分比）
		discount = amount * c.Value / 10000
		if discount > amount {
			discount = amount
		}
	}
	return discount
}

// validate 券校验唯一口：有效期 + scope + 门槛 + 限量 + 每人限次 + 优惠计算。
// db 由调用方传入（下单传事务内 tx，试算传读 db）。
func (m *orderTradeModule) validate(db *gorm.DB, userID int64, code string, plan *model.Plan, period string, amount int64) (*model.Coupon, int64, error) {
	coupon, err := m.coupons.GetByCode(db, code)
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
	planIDs, periods := m.scopeOf(coupon)
	if !m.planAllowed(planIDs, plan.ID) {
		return nil, 0, errs.ErrCoupon
	}
	if !m.periodAllowed(periods, period) {
		return nil, 0, errs.ErrCoupon
	}
	if amount < coupon.MinSpend {
		return nil, 0, errs.ErrCoupon
	}
	if coupon.TotalLimit > 0 && coupon.UsedCount >= coupon.TotalLimit {
		return nil, 0, errs.ErrCoupon
	}
	if coupon.LimitPerUser > 0 {
		n, err := m.coupons.CountUsage(db, coupon.ID, userID)
		if err != nil {
			// 查询失败保守拒绝,不跳过限用校验
			return nil, 0, errs.ErrCoupon
		}
		if n >= int64(coupon.LimitPerUser) {
			return nil, 0, errs.ErrCoupon
		}
	}
	return coupon, m.discountFor(coupon, amount), nil
}

// available 可用券列表唯一口：SQL 粗筛 + 每人限次 + scope 细筛 + 展示组装。
func (m *orderTradeModule) available(db *gorm.DB, userID, planID int64, period string) ([]model.CouponItem, error) {
	now := time.Now()
	list, err := m.coupons.ListAvailable(db, now)
	if err != nil {
		return nil, err
	}
	out := make([]model.CouponItem, 0, len(list))
	for _, c := range list {
		// 每人限用（limit=0 不限）
		if c.LimitPerUser > 0 {
			n, err := m.coupons.CountUsage(db, c.ID, userID)
			// 查询失败保守过滤,不展示该券
			if err != nil {
				continue
			}
			if n >= int64(c.LimitPerUser) {
				continue
			}
		}
		planIDs, periods := m.scopeOf(&c)
		// 适用套餐（可选过滤；空=全部）
		if planID > 0 && !m.planAllowed(planIDs, planID) {
			continue
		}
		// 适用周期（可选过滤；空=不限）
		if period != "" && !m.periodAllowed(periods, period) {
			continue
		}
		item := model.CouponItem{
			Code: c.Code, Type: c.Type,
			Value: model.FenToYuan(c.Value), MinSpend: model.FenToYuan(c.MinSpend),
			ValidPeriods: []string{}, PlanIDs: []int64{},
			StartedAt: c.StartedAt, EndedAt: c.EndedAt,
		}
		item.ValidPeriods = append(item.ValidPeriods, periods...)
		item.PlanIDs = append(item.PlanIDs, planIDs...)
		out = append(out, item)
	}
	return out, nil
}

// claim 下单占用唯一口：校验 → 原子占用 → 每人限次锁定复核 → 记流水。
// 调用方须在同一事务内随后建单；失败即超发或限用，调用方回滚即可。
func (m *orderTradeModule) claim(tx *gorm.DB, userID int64, code string, plan *model.Plan, period string, amount int64, orderNo string) (*model.Coupon, int64, error) {
	coupon, discount, err := m.validate(tx, userID, code, plan, period, amount)
	if err != nil {
		return nil, 0, err
	}
	// 原子占用总限量（条件更新），失败即超发 → 12001
	ok, err := m.coupons.Occupy(tx, coupon.ID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, errs.ErrCoupon
	}
	// 每用户限次：Occupy 已锁 coupon 行串行化并发下单，再用锁定读校验
	if coupon.LimitPerUser > 0 {
		n, err := m.coupons.CountUsageLocked(tx, coupon.ID, userID)
		if err != nil {
			return nil, 0, err
		}
		if n >= int64(coupon.LimitPerUser) {
			return nil, 0, errs.ErrCoupon
		}
	}
	if err := m.coupons.RecordUsage(tx, coupon.ID, userID, orderNo); err != nil {
		return nil, 0, err
	}
	return coupon, discount, nil
}

// release 占用释放唯一口：回退计数 + 删流水（取消/关闭/退款/超时关单共用）。
func (m *orderTradeModule) release(db *gorm.DB, couponID, userID int64, orderNo string) error {
	if err := m.coupons.Release(db, couponID); err != nil {
		return err
	}
	return m.coupons.DeleteUsage(db, couponID, userID, orderNo)
}

// codeByID 订单详情券码回填口：经由 seam 触达，不直查。
func (m *orderTradeModule) codeByID(db *gorm.DB, id int64) (string, error) {
	return m.coupons.CodeByID(db, id)
}

// fetchCouponCode 券码直读的唯一实现：adapter 与存量 wrapper 共用。
func fetchCouponCode(db *gorm.DB, id int64) (string, error) {
	var c struct {
		Code string
	}
	err := db.Model(&model.Coupon{}).Select("code").Where("id = ?", id).Scan(&c).Error
	return c.Code, err
}

// CouponCheck POST /coupons/check 纯试算。
func (s *OrderService) CouponCheck(ctx context.Context, userID int64, req *model.CouponCheckReq) (*model.CouponCheckResp, error) {
	plan, err := s.repos.Plan.GetByID(s.db, req.PlanID)
	if err != nil || !plan.IsShow {
		return nil, errs.ErrPlanNotAvailable
	}
	price := s.trade.priceOf(plan, req.Period)
	if price == nil {
		return nil, errs.ErrPlanPeriod
	}
	_, discount, err := s.trade.validate(s.db, userID, req.Code, plan, req.Period, *price)
	if err != nil {
		return nil, err
	}
	return &model.CouponCheckResp{
		Valid:          true,
		DiscountAmount: model.FenToYuan(discount),
		PayAmount:      model.FenToYuan(*price - discount),
	}, nil
}

// AvailableCoupons GET /coupons/available：当前用户可用的优惠券列表。
// planID/period 为可选过滤（下单弹窗按选中套餐+周期展示）；不传则返回全部可用券。
// 过滤：启用 + 生效期内 + 总限量未满（SQL）；每人限用未满 + 适用套餐/周期（内存按当前用户）。
func (s *OrderService) AvailableCoupons(ctx context.Context, userID, planID int64, period string) ([]model.CouponItem, error) {
	return s.trade.available(s.db, userID, planID, period)
}

// validateCoupon 校验优惠券并返回优惠金额（分）。存量薄 wrapper，实现收敛于 trade module。
func (s *OrderService) validateCoupon(db *gorm.DB, userID int64, code string, plan *model.Plan, period string, amount int64) (*model.Coupon, int64, error) {
	return s.trade.validate(db, userID, code, plan, period, amount)
}

// couponCode 存量薄 wrapper：经由集中实现回填券码，供详情展示用。
func couponCode(db *gorm.DB, id int64) (string, error) {
	return fetchCouponCode(db, id)
}

// releaseCoupon 存量薄 wrapper：实现收敛于 trade module 的 release。
// 存量调用方（管理关单/退款、超时关单）未经理门面注入，故以直连 adapter 承接同一释放序列。
func releaseCoupon(db *gorm.DB, couponID, userID int64, orderNo string) error {
	m := &orderTradeModule{db: db, repos: &repo.Repos{}, coupons: orderCouponRepoAdapter{repos: &repo.Repos{}}}
	return m.release(db, couponID, userID, orderNo)
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
