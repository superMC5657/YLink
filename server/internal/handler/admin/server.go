package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
)

// ---- 节点 ----

func (h *Admin) ListServers(c *gin.Context) {
	data, err := h.svc.ListAllServers(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateServer(c *gin.Context) {
	var req model.AdminServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateServer(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateServer(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateServer(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteServer(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteServer(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ResetNodeKey 重置节点上报密钥（审计 + 旧密钥立即失效）
// @Summary 重置节点上报密钥
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param id path int true "节点 ID"
// @Success 200 {object} resp.Body{data=object{node_key=string}}
// @Router /admin/servers/{id}/node-key/reset [post]
func (h *Admin) ResetNodeKey(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	key, err := h.svc.ResetNodeKey(c.Request.Context(), h.adminID(c), id, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"node_key": key})
}

// ---- 节点批量 / 复制 / 排序（F09） ----

// BatchServers 批量节点操作
// @Summary 批量删除/更新节点公共字段（≤500 个，返回成功数与失败原因）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminBatchServerReq true "请求"
// @Success 200 {object} resp.Body{data=model.AdminBatchServerResp}
// @Router /admin/servers/batch [post]
func (h *Admin) BatchServers(c *gin.Context) {
	var req model.AdminBatchServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.BatchServers(c.Request.Context(), h.adminID(c), &req, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// CopyServer 复制节点
// @Summary 复制节点（全字段复制，名称追加 -copy，重新生成 node_key）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param id path int true "节点 ID"
// @Success 200 {object} resp.Body{data=model.AdminServerView}
// @Router /admin/servers/{id}/copy [post]
func (h *Admin) CopyServer(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.CopyServer(c.Request.Context(), h.adminID(c), id, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// SortServers 节点排序
// @Summary 批量更新节点排序（单事务，按展示顺序传 items）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminSortServerReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/servers/sort [post]
func (h *Admin) SortServers(c *gin.Context) {
	var req model.AdminSortServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.SortServers(c.Request.Context(), h.adminID(c), req.Items, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 节点分组 ----

func (h *Admin) ListServerGroups(c *gin.Context) {
	data, err := h.svc.ListAllServerGroups(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateServerGroup(c *gin.Context) {
	var req model.AdminServerGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateServerGroup(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateServerGroup(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminServerGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateServerGroup(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteServerGroup(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteServerGroup(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 流量 ----

// ImportTraffic 流量导入（模式 B）
// @Summary 流量导入
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.TrafficImportReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/traffic/import [post]
func (h *Admin) ImportTraffic(c *gin.Context) {
	var req model.TrafficImportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.ImportTraffic(c.Request.Context(), h.adminID(c), &req, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ResetTraffic 流量重置（F16）
// @Summary 按用户批量重置流量（清零用量/重新给量，写重置记录与审计；保留节点上报快照防重复计费）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminTrafficResetReq true "请求"
// @Success 200 {object} resp.Body{data=model.AdminTrafficResetResp}
// @Router /admin/traffic/reset [post]
func (h *Admin) ResetTraffic(c *gin.Context) {
	var req model.AdminTrafficResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.ResetTraffic(c.Request.Context(), h.adminID(c), &req, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// ListTrafficResets 流量重置记录（F16）
// @Summary 重置记录分页（可按用户筛选，联表取用户邮箱）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param user_id query int false "用户 ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数（≤100）"
// @Success 200 {object} resp.Body{data=resp.Page}
// @Router /admin/traffic/resets [get]
func (h *Admin) ListTrafficResets(c *gin.Context) {
	page, pageSize := pageParams(c)
	var userID *int64
	if v := c.Query("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			resp.FailWithCode(c, 40000, "参数校验失败: user_id 不合法")
			return
		}
		userID = &id
	}
	list, total, err := h.svc.ListTrafficResets(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}
