package service

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// ---- F02 佣金提现 ----

func newWithdrawEnv(t *testing.T) (*testEnv, *InviteService) {
	e := newTestEnv(t)
	svc := NewInviteService(e.db, e.rdb, &repo.Repos{}, nil)
	return e, svc
}

func TestWithdrawSubmitAgent(t *testing.T) {
	e, svc := newWithdrawEnv(t)
	now := time.Now()
	agent := &model.User{ID: 9, Email: "agent@b.com", Role: model.RoleAgent, CommissionBalance: 5000, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(agent))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"tickets\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(77))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"ticket_messages\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"commission_withdraws\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(55))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"commission_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	w, err := svc.SubmitWithdraw(context.Background(), 9, &model.WithdrawCreateReq{
		Amount: 50, Method: "alipay", Account: "agent@b.com",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(77), w.TicketID)
	assert.Equal(t, 50.00, w.Amount)
	assert.Equal(t, model.WithdrawPending, w.Status)
}

func TestWithdrawSubmitNonAgentForbidden(t *testing.T) {
	e, svc := newWithdrawEnv(t)
	now := time.Now()
	u := &model.User{ID: 8, Email: "u@b.com", Role: model.RoleUser, CommissionBalance: 5000, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	e.mock.ExpectRollback()

	_, err := svc.SubmitWithdraw(context.Background(), 8, &model.WithdrawCreateReq{
		Amount: 10, Method: "alipay", Account: "u@b.com",
	})
	require.Error(t, err)
	assert.Equal(t, 13003, codeOf(err))
}

func TestWithdrawSubmitInsufficient(t *testing.T) {
	e, svc := newWithdrawEnv(t)
	now := time.Now()
	agent := &model.User{ID: 9, Email: "agent@b.com", Role: model.RoleAgent, CommissionBalance: 100, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(agent))
	e.mock.ExpectRollback()

	_, err := svc.SubmitWithdraw(context.Background(), 9, &model.WithdrawCreateReq{
		Amount: 50, Method: "alipay", Account: "agent@b.com",
	})
	require.Error(t, err)
	assert.Equal(t, 13002, codeOf(err))
}

func withdrawTicketRows(tk *model.Ticket) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "user_id", "subject", "level", "type", "status", "reopen_count", "last_reply_at", "created_at",
	}).AddRow(tk.ID, tk.UserID, tk.Subject, tk.Level, tk.Type, tk.Status, tk.ReopenCount, tk.LastReplyAt, tk.CreatedAt)
}

func withdrawRows(w *model.CommissionWithdraw) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "user_id", "ticket_id", "amount", "method", "account", "status",
		"review_remark", "reviewed_at", "created_at", "updated_at",
	}).AddRow(w.ID, w.UserID, w.TicketID, w.Amount, w.Method, w.Account, w.Status,
		w.ReviewRemark, w.ReviewedAt, w.CreatedAt, w.UpdatedAt)
}

// newWithdrawSvc 构造 AdminService（设置走 nil,审计写 audit_logs 由 mock 承接）。
func newWithdrawAdminSvc(t *testing.T) (*testEnv, *AdminService) {
	e := newTestEnv(t)
	svc := NewAdminService(e.db, e.rdb, &repo.Repos{}, nil, nil, nil)
	return e, svc
}

func TestWithdrawRejectRefundsCommission(t *testing.T) {
	e, svc := newWithdrawAdminSvc(t)
	now := time.Now()
	tk := &model.Ticket{ID: 77, UserID: 9, Subject: "佣金提现申请", Level: 0, Type: model.TicketTypeWithdraw, Status: 0, CreatedAt: now}
	w := &model.CommissionWithdraw{ID: 55, UserID: 9, TicketID: 77, Amount: 5000, Method: "alipay", Account: "a@b.com", Status: model.WithdrawPending, CreatedAt: now}
	u := &model.User{ID: 9, Email: "agent@b.com", CommissionBalance: 0, CreatedAt: now, UpdatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(withdrawTicketRows(tk))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_withdraws\"")).WillReturnRows(withdrawRows(w))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"users\"")).WillReturnRows(userRow(u))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"users\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"commission_withdraws\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"commission_logs\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"ticket_messages\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	require.NoError(t, svc.ReviewWithdraw(context.Background(), 1, 77, false, "账号有误", "1.2.3.4"))
}

func TestWithdrawPay(t *testing.T) {
	e, svc := newWithdrawAdminSvc(t)
	now := time.Now()
	tk := &model.Ticket{ID: 77, UserID: 9, Subject: "佣金提现申请", Level: 0, Type: model.TicketTypeWithdraw, Status: 0, CreatedAt: now}
	w := &model.CommissionWithdraw{ID: 55, UserID: 9, TicketID: 77, Amount: 5000, Method: "alipay", Account: "a@b.com", Status: model.WithdrawPending, CreatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(withdrawTicketRows(tk))
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"commission_withdraws\"")).WillReturnRows(withdrawRows(w))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"commission_withdraws\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"commission_logs\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"ticket_messages\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	require.NoError(t, svc.ReviewWithdraw(context.Background(), 1, 77, true, "", "1.2.3.4"))
}

