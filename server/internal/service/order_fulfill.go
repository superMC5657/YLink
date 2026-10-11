package service

import (
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// ---- 履约域 module：订阅开通/续期与佣金入账（支付后置调用） ----
// 渠道判定（余额直付 vs 在线渠道）收敛于本 module，
// 券×周期×渠道的 N×M 分支不再散落于收银台与佣金各处。

// isBalanceChannel 渠道判定唯一口：余额直付渠道。
func (m *orderTradeModule) isBalanceChannel(method string) bool {
	return method == "balance"
}

// commissionEligible 佣金准入唯一口：余额直付不写佣金，仅在线渠道写佣金。
func (m *orderTradeModule) commissionEligible(payMethod *string) bool {
	return payMethod == nil || *payMethod != "balance"
}

// apply 开通/续期唯一口（core-flows 2.1）。
func (m *orderTradeModule) apply(tx *gorm.DB, order *model.Order) error {
	user, err := m.repos.User.GetByID(tx, order.UserID)
	if err != nil {
		return err
	}
	plan, err := m.repos.Plan.GetByID(tx, order.PlanID)
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
	return m.repos.User.Save(tx, user)
}

// grant 佣金唯一口：下单支付后写佣金（确认中）。payAmount 为实际支付金额（余额支付为扣减前的应付额）。
// 余额支付不产生佣金（2026-08-12 需求变更）：仅在线渠道写佣金。
// 邀请人缺失不上抛（返回 nil 不阻塞支付主流程）；佣金缺失上抛语义由上层保持。
func (m *orderTradeModule) grant(tx *gorm.DB, order *model.Order, payAmount int64) error {
	if !m.commissionEligible(order.PayMethod) {
		return nil
	}
	user, err := m.repos.User.GetByID(tx, order.UserID)
	if err != nil {
		return err
	}
	if user.InviteByID == nil {
		return nil
	}
	inviter, err := m.repos.User.GetByID(tx, *user.InviteByID)
	if err != nil {
		return nil // 邀请人异常不阻塞支付
	}
	rate := commissionRateFor(tx, inviter.Role)
	amount := payAmount * int64(rate) / 100
	if amount <= 0 {
		return nil
	}
	return m.repos.Commission.Create(tx, &model.CommissionLog{
		InviteUserID: *user.InviteByID,
		FromUserID:   user.ID,
		OrderNo:      order.OrderNo,
		OrderAmount:  payAmount,
		Rate:         rate,
		Amount:       amount,
		Status:       model.CommissionPending,
	})
}

// applySubscription 存量薄 wrapper，实现收敛于 trade module。
func (s *OrderService) applySubscription(tx *gorm.DB, order *model.Order) error {
	return s.trade.apply(tx, order)
}

// grantCommission 存量薄 wrapper，实现收敛于 trade module。
func (s *OrderService) grantCommission(tx *gorm.DB, order *model.Order, payAmount int64) error {
	return s.trade.grant(tx, order, payAmount)
}
