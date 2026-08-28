package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	jwtpkg "ylink-backend/internal/pkg/jwt"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/pkg/passwd"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

const (
	captchaTTL    = 10 * time.Minute
	captchaResend = 60 * time.Second
	loginFailMax  = 5
	loginLockTTL  = 10 * time.Minute
	ipCaptchaMax  = 20 // 同 IP 每日验证码上限
)

// AuthService 账户域：验证码、注册、登录、刷新、找回、登出。
type AuthService struct {
	db     *gorm.DB
	rdb    *redis.Client
	repos  *repo.Repos
	jwt    *jwtpkg.Manager
	mailer *mailer.Mailer
	cfg    *config.Config
}

func NewAuthService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, jm *jwtpkg.Manager, m *mailer.Mailer, cfg *config.Config) *AuthService {
	return &AuthService{db: db, rdb: rdb, repos: repos, jwt: jm, mailer: m, cfg: cfg}
}

// ---- 验证码 ----

// SendCaptcha 发送邮箱验证码；type=register/forgot。
func (s *AuthService) SendCaptcha(ctx context.Context, email, typ string) (*model.CaptchaResp, error) {
	// 同邮箱 60s 限频
	rateKey := redispkg.Key("captcha", "rate", email)
	if n, _ := s.rdb.Incr(ctx, rateKey).Result(); n == 1 {
		s.rdb.Expire(ctx, rateKey, captchaResend)
	} else if n > 1 {
		return nil, errs.ErrCaptchaFrequent
	}
	// 同 IP 每日上限
	ip := ctxIP(ctx)
	if ip != "" {
		ipKey := redispkg.Key("captcha", "rate", "ip", ip)
		day, _ := s.rdb.Incr(ctx, ipKey).Result()
		if day == 1 {
			s.rdb.Expire(ctx, ipKey, 24*time.Hour)
		}
		if day > ipCaptchaMax {
			return nil, errs.ErrCaptchaFrequent
		}
	}

	// register 时邮箱已存在直接报错
	if typ == "register" {
		exists, err := s.repos.User.ExistsEmail(s.db, email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errs.ErrEmailTaken
		}
	}

	code := randomCode(6)
	captchaKey := redispkg.Key("captcha", "email", typ, email)
	if err := s.rdb.Set(ctx, captchaKey, code, captchaTTL).Err(); err != nil {
		return nil, err
	}

	// 异步发送邮件（不阻塞主流程，失败重试 2 次）
	go s.sendCaptchaMail(email, code)
	return &model.CaptchaResp{ExpireIn: int(captchaTTL.Seconds()), ResendAfter: int(captchaResend.Seconds())}, nil
}

