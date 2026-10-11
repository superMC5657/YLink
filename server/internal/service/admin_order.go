package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
	"ylink-backend/internal/repo"
)

// 订单域 module：列表/关闭/退款 + 订阅收回。

// ListOrders GET /admin/orders。
func (m *adminOrderModule) ListOrders(ctx context.Context, status *int, page, pageSize int) ([]model.AdminOrderItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := m.repos.Order.ListByPage(m.db, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	emails := m.audit.emailsOf(userIDsOf(list))
	planNames := map[int64]string{}
	// 批量查佣金:order_no → amount(一个订单至多一条佣金记录,uniqueIndex)。
	// 查询失败必须上抛,否则接口会以成功响应返回 commission_amount 全 null,
	// 把数据缺失误表现为「无佣金」(review-0.5.0 P2)。
	commissionByOrder := map[string]float64{}
	comms, err := m.repos.Commission.ListByOrderNos(m.db, orderNosOf(list))
	if err != nil {
		return nil, 0, err
	}
	for _, cl := range comms {
		commissionByOrder[cl.OrderNo] = model.FenToYuan(cl.Amount)
	}
	out := make([]model.AdminOrderItem, 0, len(list))
	for _, o := range list {
		name, ok := planNames[o.PlanID]
		if !ok {
			if p, err := m.repos.Plan.GetByID(m.db, o.PlanID); err == nil {
				name = p.Name
				planNames[o.PlanID] = name
			}
		}
		var commission *float64
		if v, has := commissionByOrder[o.OrderNo]; has {
			commission = &v
		}
		out = append(out, model.AdminOrderItem{
			OrderNo: o.OrderNo, UserID: o.UserID, UserEmail: emails[o.UserID], PlanName: name,
			Period: o.Period, Amount: model.FenToYuan(o.Amount), DiscountAmount: model.FenToYuan(o.DiscountAmount),
			BalanceUsed: model.FenToYuan(o.BalanceUsed), PayAmount: model.FenToYuan(o.PayAmount),
			CommissionAmount: commission,
			Status:           o.Status, PayMethod: o.PayMethod, PaidAt: o.PaidAt, CreatedAt: o.CreatedAt,
		})
	}
	return out, total, nil
}

// CloseOrder POST /admin/orders/{no}/close：管理员关闭待支付订单（回退优惠券占用 + 审计）。
func (m *adminOrderModule) CloseOrder(ctx context.Context, adminID int64, orderNo, remark, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		order, err := m.repos.Order.GetByNoForUpdate(tx, orderNo)
		if err != nil {
			return errs.ErrNotFound
		}
		if order.Status != model.OrderPending {
			return errs.ErrOrderStatus
		}
		// 条件更新（防与支付回调竞态）：影响行数为 0 说明已被并发完成/取消
		affected, err := m.repos.Order.UpdateStatusIfPending(tx, orderNo, model.OrderCanceled)
		if err != nil {
			return err
		}
		if affected == 0 {
			return errs.ErrOrderStatus
		}
		if order.CouponID != nil {
			if err := releaseCoupon(tx, *order.CouponID, order.UserID, orderNo); err != nil {
				return err
			}
		}
		return m.audit.write(tx, adminID, "close_order", orderNo, ip, map[string]any{
			"remark": sanitize.Text(remark),
		})
	})
}

