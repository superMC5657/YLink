package repo

import (
	"gorm.io/gorm"

	"ylink/internal/model"
)

// AgentApplyRepo 代理商申请数据访问。
type AgentApplyRepo struct{}

func (AgentApplyRepo) GetByUser(db *gorm.DB, userID int64) (*model.AgentApply, error) {
	var a model.AgentApply
	if err := db.Where("user_id = ?", userID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (AgentApplyRepo) Create(db *gorm.DB, a *model.AgentApply) error { return db.Create(a).Error }

func (AgentApplyRepo) Save(db *gorm.DB, a *model.AgentApply) error { return db.Save(a).Error }

// CountValidInvited 统计邀请的有效用户数：
// invite_by_id=我 且（有已完成订单 或 注册满 3 天未封禁）。
func (UserRepo) CountValidInvited(db *gorm.DB, inviterID int64) (int64, error) {
	var n int64
	err := db.Model(&model.User{}).
		Where("invite_by_id = ? AND (is_banned = 0 AND created_at < DATE_SUB(NOW(), INTERVAL 3 DAY) "+
			"OR EXISTS (SELECT 1 FROM orders o WHERE o.user_id = users.id AND o.status = 1))", inviterID).
		Count(&n).Error
	return n, err
}
