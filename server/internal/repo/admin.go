package repo

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/passwd"
)

// ---- 管理端 · 用户 ----

// ListByPage 用户分页（email 关键字可空）。
func (UserRepo) ListByPage(db *gorm.DB, keyword string, page, pageSize int) ([]model.User, int64, error) {
	var list []model.User
	var total int64
	q := db.Model(&model.User{})
	if keyword != "" {
		q = q.Where("email LIKE ?", "%"+keyword+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// SetBanned 封禁/解封。
func (UserRepo) SetBanned(db *gorm.DB, id int64, banned bool) error {
	return db.Model(&model.User{}).Where("id = ?", id).Update("is_banned", banned).Error
}

// UpdateRole 更新角色。
func (UserRepo) UpdateRole(db *gorm.DB, id int64, role int) error {
	return db.Model(&model.User{}).Where("id = ?", id).Update("role", role).Error
}

// ---- 管理端 · 套餐 ----

func (PlanRepo) ListAll(db *gorm.DB) ([]model.Plan, error) {
	var list []model.Plan
	err := db.Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (PlanRepo) Create(db *gorm.DB, p *model.Plan) error { return db.Create(p).Error }

// UpdateMap 按 ID 更新指定列（map 更新，支持 false/空值落库）。
func (PlanRepo) UpdateMap(db *gorm.DB, id int64, updates map[string]any) error {
	return db.Model(&model.Plan{}).Where("id = ?", id).Updates(updates).Error
}

func (PlanRepo) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&model.Plan{}, id).Error
}

// ---- 管理端 · 节点 ----

func (ServerRepo) ListAll(db *gorm.DB) ([]model.Server, error) {
	var list []model.Server
	err := db.Order("group_id ASC, sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (ServerRepo) Create(db *gorm.DB, s *model.Server) error { return db.Create(s).Error }

func (ServerRepo) GetByID(db *gorm.DB, id int64) (*model.Server, error) {
	var s model.Server
	if err := db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateMap 按 ID 更新指定列（map 更新，支持 false/空值落库）。
func (ServerRepo) UpdateMap(db *gorm.DB, id int64, updates map[string]any) error {
	return db.Model(&model.Server{}).Where("id = ?", id).Updates(updates).Error
}

func (ServerRepo) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&model.Server{}, id).Error
}

func (ServerRepo) ListAllGroups(db *gorm.DB) ([]model.ServerGroup, error) {
	var list []model.ServerGroup
	err := db.Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (ServerRepo) CreateGroup(db *gorm.DB, g *model.ServerGroup) error { return db.Create(g).Error }

func (ServerRepo) GetGroupByID(db *gorm.DB, id int64) (*model.ServerGroup, error) {
	var g model.ServerGroup
	if err := db.First(&g, id).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (ServerRepo) UpdateGroup(db *gorm.DB, g *model.ServerGroup) error {
	return db.Model(&model.ServerGroup{}).Where("id = ?", g.ID).Updates(g).Error
}

func (ServerRepo) DeleteGroup(db *gorm.DB, id int64) error {
	return db.Delete(&model.ServerGroup{}, id).Error
}

// ---- 管理端 · 订单 ----

// ListByPage 全量订单分页（status 可空）。
func (OrderRepo) ListByPage(db *gorm.DB, status *int, page, pageSize int) ([]model.Order, int64, error) {
	var list []model.Order
	var total int64
	q := db.Model(&model.Order{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ---- 管理端 · 公告 / 知识库 ----

func (NoticeRepo) Create(db *gorm.DB, n *model.Notice) error { return db.Create(n).Error }

func (NoticeRepo) GetByID(db *gorm.DB, id int64) (*model.Notice, error) {
	var n model.Notice
	if err := db.First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

// UpdateMap 按 ID 更新指定列（map 更新，支持 false/空值落库）。
func (NoticeRepo) UpdateMap(db *gorm.DB, id int64, updates map[string]any) error {
	return db.Model(&model.Notice{}).Where("id = ?", id).Updates(updates).Error
}

func (NoticeRepo) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&model.Notice{}, id).Error
}

func (KnowledgeRepo) Create(db *gorm.DB, k *model.Knowledge) error { return db.Create(k).Error }

// UpdateMap 按 ID 更新指定列（map 更新，支持 false/空值落库）。
func (KnowledgeRepo) UpdateMap(db *gorm.DB, id int64, updates map[string]any) error {
	return db.Model(&model.Knowledge{}).Where("id = ?", id).Updates(updates).Error
}

func (KnowledgeRepo) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&model.Knowledge{}, id).Error
}

// ---- 管理端 · 工单 ----

// ListByPage 全量工单分页。
func (TicketRepo) ListByPage(db *gorm.DB, page, pageSize int) ([]model.Ticket, int64, error) {
	var list []model.Ticket
	var total int64
	if err := db.Model(&model.Ticket{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ---- 管理端 · 代理申请 ----

func (AgentApplyRepo) ListByStatus(db *gorm.DB, status int, page, pageSize int) ([]model.AgentApply, int64, error) {
	var list []model.AgentApply
	var total int64
	q := db.Model(&model.AgentApply{})
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (AgentApplyRepo) GetByID(db *gorm.DB, id int64) (*model.AgentApply, error) {
	var a model.AgentApply
	if err := db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// GetByIDForUpdate 行锁读取（审批并发防重）。
func (AgentApplyRepo) GetByIDForUpdate(db *gorm.DB, id int64) (*model.AgentApply, error) {
	var a model.AgentApply
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, id).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ---- 管理端 · 佣金日志 ----

func (CommissionRepo) ListByPage(db *gorm.DB, status *int, page, pageSize int) ([]model.CommissionLog, int64, error) {
	var list []model.CommissionLog
	var total int64
	q := db.Model(&model.CommissionLog{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ---- 管理端 · 审计 ----

// AuditLogRepo 审计日志。
type AuditLogRepo struct{}

func (AuditLogRepo) Create(db *gorm.DB, l *model.AuditLog) error { return db.Create(l).Error }

// ---- 管理端 · 流量 ----

// Upsert 按 (user_id, date) 覆盖导入。
func (TrafficLogRepo) Upsert(db *gorm.DB, t *model.TrafficLog) error {
	var count int64
	db.Model(&model.TrafficLog{}).Where("user_id = ? AND date = ?", t.UserID, t.Date).Count(&count)
	if count > 0 {
		return db.Model(&model.TrafficLog{}).Where("user_id = ? AND date = ?", t.UserID, t.Date).
			Updates(map[string]any{"u": t.U, "d": t.D}).Error
	}
	return db.Create(t).Error
}

// ---- Repos 聚合更新 ----

// Repos 聚合全部仓储，注入 service 层。
type Repos struct {
	User       UserRepo
	Invite     InviteCodeRepo
	Setting    SettingRepo
	Notice     NoticeRepo
	Knowledge  KnowledgeRepo
	Plan       PlanRepo
	Server     ServerRepo
	Order      OrderRepo
	Payment    PaymentRepo
	Coupon     CouponRepo
	Commission CommissionRepo
	Traffic    TrafficLogRepo
	AgentApply AgentApplyRepo
	Ticket     TicketRepo
	Audit      AuditLogRepo
}

// EnsureAdmin 幂等初始化首个管理员：users 表无管理员且环境变量齐全时创建。
func EnsureAdmin(db *gorm.DB, email, password string) error {
	if email == "" || password == "" {
		return errors.New("ADMIN_EMAIL / ADMIN_PASSWORD 未设置")
	}
	var count int64
	if err := db.Model(&model.User{}).Where("role = ?", model.RoleAdmin).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := passwd.Hash(password)
	if err != nil {
		return err
	}
	admin := &model.User{
		Email:        email,
		PasswordHash: hash,
		Role:         model.RoleAdmin,
		SubToken:     uuid.NewString(),
	}
	return db.Create(admin).Error
}

// EnsureDemoUser 幂等初始化演示账号：该邮箱不存在且环境变量齐全时创建（普通用户）。
func EnsureDemoUser(db *gorm.DB, email, password string) error {
	if email == "" || password == "" {
		return errors.New("DEMO_EMAIL / DEMO_PASSWORD 未设置")
	}
	var count int64
	if err := db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := passwd.Hash(password)
	if err != nil {
		return err
	}
	demo := &model.User{
		Email:        email,
		PasswordHash: hash,
		Role:         model.RoleUser,
		SubToken:     uuid.NewString(),
	}
	return db.Create(demo).Error
}
