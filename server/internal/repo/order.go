package repo

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"nanocloud/internal/model"
)

// OrderRepo 订单数据访问。
type OrderRepo struct{}

func (OrderRepo) Create(db *gorm.DB, o *model.Order) error { return db.Create(o).Error }

func (OrderRepo) GetByNo(db *gorm.DB, orderNo string) (*model.Order, error) {
	var o model.Order
	if err := db.Where("order_no = ?", orderNo).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetByNoForUpdate 行锁读取（并发回调/余额支付防重）。
func (OrderRepo) GetByNoForUpdate(db *gorm.DB, orderNo string) (*model.Order, error) {
	var o model.Order
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_no = ?", orderNo).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (OrderRepo) GetByNoAndUser(db *gorm.DB, orderNo string, userID int64) (*model.Order, error) {
	var o model.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, userID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetByIdempotencyKey 幂等键查订单（限本人）。
func (OrderRepo) GetByIdempotencyKey(db *gorm.DB, key string, userID int64) (*model.Order, error) {
	var o model.Order
	if err := db.Where("idempotency_key = ? AND user_id = ?", key, userID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// ListByUser 用户订单分页（status 可空）。
func (OrderRepo) ListByUser(db *gorm.DB, userID int64, status *int, page, pageSize int) ([]model.Order, int64, error) {
	var list []model.Order
	var total int64
	q := db.Model(&model.Order{}).Where("user_id = ?", userID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (OrderRepo) Save(db *gorm.DB, o *model.Order) error { return db.Save(o).Error }

// UpdateStatus 更新订单状态。
func (OrderRepo) UpdateStatus(db *gorm.DB, orderNo string, status int) error {
	return db.Model(&model.Order{}).Where("order_no = ?", orderNo).Update("status", status).Error
}

// UpdateStatusIfPending 仅在待支付状态下更新（防关单与支付回调竞态吞单）。
func (OrderRepo) UpdateStatusIfPending(db *gorm.DB, orderNo string, status int) error {
	return db.Model(&model.Order{}).
		Where("order_no = ? AND status = ?", orderNo, model.OrderPending).
		Update("status", status).Error
}

// ListPendingBefore 指定时间前仍未支付的订单（cron 关单）。
func (OrderRepo) ListPendingBefore(db *gorm.DB, before interface{}) ([]model.Order, error) {
	var list []model.Order
	err := db.Where("status = ? AND created_at < ?", model.OrderPending, before).Find(&list).Error
	return list, err
}

// PaymentRepo 支付单数据访问。
type PaymentRepo struct{}

func (PaymentRepo) Create(db *gorm.DB, p *model.Payment) error { return db.Create(p).Error }

func (PaymentRepo) GetByOrderAndMethod(db *gorm.DB, orderNo, method string) (*model.Payment, error) {
	var p model.Payment
	if err := db.Where("order_no = ? AND method = ?", orderNo, method).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (PaymentRepo) GetByTradeNo(db *gorm.DB, tradeNo string) (*model.Payment, error) {
	var p model.Payment
	if err := db.Where("trade_no = ?", tradeNo).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (PaymentRepo) Save(db *gorm.DB, p *model.Payment) error { return db.Save(p).Error }

// ListPending 全部待支付支付单（cron 查单）。
func (PaymentRepo) ListPending(db *gorm.DB) ([]model.Payment, error) {
	var list []model.Payment
	err := db.Where("status = ?", model.PayPending).Find(&list).Error
	return list, err
}

// ListPendingWithPayments 有待支付支付单的待支付订单（cron 查单）。
func (PaymentRepo) ListPendingOrderNos(db *gorm.DB) ([]string, error) {
	var nos []string
	err := db.Model(&model.Payment{}).
		Where("status = ?", model.PayPending).
		Distinct().Pluck("order_no", &nos).Error
	return nos, err
}

// CouponRepo 优惠券数据访问。
type CouponRepo struct{}

func (CouponRepo) GetByCode(db *gorm.DB, code string) (*model.Coupon, error) {
	var c model.Coupon
	if err := db.Where("code = ? AND is_enable = 1", code).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (CouponRepo) CountUsage(db *gorm.DB, couponID, userID int64) (int64, error) {
	var n int64
	err := db.Model(&model.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", couponID, userID).Count(&n).Error
	return n, err
}

func (CouponRepo) IncrUsed(db *gorm.DB, couponID int64) error {
	return db.Model(&model.Coupon{}).Where("id = ?", couponID).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

func (CouponRepo) RecordUsage(db *gorm.DB, couponID, userID int64, orderNo string) error {
	return db.Create(&model.CouponUsage{CouponID: couponID, UserID: userID, OrderNo: orderNo}).Error
}

// CommissionRepo 佣金数据访问。
type CommissionRepo struct{}

func (CommissionRepo) Create(db *gorm.DB, cl *model.CommissionLog) error { return db.Create(cl).Error }

func (CommissionRepo) Save(db *gorm.DB, cl *model.CommissionLog) error { return db.Save(cl).Error }

func (CommissionRepo) GetByOrderNo(db *gorm.DB, orderNo string) (*model.CommissionLog, error) {
	var cl model.CommissionLog
	if err := db.Where("order_no = ?", orderNo).First(&cl).Error; err != nil {
		return nil, err
	}
	return &cl, nil
}

func (CommissionRepo) ListByInvite(db *gorm.DB, inviteUserID int64, page, pageSize int) ([]model.CommissionLog, int64, error) {
	var list []model.CommissionLog
	var total int64
	q := db.Model(&model.CommissionLog{}).Where("invite_user_id = ?", inviteUserID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ListByInvite 佣金记录分页（按状态过滤）。
func (CommissionRepo) ListByInviteStatus(db *gorm.DB, inviteUserID int64, status int, page, pageSize int) ([]model.CommissionLog, int64, error) {
	var list []model.CommissionLog
	var total int64
	q := db.Model(&model.CommissionLog{}).Where("invite_user_id = ? AND status = ?", inviteUserID, status)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// SumAmount 佣金金额汇总（按状态）。
func (CommissionRepo) SumAmount(db *gorm.DB, inviteUserID int64, status int) (int64, error) {
	var sum int64
	err := db.Model(&model.CommissionLog{}).
		Where("invite_user_id = ? AND status = ?", inviteUserID, status).
		Select("COALESCE(SUM(amount),0)").Scan(&sum).Error
	return sum, err
}

// ListPendingConfirmBefore 确认中且超过确认期的佣金（cron 转已发放）。
func (CommissionRepo) ListPendingConfirmBefore(db *gorm.DB, paidBefore interface{}) ([]model.CommissionLog, error) {
	var list []model.CommissionLog
	err := db.Where("status = ? AND created_at < ?", model.CommissionPending, paidBefore).Find(&list).Error
	return list, err
}
