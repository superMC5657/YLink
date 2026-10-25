package repo

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/model"
)

// paginate 收敛缝测试：Count + 排序分页的语句形状固定，缺省 id 倒序、
// 自定义 orderBy 透传、Count 失败上抛。

func TestListPageDefaultOrder(t *testing.T) {
	db, mock := newRepoTestDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" ORDER BY id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	list, total, err := listPage[model.User](db.Model(&model.User{}), 1, 10, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, int64(7), list[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListPageCustomOrderAndFilter(t *testing.T) {
	db, mock := newRepoTestDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "notices" WHERE is_show = true`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "notices" WHERE is_show = true ORDER BY sort ASC, created_at DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	list, total, err := listPage[model.Notice](db.Model(&model.Notice{}).Where("is_show = true"), 2, 20, "sort ASC, created_at DESC")
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListPageCountError(t *testing.T) {
	db, mock := newRepoTestDB(t)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
		WillReturnError(assert.AnError)

	_, _, err := listPage[model.User](db.Model(&model.User{}), 1, 10, "")
	assert.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
