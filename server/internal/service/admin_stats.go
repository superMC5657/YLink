package service

import (
	"context"
	"time"

	"ylink-backend/internal/model"
)

// ---- 管理端 · 统计报表（F04） ----

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

// StatOrders GET /admin/stat/orders：订单/营收/退款日趋势（含无数据日补零）。
func (s *AdminService) StatOrders(ctx context.Context, days int) (*model.AdminStatOrdersResp, error) {
	days = clampStatDays(days)
	since, dates := statDayRange(days)
	counts, err := s.repos.Stat.OrderCountByDay(s.db, since)
	if err != nil {
		return nil, err
	}
	revenues, err := s.repos.Stat.RevenueByDay(s.db, since)
	if err != nil {
		return nil, err
	}
	refunds, err := s.repos.Stat.RefundByDay(s.db, since)
	if err != nil {
		return nil, err
	}
	balanceRevenues, err := s.repos.Stat.BalanceRevenueByDay(s.db, since)
	if err != nil {
		return nil, err
	}
	balanceRefunds, err := s.repos.Stat.BalanceRefundByDay(s.db, since)
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
func (s *AdminService) StatUsers(ctx context.Context, days int) (*model.AdminStatUsersResp, error) {
	days = clampStatDays(days)
	since, dates := statDayRange(days)
	registers, err := s.repos.Stat.RegisterByDay(s.db, since)
	if err != nil {
		return nil, err
	}
	trend := make([]model.AdminStatUserPoint, 0, len(dates))
	for _, d := range dates {
		trend = append(trend, model.AdminStatUserPoint{Date: d, Count: registers[d]})
	}
	slices, err := s.repos.Stat.PlanDistribution(s.db)
	if err != nil {
		return nil, err
	}
	dist := make([]model.AdminStatPlanSlice, 0, len(slices))
	for _, r := range slices {
		name := ""
		if p, err := s.repos.Plan.GetByID(s.db, r.PlanID); err == nil {
			name = p.Name
		}
		dist = append(dist, model.AdminStatPlanSlice{PlanID: r.PlanID, PlanName: name, Users: r.Users})
	}
	return &model.AdminStatUsersResp{Days: days, RegisterTrend: trend, PlanDistribution: dist}, nil
}

// StatTraffic GET /admin/stat/traffic：用户流量消耗 TopN（时间范围内）+ 节点流量分布 TopN。
func (s *AdminService) StatTraffic(ctx context.Context, days int) (*model.AdminStatTrafficResp, error) {
	days = clampStatDays(days)
	since, _ := statDayRange(days)
	const topN = 10
	userRows, err := s.repos.Stat.UserTrafficTop(s.db, since, topN)
	if err != nil {
		return nil, err
	}
	userTop := make([]model.AdminStatUserTraffic, 0, len(userRows))
	for _, r := range userRows {
		userTop = append(userTop, model.AdminStatUserTraffic{UserID: r.UserID, Email: r.Email, TotalBytes: r.Total})
	}
	nodeRows, err := s.repos.Stat.NodeTrafficTop(s.db, topN)
	if err != nil {
		return nil, err
	}
	nodeTop := make([]model.AdminStatNodeTraffic, 0, len(nodeRows))
	for _, r := range nodeRows {
		nodeTop = append(nodeTop, model.AdminStatNodeTraffic{ServerID: r.ServerID, Name: r.Name, Bytes: r.Total})
	}
	return &model.AdminStatTrafficResp{Days: days, UserTop: userTop, NodeTop: nodeTop}, nil
}
