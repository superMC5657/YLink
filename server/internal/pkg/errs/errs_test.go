package errs

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPSStatusMapping(t *testing.T) {
	cases := []struct {
		code int
		want int
	}{
		{0, http.StatusOK},
		{40000, http.StatusBadRequest},
		{40100, http.StatusUnauthorized},
		{40101, http.StatusUnauthorized},
		{40300, http.StatusForbidden},
		{40400, http.StatusNotFound},
		{40900, http.StatusConflict},
		{42900, http.StatusTooManyRequests},
		{50000, http.StatusInternalServerError},
		{10001, http.StatusBadRequest},
		{10003, http.StatusTooManyRequests},
		{11003, http.StatusConflict},
		{14001, http.StatusConflict},
		{15002, http.StatusConflict},
		{12001, http.StatusBadRequest},
	}
	for _, c := range cases {
		e := New(c.code, "x")
		assert.Equal(t, c.want, e.HTTP, "code=%d", c.code)
	}
}

func TestFrom(t *testing.T) {
	assert.Equal(t, 40100, From(ErrUnauthorized).Code)
	// 未知错误统一 50000，不外泄
	assert.Equal(t, 50000, From(assert.AnError).Code)
	assert.Equal(t, "服务器内部错误", From(assert.AnError).Message)
	assert.Equal(t, 0, From(nil).Code)
}
