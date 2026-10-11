package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/model"
	"ylink-backend/internal/repo"
)

// claim ica2-5：行为变更=提现工单提交/审核/关重开不变式收敛生命周期深 Module；
// 必须不变=提交即扣佣四写同序、pay/reject 审核闭环语义、普通工单关/重开不受影响；
// 最怕的失败=用户關单致佣金卡住、管理端绕过审核直接關单、拒绝未退回佣金。
func TestLifecycleGuards(t *testing.T) {
	withdraw := &model.Ticket{Type: model.TicketTypeWithdraw}
	normal := &model.Ticket{Type: model.TicketTypeNormal}
	assert.Error(t, guardUserCloseWithdraw(withdraw))
	assert.NoError(t, guardUserCloseWithdraw(normal))
	assert.Error(t, guardUserReopenWithdraw(withdraw))
	assert.NoError(t, guardUserReopenWithdraw(normal))
	assert.Error(t, guardAdminCloseWithdraw(withdraw))
	assert.NoError(t, guardAdminCloseWithdraw(normal))

	// 读 seam：非提现 nil，提现查询失败降级 nil
	e := newTestEnv(t)
	assert.Nil(t, withdrawInfo(e.db, normal))
}

// TestAdminCloseWithdrawTicketRejected 管理端直接關提现工单被拒（14003），
// 必须走 pay/reject 审核闭环；普通工单直接關仍放行。
func TestAdminCloseWithdrawTicketRejected(t *testing.T) {
	e := newTestEnv(t)
	repos := &repo.Repos{}
	set := NewSettingService(e.db, e.rdb, repos)
	svc := NewAdminService(e.db, e.rdb, repos, nil, set, nil)
	now := time.Now()

	wt := &model.Ticket{ID: 77, UserID: 9, Subject: "佣金提现申请", Level: 0, Type: model.TicketTypeWithdraw, Status: 0, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(withdrawTicketRows(wt))
	err := svc.CloseTicket(context.Background(), 77)
	assert.Equal(t, 14003, codeOf(err))

	nt := &model.Ticket{ID: 78, UserID: 9, Subject: "普通", Level: 1, Type: model.TicketTypeNormal, Status: 1, CreatedAt: now}
	e.mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM \"tickets\"")).WillReturnRows(withdrawTicketRows(nt))
	e.mock.ExpectBegin()
	e.mock.ExpectExec(regexp.QuoteMeta("UPDATE \"tickets\"")).WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()
	require.NoError(t, svc.CloseTicket(context.Background(), 78))
}
