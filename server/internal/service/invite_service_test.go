package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/repo"
)

func newInviteEnv(t *testing.T) (*testEnv, *InviteService) {
	e := newTestEnv(t)
	svc := NewInviteService(e.db, e.rdb, &repo.Repos{}, &config.Config{App: config.AppConfig{BaseURL: "https://panel.example.com"}})
	return e, svc
}

func TestTransferSuccess(t *testing.T) {
	e, svc := newInviteEnv(t)
	now := time.Now()
	u := &model.User{ID: 1, Email: "a@b.com", Balance: 100, CommissionBalance: 2000, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	resp, err := svc.Transfer(context.Background(), 1, 1500)
	require.NoError(t, err)
	assert.Equal(t, 16.00, resp.Balance)          // 100 + 1500 分
	assert.Equal(t, 5.00, resp.CommissionBalance) // 2000 - 1500 分
}

func TestTransferInsufficient(t *testing.T) {
	e, svc := newInviteEnv(t)
	now := time.Now()
	u := &model.User{ID: 1, Email: "a@b.com", CommissionBalance: 100, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	e.mock.ExpectRollback()

	_, err := svc.Transfer(context.Background(), 1, 5000)
	assert.Equal(t, 13002, codeOf(err))
}

// TestTransferAmountOverflow 超大金额元→分转换回绕为负数（或超安全域）必须整体拒绝（13005），
// 不得进入事务——负 fen 会绕过「佣金余额 < 金额」校验并反向增加余额。
func TestTransferAmountOverflow(t *testing.T) {
	_, svc := newInviteEnv(t)

	_, err := svc.Transfer(context.Background(), 1, model.YuanToFen(1e19))
	assert.Equal(t, 13005, codeOf(err))

	_, err = svc.Transfer(context.Background(), 1, -100)
	assert.Equal(t, 13005, codeOf(err))

	_, err = svc.Transfer(context.Background(), 1, 0)
	assert.Equal(t, 13005, codeOf(err))

	_, err = svc.Transfer(context.Background(), 1, maxMoneyFen+1)
	assert.Equal(t, 13005, codeOf(err))
}

// TestSubmitWithdrawAmountOverflow 提现金额超过安全域必须拒绝（13005）：
// 溢出为负 fen 时「佣金余额 < 负金额」恒不成立，提交即扣减反而会大幅增加佣金余额。
func TestSubmitWithdrawAmountOverflow(t *testing.T) {
	_, svc := newInviteEnv(t)

	_, err := svc.SubmitWithdraw(context.Background(), 1, &model.WithdrawCreateReq{
		Amount: 1e19, Method: "alipay", Account: "a@b.com",
	})
	assert.Equal(t, 13005, codeOf(err))

	_, err = svc.SubmitWithdraw(context.Background(), 1, &model.WithdrawCreateReq{
		Amount: 1e14, Method: "alipay", Account: "a@b.com", // 1e16 分：未回绕但超出 maxMoneyFen 安全域
	})
	assert.Equal(t, 13005, codeOf(err))
}

func TestCreateCodeLimit(t *testing.T) {
	e, svc := newInviteEnv(t)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"invite_codes\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"invite_code_limit":5}`))
	_, err := svc.CreateCode(context.Background(), 1)
	assert.Equal(t, 13001, codeOf(err))
}

func TestApplyAgentNotQualified(t *testing.T) {
	e, svc := newInviteEnv(t)
	// 读取 agent 策略（required=50，valid_invite_days 缺省 3）
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"required_valid_invites":50}`))
	// 有效邀请数 0 < 50
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	_, err := svc.ApplyAgent(context.Background(), 1)
	assert.Equal(t, 15001, codeOf(err))
}

func TestApplyAgentDuplicated(t *testing.T) {
	e, svc := newInviteEnv(t)
	// 读取 agent 策略
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"required_valid_invites":50}`))
	// 有效邀请数达标
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(60))
	// 已有待审核申请
	now := time.Now()
	apply := &model.AgentApply{ID: 1, UserID: 1, Status: 0, CreatedAt: now}
	applyRows := sqlmock.NewRows([]string{"id", "user_id", "status", "remark", "reviewed_at", "created_at", "updated_at"}).
		AddRow(apply.ID, apply.UserID, apply.Status, nil, nil, now, now)
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"agent_applies\"")).WillReturnRows(applyRows)

	_, err := svc.ApplyAgent(context.Background(), 1)
	assert.Equal(t, 15002, codeOf(err))
}

func TestAgentStatus(t *testing.T) {
	e, svc := newInviteEnv(t)
	now := time.Now()
	u := &model.User{ID: 1, Email: "a@b.com", Role: 0, CreatedAt: now, UpdatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	// settings agent：required=50，valid_invite_days 缺省 3
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT \"value\" FROM \"settings\"")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"required_valid_invites":50}`))
	// 有效邀请数
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM \"users\"")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	// agent_applies 无记录
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"agent_applies\"")).WillReturnError(assert.AnError)

	resp, err := svc.AgentStatus(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, resp.IsAgent)
	assert.Equal(t, "none", resp.ApplyStatus)
	assert.False(t, resp.Qualified)
	assert.Equal(t, int64(3), resp.ValidInvites)
	require.Len(t, resp.Conditions, 2)
	assert.False(t, resp.Conditions[0].Met)
}
