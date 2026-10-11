package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/sanitize"
)

// 台账域 module（crud 向量）：套餐/节点基础/分组/优惠券。
// 节点批量/复制/排序/密钥轮换收敛于 node 批量 module；内容类收敛于 content module。

// ListAllPlans GET /admin/plans（含隐藏,价格统一为元,展开 group_ids/is_show）。
func (m *adminCatalogModule) ListAllPlans(ctx context.Context) ([]model.AdminPlanView, error) {
	list, err := m.repos.Plan.ListAll(m.db)
	if err != nil {
		return nil, err
	}
	views := make([]model.AdminPlanView, 0, len(list))
	for _, p := range list {
		views = append(views, toPlanView(&p))
	}
	return views, nil
}

func toPlanView(p *model.Plan) model.AdminPlanView {
	v := model.AdminPlanView{
		ID: p.ID, Name: p.Name, Content: p.Content,
		MonthPrice: yuanPtr(p.MonthPrice), QuarterPrice: yuanPtr(p.QuarterPrice),
		HalfYearPrice: yuanPtr(p.HalfYearPrice), YearPrice: yuanPtr(p.YearPrice),
		OnetimePrice: yuanPtr(p.OnetimePrice),
		TrafficGB:    p.TrafficGB, SpeedLimit: p.SpeedLimit, DeviceLimit: p.DeviceLimit,
		IsShow: p.IsShow, Sort: p.Sort,
	}
	if p.GroupIDs != "" {
		_ = json.Unmarshal([]byte(p.GroupIDs), &v.GroupIDs)
	}
	return v
}

func yuanPtr(fen *int64) *float64 {
	if fen == nil {
		return nil
	}
	v := model.FenToYuan(*fen)
	return &v
}