func TestWithdrawReviewWrongTicketType(t *testing.T) {
	e, svc := newWithdrawAdminSvc(t)
	now := time.Now()
	tk := &model.Ticket{ID: 78, UserID: 9, Subject: "普通工单", Level: 1, Type: model.TicketTypeNormal, Status: 0, CreatedAt: now}

	e.mock.ExpectBegin()
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(withdrawTicketRows(tk))
	e.mock.ExpectRollback()

	err := svc.ReviewWithdraw(context.Background(), 1, 78, true, "", "1.2.3.4")
	assert.Equal(t, 40900, codeOf(err))
}

// ---- F14 会话管理 ----

func TestListAndRevokeSessions(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()
	metaA, _ := json.Marshal(map[string]any{"ip": "1.2.3.4", "ua": "Chrome", "ts": time.Now()})
	e.mr.Set("refresh:7:sess-a", string(metaA))
	e.mr.Set("refresh:7:sess-b", "1") // 历史版本值（升级前的白名单）

	// 构造 UserService（走 AuthService 的 cfg）
	svc := NewUserService(e.db, e.rdb, &repo.Repos{}, e.svc)

	items, err := svc.ListSessions(ctx, 7, "sess-a")
	require.NoError(t, err)
	assert.Len(t, items, 2)
	current := 0
	for _, it := range items {
		if it.Current {
			current++
			assert.Equal(t, "sess-a", it.JTI)
			assert.Equal(t, "1.2.3.4", it.IP)
			assert.NotNil(t, it.CreatedAt, "有元数据的会话应返回登录时间")
		} else {
			// 历史会话（白名单值 "1"）无元数据：CreatedAt 必须为 nil（JSON null），
			// 不得透出 Go 零值时间（会被渲染成 0001/1/1）
			assert.Nil(t, it.CreatedAt)
			assert.Equal(t, "sess-b", it.JTI)
		}
	}
	assert.Equal(t, 1, current)

	// 踢下线指定会话
	require.NoError(t, svc.RevokeSession(ctx, 7, "sess-b", "sess-a"))
	v, err := e.mr.Get("refresh:7:sess-b")
	assert.Equal(t, "", v)
	killed := e.mr.HGet("auth:kill:7", "sess-b")
	assert.Equal(t, "1", killed)

	// 不能踢当前会话 / 不存在的会话
	assert.Equal(t, 40000, codeOf(svc.RevokeSession(ctx, 7, "sess-a", "sess-a")))
	assert.Equal(t, 40400, codeOf(svc.RevokeSession(ctx, 7, "sess-b", "sess-a")))
}

// ---- F11 邮件模板 ----

func mailTemplateRows(mt *model.MailTemplate) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"name", "subject", "body", "updated_at"}).
		AddRow(mt.Name, mt.Subject, mt.Body, mt.UpdatedAt)
}

func TestRenderMailTemplateFallback(t *testing.T) {
	e := newTestEnv(t)
	// 自定义模板查询失败（无自定义行）→ 回退内置文案
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"mail_templates\"")).WillReturnError(assert.AnError)

	subject, body, err := renderMailTemplate(e.db, "YLink", "captcha", map[string]string{"code": "654321"})
	require.NoError(t, err)
	assert.Contains(t, subject, "邮箱验证码")
	assert.Contains(t, subject, "YLink")
	assert.Contains(t, body, "654321")
}

func TestRenderMailTemplateCustom(t *testing.T) {
	e := newTestEnv(t)
	mt := &model.MailTemplate{Name: "captcha", Subject: "[{{.site_name}}] 自定义主题", Body: "自定义内容 {{.code}}", UpdatedAt: time.Now()}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"mail_templates\"")).WillReturnRows(mailTemplateRows(mt))

	subject, body, err := renderMailTemplate(e.db, "YLink", "captcha", map[string]string{"code": "654321"})
	require.NoError(t, err)
	assert.Equal(t, "[YLink] 自定义主题", subject)
	assert.Contains(t, body, "自定义内容 654321")
}

func TestRenderMailTemplateBrokenCustomFallsBack(t *testing.T) {
	e := newTestEnv(t)
	// 语法错误的自定义模板 → 渲染失败回退内置（先查自定义行成功，渲染炸了再回退）
	mt := &model.MailTemplate{Name: "captcha", Subject: "[{{.site_name}}] 主题", Body: "坏模板 {{.code", UpdatedAt: time.Now()}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"mail_templates\"")).WillReturnRows(mailTemplateRows(mt))

	_, body, err := renderMailTemplate(e.db, "YLink", "captcha", map[string]string{"code": "654321"})
	require.NoError(t, err)
	assert.Contains(t, body, "654321")
}

// ---- F15 内容排序 ----

func TestSortNotices(t *testing.T) {
	e, svc := newWithdrawAdminSvc(t)
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"notices\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"notices\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO \"audit_logs\"")).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	e.mock.ExpectCommit()

	require.NoError(t, svc.SortNotices(context.Background(), 1, []model.AdminSortItem{
		{ID: 1, Sort: 0}, {ID: 2, Sort: 1},
	}, "1.2.3.4"))
}

func TestWithdrawErrsHTTP(t *testing.T) {
	assert.Equal(t, 403, errs.ErrWithdrawForbidden.HTTP)
	assert.Equal(t, 13003, errs.ErrWithdrawForbidden.Code)
	assert.Equal(t, 13004, errs.ErrWithdrawStatus.Code)
}
