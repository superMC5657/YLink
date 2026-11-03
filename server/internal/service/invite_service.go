package service

import (
	"context"
	"crypto/rand"
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
	set   *SettingService
}

func NewInviteService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config, set *SettingService) *InviteService {
	return &InviteService{db: db, rdb: rdb, repos: repos, cfg: cfg, set: set}
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
		CommissionRate:    commissionRateFor(ctx, s.set, user.Role),
		RegisteredCount:   registered,
		TotalCommission:   model.FenToYuan(total),
		PendingCommission: model.FenToYuan(pending),
	}, nil
}

// codeLimit 邀请码数量上限（invite.invite_code_limit，缺省 5）。
func (s *InviteService) codeLimit(ctx context.Context) int {
	return s.set.GetInt(ctx, "invite", "invite_code_limit", 5)
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
	// register_url_prefix 仅返回路径后缀:API 地址(APP_BASE_URL)≠ 前端站点地址,
	// 完整链接由前端按当前页面 origin 拼接(effectiveRegisterUrlPrefix),本字段仅供契约/占位。
	return &model.InviteCodesResp{
		List:              items,
		Limit:             s.codeLimit(ctx),
		RegisterURLPrefix: "/#/register?code=",
	}, nil
}

// CreateCode POST /invite/codes 生成新邀请码。
func (s *InviteService) CreateCode(ctx context.Context, userID int64) (*model.InviteCodeItem, error) {
	count, err := s.repos.Invite.CountByUser(s.db, userID)
	if err != nil {
		return nil, err
	}
	if count >= int64(s.codeLimit(ctx)) {
		return nil, errs.ErrInviteMax
	}
	code, err := randomInviteCode()
	if err != nil {
		return nil, err
	}
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

// randomInviteCode 8 位大写字母数字；熵源失败显式报错（拒绝服务优于静默弱随机）。
func randomInviteCode() (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return string(b), nil
}

// ---- 佣金记录 ----

// Records GET /invite/records 仅展示已发放（status=1）。
func (s *InviteService) Records(ctx context.Context, userID int64, page, pageSize int) ([]model.CommissionRecordResp, int64, error) {
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

// maxMoneyFen 薄 adapter：安全上限收敛于 money 深 Module（model.MaxMoneyFen）。
// 仅作 float64 元→分转换的溢出/精度防护：YuanToFen 对超大金额会回绕为负数
// （负 fen 绕过「余额 < 金额」校验后反而增加余额），或精度丢失；非业务限额（spec F02 不引入限额设置）。
const maxMoneyFen = model.MaxMoneyFen

// validMoneyFen 薄 adapter：转发到 money 深 Module，非法金额入口即不可表达。
func validMoneyFen(fen int64) bool { return model.ValidMoneyFen(fen) }

// Transfer POST /invite/transfer 佣金划转余额。
// 金额转换（元→分）在 handler 完成，此处校验转换结果，防超大金额回绕成负 fen 绕过余额校验。
func (s *InviteService) Transfer(ctx context.Context, userID int64, amountFen int64) (*model.TransferResp, error) {
	if !validMoneyFen(amountFen) {
		return nil, errs.ErrAmountInvalid
	}
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

// ---- 佣金提现（F02，仅代理商） ----

// SubmitWithdraw POST /invite/withdraw 提交佣金提现工单。
// 最小闭环口径（spec F02）：仅代理商可发起；提交即扣减 commission_balance（同事务行锁防双花），
// 写 commission_logs 提现流水（biz_type=1, status=0）与 commission_withdraws 提现单；
// 工单首条消息为结构化提现信息，管理员手动确认打款或拒绝退回。
func (s *InviteService) SubmitWithdraw(ctx context.Context, userID int64, req *model.WithdrawCreateReq) (*model.WithdrawItem, error) {
	// 深 Module checked 入口：转换+校验一次完成，非法金额在此即不可表达。
	amountFen, ok := model.CheckedYuanToFen(req.Amount)
	if !ok {
		return nil, errs.ErrAmountInvalid
	}
	var out *model.WithdrawItem
	err := repo.WithTx(s.db, func(tx *gorm.DB) error {
		// 生命周期深 Module 拥有提交四写（行锁→身份→余额→扣减→四写同序）
		item, err := submitWithdrawTx(tx, s.repos, userID, amountFen, req.Method, req.Account)
		if err != nil {
			return err
		}
		out = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Withdraws GET /invite/withdraws 本人提现记录分页。
func (s *InviteService) Withdraws(ctx context.Context, userID int64, page, pageSize int) ([]model.WithdrawItem, int64, error) {
	list, total, err := s.repos.Withdraw.ListByUser(s.db, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.WithdrawItem, 0, len(list))
	for i := range list {
		out = append(out, *toWithdrawItem(&list[i]))
	}
	return out, total, nil
}

func toWithdrawItem(w *model.CommissionWithdraw) *model.WithdrawItem {
	return &model.WithdrawItem{
		ID: w.ID, Amount: model.FenToYuan(w.Amount), Method: w.Method, Account: w.Account,
		Status: w.Status, ReviewRemark: w.ReviewRemark, TicketID: w.TicketID,
		ReviewedAt: w.ReviewedAt, CreatedAt: w.CreatedAt,
	}
}

// ---- 代理商 ----

// AgentStatus GET /agent/status。
func (s *InviteService) AgentStatus(ctx context.Context, userID int64) (*model.AgentStatusResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	required, validDays := agentPolicy(ctx, s.set)
	valid, err := s.repos.User.CountValidInvited(s.db, userID, validDays)
	if err != nil {
		return nil, err
	}
	qualified := valid >= int64(required)

	applyStatus := "none"
	if apply, err := s.repos.AgentApply.GetByUser(s.db, userID); err == nil {
		switch apply.Status {
		case model.AgentApplyPending:
			applyStatus = "pending"
		case model.AgentApplyApproved:
			applyStatus = "approved"
		case model.AgentApplyRejected:
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

// agentPolicy 代理商策略唯一读取口：required_valid_invites（默认 50）与
// valid_invite_days（默认 3 天），经 SettingService 缓存读取。
func agentPolicy(ctx context.Context, set *SettingService) (required, validDays int) {
	required = set.GetInt(ctx, "agent", "required_valid_invites", 50)
	validDays = set.GetInt(ctx, "agent", "valid_invite_days", 3)
	return
}

// ApplyAgent POST /agent/apply。
func (s *InviteService) ApplyAgent(ctx context.Context, userID int64) (string, error) {
	required, validDays := agentPolicy(ctx, s.set)
	valid, err := s.repos.User.CountValidInvited(s.db, userID, validDays)
	if err != nil {
		return "", err
	}
	if valid < int64(required) {
		return "", errs.ErrAgentNotQualified
	}
	apply, err := s.repos.AgentApply.GetByUser(s.db, userID)
	if err == nil {
		if apply.Status != model.AgentApplyRejected { // 审核中或已通过，不可重复提交
			return "", errs.ErrAgentDuplicated
		}
		// 被拒绝后重新提交
		apply.Status = model.AgentApplyPending
		apply.Remark = nil
		if err := s.repos.AgentApply.Save(s.db, apply); err != nil {
			return "", err
		}
		return "pending", nil
	}
	if err := s.repos.AgentApply.Create(s.db, &model.AgentApply{UserID: userID, Status: model.AgentApplyPending}); err != nil {
		return "", err
	}
	return "pending", nil
}
