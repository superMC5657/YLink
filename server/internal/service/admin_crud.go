package service

import (
	"context"
	"encoding/json"

	"ylink/internal/model"
	"ylink/internal/pkg/errs"
	"ylink/internal/pkg/sanitize"
)

// ---- 管理端 · 套餐 CRUD ----

// ListAllPlans GET /admin/plans（含隐藏）。
func (s *AdminService) ListAllPlans(ctx context.Context) ([]model.Plan, error) {
	return s.repos.Plan.ListAll(s.db)
}

func (s *AdminService) CreatePlan(ctx context.Context, req *model.AdminPlanReq) (*model.Plan, error) {
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
	if len(req.GroupIDs) > 0 {
		b, _ := json.Marshal(req.GroupIDs)
		p.GroupIDs = string(b)
	}
	if err := s.repos.Plan.Create(s.db, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AdminService) UpdatePlan(ctx context.Context, id int64, req *model.AdminPlanReq) error {
	p, err := s.repos.Plan.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	p.Name = sanitize.Text(req.Name)
	p.Content = sanitize.Markdown(req.Content)
	p.TrafficGB = req.TrafficGB
	p.SpeedLimit = req.SpeedLimit
	p.DeviceLimit = req.DeviceLimit
	p.Sort = req.Sort
	if req.IsShow != nil {
		p.IsShow = *req.IsShow
	}
	p.MonthPrice = fenPtr(req.MonthPrice)
	p.QuarterPrice = fenPtr(req.QuarterPrice)
	p.HalfYearPrice = fenPtr(req.HalfYearPrice)
	p.YearPrice = fenPtr(req.YearPrice)
	p.OnetimePrice = fenPtr(req.OnetimePrice)
	if len(req.GroupIDs) > 0 {
		b, _ := json.Marshal(req.GroupIDs)
		p.GroupIDs = string(b)
	}
	return s.repos.Plan.Update(s.db, p)
}

func (s *AdminService) DeletePlan(ctx context.Context, id int64) error {
	return s.repos.Plan.Delete(s.db, id)
}

func fenPtr(yuan *float64) *int64 {
	if yuan == nil {
		return nil
	}
	v := model.YuanToFen(*yuan)
	return &v
}

// ---- 管理端 · 节点 CRUD ----

func (s *AdminService) ListAllServers(ctx context.Context) ([]model.Server, error) {
	return s.repos.Server.ListAll(s.db)
}

func (s *AdminService) ListAllServerGroups(ctx context.Context) ([]model.ServerGroup, error) {
	return s.repos.Server.ListAllGroups(s.db)
}

func (s *AdminService) CreateServerGroup(ctx context.Context, req *model.AdminServerGroupReq) (*model.ServerGroup, error) {
	g := &model.ServerGroup{Name: sanitize.Text(req.Name), Sort: req.Sort}
	if err := s.repos.Server.CreateGroup(s.db, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *AdminService) UpdateServerGroup(ctx context.Context, id int64, req *model.AdminServerGroupReq) error {
	g, err := s.repos.Server.GetGroupByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	g.Name = sanitize.Text(req.Name)
	g.Sort = req.Sort
	return s.repos.Server.UpdateGroup(s.db, g)
}

func (s *AdminService) DeleteServerGroup(ctx context.Context, id int64) error {
	return s.repos.Server.DeleteGroup(s.db, id)
}

func (s *AdminService) CreateServer(ctx context.Context, req *model.AdminServerReq) (*model.Server, error) {
	srv := srvFromReq(req)
	srv.IsShow = true
	if req.IsShow != nil {
		srv.IsShow = *req.IsShow
	}
	if err := s.repos.Server.Create(s.db, srv); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *AdminService) UpdateServer(ctx context.Context, id int64, req *model.AdminServerReq) error {
	srv, err := s.repos.Server.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	updated := srvFromReq(req)
	updated.ID = srv.ID
	return s.repos.Server.Update(s.db, updated)
}

func (s *AdminService) DeleteServer(ctx context.Context, id int64) error {
	return s.repos.Server.Delete(s.db, id)
}

func srvFromReq(req *model.AdminServerReq) *model.Server {
	tags := "[]"
	if len(req.Tags) > 0 {
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

// ---- 管理端 · 优惠券 CRUD ----

func (s *AdminService) ListAllCoupons(ctx context.Context) ([]model.Coupon, error) {
	var list []model.Coupon
	err := s.db.Order("id DESC").Find(&list).Error
	return list, err
}

func (s *AdminService) CreateCoupon(ctx context.Context, req *model.AdminCouponReq) (*model.Coupon, error) {
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
	if err := s.repos.Coupon.Create(s.db, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *AdminService) UpdateCoupon(ctx context.Context, id int64, req *model.AdminCouponReq) error {
	var exist model.Coupon
	if err := s.db.First(&exist, id).Error; err != nil {
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
	return s.db.Model(&model.Coupon{}).Where("id = ?", id).Updates(updates).Error
}

func (s *AdminService) DeleteCoupon(ctx context.Context, id int64) error {
	return s.repos.Coupon.Delete(s.db, id)
}

// ---- 管理端 · 公告 / 知识库 CRUD ----

func (s *AdminService) CreateNotice(ctx context.Context, req *model.AdminNoticeReq) (*model.Notice, error) {
	n := &model.Notice{
		Title: sanitize.Text(req.Title), Content: sanitize.Markdown(req.Content), IsShow: true, Sort: req.Sort,
	}
	if req.IsShow != nil {
		n.IsShow = *req.IsShow
	}
	if err := s.repos.Notice.Create(s.db, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *AdminService) UpdateNotice(ctx context.Context, id int64, req *model.AdminNoticeReq) error {
	n, err := s.repos.Notice.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	n.Title = sanitize.Text(req.Title)
	n.Content = sanitize.Markdown(req.Content)
	n.Sort = req.Sort
	if req.IsShow != nil {
		n.IsShow = *req.IsShow
	}
	return s.repos.Notice.Update(s.db, n)
}

func (s *AdminService) DeleteNotice(ctx context.Context, id int64) error {
	return s.repos.Notice.Delete(s.db, id)
}

func (s *AdminService) CreateKnowledge(ctx context.Context, req *model.AdminKnowledgeReq) (*model.Knowledge, error) {
	lang := req.Language
	if lang == "" {
		lang = "zh-CN"
	}
	k := &model.Knowledge{
		Category: sanitize.Text(req.Category), Title: sanitize.Text(req.Title),
		Body: sanitize.Markdown(req.Body), Language: lang, IsShow: true, Sort: req.Sort,
	}
	if req.IsShow != nil {
		k.IsShow = *req.IsShow
	}
	if err := s.repos.Knowledge.Create(s.db, k); err != nil {
		return nil, err
	}
	return k, nil
}

func (s *AdminService) UpdateKnowledge(ctx context.Context, id int64, req *model.AdminKnowledgeReq) error {
	k, err := s.repos.Knowledge.GetByID(s.db, id)
	if err != nil {
		return errs.ErrNotFound
	}
	lang := req.Language
	if lang == "" {
		lang = k.Language
	}
	k.Category = sanitize.Text(req.Category)
	k.Title = sanitize.Text(req.Title)
	k.Body = sanitize.Markdown(req.Body)
	k.Language = lang
	k.Sort = req.Sort
	if req.IsShow != nil {
		k.IsShow = *req.IsShow
	}
	return s.repos.Knowledge.Update(s.db, k)
}

func (s *AdminService) DeleteKnowledge(ctx context.Context, id int64) error {
	return s.repos.Knowledge.Delete(s.db, id)
}
