package service

import (
	"context"
	"time"

	"ylink-backend/internal/model"
)

// 统计域 module：总览 + F04 三张报表。

// clampStatDays 报表时间范围收敛：缺省 30 天，上限 365 天。
func clampStatDays(days int) int {
	if days <= 0 {
		return 30
	}
	if days > 365 {
		return 365
	}
	return days
}

// statDayRange 返回近 days 天的起始时刻（含首日本地 00:00）与逐日日期列表（升序）。
func statDayRange(days int) (time.Time, []string) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := today.AddDate(0, 0, -(days - 1))
	dates := make([]string, 0, days)
	for i := 0; i < days; i++ {
		dates = append(dates, since.AddDate(0, 0, i).Format("2006-01-02"))
	}
	return since, dates
}

// Overview GET /admin/stat/overview。
func (m *adminStatsModule) Overview(ctx context.Context) (*model.AdminOverviewResp, error) {
	out := &model.AdminOverviewResp{}
	today := time.Now().Format("2006-01-02")
	var usr, agent, order, completed, plan int64
	var revTotal, revToday, balance int64
	if err := m.db.Model(&model.User{}).Count(&usr).Error; err != nil {
		return nil, err
	}
	if err := m.db.Model(&model.User{}).Where("role = ?", model.RoleAgent).Count(&agent).Error; err != nil {
		return nil, err
	}
	if err := m.db.Model(&model.Order{}).Count(&order).Error; err != nil {
		return nil, err
	}
	if err := m.db.Model(&model.Order{}).Where("status = ?", model.OrderCompleted).Count(&completed).Error; err != nil {
		return nil, err
	}
	if err := m.db.Model(&model.Plan{}).Count(&plan).Error; err != nil {
		return nil, err
	}
	if err := m.db.Model(&model.Order{}).Where("status = ?", model.OrderCompleted).
		Select("COALESCE(SUM(pay_amount),0)").Scan(&revTotal).Error; err != nil {
		return nil, err
	}
	if err := m.db.Model(&model.Order{}).Where("status = ? AND paid_at::date = ?", model.OrderCompleted, today).
		Select("COALESCE(SUM(pay_amount),0)").Scan(&revToday).Error; err != nil {
		return nil, err
	}
	if err := m.db.Model(&model.User{}).
		Select("COALESCE(SUM(balance),0)").Scan(&balance).Error; err != nil {
		return nil, err
	}
	out.UserCount = usr
	out.AgentCount = agent
	out.OrderCount = order
	out.CompletedOrders = completed
	out.PlanCount = plan
	out.TotalRevenue = model.FenToYuan(revTotal)
	out.TodayRevenue = model.FenToYuan(revToday)
	out.TotalBalance = model.FenToYuan(balance)
	return out, nil
}

// StatOrders GET /admin/stat/orders：订单/营收/退款日趋势（含无数据日补零）。
func (m *adminStatsModule) StatOrders(ctx context.Context, days int) (*model.AdminStatOrdersResp, error) {
	days = clampStatDays(days)
	since, dates := statDayRange(days)
	counts, err := m.repos.Stat.OrderCountByDay(m.db, since)
	if err != nil {
		return nil, err
	}
	revenues, err := m.repos.Stat.RevenueByDay(m.db, since)
	if err != nil {
		return nil, err
	}
	refunds, err := m.repos.Stat.RefundByDay(m.db, since)
	if err != nil {
		return nil, err
	}
	balanceRevenues, err := m.repos.Stat.BalanceRevenueByDay(m.db, since)
	if err != nil {
		return nil, err
	}
	balanceRefunds, err := m.repos.Stat.BalanceRefundByDay(m.db, since)
	if err != nil {
		return nil, err
	}
	items := make([]model.AdminStatOrderPoint, 0, len(dates))
	for _, d := range dates {
		p := model.AdminStatOrderPoint{Date: d, OrderCount: counts[d]}
		if r, ok := revenues[d]; ok {
			p.CompletedCount = r.Count
			p.Revenue = model.FenToYuan(r.Amount)
		}
		if r, ok := balanceRevenues[d]; ok {
			p.BalanceUsed = model.FenToYuan(r)
		}
		p.Refunded = model.FenToYuan(refunds[d])
		p.BalanceRefunded = model.FenToYuan(balanceRefunds[d])
		items = append(items, p)
	}
	return &model.AdminStatOrdersResp{Days: days, Items: items}, nil
}

// StatUsers GET /admin/stat/users：注册趋势（含无数据日补零）+ 套餐分布。
func (m *adminStatsModule) StatUsers(ctx context.Context, days int) (*model.AdminStatUsersResp, error) {
	days = clampStatDays(days)
	since, dates := statDayRange(days)
	registers, err := m.repos.Stat.RegisterByDay(m.db, since)
	if err != nil {
		return nil, err
	}
	trend := make([]model.AdminStatUserPoint, 0, len(dates))
	for _, d := range dates {
		trend = append(trend, model.AdminStatUserPoint{Date: d, Count: registers[d]})
	}
	slices, err := m.repos.Stat.PlanDistribution(m.db)
	if err != nil {
		return nil, err
	}
	dist := make([]model.AdminStatPlanSlice, 0, len(slices))
	for _, r := range slices {
		name := ""
		if p, err := m.repos.Plan.GetByID(m.db, r.PlanID); err == nil {
			name = p.Name
		}
		dist = append(dist, model.AdminStatPlanSlice{PlanID: r.PlanID, PlanName: name, Users: r.Users})
	}
	return &model.AdminStatUsersResp{Days: days, RegisterTrend: trend, PlanDistribution: dist}, nil
}

// StatTraffic GET /admin/stat/traffic：用户流量消耗 TopN（时间范围内）+ 节点流量分布 TopN。
func (m *adminStatsModule) StatTraffic(ctx context.Context, days int) (*model.AdminStatTrafficResp, error) {
	days = clampStatDays(days)
	since, _ := statDayRange(days)
	const topN = 10
	userRows, err := m.repos.Stat.UserTrafficTop(m.db, since, topN)
	if err != nil {
		return nil, err
	}
	userTop := make([]model.AdminStatUserTraffic, 0, len(userRows))
	for _, r := range userRows {
		userTop = append(userTop, model.AdminStatUserTraffic{UserID: r.UserID, Email: r.Email, TotalBytes: r.Total})
	}
	nodeRows, err := m.repos.Stat.NodeTrafficTop(m.db, topN)
	if err != nil {
		return nil, err
	}
	nodeTop := make([]model.AdminStatNodeTraffic, 0, len(nodeRows))
	for _, r := range nodeRows {
		nodeTop = append(nodeTop, model.AdminStatNodeTraffic{ServerID: r.ServerID, Name: r.Name, Bytes: r.Total})
	}
	return &model.AdminStatTrafficResp{Days: days, UserTop: userTop, NodeTop: nodeTop}, nil
}
