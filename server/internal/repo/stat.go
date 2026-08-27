package repo

import (
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// StatRepo 管理端统计报表数据访问（F04）。
// 口径:订单创建数按 orders.created_at;实收按 status=completed 的 paid_at;
// 退款额按 status=refunded 的 updated_at(退款时更新,近似);注册按 users.created_at;
// 用户流量按 traffic_logs 日明细 u+d;节点流量按 node_user_stats 上报累计值(未乘倍率)。
type StatRepo struct{}

// statDateRow 通用「日期 + 计数」扫描行。
type statDateRow struct {
	Date  string `gorm:"column:d"`
	Count int64  `gorm:"column:c"`
}

// statDateAmountRow 通用「日期 + 计数 + 金额(分)」扫描行。
type statDateAmountRow struct {
	Date   string `gorm:"column:d"`
	Count  int64  `gorm:"column:c"`
	Amount int64  `gorm:"column:a"`
}

// statPlanRow 套餐分布扫描行。
type statPlanRow struct {
	PlanID int64 `gorm:"column:plan_id"`
	Users  int64 `gorm:"column:users"`
}

// statUserTrafficRow 用户流量 Top 扫描行。
type statUserTrafficRow struct {
	UserID int64  `gorm:"column:user_id"`
	Email  string `gorm:"column:email"`
	Total  int64  `gorm:"column:total"`
}

// statNodeTrafficRow 节点流量 Top 扫描行。
type statNodeTrafficRow struct {
	ServerID int64  `gorm:"column:server_id"`
	Name     string `gorm:"column:name"`
	Total    int64  `gorm:"column:total"`
}

// OrderCountByDay 时间范围内每日创建订单数。
func (StatRepo) OrderCountByDay(db *gorm.DB, since time.Time) (map[string]int64, error) {
	var rows []statDateRow
	err := db.Model(&model.Order{}).
		Select("to_char(created_at::date, 'YYYY-MM-DD') AS d, COUNT(*) AS c").
		Where("created_at >= ?", since).
		Group("created_at::date").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Date] = r.Count
	}
	return out, nil
}

// RevenueByDay 时间范围内每日完成订单数与实收（按 paid_at，分）。
func (StatRepo) RevenueByDay(db *gorm.DB, since time.Time) (map[string]statDateAmountRow, error) {
	var rows []statDateAmountRow
	err := db.Model(&model.Order{}).
		Select("to_char(paid_at::date, 'YYYY-MM-DD') AS d, COUNT(*) AS c, COALESCE(SUM(pay_amount),0) AS a").
		Where("status = ? AND paid_at >= ?", model.OrderCompleted, since).
		Group("paid_at::date").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]statDateAmountRow, len(rows))
	for _, r := range rows {
		out[r.Date] = r
	}
	return out, nil
}

// RefundByDay 时间范围内每日退款额（按 updated_at 近似，分）。
func (StatRepo) RefundByDay(db *gorm.DB, since time.Time) (map[string]int64, error) {
	var rows []statDateAmountRow
	err := db.Model(&model.Order{}).
		Select("to_char(updated_at::date, 'YYYY-MM-DD') AS d, 0 AS c, COALESCE(SUM(pay_amount),0) AS a").
		Where("status = ? AND updated_at >= ?", model.OrderRefunded, since).
		Group("updated_at::date").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Date] = r.Amount
	}
	return out, nil
}

// RegisterByDay 时间范围内每日注册数。
func (StatRepo) RegisterByDay(db *gorm.DB, since time.Time) (map[string]int64, error) {
	var rows []statDateRow
	err := db.Model(&model.User{}).
		Select("to_char(created_at::date, 'YYYY-MM-DD') AS d, COUNT(*) AS c").
		Where("created_at >= ?", since).
		Group("created_at::date").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Date] = r.Count
	}
	return out, nil
}

// PlanDistribution 当前生效订阅按套餐聚合（活跃用户口径：plan_id 非空）。
func (StatRepo) PlanDistribution(db *gorm.DB) ([]statPlanRow, error) {
	var rows []statPlanRow
	err := db.Model(&model.User{}).
		Select("plan_id, COUNT(*) AS users").
		Where("plan_id IS NOT NULL").
		Group("plan_id").Order("users DESC").Scan(&rows).Error
	return rows, err
}

// UserTrafficTop 时间范围内用户流量消耗 TopN（字节）。
func (StatRepo) UserTrafficTop(db *gorm.DB, since time.Time, limit int) ([]statUserTrafficRow, error) {
	var rows []statUserTrafficRow
	err := db.Model(&model.TrafficLog{}).
		Select("traffic_logs.user_id, users.email, COALESCE(SUM(traffic_logs.u + traffic_logs.d),0) AS total").
		Joins("JOIN users ON users.id = traffic_logs.user_id").
		Where("traffic_logs.date >= ?", since.Format("2006-01-02")).
		Group("traffic_logs.user_id, users.email").
		Order("total DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

// NodeTrafficTop 节点上报累计流量 TopN（字节，未乘倍率；快照为节点全周期累计，无时间维度）。
func (StatRepo) NodeTrafficTop(db *gorm.DB, limit int) ([]statNodeTrafficRow, error) {
	var rows []statNodeTrafficRow
	err := db.Model(&model.NodeUserStat{}).
		Select("node_user_stats.server_id, servers.name, COALESCE(SUM(node_user_stats.last_u + node_user_stats.last_d),0) AS total").
		Joins("JOIN servers ON servers.id = node_user_stats.server_id").
		Group("node_user_stats.server_id, servers.name").
		Order("total DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}