// Refund POST /admin/orders/{no}/refund：退款 + 收回订阅 + 佣金回滚 + 优惠券回退 + 审计。
// 退款仅在系统内原路处理余额部分;现金部分需管理员线下在网关侧操作退款,
// 并将网关退款单号经 gatewayRefundID 回填审计(系统不做网关退款调用)。
func (m *adminOrderModule) Refund(ctx context.Context, adminID int64, orderNo, remark, gatewayRefundID, ip string) error {
	return repo.WithTx(m.db, func(tx *gorm.DB) error {
		order, err := m.repos.Order.GetByNoForUpdate(tx, orderNo)
		if err != nil {
			return errs.ErrNotFound
		}
		if order.Status != model.OrderCompleted {
			return errs.ErrOrderStatus
		}
		// 余额支付退回余额
		if order.BalanceUsed > 0 {
			user, err := m.repos.User.GetByIDForUpdate(tx, order.UserID)
			if err != nil {
				return err
			}
			user.Balance += order.BalanceUsed
			if err := m.repos.User.Save(tx, user); err != nil {
				return err
			}
		}
		// 优惠券占用回退
		if order.CouponID != nil {
			if err := releaseCoupon(tx, *order.CouponID, order.UserID, order.OrderNo); err != nil {
				return err
			}
		}
		// 收回订阅：退款后用户不应继续享有该订单对应的订阅服务
		if err := m.revokeSubscriptionOnRefund(tx, order); err != nil {
			return err
		}
		order.Status = model.OrderRefunded
		if err := m.repos.Order.Save(tx, order); err != nil {
			return err
		}
		// 佣金回滚（core-flows 第 4 节）
		if cl, err := m.repos.Commission.GetByOrderNo(tx, order.OrderNo); err == nil {
			switch cl.Status {
			case model.CommissionPending:
				cl.Status = model.CommissionRevoked
				if err := m.repos.Commission.Save(tx, cl); err != nil {
					return err
				}
			case model.CommissionGranted:
				inviter, err := m.repos.User.GetByIDForUpdate(tx, cl.InviteUserID)
				if err != nil {
					return err
				}
				inviter.CommissionBalance -= cl.Amount // 可为负，记录审计
				if err := m.repos.User.Save(tx, inviter); err != nil {
					return err
				}
				cl.Status = model.CommissionRevoked
				if err := m.repos.Commission.Save(tx, cl); err != nil {
					return err
				}
			}
		}
		return m.audit.write(tx, adminID, "refund", orderNo, ip, map[string]any{
			"pay_amount": order.PayAmount, "remark": sanitize.Text(remark),
			"gateway_refund_id": sanitize.Text(gatewayRefundID),
		})
	})
}

// revokeSubscriptionOnRefund 退款收回订阅（core-flows 2.1 的逆操作）：
//   - onetime：仅扣回本次叠加的流量（下限 0），不动到期时间
//   - 周期套餐：若用户当前生效订阅正是该订单套餐（plan_id 相同且未过期），清除订阅
//     （plan_id/expired_at 置空、流量清零、限速/设备数清空）
//
// 说明：系统未保存订单开通前的订阅快照，因此「同套餐续期叠加」场景退款后会整体清除订阅
// （用户原有未过期订阅一并收回）；管理员对续期叠加订单退款时需知悉此行为。
func (m *adminOrderModule) revokeSubscriptionOnRefund(tx *gorm.DB, order *model.Order) error {
	user, err := m.repos.User.GetByIDForUpdate(tx, order.UserID)
	if err != nil {
		return err
	}
	plan, err := m.repos.Plan.GetByID(tx, order.PlanID)
	if err != nil {
		return err
	}
	trafficBytes := int64(plan.TrafficGB) * 1024 * 1024 * 1024

	if order.Period == "onetime" {
		// 一次性流量包：扣回本次流量
		user.TransferEnable -= trafficBytes
		if user.TransferEnable < 0 {
			user.TransferEnable = 0
		}
		return m.repos.User.Save(tx, user)
	}
	// 周期订阅：仅当当前生效订阅正是该订单套餐时清除
	if user.PlanID != nil && *user.PlanID == order.PlanID &&
		user.ExpiredAt != nil && user.ExpiredAt.After(time.Now()) {
		user.PlanID = nil
		user.ExpiredAt = nil
		user.TransferEnable = 0
		user.U = 0
		user.D = 0
		user.SpeedLimit = nil
		user.DeviceLimit = nil
		return m.repos.User.Save(tx, user)
	}
	return nil
}

func userIDsOf(list []model.Order) []int64 {
	ids := make([]int64, 0, len(list))
	for _, o := range list {
		ids = append(ids, o.UserID)
	}
	return ids
}

func orderNosOf(list []model.Order) []string {
	nos := make([]string, 0, len(list))
	for _, o := range list {
		nos = append(nos, o.OrderNo)
	}
	return nos
}
