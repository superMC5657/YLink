package service

import (
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// ---- 履约:订阅开通/续期与佣金入账(支付成功后调用) ----

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

// grantCommission 下单支付成功后写佣金（确认中）。payAmount 为实际支付金额（余额支付为扣减前的应付额）。
// 余额支付不产生佣金（2026-08-12 需求变更）：仅在线支付渠道(epay 等)写佣金。
func (s *OrderService) grantCommission(tx *gorm.DB, order *model.Order, payAmount int64) error {
	if order.PayMethod != nil && *order.PayMethod == "balance" {
		return nil
	}
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
	rate := commissionRateFor(tx, inviter.Role)
	amount := payAmount * int64(rate) / 100
	if amount <= 0 {
		return nil
	}
	return s.repos.Commission.Create(tx, &model.CommissionLog{
		InviteUserID: *user.InviteByID,
		FromUserID:   user.ID,
		OrderNo:      order.OrderNo,
		OrderAmount:  payAmount,
		Rate:         rate,
		Amount:       amount,
		Status:       model.CommissionPending,
	})
}
