package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/pkg/sanitize"
)

// ---- 管理端 · 套餐 CRUD ----

// ListAllPlans GET /admin/plans（含隐藏,价格统一为元,展开 group_ids/is_show）。
func (s *AdminService) ListAllPlans(ctx context.Context) ([]model.AdminPlanView, error) {
	list, err := s.repos.Plan.ListAll(s.db)
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
	if req.GroupIDs != nil {
		b, _ := json.Marshal(req.GroupIDs)
		p.GroupIDs = string(b)
	}
	if err := s.repos.Plan.Create(s.db, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AdminService) UpdatePlan(ctx context.Context, id int64, req *model.AdminPlanReq) error {
	if _, err := s.repos.Plan.GetByID(s.db, id); err != nil {
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
	return s.repos.Plan.UpdateMap(s.db, id, updates)
}

func (s *AdminService) DeletePlan(ctx context.Context, id int64) error {
	// 有订单引用的套餐禁止物理删除（历史订单详情依赖套餐名）
	n, err := s.repos.Order.CountByPlan(s.db, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.ErrPlanInUse
	}
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

// ListAllServers GET /admin/servers（含隐藏,展开 group_id/host/port/config/tags）。
func (s *AdminService) ListAllServers(ctx context.Context) ([]model.AdminServerView, error) {
	list, err := s.repos.Server.ListAll(s.db)
	if err != nil {
		return nil, err
	}
	views := make([]model.AdminServerView, 0, len(list))
	for _, srv := range list {
		views = append(views, toServerView(&srv))
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
	srv.NodeKey = newNodeKey()
	if err := s.repos.Server.Create(s.db, srv); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *AdminService) UpdateServer(ctx context.Context, id int64, req *model.AdminServerReq) error {
	if _, err := s.repos.Server.GetByID(s.db, id); err != nil {
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
	return s.repos.Server.UpdateMap(s.db, id, updates)
}

func (s *AdminService) DeleteServer(ctx context.Context, id int64) error {
	return s.repos.Server.Delete(s.db, id)
}

// ResetNodeKey POST /admin/servers/{id}/node-key/reset 重置节点上报密钥(旧密钥立即失效)。
func (s *AdminService) ResetNodeKey(ctx context.Context, adminID, serverID int64, ip string) (string, error) {
	srv, err := s.repos.Server.GetByID(s.db, serverID)
	if err != nil {
		return "", errs.ErrNotFound
	}
	oldKey := srv.NodeKey
	newKey := newNodeKey()
	if err := s.repos.Server.UpdateMap(s.db, serverID, map[string]any{"node_key": newKey}); err != nil {
		return "", err
	}
	_ = s.audit(s.db, adminID, "reset_node_key", fmt.Sprint(serverID), ip, map[string]any{})
	// 旧密钥的鉴权缓存立即失效(NodeAuth 中间件 node:key:{k})
	if oldKey != "" {
		s.rdb.Del(ctx, redispkg.Key("node", "key", oldKey))
	}
	return newKey, nil
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

// ---- 管理端 · 优惠券 CRUD ----

func (s *AdminService) ListAllCoupons(ctx context.Context) ([]model.AdminCouponView, error) {
	var list []model.Coupon
	if err := s.db.Order("id DESC").Find(&list).Error; err != nil {
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
	if _, err := s.repos.Notice.GetByID(s.db, id); err != nil {
		return errs.ErrNotFound
	}
	updates := map[string]any{
		"title":   sanitize.Text(req.Title),
		"content": sanitize.Markdown(req.Content),
		"sort":    req.Sort,
	}
	if req.IsShow != nil {
		updates["is_show"] = *req.IsShow
	}
	return s.repos.Notice.UpdateMap(s.db, id, updates)
}

func (s *AdminService) DeleteNotice(ctx context.Context, id int64) error {
	return s.repos.Notice.Delete(s.db, id)
}

// resolveKnowledgeCategory F15：按 (language, name) 找到或创建分类，返回分类 ID 与展示名。
// 显式 categoryID 优先；无匹配 ID 时按名称归并（首次出现的新分类自动建行，存量数据兜底）。
func (s *AdminService) resolveKnowledgeCategory(db *gorm.DB, language, name string, categoryID *int64) (catID *int64, displayName string, err error) {
	displayName = name
	if categoryID != nil && *categoryID > 0 {
		if kc, e := s.repos.KnowledgeCat.GetByID(db, *categoryID); e == nil {
			return &kc.ID, kc.Name, nil
		}
	}
	if name == "" {
		return nil, "", nil
	}
	if kc, e := s.repos.KnowledgeCat.GetByLanguageAndName(db, language, name); e == nil {
		return &kc.ID, kc.Name, nil
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, "", e
	}
	kc := &model.KnowledgeCategory{Language: language, Name: name}
	if err := s.repos.KnowledgeCat.Create(db, kc); err != nil {
		return nil, "", err
	}
	return &kc.ID, kc.Name, nil
}

func (s *AdminService) CreateKnowledge(ctx context.Context, req *model.AdminKnowledgeReq) (*model.Knowledge, error) {
	lang := req.Language
	if lang == "" {
		lang = "zh-CN"
	}
	catName := sanitize.Text(req.Category)
	catID, displayName, err := s.resolveKnowledgeCategory(s.db, lang, catName, req.CategoryID)
	if err != nil {
		return nil, err
	}
	k := &model.Knowledge{
		Category: displayName, CategoryID: catID,
		Title: sanitize.Text(req.Title),
		Body:  sanitize.Markdown(req.Body), Language: lang, IsShow: true, Sort: req.Sort,
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
	catName := sanitize.Text(req.Category)
	// 未显式传 category_id 时沿用原归属（仅当分类名未变），避免改标题等字段丢失归类
	var catIDArg *int64
	if req.CategoryID != nil {
		catIDArg = req.CategoryID
	} else if catName == k.Category && k.CategoryID != nil {
		catIDArg = k.CategoryID
	}
	catID, displayName, err := s.resolveKnowledgeCategory(s.db, lang, catName, catIDArg)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"category":    displayName,
		"category_id": catID,
		"title":       sanitize.Text(req.Title),
		"body":        sanitize.Markdown(req.Body),
		"language":    lang,
		"sort":        req.Sort,
	}
	if req.IsShow != nil {
		updates["is_show"] = *req.IsShow
	}
	return s.repos.Knowledge.UpdateMap(s.db, id, updates)
}

func (s *AdminService) DeleteKnowledge(ctx context.Context, id int64) error {
	return s.repos.Knowledge.Delete(s.db, id)
}

// ListAllNotices GET /admin/notices（含隐藏,创建时间倒序）。
func (s *AdminService) ListAllNotices(ctx context.Context) ([]model.AdminNoticeItem, error) {
	var list []model.Notice
	if err := s.db.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]model.AdminNoticeItem, 0, len(list))
	for _, n := range list {
		out = append(out, model.AdminNoticeItem{
			ID: n.ID, Title: n.Title, Content: n.Content, IsShow: n.IsShow, Sort: n.Sort, CreatedAt: n.CreatedAt,
		})
	}
	return out, nil
}

// ListAllKnowledges GET /admin/knowledges（含隐藏）。
func (s *AdminService) ListAllKnowledges(ctx context.Context) ([]model.AdminKnowledgeItem, error) {
	var list []model.Knowledge
	if err := s.db.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]model.AdminKnowledgeItem, 0, len(list))
	for _, k := range list {
		out = append(out, model.AdminKnowledgeItem{
			ID: k.ID, Category: k.Category, Title: k.Title, Body: k.Body,
			Language: k.Language, IsShow: k.IsShow, Sort: k.Sort, UpdatedAt: k.UpdatedAt,
		})
	}
	return out, nil
}
