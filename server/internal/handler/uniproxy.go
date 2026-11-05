package handler

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/service"
)

// UniProxy V2Board UniProxy 协议兼容处理端点。
type UniProxy struct {
	svc *service.UniProxyService
}

// NewUniProxy 创建 UniProxy 处理端点。
func NewUniProxy(svc *service.UniProxyService) *UniProxy {
	return &UniProxy{svc: svc}
}

// checkAndSetETag 计算并设置 ETag Header。若客户端 If-None-Match 命中则返回 304 并返回 true。
func checkAndSetETag(c *gin.Context, data []byte) bool {
	etag := fmt.Sprintf("%x", sha1.Sum(data))
	quotedETag := `"` + etag + `"`
	c.Header("ETag", quotedETag)

	ifNoneMatch := c.GetHeader("If-None-Match")
	if ifNoneMatch != "" && (ifNoneMatch == etag || ifNoneMatch == quotedETag || strings.Contains(ifNoneMatch, etag)) {
		c.Status(http.StatusNotModified)
		return true
	}
	return false
}

// Config 节点配置下发
// @Summary 节点配置下发（V2Board UniProxy 协议兼容）
// @Tags UniProxy
// @Produce json
// @Param node_id query int true "节点 ID"
// @Param token query string false "节点密钥"
// @Param node_type query string false "协议类型"
// @Success 200 {object} model.UniProxyConfigResp
// @Failure 400 {object} resp.Body
// @Failure 401 {object} resp.Body
// @Failure 404 {object} resp.Body
// @Router /api/v1/server/UniProxy/config [get]
func (h *UniProxy) Config(c *gin.Context) {
	serverID := middleware.UniProxyServerID(c)
	nodeType := c.Query("node_type")

	data, err := h.svc.GetConfig(c.Request.Context(), serverID, nodeType)
	if err != nil {
		resp.Fail(c, err)
		return
	}

	rawBytes, err := json.Marshal(data)
	if err != nil {
		resp.Fail(c, err)
		return
	}

	if checkAndSetETag(c, rawBytes) {
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", rawBytes)
}

// User 节点有效用户凭证列表下发
// @Summary 节点有效用户凭证列表下发（V2Board UniProxy 协议兼容）
// @Tags UniProxy
// @Produce json
// @Param node_id query int true "节点 ID"
// @Param token query string false "节点密钥"
// @Param node_type query string false "协议类型"
// @Success 200 {object} model.UniProxyUserResp
// @Failure 401 {object} resp.Body
// @Failure 404 {object} resp.Body
// @Router /api/v1/server/UniProxy/user [get]
func (h *UniProxy) User(c *gin.Context) {
	serverID := middleware.UniProxyServerID(c)

	data, err := h.svc.GetUsers(c.Request.Context(), serverID)
	if err != nil {
		resp.Fail(c, err)
		return
	}

	rawBytes, err := json.Marshal(data)
	if err != nil {
		resp.Fail(c, err)
		return
	}

	if checkAndSetETag(c, rawBytes) {
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", rawBytes)
}

// Push 节点增量流量上报
// @Summary 节点增量流量上报（V2Board UniProxy 协议兼容）
// @Tags UniProxy
// @Accept json
// @Produce json
// @Param node_id query int true "节点 ID"
// @Param token query string false "节点密钥"
// @Success 200 {object} model.UniProxyPushResp
// @Failure 400 {object} resp.Body
// @Failure 401 {object} resp.Body
// @Failure 404 {object} resp.Body
// @Router /api/v1/server/UniProxy/push [post]
func (h *UniProxy) Push(c *gin.Context) {
	serverID := middleware.UniProxyServerID(c)

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resp.FailWithCode(c, 40000, "failed to read request body")
		return
	}

	items, err := model.ParseUniProxyPushTraffic(bodyBytes)
	if err != nil {
		resp.FailWithCode(c, 40000, err.Error())
		return
	}

	_, err = h.svc.PushTraffic(c.Request.Context(), serverID, items)
	if err != nil {
		resp.Fail(c, err)
		return
	}

	c.JSON(http.StatusOK, model.UniProxyPushResp{
		Data: true,
	})
}
