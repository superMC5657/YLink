package repo

import (
	"encoding/json"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ylink-backend/internal/model"
)

// SettingRepo 站点配置数据访问。
type SettingRepo struct{}

// Get 读取单个配置项（JSON 字符串）。
func (SettingRepo) Get(db *gorm.DB, key string) (string, error) {
	var s model.Setting
	if err := db.Select("value").Where("\"key\" = ?", key).First(&s).Error; err != nil {
		return "", err
	}
	return s.Value, nil
}

// GetJSON 读取并反序列化配置项。
func (r SettingRepo) GetJSON(db *gorm.DB, key string, out any) error {
	raw, err := r.Get(db, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}

// Set 写入配置项。
func (SettingRepo) Set(db *gorm.DB, key, value string) error {
	s := &model.Setting{Key: key, Value: value}
	return db.Save(s).Error
}

// UserRepo 用户数据访问。
type UserRepo struct{}

func (UserRepo) GetByID(db *gorm.DB, id int64) (*model.User, error) {
	var u model.User
	if err := db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByIDForUpdate 行锁读取（余额扣减并发安全）。
func (UserRepo) GetByIDForUpdate(db *gorm.DB, id int64) (*model.User, error) {
	var u model.User
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (UserRepo) GetByEmail(db *gorm.DB, email string) (*model.User, error) {
	var u model.User
	if err := db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (UserRepo) GetBySubToken(db *gorm.DB, token string) (*model.User, error) {
	var u model.User
	if err := db.Where("sub_token = ?", token).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (UserRepo) Create(db *gorm.DB, u *model.User) error { return db.Create(u).Error }

// Update 按 ID 更新非零字段。
func (UserRepo) Update(db *gorm.DB, u *model.User) error {
	return db.Model(u).Updates(u).Error
}

// UpdateProfile 更新通知设置（map 更新，支持 false 值持久化）。
func (UserRepo) UpdateProfile(db *gorm.DB, id int64, remindExpire, remindTraffic *bool) error {
	updates := map[string]any{}
	if remindExpire != nil {
		updates["remind_expire"] = *remindExpire
	}
	if remindTraffic != nil {
		updates["remind_traffic"] = *remindTraffic
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

// Save 全字段保存（用于余额等可能为 0 的字段）。
func (UserRepo) Save(db *gorm.DB, u *model.User) error { return db.Save(u).Error }

// ExistsEmail 邮箱是否已存在。
func (UserRepo) ExistsEmail(db *gorm.DB, email string) (bool, error) {
	var n int64
	err := db.Model(&model.User{}).Where("email = ?", email).Count(&n).Error
	return n > 0, err
}

// CountInvitedBy 统计某用户邀请的人数。
func (UserRepo) CountInvitedBy(db *gorm.DB, inviterID int64) (int64, error) {
	var n int64
	err := db.Model(&model.User{}).Where("invite_by_id = ?", inviterID).Count(&n).Error
	return n, err
}

// CountByStatus 统计用户订单数（按状态）。
func (UserRepo) CountOrdersByStatus(db *gorm.DB, uid int64, status int) (int64, error) {
	var n int64
	err := db.Model(&model.Order{}).Where("user_id = ? AND status = ?", uid, status).Count(&n).Error
	return n, err
}

// CountOpenTickets 统计用户未关闭工单数。
func (UserRepo) CountOpenTickets(db *gorm.DB, uid int64) (int64, error) {
	var n int64
	err := db.Model(&model.Ticket{}).Where("user_id = ? AND status != ?", uid, 2).Count(&n).Error
	return n, err
}

// InviteCodeRepo 邀请码数据访问。
type InviteCodeRepo struct{}

func (InviteCodeRepo) GetByCode(db *gorm.DB, code string) (*model.InviteCode, error) {
	var ic model.InviteCode
	if err := db.Where("code = ? AND status = 1", code).First(&ic).Error; err != nil {
		return nil, err
	}
	return &ic, nil
}

func (InviteCodeRepo) Create(db *gorm.DB, ic *model.InviteCode) error { return db.Create(ic).Error }

// DeleteByUser 删除指定用户自己的邀请码,返回受影响行数。
func (InviteCodeRepo) DeleteByUser(db *gorm.DB, uid int64, code string) (int64, error) {
	res := db.Where("user_id = ? AND code = ?", uid, code).Delete(&model.InviteCode{})
	return res.RowsAffected, res.Error
}

func (InviteCodeRepo) ListByUser(db *gorm.DB, uid int64) ([]model.InviteCode, error) {
	var list []model.InviteCode
	err := db.Where("user_id = ?", uid).Order("id DESC").Find(&list).Error
	return list, err
}

func (InviteCodeRepo) CountByUser(db *gorm.DB, uid int64) (int64, error) {
	var n int64
	err := db.Model(&model.InviteCode{}).Where("user_id = ?", uid).Count(&n).Error
	return n, err
}

// IncrUsedCount 使用次数 +1。
func (InviteCodeRepo) IncrUsedCount(db *gorm.DB, id int64) error {
	return db.Model(&model.InviteCode{}).Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

// ---- Repos 聚合（见 content.go） ----

// WithTx 在事务内执行 fn。
func WithTx(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error { return fn(tx) })
}
