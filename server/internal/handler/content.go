package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink/internal/middleware"
	"ylink/internal/pkg/resp"
	"ylink/internal/service"
)

// Content 站点配置/公告/知识库端点。
type Content struct {
	svc *service.ContentService
}

func NewContent(svc *service.ContentService) *Content { return &Content{svc: svc} }

// Config 获取站点配置（免登录）
// @Summary 获取站点配置
// @Tags 站点
// @Produce json
// @Success 200 {object} resp.Body{data=model.SiteConfigResp}
// @Router /config [get]
func (h *Content) Config(c *gin.Context) {
	data, err := h.svc.SiteConfig(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Notices 公告列表
// @Summary 公告列表
// @Tags 内容
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} resp.Body{data=resp.Page}
// @Router /notices [get]
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

// Knowledges 知识库列表（按分类分组）
// @Summary 知识库列表
// @Tags 内容
// @Produce json
// @Param language query string false "语言"
// @Param keyword query string false "关键字"
// @Success 200 {object} resp.Body{data=object{groups=[]model.KnowledgeGroup}}
// @Router /knowledges [get]
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

// KnowledgeDetail 知识库详情
// @Summary 知识库详情
// @Tags 内容
// @Produce json
// @Param id path int true "知识 ID"
// @Success 200 {object} resp.Body{data=model.Knowledge}
// @Failure 404 {object} resp.Body
// @Router /knowledges/{id} [get]
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

// List 节点状态列表
// @Summary 节点状态列表
// @Tags 节点
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=object{groups=[]model.ServerGroupResp}}
// @Router /servers [get]
func (h *Server) List(c *gin.Context) {
	data, err := h.svc.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"groups": data})
}
