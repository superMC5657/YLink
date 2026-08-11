package service

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink/internal/model"
	"ylink/internal/pkg/errs"
	"ylink/internal/pkg/passwd"
	"ylink/internal/repo"
)

// UserService 用户域：信息、设置、改密。
type UserService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	auth  *AuthService
}

func NewUserService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, auth *AuthService) *UserService {
	return &UserService{db: db, rdb: rdb, repos: repos, auth: auth}
}

// Stat GET /user/stat 仪表板统计。
func (s *UserService) Stat(ctx context.Context, userID int64) (*model.UserStatResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	pending, err := s.repos.User.CountOrdersByStatus(s.db, userID, model.OrderPending)
	if err != nil {
		return nil, err
	}
	openTickets, err := s.repos.User.CountOpenTickets(s.db, userID)
	if err != nil {
		return nil, err
	}
	invited, err := s.repos.User.CountInvitedBy(s.db, userID)
	if err != nil {
		return nil, err
	}
	return &model.UserStatResp{
		Email:             user.Email,
		Balance:           model.FenToYuan(user.Balance),
		CommissionBalance: model.FenToYuan(user.CommissionBalance),
		PendingOrderCount: pending,
		OpenTicketCount:   openTickets,
		InvitedCount:      invited,
		IsAgent:           user.Role == model.RoleAgent,
	}, nil
}

// UpdateProfile PUT /user/profile 更新通知设置。
func (s *UserService) UpdateProfile(ctx context.Context, userID int64, req *model.UpdateProfileReq) (*model.UserProfileResp, error) {
	if err := s.repos.User.UpdateProfile(s.db, userID, req.RemindExpire, req.RemindTraffic); err != nil {
		return nil, err
	}
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	return &model.UserProfileResp{RemindExpire: user.RemindExpire, RemindTraffic: user.RemindTraffic}, nil
}

// Profile GET /user/profile 读取通知设置（前端挂载时回填）。
func (s *UserService) Profile(ctx context.Context, userID int64) (*model.UserProfileResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	return &model.UserProfileResp{RemindExpire: user.RemindExpire, RemindTraffic: user.RemindTraffic}, nil
}

// ChangePassword POST /user/password/change 修改密码；成功后吊销其他会话。
func (s *UserService) ChangePassword(ctx context.Context, userID int64, jti string, req *model.ChangePasswordReq) error {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return errs.ErrNotFound
	}
	if !passwd.Verify(user.PasswordHash, req.OldPassword) {
		return errs.ErrBadCredentials
	}
	hash, err := passwd.Hash(req.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	if err := s.repos.User.Update(s.db, user); err != nil {
		return err
	}
	return s.auth.revokeOtherSessions(ctx, userID, jti)
}
