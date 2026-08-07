package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"nanocloud/internal/middleware"
	"nanocloud/internal/pkg/resp"
	"nanocloud/internal/service"
)

// Content 站点配置/公告/知识库端点。
type Content struct {
	svc *service.ContentService
}

func NewContent(svc *service.ContentService) *Content { return &Content{svc: svc} }

// Config GET /config（免登录）。
func (h *Content) Config(c *gin.Context) {
	data, err := h.svc.SiteConfig(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Notices GET /notices。
func (h *Content) Notices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	list, total, err := h.svc.Notices(c.Request.Context(), page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

// Knowledges GET /knowledges。
func (h *Content) Knowledges(c *gin.Context) {
	language := c.DefaultQuery("language", "zh-CN")
	keyword := c.Query("keyword")
	data, err := h.svc.Knowledges(c.Request.Context(), language, keyword)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"groups": data})
}

// KnowledgeDetail GET /knowledges/:id。
func (h *Content) KnowledgeDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return
	}
	data, err := h.svc.KnowledgeDetail(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Server 节点列表端点。
type Server struct {
	svc *service.ServerService
}

func NewServer(svc *service.ServerService) *Server { return &Server{svc: svc} }

// List GET /servers（需鉴权，按套餐可见分组）。
func (h *Server) List(c *gin.Context) {
	data, err := h.svc.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"groups": data})
}
