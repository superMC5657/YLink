package repo

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
)

// 缝测试：种子账号创建（EnsureAdmin → EnsureDemoUser）。
// 最怕的失败=users.uuid 零值空串落库——第二个账号撞 uq_users_uuid 唯一约束、
// 两账号订阅凭证缺失（模式 A 流量无法归因）。users.uuid 为 NOT NULL+UNIQUE。

func newRepoTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	return db, mock
}

// expectSeedInsert mock 一次「count=0 → 事务内 INSERT」的种子创建。
func expectSeedInsert(mock sqlmock.Sqlmock, countWhere string, id int64) {
	mock.ExpectQuery(`SELECT count\(\*\) FROM "users" WHERE ` + countWhere).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
	mock.ExpectCommit()
}

// captureUserUUID 经 gorm create 回调捕获即将落库的 User.UUID（SQL 构建前最后的模型态）。
func captureUserUUID(t *testing.T, db *gorm.DB) map[string]string {
	t.Helper()
	captured := map[string]string{}
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("test:capture_seed_uuid", func(tx *gorm.DB) {
		if u, ok := tx.Statement.Dest.(*model.User); ok {
			captured[u.Email] = u.UUID
		}
	}))
	return captured
}

func TestEnsureSeedUsersCarryUUID(t *testing.T) {
	db, mock := newRepoTestDB(t)
	uuids := captureUserUUID(t, db)

	// 全新库启动顺序：admin 先建、demo 后建
	expectSeedInsert(mock, "role", 1)
	expectSeedInsert(mock, "email", 2)
	require.NoError(t, EnsureAdmin(db, "admin@example.com", "Admin@123456"))
	require.NoError(t, EnsureDemoUser(db, "demo@test.com", "Passw0rd"))
	require.NoError(t, mock.ExpectationsWereMet())

	adminUUID, demoUUID := uuids["admin@example.com"], uuids["demo@test.com"]
	for email, u := range map[string]string{"admin": adminUUID, "demo": demoUUID} {
		assert.NotEmpty(t, u, "%s 账号 uuid 不得为空串（NOT NULL 且 UNIQUE，空串第二个账号必撞约束）", email)
		_, err := uuid.Parse(u)
		assert.NoError(t, err, "%s 账号 uuid 应为合法 UUID", email)
	}
	assert.NotEqual(t, adminUUID, demoUUID, "两账号 uuid 必须互不相同")
}

// 幂等语义不回归：已有管理员/演示账号时不再 INSERT。
func TestEnsureSeedUsersIdempotent(t *testing.T) {
	db, mock := newRepoTestDB(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "users" WHERE role`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "users" WHERE email`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	require.NoError(t, EnsureAdmin(db, "admin@example.com", "Admin@123456"))
	require.NoError(t, EnsureDemoUser(db, "demo@test.com", "Passw0rd"))
	require.NoError(t, mock.ExpectationsWereMet())
}
