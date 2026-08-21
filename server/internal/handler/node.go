package handler

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
	"ylink-backend/internal/service"
)

// Node 节点上报端点(模式 A,X-Node-Key 鉴权)。
type Node struct {
	svc *service.NodeService
}

func NewNode(svc *service.NodeService) *Node { return &Node{svc: svc} }

// Users 节点用户同步
// @Summary 节点用户同步(有效订阅用户与凭证)
// @Tags 节点
// @Produce json
// @Success 200 {object} resp.Body{data=service.NodeUsersResp}
// @Router /node/users [get]
func (h *Node) Users(c *gin.Context) {
	data, err := h.svc.Users(c.Request.Context(), middleware.ServerID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Report 节点流量上报
// @Summary 节点流量上报(每用户累计值,幂等)
// @Tags 节点
// @Accept json
// @Produce json
// @Param body body service.NodeReportReq true "请求"
// @Success 200 {object} resp.Body{data=service.NodeReportResp}
// @Router /node/report [post]
func (h *Node) Report(c *gin.Context) {
	var req service.NodeReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.Report(c.Request.Context(), middleware.ServerID(c), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}
