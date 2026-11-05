package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/repo"
	"ylink-backend/internal/service"
)

func newUniProxyHandlerTestEnv(t *testing.T) (*gin.Engine, *UniProxy, sqlmock.Sqlmock, *redis.Client) {
	gin.SetMode(gin.TestMode)
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := service.NewUniProxyService(db, rdb, &repo.Repos{})
	h := NewUniProxy(svc)

	r := gin.New()
	// 模拟已通过鉴权并注入 server_id=1
	r.Use(func(c *gin.Context) {
		c.Set("server_id", int64(1))
		c.Set("uniproxy_server_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/server/UniProxy/config", h.Config)
	r.GET("/api/v1/server/UniProxy/user", h.User)
	r.POST("/api/v1/server/UniProxy/push", h.Push)

	return r, h, mock, rdb
}

func serverRow(s *model.Server) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "group_id", "name", "type", "host", "port", "config", "rate", "tags", "status", "is_show", "sort", "node_key",
	}).AddRow(s.ID, s.GroupID, s.Name, s.Type, s.Host, s.Port, s.Config, s.Rate, s.Tags, s.Status, s.IsShow, s.Sort, s.NodeKey)
}

func planRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "group_ids", "is_show"}).AddRow(int64(10), "[1]", true)
}

func userRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "uuid", "speed_limit", "device_limit"}).
		AddRow(int64(101), "uuid-101", 100, 2)
}

func pushUserRows() *sqlmock.Rows {
	future := time.Now().Add(time.Hour)
	return sqlmock.NewRows([]string{"id", "uuid", "plan_id", "expired_at", "sub_token", "transfer_enable"}).
		AddRow(int64(101), "uuid-101", int64(10), future, "sub-token-101", int64(10000000))
}

func TestUniProxyHandler_Config(t *testing.T) {
	r, _, mock, _ := newUniProxyHandlerTestEnv(t)

	srv := &model.Server{
		ID:       1,
		GroupID:  1,
		Name:     "HK-01",
		Type:     "vless",
		Host:     "hk.example.com",
		Port:     443,
		Config:   `{"network":"tcp","security":"reality","sni":"hk.example.com"}`,
		Rate:     1.0,
		Status:   1,
		NodeKey:  "key1",
	}

	// 1. 正常请求
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(srv))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/UniProxy/config?node_type=vless", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	etag := w.Header().Get("ETag")
	assert.NotEmpty(t, etag)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(443), body["server_port"])

	// 2. 携带 If-None-Match 请求命中 304
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(srv))
	req304 := httptest.NewRequest(http.MethodGet, "/api/v1/server/UniProxy/config?node_type=vless", nil)
	req304.Header.Set("If-None-Match", etag)
	w304 := httptest.NewRecorder()
	r.ServeHTTP(w304, req304)

	assert.Equal(t, http.StatusNotModified, w304.Code)
	assert.Empty(t, w304.Body.String())

	// 3. 不兼容的 node_type 请求返回 400
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(srv))
	reqBadType := httptest.NewRequest(http.MethodGet, "/api/v1/server/UniProxy/config?node_type=hysteria2", nil)
	wBadType := httptest.NewRecorder()
	r.ServeHTTP(wBadType, reqBadType)
	assert.Equal(t, http.StatusBadRequest, wBadType.Code)
}

func TestUniProxyHandler_User(t *testing.T) {
	r, _, mock, _ := newUniProxyHandlerTestEnv(t)

	srv := &model.Server{ID: 1, GroupID: 1, Status: 1}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(srv))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).WillReturnRows(planRows())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE plan_id IN`)).WillReturnRows(userRows())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/UniProxy/user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	etag := w.Header().Get("ETag")
	assert.NotEmpty(t, etag)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	users, ok := body["users"].([]any)
	require.True(t, ok)
	assert.Len(t, users, 1)

	// 携带 ETag 测试 304（重新生成 rows）
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(srv))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).WillReturnRows(planRows())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE plan_id IN`)).WillReturnRows(userRows())

	req304 := httptest.NewRequest(http.MethodGet, "/api/v1/server/UniProxy/user", nil)
	req304.Header.Set("If-None-Match", etag)
	w304 := httptest.NewRecorder()
	r.ServeHTTP(w304, req304)

	assert.Equal(t, http.StatusNotModified, w304.Code)
}

func TestUniProxyHandler_Push(t *testing.T) {
	r, _, mock, _ := newUniProxyHandlerTestEnv(t)

	srv := &model.Server{ID: 1, GroupID: 1, Rate: 1.0, Status: 1}

	// 1. Array 格式上报
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(srv))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id IN`)).WillReturnRows(pushUserRows())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).WillReturnRows(planRows())
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "traffic_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	arrayBody := `[{"user_id": 101, "u": 500, "d": 1500}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/server/UniProxy/push", bytes.NewBufferString(arrayBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var respData map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respData))
	assert.Equal(t, true, respData["data"])

	// 2. Map 格式上报（V2bX 原生 {"101": [500, 1500]}）
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "servers"`)).WillReturnRows(serverRow(srv))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id IN`)).WillReturnRows(pushUserRows())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "plans"`)).WillReturnRows(planRows())
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "traffic_logs"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	mapBody := `{"101": [500, 1500]}`
	reqMap := httptest.NewRequest(http.MethodPost, "/api/v1/server/UniProxy/push", bytes.NewBufferString(mapBody))
	reqMap.Header.Set("Content-Type", "application/json")
	wMap := httptest.NewRecorder()
	r.ServeHTTP(wMap, reqMap)

	assert.Equal(t, http.StatusOK, wMap.Code)
	require.NoError(t, json.Unmarshal(wMap.Body.Bytes(), &respData))
	assert.Equal(t, true, respData["data"])

	// 3. 错误格式上报返回 400
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/server/UniProxy/push", bytes.NewBufferString("invalid json"))
	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, reqBad)
	assert.Equal(t, http.StatusBadRequest, wBad.Code)
}
