package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func pageTestContext(t *testing.T, rawQuery string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x?"+rawQuery, nil)
	return c
}

// 缝测试：解析即钳制（分页唯一应用点），各 List 不再自备 case。
// 最怕的失败：新 List 绕过 PageParams 手 parse，导致 0/-1/超大穿透到 SQL。
func TestPageParams(t *testing.T) {
	cases := []struct {
		query string
		page  int
		size  int
	}{
		{"", 1, 10},
		{"page=0&page_size=99", 1, 10},
		{"page=-1&page_size=-5", 1, 10},
		{"page=abc&page_size=xyz", 1, 10},
		{"page=2&page_size=20", 2, 20},
		{"page=1&page_size=50", 1, 50},
		{"page=1&page_size=51", 1, 10},
		{"page=3&page_size=1000000000", 3, 10},
	}
	for _, c := range cases {
		page, size := PageParams(pageTestContext(t, c.query))
		if page != c.page || size != c.size {
			t.Errorf("PageParams(%q) = (%d, %d), want (%d, %d)", c.query, page, size, c.page, c.size)
		}
	}
}

func TestPageParamsAudit(t *testing.T) {
	cases := []struct {
		query string
		page  int
		size  int
	}{
		{"", 1, 20},
		{"page=0&page_size=101", 1, 20},
		{"page=abc", 1, 20},
		{"page=2&page_size=100", 2, 100},
		{"page=1&page_size=51", 1, 51},
		{"page=1&page_size=101", 1, 20},
	}
	for _, c := range cases {
		page, size := PageParamsAudit(pageTestContext(t, c.query))
		if page != c.page || size != c.size {
			t.Errorf("PageParamsAudit(%q) = (%d, %d), want (%d, %d)", c.query, page, size, c.page, c.size)
		}
	}
}
