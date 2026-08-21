package repo

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ylink-backend/internal/model"
)

// ---- 模式 A · 节点上报(见 docs/backend/core-flows.md §8) ----

// GetByNodeKey 按上报密钥查节点(X-Node-Key 鉴权)。
func (ServerRepo) GetByNodeKey(db *gorm.DB, key string) (*model.Server, error) {
	var s model.Server
	if err := db.Where("node_key = ?", key).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ListByUUIDs 按订阅凭证批量查用户(上报归因)。
func (UserRepo) ListByUUIDs(db *gorm.DB, uuids []string) ([]model.User, error) {
	var list []model.User
	err := db.Where("uuid IN ?", uuids).Find(&list).Error
	return list, err
}

// ListActiveByPlanIDs 套餐内有效订阅且未封禁的用户(过期时间为空视为不限期,如 onetime)。
func (UserRepo) ListActiveByPlanIDs(db *gorm.DB, planIDs []int64, now time.Time) ([]model.User, error) {
	var list []model.User
	err := db.Where("plan_id IN ? AND is_banned = false AND (expired_at IS NULL OR expired_at > ?)", planIDs, now).
		Order("id ASC").Find(&list).Error
	return list, err
}

// IncrTraffic 原子累加用户已用流量(SQL 侧自增,免行锁读改写)。
func (UserRepo) IncrTraffic(db *gorm.DB, userID int64, du, dd int64) error {
	return db.Model(&model.User{}).Where("id = ?", userID).
		Updates(map[string]any{"u": gorm.Expr("u + ?", du), "d": gorm.Expr("d + ?", dd)}).Error
}

// AdditiveUpsert 按 (user_id, date) 增量聚合:存在则累加,不存在则插入。
// 与模式 B 的覆盖式 Upsert(repo/admin.go)区分:节点上报使用本方法。
func (TrafficLogRepo) AdditiveUpsert(db *gorm.DB, userID int64, date string, du, dd int64) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"u": gorm.Expr("traffic_logs.u + EXCLUDED.u"),
			"d": gorm.Expr("traffic_logs.d + EXCLUDED.d"),
		}),
	}).Create(&model.TrafficLog{UserID: userID, Date: date, U: du, D: dd}).Error
}

// NodeUserStatRepo 节点上报快照数据访问。
type NodeUserStatRepo struct{}

// GetForUpdate 行锁读取快照(上报事务内差分,防并发上报交错);不存在返回 gorm.ErrRecordNotFound(视为首次上报,基线 0)。
func (NodeUserStatRepo) GetForUpdate(db *gorm.DB, serverID, userID int64) (*model.NodeUserStat, error) {
	var st model.NodeUserStat
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("server_id = ? AND user_id = ?", serverID, userID).First(&st).Error
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// Upsert 覆盖快照(累计值 + 时间):存在则更新,不存在则插入。
func (NodeUserStatRepo) Upsert(db *gorm.DB, serverID, userID int64, lastU, lastD int64) error {
	now := time.Now()
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "server_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"last_u": lastU, "last_d": lastD, "updated_at": now,
		}),
	}).Create(&model.NodeUserStat{
		ServerID: serverID, UserID: userID, LastU: lastU, LastD: lastD, UpdatedAt: now,
	}).Error
}
