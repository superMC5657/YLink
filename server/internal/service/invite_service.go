package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// InviteService 营销域：邀请码、佣金、划转、代理申请。
type InviteService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	cfg   *config.Config
}

func NewInviteService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config) *InviteService {
	return &InviteService{db: db, rdb: rdb, repos: repos, cfg: cfg}
}

// ---- 邀请总览 ----

// Summary GET /invite/summary。
func (s *InviteService) Summary(ctx context.Context, userID int64) (*model.InviteSummaryResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	registered, err := s.repos.User.CountInvitedBy(s.db, userID)
	if err != nil {
		return nil, err
	}
	total, err := s.repos.Commission.SumAmount(s.db, userID, model.CommissionGranted)
	if err != nil {
		return nil, err
	}
	pending, err := s.repos.Commission.SumAmount(s.db, userID, model.CommissionPending)
	if err != nil {
		return nil, err
	}
	return &model.InviteSummaryResp{
		CommissionBalance: model.FenToYuan(user.CommissionBalance),
		CommissionRate:    commissionRateFor(s.db, user.Role),
		RegisteredCount:   registered,
		TotalCommission:   model.FenToYuan(total),
		PendingCommission: model.FenToYuan(pending),
	}, nil
}

func (s *InviteService) codeLimit() int {
	type inviteCfg struct {
		InviteCodeLimit int `json:"invite_code_limit"`
	}
	var cfg inviteCfg
	if raw, err := s.repos.Setting.Get(s.db, "invite"); err == nil {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.InviteCodeLimit <= 0 {
		return 5
	}
	return cfg.InviteCodeLimit
}

// ---- 邀请码 ----

// Codes GET /invite/codes。
func (s *InviteService) Codes(ctx context.Context, userID int64) (*model.InviteCodesResp, error) {
	list, err := s.repos.Invite.ListByUser(s.db, userID)
	if err != nil {
		return nil, err
	}
	items := make([]model.InviteCodeItem, 0, len(list))
	for _, ic := range list {
		items = append(items, model.InviteCodeItem{Code: ic.Code, UsedCount: ic.UsedCount, CreatedAt: ic.CreatedAt})
	}
	return &model.InviteCodesResp{
		List:              items,
		Limit:             s.codeLimit(),
		RegisterURLPrefix: s.cfg.App.BaseURL + "/register?code=",
	}, nil
}

// CreateCode POST /invite/codes 生成新邀请码。
func (s *InviteService) CreateCode(ctx context.Context, userID int64) (*model.InviteCodeItem, error) {
	count, err := s.repos.Invite.CountByUser(s.db, userID)
	if err != nil {
		return nil, err
	}
	if count >= int64(s.codeLimit()) {
		return nil, errs.ErrInviteMax
	}
	code := randomInviteCode()
	ic := &model.InviteCode{UserID: userID, Code: code, Status: 1}
	if err := s.repos.Invite.Create(s.db, ic); err != nil {
		return nil, err
	}
	return &model.InviteCodeItem{Code: ic.Code, UsedCount: 0, CreatedAt: ic.CreatedAt}, nil
}

// DeleteCode DELETE /invite/codes/:code 删除当前用户自己的邀请码。
func (s *InviteService) DeleteCode(ctx context.Context, userID int64, code string) error {
	if code == "" {
		return errs.ErrInviteCode
	}
	affected, err := s.repos.Invite.DeleteByUser(s.db, userID, code)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errs.ErrNotFound
	}
	return nil
}

// randomInviteCode 8 位大写字母数字。
func randomInviteCode() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

// ---- 佣金记录 ----

// Records GET /invite/records 仅展示已发放（status=1）。
func (s *InviteService) Records(ctx context.Context, userID int64, page, pageSize int) ([]model.CommissionRecordResp, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	list, total, err := s.repos.Commission.ListByInviteStatus(s.db, userID, model.CommissionGranted, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.CommissionRecordResp, 0, len(list))
	for _, cl := range list {
		out = append(out, model.CommissionRecordResp{
			OrderNo:     cl.OrderNo,
			Amount:      model.FenToYuan(cl.Amount),
			Rate:        cl.Rate,
			Status:      cl.Status,
			ConfirmedAt: cl.ConfirmedAt,
			CreatedAt:   cl.CreatedAt,
		})
	}
	return out, total, nil
}