func (s *AuthService) sendCaptchaMail(email, code string) {
	// F11：验证码邮件走模板渲染（自定义模板优先，缺失/错误回退内置文案）
	subject, body, err := renderMailTemplate(s.db, s.cfg.App.Name, "captcha", map[string]string{"code": code})
	if err != nil {
		logger.L().Error("render captcha mail", zap.Error(err))
		return
	}
	for i := 0; i < 3; i++ {
		if err := s.mailer.Send(email, subject, body); err == nil {
			return
		} else if i == 2 {
			logger.L().Error("send captcha mail failed", zap.String("email", email), zap.Error(err))
		} else {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}
}

// verifyCaptcha 校验并一次性删除验证码。
func (s *AuthService) verifyCaptcha(ctx context.Context, email, typ, code string) error {
	key := redispkg.Key("captcha", "email", typ, email)
	got, err := s.rdb.Get(ctx, key).Result()
	if err != nil || got != code {
		return errs.ErrCaptcha
	}
	s.rdb.Del(ctx, key)
	return nil
}

// ---- 注册 ----

// Register 注册：校验验证码 → 创建用户 → 邀请绑定 → 签发 token。
func (s *AuthService) Register(ctx context.Context, req *model.AuthRegisterReq) (*model.TokenResp, error) {
	// 站点开启强制邀请时，注册必须携带有效邀请码
	if req.InviteCode == "" {
		var site siteSettings
		if err := s.repos.Setting.GetJSON(s.db, "site", &site); err == nil &&
			site.InviteCodeRequired != nil && *site.InviteCodeRequired {
			return nil, errs.ErrInviteCode
		}
	}
	if err := s.verifyCaptcha(ctx, req.Email, "register", req.EmailCode); err != nil {
		return nil, err
	}
	exists, err := s.repos.User.ExistsEmail(s.db, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errs.ErrEmailTaken
	}
	hash, err := passwd.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: hash,
		SubToken:     uuid.NewString(),
		UUID:         uuid.NewString(), // 每用户订阅凭证(模式 A 归因)
	}

	err = repo.WithTx(s.db, func(tx *gorm.DB) error {
		// 邀请绑定：有效且非本人 → 记一级
		if req.InviteCode != "" {
			ic, err := s.repos.Invite.GetByCode(tx, req.InviteCode)
			if err != nil {
				return errs.ErrInviteCode
			}
			inviter, err := s.repos.User.GetByID(tx, ic.UserID)
			if err != nil {
				return errs.ErrInviteCode
			}
			if inviter.Email != user.Email {
				user.InviteByID = &inviter.ID
				if err := s.repos.Invite.IncrUsedCount(tx, ic.ID); err != nil {
					return err
				}
			}
		}
		return s.repos.User.Create(tx, user)
	})
	if err != nil {
		return nil, err
	}
	return s.issueSession(ctx, user)
}

// ---- 登录 ----

// Login 登录：密码校验 + 失败锁定（5 次锁 10 分钟）。
func (s *AuthService) Login(ctx context.Context, req *model.AuthLoginReq) (*model.TokenResp, error) {
	user, err := s.repos.User.GetByEmail(s.db, req.Email)
	if err != nil {
		return nil, errs.ErrBadCredentials
	}
	if user.IsBanned {
		return nil, errs.ErrForbidden
	}
	failKey := redispkg.Key("login", "fail", req.Email)
	// 已锁定
	if lockedIn, err := s.rdb.TTL(ctx, failKey).Result(); err == nil && lockedIn > 0 {
		if n, _ := s.rdb.Get(ctx, failKey).Int64(); n >= loginFailMax {
			return nil, &errs.Error{Code: 42900, Message: fmt.Sprintf("登录失败次数过多，请 %d 秒后重试", int(lockedIn.Seconds())), HTTP: 429}
		}
	}

	if !passwd.Verify(user.PasswordHash, req.Password) {
		n, _ := s.rdb.Incr(ctx, failKey).Result()
		if n == 1 {
			s.rdb.Expire(ctx, failKey, loginLockTTL)
		}
		if n >= loginFailMax {
			return nil, &errs.Error{Code: 42900, Message: "登录失败次数过多，已锁定 10 分钟", HTTP: 429}
		}
		return nil, errs.ErrBadCredentials
	}
	s.rdb.Del(ctx, failKey)
	return s.issueSession(ctx, user)
}

// ---- 刷新 ----

// Refresh 校验 refresh 白名单并旋转签发（旧 jti 立即失效）。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenResp, error) {
	claims, err := s.jwt.Parse(refreshToken)
	if err != nil || claims.TokenType != "refresh" {
		return nil, errs.ErrUnauthorized
	}
	user, err := s.repos.User.GetByID(s.db, claims.UserID)
	if err != nil || user.IsBanned {
		return nil, errs.ErrUnauthorized
	}
	whitelistKey := refreshKey(user.ID, claims.JTI)
	if ok, _ := s.rdb.Exists(ctx, whitelistKey).Result(); ok == 0 {
		return nil, errs.ErrUnauthorized
	}
	s.rdb.Del(ctx, whitelistKey)
	return s.issueSession(ctx, user)
}

// ---- 找回密码 ----

