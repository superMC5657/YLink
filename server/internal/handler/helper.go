package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	pagepkg "ylink-backend/internal/pkg/page"
	"ylink-backend/internal/pkg/resp"
)

// param 解析唯一 owner：handler 与 handler/admin 双份 pageParams/idParam 在此收敛，
// admin 侧仅留薄 adapter。分页钳制唯一 owner 为 pkg/page（ADR-0004，ADR-0001
// PageOK 边界不动）：解析即钳制，service/repo 不再逐 List 手写。

// PageParams 分页查询解析：page/page_size，缺省 1/10，越界回落（标准档）。
func PageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return pagepkg.Clamp(page, pageSize)
}

// PageParamsAudit 审计类分页解析：缺省 20，上限 100（审计日志/重置记录翻阅场景）。
func PageParamsAudit(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return pagepkg.ClampPage(page), pagepkg.Audit.ClampSize(pageSize)
}

// IDParam 路径 id 解析：非法即 40000 短路。
func IDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return 0, false
	}
	return id, true
}

func pageParams(c *gin.Context) (int, int) { return PageParams(c) }
