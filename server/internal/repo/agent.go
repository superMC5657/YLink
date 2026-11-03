package repo

import (
	"gorm.io/gorm"

	"ylink-backend/internal/model"
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

// validInvitedPred 有效用户判定谓词（CountValidInvited 两处共用，改一处即两处一致）：
// 注册满 validDays 天未封禁 或 有已完成订单。
const validInvitedPred = "(is_banned = false AND created_at < now() - (? * interval '1 day') " +
	"OR EXISTS (SELECT 1 FROM orders o WHERE o.user_id = users.id AND o.status = 1))"

// CountValidInvited 统计邀请的有效用户数：invite_by_id=我 且命中 validInvitedPred。
func (UserRepo) CountValidInvited(db *gorm.DB, inviterID int64, validDays int) (int64, error) {
	var n int64
	err := db.Model(&model.User{}).
		Where("invite_by_id = ? AND "+validInvitedPred, inviterID, validDays).
		Count(&n).Error
	return n, err
}

// CountValidInvitedBatch 批量统计各邀请人的有效用户数（同一 validInvitedPred），
// 照 emailsOf 批量形状收敛逐人 Count 的 N+1。map 缺失的邀请人即 0。
func (UserRepo) CountValidInvitedBatch(db *gorm.DB, inviterIDs []int64, validDays int) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(inviterIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		InviterID int64 `gorm:"column:invite_by_id"`
		Cnt       int64 `gorm:"column:c"`
	}
	err := db.Model(&model.User{}).
		Select("invite_by_id, COUNT(*) AS c").
		Where("invite_by_id IN ? AND "+validInvitedPred, inviterIDs, validDays).
		Group("invite_by_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.InviterID] = r.Cnt
	}
	return out, nil
}
