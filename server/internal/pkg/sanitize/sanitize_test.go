package sanitize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	// 富文本：脚本/事件被剥离，基础标签保留
	html := `<p onclick="alert(1)">hello <b>bold</b><script>alert(2)</script><img src=x onerror=alert(3)></p>`
	out := Markdown(html)
	assert.NotContains(t, out, "script")
	assert.NotContains(t, out, "onclick")
	assert.NotContains(t, out, "onerror")
	assert.Contains(t, out, "<b>bold</b>")

	// 纯文本：全部标签剥离
	assert.Equal(t, "hello", Text("<b>hello</b>"))
}