// ---- 佣金划转 ----

// Transfer POST /invite/transfer 佣金划转余额。
func (s *InviteService) Transfer(ctx context.Context, userID int64, amountFen int64) (*model.TransferResp, error) {
	var resp *model.TransferResp
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		user, err := s.repos.User.GetByIDForUpdate(tx, userID)
		if err != nil {
			return err
		}
		if user.CommissionBalance < amountFen {
			return errs.ErrCommissionInsufficient
		}
		user.CommissionBalance -= amountFen
		user.Balance += amountFen
		if err := s.repos.User.Save(tx, user); err != nil {
			return err
		}
		resp = &model.TransferResp{
			Balance:           model.FenToYuan(user.Balance),
			CommissionBalance: model.FenToYuan(user.CommissionBalance),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ---- 代理商 ----

// AgentStatus GET /agent/status。
func (s *InviteService) AgentStatus(ctx context.Context, userID int64) (*model.AgentStatusResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	required, validDays := agentPolicy(s.db)
	valid, err := s.repos.User.CountValidInvited(s.db, userID, validDays)
	if err != nil {
		return nil, err
	}
	qualified := valid >= int64(required)

	applyStatus := "none"
	if apply, err := s.repos.AgentApply.GetByUser(s.db, userID); err == nil {
		switch apply.Status {
		case 0:
			applyStatus = "pending"
		case 1:
			applyStatus = "approved"
		case 2:
			applyStatus = "rejected"
		}
	}

	conditions := []model.AgentCondition{
		{Met: qualified, Text: "邀请有效用户：≥ " + fmt.Sprint(required) + " 人，且没有过被禁封记录。"},
		{Met: qualified, Text: fmt.Sprintf("当前有效人数：已邀请 %d 人，还需邀请 %d 人。", valid, int64(required)-valid)},
	}
	return &model.AgentStatusResp{
		IsAgent:              user.Role == model.RoleAgent,
		ApplyStatus:          applyStatus,
		Qualified:            qualified,
		ValidInvites:         valid,
		RequiredValidInvites: required,
		Conditions:           conditions,
	}, nil
}

// agentPolicy 读取代理商策略：required_valid_invites（默认 50）与 valid_invite_days（默认 3 天）。
func agentPolicy(db *gorm.DB) (required, validDays int) {
	required = 50
	validDays = 3
	type agentCfg struct {
		RequiredValidInvites int `json:"required_valid_invites"`
		ValidInviteDays      int `json:"valid_invite_days"`
	}
	var cfg agentCfg
	if raw, err := (repo.SettingRepo{}).Get(db, "agent"); err == nil {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.RequiredValidInvites > 0 {
		required = cfg.RequiredValidInvites
	}
	if cfg.ValidInviteDays > 0 {
		validDays = cfg.ValidInviteDays
	}
	return
}

// ApplyAgent POST /agent/apply。
func (s *InviteService) ApplyAgent(ctx context.Context, userID int64) (string, error) {
	required, validDays := agentPolicy(s.db)
	valid, err := s.repos.User.CountValidInvited(s.db, userID, validDays)
	if err != nil {
		return "", err
	}
	if valid < int64(required) {
		return "", errs.ErrAgentNotQualified
	}
	apply, err := s.repos.AgentApply.GetByUser(s.db, userID)
	if err == nil {
		if apply.Status != 2 { // 审核中或已通过，不可重复提交
			return "", errs.ErrAgentDuplicated
		}
		// 被拒绝后重新提交
		apply.Status = 0
		apply.Remark = nil
		if err := s.repos.AgentApply.Save(s.db, apply); err != nil {
			return "", err
		}
		return "pending", nil
	}
	if err := s.repos.AgentApply.Create(s.db, &model.AgentApply{UserID: userID, Status: 0}); err != nil {
		return "", err
	}
	return "pending", nil
}
