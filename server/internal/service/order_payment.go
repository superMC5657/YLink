package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/pkg/payment"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

// ---- 收银台与支付回调(Checkout / HandleNotify / 回执邮件) ----
// 渠道分支与履约后置经由 trade module 的 seam 收口，不再散落于各处。

// Checkout POST /orders/{no}/checkout 拉起支付或余额直付。
func (s *OrderService) Checkout(ctx context.Context, userID int64, orderNo, method string) (*model.CheckoutResp, error) {
	order, err := s.repos.Order.GetByNoAndUser(s.db, orderNo, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if order.Status != model.OrderPending {
		return nil, errs.ErrOrderStatus
	}

	if s.trade.isBalanceChannel(method) {
		return s.checkoutBalance(ctx, order)
	}

	driver := payment.Get(method)
	if driver == nil {
		return nil, errs.ErrPayMethod
	}
	// 30 分钟内同用户同渠道重复 checkout 返回原支付单（键含 method，避免换渠道拿到旧 URL）
	cacheKey := redispkg.Key("order", "paying", fmt.Sprint(userID), orderNo, method)
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
	res := &model.CheckoutResp{Type: result.Type, Content: &result.Content, ExpireIn: result.ExpireIn}
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
		payAmount := locked.PayAmount
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
		if err := s.trade.apply(tx, locked); err != nil {
			return err
		}
		return s.trade.grant(tx, locked, payAmount)
	})
	if err != nil {
		return nil, err
	}
	s.sendReceiptMailAsync(order.OrderNo)
	return &model.CheckoutResp{Type: "paid", Content: nil, ExpireIn: 0}, nil
}

// HandleNotify 处理异步通知：验签在 handler 层完成。
// 每条验签通过的回调原文落 payments.notify_payload（对账证据）；
// 非成功终态仅落库后 ack（不做账务、不报错，由 handler 回 success 防网关重试风暴）。
func (s *OrderService) HandleNotify(ctx context.Context, method string, nr *payment.NotifyResult) error {
	if nr == nil || nr.OrderNo == "" {
		return errors.New("notify missing order")
	}
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		// 读支付单(先于订单,非成功终态也要落 payload)
		paymentRec, err := s.repos.Payment.GetByOrderAndMethod(tx, nr.OrderNo, method)
		if err != nil {
			return err
		}
		paymentRec.NotifyPayload = nr.Raw
		if !nr.Paid || nr.TradeNo == "" {
			return s.repos.Payment.Save(tx, paymentRec)
		}
		order, err := s.repos.Order.GetByNoForUpdate(tx, nr.OrderNo)
		if err != nil {
			return err
		}
		if order.Status != model.OrderPending {
			// 已处理(幂等):仍保存回调原文,补充对账证据
			return s.repos.Payment.Save(tx, paymentRec)
		}
		if order.PayAmount != nr.Amount {
			return errors.New("amount mismatch")
		}
		// 支付单落成功（trade_no 唯一约束兜底并发）
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
		if err := s.trade.apply(tx, order); err != nil {
			return err
		}
		return s.trade.grant(tx, order, order.PayAmount)
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
