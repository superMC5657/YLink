package repo

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ylink/internal/model"
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
// 返回受影响行数：0 表示状态已非待支付（并发已处理）。
func (OrderRepo) UpdateStatusIfPending(db *gorm.DB, orderNo string, status int) (int64, error) {
	res := db.Model(&model.Order{}).
		Where("order_no = ? AND status = ?", orderNo, model.OrderPending).
		Update("status", status)
	return res.RowsAffected, res.Error
}

// ListPendingBefore 指定时间前仍未支付的订单（cron 关单）。
func (OrderRepo) ListPendingBefore(db *gorm.DB, before interface{}) ([]model.Order, error) {
	var list []model.Order
	err := db.Where("status = ? AND created_at < ?", model.OrderPending, before).Find(&list).Error
	return list, err
}

// CountByPlan 统计引用某套餐的订单数（删除套餐前置校验）。
func (OrderRepo) CountByPlan(db *gorm.DB, planID int64) (int64, error) {
	var n int64
	err := db.Model(&model.Order{}).Where("plan_id = ?", planID).Count(&n).Error
	return n, err
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

// ClosePendingByOrderNo 将该订单所有待支付支付单置为已关闭(订单超时关单/已取消时清理,
// 避免残留 pending 支付单被查单任务反复轮询)。返回受影响行数。
func (PaymentRepo) ClosePendingByOrderNo(db *gorm.DB, orderNo string) (int64, error) {
	res := db.Model(&model.Payment{}).
		Where("order_no = ? AND status = ?", orderNo, model.PayPending).
		Update("status", model.PayClosed)
	return res.RowsAffected, res.Error
}

// CouponRepo 优惠券数据访问。
type CouponRepo struct{}

func (CouponRepo) Create(db *gorm.DB, c *model.Coupon) error { return db.Create(c).Error }

func (CouponRepo) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&model.Coupon{}, id).Error
}

func (CouponRepo) GetByCode(db *gorm.DB, code string) (*model.Coupon, error) {
	var c model.Coupon
	if err := db.Where("code = ? AND is_enable = 1", code).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ListAvailable 查询当前对用户可见的优惠券：启用 + 生效期内 + 总限量未满。
// limit_per_user、套餐/周期匹配在 service 层按当前用户与请求参数过滤（避免此处泄露用户维度 SQL 复杂度）。
func (CouponRepo) ListAvailable(db *gorm.DB, now time.Time) ([]model.Coupon, error) {
	var list []model.Coupon
	err := db.Where("is_enable = 1").
		Where("(total_limit = 0 OR used_count < total_limit)").
		Where("(started_at IS NULL OR started_at <= ?)", now).
		Where("(ended_at IS NULL OR ended_at >= ?)", now).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

func (CouponRepo) CountUsage(db *gorm.DB, couponID, userID int64) (int64, error) {
	var n int64
	err := db.Model(&model.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", couponID, userID).Count(&n).Error
	return n, err
}

// CountUsageLocked 行锁统计用户使用次数（配合 Occupy 串行化并发下单，防 limit_per_user 超限）。
func (CouponRepo) CountUsageLocked(db *gorm.DB, couponID, userID int64) (int64, error) {
	var list []model.CouponUsage
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("coupon_id = ? AND user_id = ?", couponID, userID).Find(&list).Error
	return int64(len(list)), err
}

// Occupy 原子占用优惠券：仅当未超总限量时 used_count+1（防 TOCTOU 超发）。
// 返回 false 表示已达限量。
func (CouponRepo) Occupy(db *gorm.DB, couponID int64) (bool, error) {
	res := db.Model(&model.Coupon{}).
		Where("id = ? AND (total_limit = 0 OR used_count < total_limit)", couponID).
		UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Release 回退占用（取消/退款时），防止负计数。
func (CouponRepo) Release(db *gorm.DB, couponID int64) error {
	return db.Model(&model.Coupon{}).Where("id = ? AND used_count > 0", couponID).
		UpdateColumn("used_count", gorm.Expr("used_count - 1")).Error
}

func (CouponRepo) DeleteUsage(db *gorm.DB, couponID, userID int64, orderNo string) error {
	return db.Where("coupon_id = ? AND user_id = ? AND order_no = ?", couponID, userID, orderNo).
		Delete(&model.CouponUsage{}).Error
}

func (CouponRepo) RecordUsage(db *gorm.DB, couponID, userID int64, orderNo string) error {
	return db.Create(&model.CouponUsage{CouponID: couponID, UserID: userID, OrderNo: orderNo}).Error
}

// CommissionRepo 佣金数据访问。
type CommissionRepo struct{}

func (CommissionRepo) Create(db *gorm.DB, cl *model.CommissionLog) error { return db.Create(cl).Error }

func (CommissionRepo) Save(db *gorm.DB, cl *model.CommissionLog) error { return db.Save(cl).Error }

// UpdateStatusIfPending 仅在确认中（status=0）时置为已发放，防止与退款撤销竞态覆盖。
func (CommissionRepo) UpdateStatusIfPending(db *gorm.DB, id int64) (int64, error) {
	res := db.Model(&model.CommissionLog{}).
		Where("id = ? AND status = ?", id, model.CommissionPending).
		Update("status", model.CommissionGranted)
	return res.RowsAffected, res.Error
}

func (CommissionRepo) GetByOrderNo(db *gorm.DB, orderNo string) (*model.CommissionLog, error) {
	var cl model.CommissionLog
	if err := db.Where("order_no = ?", orderNo).First(&cl).Error; err != nil {
		return nil, err
	}
	return &cl, nil
}

// ListByOrderNos 按订单号批量查佣金记录(管理端订单列表展示佣金用,一个订单至多一条)。
func (CommissionRepo) ListByOrderNos(db *gorm.DB, orderNos []string) ([]model.CommissionLog, error) {
	if len(orderNos) == 0 {
		return nil, nil
	}
	var list []model.CommissionLog
	err := db.Where("order_no IN ?", orderNos).Find(&list).Error
	return list, err
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
