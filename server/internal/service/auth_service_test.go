package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	jwtpkg "ylink-backend/internal/pkg/jwt"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/pkg/passwd"
	"ylink-backend/internal/repo"
)

func init() { logger.Nop() }

type testEnv struct {
	db   *gorm.DB
	mock sqlmock.Sqlmock
	mr   *miniredis.Miniredis
	rdb  *redis.Client
	svc  *AuthService
}

func newTestEnv(t *testing.T) *testEnv {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	cfg := &config.Config{}
	cfg.JWT.Secret = "test-secret-key-0123456789abcdef0123456789"
	cfg.JWT.AccessTTL = 2 * time.Hour
	cfg.JWT.RefreshTTL = 336 * time.Hour
	cfg.App.Name = "YLink"
	cfg.SMTP.Host = "invalid.invalid"
	cfg.SMTP.Port = 1

	svc := NewAuthService(db, rdb, &repo.Repos{}, jwtpkg.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL),
		mailer.New(cfg.SMTP), cfg)
	return &testEnv{db: db, mock: mock, mr: mr, rdb: rdb, svc: svc}
}

func userRow(u *model.User) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "email", "password_hash", "role", "balance", "commission_balance", "invite_by_id",
		"is_banned", "remind_expire", "remind_traffic", "telegram_id", "plan_id", "expired_at",
		"transfer_enable", "u", "d", "speed_limit", "device_limit", "sub_token", "uuid", "created_at", "updated_at",
	}).AddRow(u.ID, u.Email, u.PasswordHash, u.Role, u.Balance, u.CommissionBalance, u.InviteByID,
		u.IsBanned, u.RemindExpire, u.RemindTraffic, u.TelegramID, u.PlanID, u.ExpiredAt,
		u.TransferEnable, u.U, u.D, u.SpeedLimit, u.DeviceLimit, u.SubToken, u.UUID, u.CreatedAt, u.UpdatedAt)
}

func (e *testEnv) expectUserByEmail(u *model.User) {
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
}

func (e *testEnv) expectUserByID(u *model.User) {
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
}

func codeOf(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*errs.Error); ok {
		return e.Code
	}
	return -1
}

func TestSendCaptcha(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()

	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"users\"")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	data, err := e.svc.SendCaptcha(ctx, "a@b.com", "register")
	require.NoError(t, err)
	assert.Equal(t, 600, data.ExpireIn)
	assert.Equal(t, 60, data.ResendAfter)
	code, err := e.mr.Get("captcha:email:register:a@b.com")
	require.NoError(t, err)
	assert.Len(t, code, 6)

	// 60s 内重发 → 10003 限频
	_, err = e.svc.SendCaptcha(ctx, "a@b.com", "register")
	assert.Equal(t, 10003, codeOf(err))
}

func TestSendCaptchaEmailTaken(t *testing.T) {
	e := newTestEnv(t)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"users\"")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	_, err := e.svc.SendCaptcha(context.Background(), "taken@b.com", "register")
	assert.Equal(t, 10001, codeOf(err))
}

func TestRegister(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()

	// 预置验证码
	require.NoError(t, e.mr.Set("captcha:email:register:new@b.com", "123456"))
	// 读取 site 配置（无强制邀请 → 默认关闭）
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).WillReturnError(assert.AnError)
	// 邮箱不存在
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"users\"")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	// 创建用户
	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10086))
	e.mock.ExpectCommit()

	resp, err := e.svc.Register(ctx, &model.AuthRegisterReq{Email: "new@b.com", Password: "Passw0rd!", EmailCode: "123456"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, int64(10086), resp.User.ID)
	// 验证码已一次性删除
	_, err = e.mr.Get("captcha:email:register:new@b.com")
	assert.Error(t, err)
}

func TestRegisterWrongCaptcha(t *testing.T) {
	e := newTestEnv(t)
	require.NoError(t, e.mr.Set("captcha:email:register:bad@b.com", "654321"))
	_, err := e.svc.Register(context.Background(), &model.AuthRegisterReq{Email: "bad@b.com", Password: "Passw0rd!", EmailCode: "000000"})
	assert.Equal(t, 10002, codeOf(err))
}

func TestLogin(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()

	hash, _ := passwd.Hash("Passw0rd!")
	u := &model.User{ID: 1, Email: "a@b.com", PasswordHash: hash, SubToken: "t", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	// 错误密码 → 40101
	e.expectUserByEmail(u)
	_, err := e.svc.Login(ctx, &model.AuthLoginReq{Email: "a@b.com", Password: "wrong"})
	assert.Equal(t, 40101, codeOf(err))
	failVal, _ := e.mr.Get("login:fail:a@b.com")
	assert.Equal(t, "1", failVal)

	// 正确密码 → 签发 token
	e.expectUserByEmail(u)
	resp, err := e.svc.Login(ctx, &model.AuthLoginReq{Email: "a@b.com", Password: "Passw0rd!"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestLoginLockAfter5Fails(t *testing.T) {
	e := newTestEnv(t)
	hash, _ := passwd.Hash("Passw0rd!")
	u := &model.User{ID: 1, Email: "lock@b.com", PasswordHash: hash, SubToken: "t", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	for i := 0; i < 4; i++ {
		e.expectUserByEmail(u)
		_, err := e.svc.Login(context.Background(), &model.AuthLoginReq{Email: "lock@b.com", Password: "wrong"})
		assert.Equal(t, 40101, codeOf(err), "attempt %d", i+1)
	}
	// 第 5 次失败触发锁定
	e.expectUserByEmail(u)
	_, err := e.svc.Login(context.Background(), &model.AuthLoginReq{Email: "lock@b.com", Password: "wrong"})
	assert.Equal(t, 42900, codeOf(err))
	// 锁定期间即使密码正确也拒绝（先查用户再判锁定）
	e.expectUserByEmail(u)
	_, err = e.svc.Login(context.Background(), &model.AuthLoginReq{Email: "lock@b.com", Password: "Passw0rd!"})
	assert.Equal(t, 42900, codeOf(err))
}

func TestRefreshRotation(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()

	u := &model.User{ID: 7, Email: "r@b.com", SubToken: "t", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	// 先签发（写白名单）
	resp, err := e.svc.issueSession(ctx, u)
	require.NoError(t, err)
	oldJTI, err := e.svc.jwt.Parse(resp.RefreshToken)
	require.NoError(t, err)
	oldKey := "refresh:7:" + oldJTI.JTI
	v, _ := e.mr.Get(oldKey)
	assert.NotEmpty(t, v) // F14:白名单值为会话元数据 JSON

	// 刷新（需查用户）
	e.expectUserByID(u)
	newResp, err := e.svc.Refresh(ctx, resp.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newResp.AccessToken)
	// 旧白名单已删除
	v, _ = e.mr.Get(oldKey)
	assert.Equal(t, "", v)
	// 新白名单存在
	newJTI, _ := e.svc.jwt.Parse(newResp.RefreshToken)
	newV, _ := e.mr.Get("refresh:7:" + newJTI.JTI)
	assert.NotEmpty(t, newV)
}

func TestRefreshUnknownToken(t *testing.T) {
	e := newTestEnv(t)
	_, err := e.svc.Refresh(context.Background(), "not-a-real-token")
	assert.Equal(t, 40100, codeOf(err))
}
