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

// 缝测试：改密/重置订阅只写目标列。最怕的失败=请求开始读出的全量快照
// 整体 Updates，把操作期间并发的余额/流量/佣金写覆盖回去（丢钱、丢流量）。

func TestChangePasswordUpdatesOnlyPasswordColumn(t *testing.T) {
	e := newTestEnv(t)
	hash, err := hashPassword("OldPass0!")
	require.NoError(t, err)
	u := &model.User{ID: 1, Email: "a@b.com", PasswordHash: hash, Balance: 500, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).WillReturnRows(userRow(u))
	e.mock.ExpectBegin()
	// 锚定整条 UPDATE：仅 password_hash（+gorm 自动的 updated_at），断言不含 balance 等其他列
	e.mock.ExpectExec(`^UPDATE "users" SET "password_hash"=.+"updated_at"=.+ WHERE id = .+$`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	e.mock.ExpectCommit()

	svc := NewUserService(e.db, e.rdb, &repo.Repos{}, e.svc)
	require.NoError(t, svc.ChangePassword(context.Background(), 1, "jti-1", &model.ChangePasswordReq{
		OldPassword: "OldPass0!", NewPassword: "NewPass0!",
	}))
	require.NoError(t, e.mock.ExpectationsWereMet())
}

func TestChangePasswordWrongOld(t *testing.T) {
	e := newTestEnv(t)
	hash, err := hashPassword("OldPass0!")
	require.NoError(t, err)
	u := &model.User{ID: 1, Email: "a@b.com", PasswordHash: hash, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	e.mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).WillReturnRows(userRow(u))

	svc := NewUserService(e.db, e.rdb, &repo.Repos{}, e.svc)
	err = svc.ChangePassword(context.Background(), 1, "jti-1", &model.ChangePasswordReq{
		OldPassword: "wrong", NewPassword: "NewPass0!",
	})
	assert.Equal(t, 40101, codeOf(err)) // ErrBadCredentials
	require.NoError(t, e.mock.ExpectationsWereMet())
}