func (m *adminCatalogModule) CreatePlan(ctx context.Context, req *model.AdminPlanReq) (*model.Plan, error) {
	p := &model.Plan{
		Name:        sanitize.Text(req.Name),
		Content:     sanitize.Markdown(req.Content),
		TrafficGB:   req.TrafficGB,
		SpeedLimit:  req.SpeedLimit,
		DeviceLimit: req.DeviceLimit,
		IsShow:      true,
		Sort:        req.Sort,
	}
	if req.IsShow != nil {
		p.IsShow = *req.IsShow
	}
	p.MonthPrice = fenPtr(req.MonthPrice)
	p.QuarterPrice = fenPtr(req.QuarterPrice)
	p.HalfYearPrice = fenPtr(req.HalfYearPrice)
	p.YearPrice = fenPtr(req.YearPrice)
	p.OnetimePrice = fenPtr(req.OnetimePrice)
	if req.GroupIDs != nil {
		b, _ := json.Marshal(req.GroupIDs)
		p.GroupIDs = string(b)
	}
	if err := m.repos.Plan.Create(m.db, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (m *adminCatalogModule) UpdatePlan(ctx context.Context, id int64, req *model.AdminPlanReq) error {
	if _, err := m.repos.Plan.GetByID(m.db, id); err != nil {
		return errs.ErrNotFound
	}
	updates := map[string]any{
		"name":            sanitize.Text(req.Name),
		"content":         sanitize.Markdown(req.Content),
		"traffic_gb":      req.TrafficGB,
		"speed_limit":     req.SpeedLimit,
		"device_limit":    req.DeviceLimit,
		"sort":            req.Sort,
		"month_price":     fenPtr(req.MonthPrice),
		"quarter_price":   fenPtr(req.QuarterPrice),
		"half_year_price": fenPtr(req.HalfYearPrice),
		"year_price":      fenPtr(req.YearPrice),
		"onetime_price":   fenPtr(req.OnetimePrice),
	}
	if req.IsShow != nil {
		updates["is_show"] = *req.IsShow
	}
	if req.GroupIDs != nil {
		b, _ := json.Marshal(req.GroupIDs)
		updates["group_ids"] = string(b)
	}
	return m.repos.Plan.UpdateMap(m.db, id, updates)
}

func (m *adminCatalogModule) DeletePlan(ctx context.Context, id int64) error {
	// 有订单引用的套餐禁止物理删除（历史订单详情依赖套餐名）
	n, err := m.repos.Order.CountByPlan(m.db, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.ErrPlanInUse
	}
	return m.repos.Plan.Delete(m.db, id)
}

func fenPtr(yuan *float64) *int64 {
	if yuan == nil {
		return nil
	}
	v := model.YuanToFen(*yuan)
	return &v
}

// ListAllServers GET /admin/servers（含隐藏,展开 group_id/host/port/config/tags,聚合 Redis 在线状态与探针）。
func (m *adminCatalogModule) ListAllServers(ctx context.Context) ([]model.AdminServerView, error) {
	list, err := m.repos.Server.ListAll(m.db)
	if err != nil {
		return nil, err
	}
	views := make([]model.AdminServerView, 0, len(list))
	for _, srv := range list {
		views = append(views, toServerView(&srv))
	}
	// 聚合 Redis 中的实时心跳与系统探针负载数据
	if m.rdb != nil && len(views) > 0 {
		hbKeys := make([]string, len(views))
		statKeys := make([]string, len(views))
		for i, v := range views {
			hbKeys[i] = fmt.Sprintf("node:heartbeat:%d", v.ID)
			statKeys[i] = fmt.Sprintf("node:stats:%d", v.ID)
		}
		hbVals, _ := m.rdb.MGet(ctx, hbKeys...).Result()
		statVals, _ := m.rdb.MGet(ctx, statKeys...).Result()

		for i := range views {
			if i < len(hbVals) && hbVals[i] != nil {
				if tsStr, ok := hbVals[i].(string); ok {
					if ts, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
						views[i].IsOnline = true
						views[i].LastHeartbeatAt = &ts
					}
				}
			}
			if i < len(statVals) && statVals[i] != nil {
				if statStr, ok := statVals[i].(string); ok {
					var sys model.NodeSystemStats
					if err := json.Unmarshal([]byte(statStr), &sys); err == nil {
						views[i].System = &sys
					}
				}
			}
		}
	}
	return views, nil
}

func toServerView(srv *model.Server) model.AdminServerView {
	v := model.AdminServerView{
		ID: srv.ID, GroupID: srv.GroupID, Name: srv.Name, Type: srv.Type,
		Host: srv.Host, Port: srv.Port, Config: srv.Config,
		Rate: srv.Rate, Status: srv.Status, IsShow: srv.IsShow, Sort: srv.Sort,
		NodeKey: srv.NodeKey,
	}
	if srv.Tags != nil {
		_ = json.Unmarshal([]byte(*srv.Tags), &v.Tags)
	}
	return v
}

func (m *adminCatalogModule) ListAllServerGroups(ctx context.Context) ([]model.ServerGroup, error) {
	return m.repos.Server.ListAllGroups(m.db)
}

func (m *adminCatalogModule) CreateServerGroup(ctx context.Context, req *model.AdminServerGroupReq) (*model.ServerGroup, error) {
	g := &model.ServerGroup{Name: sanitize.Text(req.Name), Sort: req.Sort}
	if err := m.repos.Server.CreateGroup(m.db, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (m *adminCatalogModule) UpdateServerGroup(ctx context.Context, id int64, req *model.AdminServerGroupReq) error {
	g, err := m.repos.Server.GetGroupByID(m.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	g.Name = sanitize.Text(req.Name)
	g.Sort = req.Sort
	return m.repos.Server.UpdateGroup(m.db, g)
}

func (m *adminCatalogModule) DeleteServerGroup(ctx context.Context, id int64) error {
	return m.repos.Server.DeleteGroup(m.db, id)
}

func (m *adminCatalogModule) CreateServer(ctx context.Context, req *model.AdminServerReq) (*model.Server, error) {
	srv := srvFromReq(req)
	srv.IsShow = true
	if req.IsShow != nil {
		srv.IsShow = *req.IsShow
	}
	srv.NodeKey = newNodeKey()
	if err := m.repos.Server.Create(m.db, srv); err != nil {
		return nil, err
	}
	return srv, nil
}

func (m *adminCatalogModule) UpdateServer(ctx context.Context, id int64, req *model.AdminServerReq) error {
	if _, err := m.repos.Server.GetByID(m.db, id); err != nil {
		return errs.ErrNotFound
	}
	srv := srvFromReq(req)
	updates := map[string]any{
		"group_id": srv.GroupID,
		"name":     srv.Name,
		"type":     srv.Type,
		"host":     srv.Host,
		"port":     srv.Port,
		"config":   srv.Config,
		"rate":     srv.Rate,
		"tags":     srv.Tags,
		"status":   srv.Status,
		"sort":     srv.Sort,
	}
	if req.IsShow != nil {
		updates["is_show"] = *req.IsShow
	}
	return m.repos.Server.UpdateMap(m.db, id, updates)
}

func (m *adminCatalogModule) DeleteServer(ctx context.Context, id int64) error {
	return m.repos.Server.Delete(m.db, id)
}

// newNodeKey 生成 32 位十六进制节点上报密钥。
func newNodeKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func srvFromReq(req *model.AdminServerReq) *model.Server {
	tags := "[]"
	if req.Tags != nil {
		if b, err := json.Marshal(req.Tags); err == nil {
			tags = string(b)
		}
	}
	status := req.Status
	if status == 0 {
		status = 1
	}
	s := &model.Server{
		GroupID: req.GroupID, Name: sanitize.Text(req.Name), Type: req.Type,
		Host: req.Host, Port: req.Port, Config: req.Config,
		Rate: req.Rate, Tags: &tags, Status: status, Sort: req.Sort,
	}
	if s.Rate <= 0 {
		s.Rate = 1
	}
	return s
}

// ListAllCoupons 优惠券列表。
func (m *adminCatalogModule) ListAllCoupons(ctx context.Context) ([]model.AdminCouponView, error) {
	var list []model.Coupon
	if err := m.db.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]model.AdminCouponView, 0, len(list))
	for _, c := range list {
		v := model.AdminCouponView{
			ID: c.ID, Code: c.Code, Type: c.Type,
			Value: model.FenToYuan(c.Value), MinSpend: model.FenToYuan(c.MinSpend),
			LimitPerUser: c.LimitPerUser, TotalLimit: c.TotalLimit, UsedCount: c.UsedCount,
			StartedAt: c.StartedAt, EndedAt: c.EndedAt, IsEnable: c.IsEnable, CreatedAt: c.CreatedAt,
			ValidPeriods: []string{}, PlanIDs: []int64{},
		}
		if c.ValidPeriods != nil {
			_ = json.Unmarshal([]byte(*c.ValidPeriods), &v.ValidPeriods)
		}
		if c.PlanIDs != nil {
			_ = json.Unmarshal([]byte(*c.PlanIDs), &v.PlanIDs)
		}
		out = append(out, v)
	}
	return out, nil
}

func (m *adminCatalogModule) CreateCoupon(ctx context.Context, req *model.AdminCouponReq) (*model.Coupon, error) {
	c := &model.Coupon{
		Code: sanitize.Text(req.Code), Type: req.Type, Value: model.YuanToFen(req.Value),
		MinSpend: model.YuanToFen(req.MinSpend), LimitPerUser: req.LimitPerUser, TotalLimit: req.TotalLimit,
		StartedAt: req.StartedAt, EndedAt: req.EndedAt, IsEnable: true,
	}
	if req.IsEnable != nil {
		c.IsEnable = *req.IsEnable
	}
	if len(req.ValidPeriods) > 0 {
		b, _ := json.Marshal(req.ValidPeriods)
		str := string(b)
		c.ValidPeriods = &str
	}
	if len(req.PlanIDs) > 0 {
		b, _ := json.Marshal(req.PlanIDs)
		str := string(b)
		c.PlanIDs = &str
	}
	if err := m.repos.Coupon.Create(m.db, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (m *adminCatalogModule) UpdateCoupon(ctx context.Context, id int64, req *model.AdminCouponReq) error {
	var exist model.Coupon
	if err := m.db.First(&exist, id).Error; err != nil {
		return errs.ErrNotFound
	}
	updates := map[string]any{
		"code":           sanitize.Text(req.Code),
		"type":           req.Type,
		"value":          model.YuanToFen(req.Value),
		"min_spend":      model.YuanToFen(req.MinSpend),
		"limit_per_user": req.LimitPerUser,
		"total_limit":    req.TotalLimit,
		"started_at":     req.StartedAt,
		"ended_at":       req.EndedAt,
		"valid_periods":  nil,
		"plan_ids":       nil,
	}
	if req.IsEnable != nil {
		updates["is_enable"] = *req.IsEnable
	}
	if len(req.ValidPeriods) > 0 {
		b, _ := json.Marshal(req.ValidPeriods)
		str := string(b)
		updates["valid_periods"] = &str
	}
	if len(req.PlanIDs) > 0 {
		b, _ := json.Marshal(req.PlanIDs)
		str := string(b)
		updates["plan_ids"] = &str
	}
	return m.db.Model(&model.Coupon{}).Where("id = ?", id).Updates(updates).Error
}

func (m *adminCatalogModule) DeleteCoupon(ctx context.Context, id int64) error {
	return m.repos.Coupon.Delete(m.db, id)
}
