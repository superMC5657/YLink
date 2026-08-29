package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/passwd"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
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
	return &model.UserProfileResp{RemindExpire: user.RemindExpire, RemindTraffic: user.RemindTraffic, TelegramBound: user.TelegramID != nil}, nil
}

// Profile GET /user/profile 读取通知设置（前端挂载时回填）。
func (s *UserService) Profile(ctx context.Context, userID int64) (*model.UserProfileResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	return &model.UserProfileResp{RemindExpire: user.RemindExpire, RemindTraffic: user.RemindTraffic, TelegramBound: user.TelegramID != nil}, nil
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

// ---- 会话管理（F14） ----

// ListSessions GET /user/sessions 活跃会话列表（refresh 白名单维度，Redis SCAN）。
// currentJTI 标记当前会话；历史版本白名单值（"1"）解析失败按未知设备降级展示。
func (s *UserService) ListSessions(ctx context.Context, userID int64, currentJTI string) ([]model.UserSessionItem, error) {
	pattern := redispkg.Key("refresh", fmt.Sprint(userID), "*")
	iter := s.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	out := make([]model.UserSessionItem, 0, 4)
	for iter.Next(ctx) {
		key := iter.Val()
		jti := key[strings.LastIndex(key, ":")+1:]
		if jti == "" {
			continue
		}
		item := model.UserSessionItem{JTI: jti, Current: jti == currentJTI}
		if raw, err := s.rdb.Get(ctx, key).Result(); err == nil {
			var meta struct {
				IP        string    `json:"ip"`
				UserAgent string    `json:"ua"`
				CreatedAt time.Time `json:"ts"`
			}
			if json.Unmarshal([]byte(raw), &meta) == nil {
				item.IP = meta.IP
				item.UserAgent = meta.UserAgent
				// 无元数据（解析失败/无时间戳）保持 nil → JSON null，前端显示「--」，
				// 不透出 Go 零值时间（会被渲染成 0001/1/1）
				if !meta.CreatedAt.IsZero() {
					item.CreatedAt = &meta.CreatedAt
				}
			}
		}
		out = append(out, item)
	}
	return out, iter.Err()
}

// RevokeSession DELETE /user/sessions/{jti} 踢下线指定会话：删除 refresh 白名单 +
// 写踢下线标记（access 立即失效）；当前会话不可自行踢除，其余会话不受影响。
func (s *UserService) RevokeSession(ctx context.Context, userID int64, jti, currentJTI string) error {
	if jti == "" {
		return errs.ErrNotFound
	}
	if jti == currentJTI {
		return errs.New(40000, "不能踢除当前登录会话")
	}
	n, err := s.rdb.Del(ctx, refreshKey(userID, jti)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return errs.ErrNotFound
	}
	// access 立即失效：Auth 中间件 HExists 命中即 401；Hash 随 refresh TTL 自动清理
	killKey := redispkg.SessionKillKey(userID)
	s.rdb.HSet(ctx, killKey, jti, 1)
	s.rdb.Expire(ctx, killKey, s.auth.cfg.JWT.RefreshTTL)
	return nil
}
