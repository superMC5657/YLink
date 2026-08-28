package resp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(list any) (*gin.Engine, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/page", func(c *gin.Context) {
		PageOK(c, list, 0, 1, 10)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	r.ServeHTTP(w, req)
	return r, w
}

// PageOK 必须把 nil slice 序列化为 [] 而不是 null:
// 前端以 list.length 判空,null 会让页面渲染中断(列表区转圈卡死)。
func TestPageOKNilSliceBecomesEmptyArray(t *testing.T) {
	var nilList []map[string]any // 模拟 GORM 空查询返回的 nil slice
	_, w := setupRouter(nilList)

	var body struct {
		Code int `json:"code"`
		Data struct {
			List json.RawMessage `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是合法 JSON: %v, body=%s", err, w.Body.String())
	}
	if string(body.Data.List) != "[]" {
		t.Fatalf("nil slice 应序列化为 [], 实际: %s", string(body.Data.List))
	}
}

func TestPageOKNonNilSliceKept(t *testing.T) {
	list := []map[string]any{{"id": int64(1)}}
	_, w := setupRouter(list)

	var body struct {
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if len(body.Data.List) != 1 || body.Data.List[0]["id"] != float64(1) {
		t.Fatalf("非空列表应原样返回, 实际: %s", w.Body.String())
	}
}

func TestPageOKEmptySliceKept(t *testing.T) {
	_, w := setupRouter([]map[string]any{})
	if w.Body.String() == "" {
		t.Fatal("无响应")
	}
	var body struct {
		Data struct {
			List json.RawMessage `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if string(body.Data.List) != "[]" {
		t.Fatalf("空切片应序列化为 [], 实际: %s", string(body.Data.List))
	}
}