// Forgot 验证码重置密码，并吊销该用户全部会话。
func (s *AuthService) Forgot(ctx context.Context, req *model.ForgotReq) error {
	if err := s.verifyCaptcha(ctx, req.Email, "forgot", req.EmailCode); err != nil {
		return err
	}
	user, err := s.repos.User.GetByEmail(s.db, req.Email)
	if err != nil {
		return nil // 邮箱不存在不暴露
	}
	hash, err := passwd.Hash(req.Password)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	if err := s.repos.User.Update(s.db, user); err != nil {
		return err
	}
	return s.revokeAllSessions(ctx, user.ID)
}

// ---- 登出 ----

// Logout 吊销当前 refresh 并 bump 会话版本号（access 立即失效）。
func (s *AuthService) Logout(ctx context.Context, userID int64, jti string) error {
	if jti != "" {
		s.rdb.Del(ctx, refreshKey(userID, jti))
	}
	bumpSessionVersion(ctx, s.rdb, userID)
	return nil
}

// ---- 会话 ----

// refreshMeta refresh 白名单值（F14 会话管理元数据；历史版本为字符串 "1"，解析失败按未知会话展示）。
type refreshMeta struct {
	IP        string    `json:"ip"`
	UserAgent string    `json:"ua"`
	CreatedAt time.Time `json:"ts"`
}

// issueSession 签发 token 对并写 refresh 白名单（含设备/IP/时间元数据）；
// access 携带当前会话版本号快照。
func (s *AuthService) issueSession(ctx context.Context, user *model.User) (*model.TokenResp, error) {
	jti := uuid.NewString()
	sv := sessionVersion(ctx, s.rdb, user.ID)
	access, refresh, err := s.jwt.Generate(user.ID, user.Role, jti, sv)
	if err != nil {
		return nil, err
	}
	meta, _ := json.Marshal(refreshMeta{
		IP: ctxIP(ctx), UserAgent: ctxUserAgent(ctx), CreatedAt: time.Now(),
	})
	// refresh 白名单：14d
	s.rdb.Set(ctx, refreshKey(user.ID, jti), string(meta), s.cfg.JWT.RefreshTTL)
	return &model.TokenResp{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwt.AccessTTL().Seconds()),
		User:         model.UserBrief{ID: user.ID, Email: user.Email, Role: user.Role},
	}, nil
}

// revokeAllSessions 删除用户全部 refresh 白名单，并 bump 会话版本号（access 立即失效）。
func (s *AuthService) revokeAllSessions(ctx context.Context, userID int64) error {
	pattern := redispkg.Key("refresh", fmt.Sprint(userID), "*")
	iter := s.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		s.rdb.Del(ctx, iter.Val())
	}
	bumpSessionVersion(ctx, s.rdb, userID)
	return iter.Err()
}

// revokeOtherSessions 吊销除当前 jti 外的会话。
func (s *AuthService) revokeOtherSessions(ctx context.Context, userID int64, keepJTI string) error {
	pattern := redispkg.Key("refresh", fmt.Sprint(userID), "*")
	iter := s.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if iter.Val() != refreshKey(userID, keepJTI) {
			s.rdb.Del(ctx, iter.Val())
		}
	}
	return iter.Err()
}

func refreshKey(userID int64, jti string) string {
	return redispkg.Key("refresh", fmt.Sprint(userID), jti)
}

// randomCode 生成 n 位数字验证码。
func randomCode(n int) string {
	max := big.NewInt(int64(pow10(n)))
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%0*d", n, num.Int64())
}

func pow10(n int) int {
	r := 1
	for i := 0; i < n; i++ {
		r *= 10
	}
	return r
}

// ctxIP 从上下文取客户端 IP（handler 注入）。
func ctxIP(ctx context.Context) string {
	if v, ok := ctx.Value("client_ip").(string); ok {
		return v
	}
	return ""
}

// ctxUserAgent 从上下文取 User-Agent（handler 注入，F14 会话元数据）。
func ctxUserAgent(ctx context.Context) string {
	if v, ok := ctx.Value("user_agent").(string); ok {
		return v
	}
	return ""
}
