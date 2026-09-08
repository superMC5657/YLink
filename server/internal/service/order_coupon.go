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

// ---- 优惠券(CouponCheck / AvailableCoupons / 校验与占用回退) ----

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

// AvailableCoupons GET /coupons/available：当前用户可用的优惠券列表。
// planID/period 为可选过滤（下单弹窗按选中套餐+周期展示）；不传则返回全部可用券。
// 过滤：启用 + 生效期内 + 总限量未满（SQL）；每人限用未满 + 适用套餐/周期（内存按当前用户）。
func (s *OrderService) AvailableCoupons(ctx context.Context, userID, planID int64, period string) ([]model.CouponItem, error) {
	now := time.Now()
	list, err := s.repos.Coupon.ListAvailable(s.db, now)
	if err != nil {
		return nil, err
	}
	out := make([]model.CouponItem, 0, len(list))
	for _, c := range list {
		// 每人限用（limit=0 不限）
		if c.LimitPerUser > 0 {
			n, err := s.repos.Coupon.CountUsage(s.db, c.ID, userID)
			// 查询失败保守过滤,不展示该券
			if err != nil {
				continue
			}
			if n >= int64(c.LimitPerUser) {
				continue
			}
		}
		// 适用套餐（可选过滤；券 plan_ids 为空=全部）
		if planID > 0 && c.PlanIDs != nil {
			var ids []int64
			if json.Unmarshal([]byte(*c.PlanIDs), &ids) == nil && len(ids) > 0 && !containsInt64(ids, planID) {
				continue
			}
		}
		// 适用周期（可选过滤；券 valid_periods 为空=全部）
		if period != "" && c.ValidPeriods != nil {
			var periods []string
			if json.Unmarshal([]byte(*c.ValidPeriods), &periods) == nil && len(periods) > 0 && !containsStr(periods, period) {
				continue
			}
		}
		item := model.CouponItem{
			Code: c.Code, Type: c.Type,
			Value: model.FenToYuan(c.Value), MinSpend: model.FenToYuan(c.MinSpend),
			ValidPeriods: []string{}, PlanIDs: []int64{},
			StartedAt: c.StartedAt, EndedAt: c.EndedAt,
		}
		if c.ValidPeriods != nil {
			_ = json.Unmarshal([]byte(*c.ValidPeriods), &item.ValidPeriods)
		}
		if c.PlanIDs != nil {
			_ = json.Unmarshal([]byte(*c.PlanIDs), &item.PlanIDs)
		}
		out = append(out, item)
	}
	return out, nil
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
		if err != nil {
			// 查询失败保守拒绝,不跳过限用校验
			return nil, 0, errs.ErrCoupon
		}
		if n >= int64(coupon.LimitPerUser) {
			return nil, 0, errs.ErrCoupon
		}
	}
	// 计算优惠
	var discount int64
	if coupon.Type == 1 { // 固定金额（Value 为分，直接与订单金额分比较）
		discount = min64(coupon.Value, amount)
	} else { // 百分比（Value 为「百分比×100」的分存储：10% 券存 1000 分 → /10000 还原百分比）
		discount = amount * coupon.Value / 10000
		if discount > amount {
			discount = amount
		}
	}
	return coupon, discount, nil
}

func couponCode(db *gorm.DB, id int64) (string, error) {
	var c struct {
		Code string
	}
	err := db.Model(&model.Coupon{}).Select("code").Where("id = ?", id).Scan(&c).Error
	return c.Code, err
}

// releaseCoupon 回退优惠券占用（取消/关闭/退款/超时关单共用，避免四处重复）。
func releaseCoupon(db *gorm.DB, couponID, userID int64, orderNo string) error {
	r := repo.CouponRepo{}
	if err := r.Release(db, couponID); err != nil {
		return err
	}
	return r.DeleteUsage(db, couponID, userID, orderNo)
}
