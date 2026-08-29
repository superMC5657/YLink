package admin

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/pkg/resp"
)

// ---- 仪表盘 ----

// Overview 仪表盘统计
// @Summary 管理端仪表盘
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.AdminOverviewResp}
// @Router /admin/stat/overview [get]
func (h *Admin) Overview(c *gin.Context) {
	data, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// ---- 统计报表（F04） ----

// StatOrders 订单统计（F04）
// @Summary 订单/营收/退款日趋势（默认近 30 天，含无数据日补零）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param days query int false "天数（1-365，默认 30）"
// @Success 200 {object} resp.Body{data=model.AdminStatOrdersResp}
// @Router /admin/stat/orders [get]
func (h *Admin) StatOrders(c *gin.Context) {
	data, err := h.svc.StatOrders(c.Request.Context(), statDays(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// StatUsers 用户统计（F04）
// @Summary 注册趋势 + 套餐分布（默认近 30 天）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param days query int false "天数（1-365，默认 30）"
// @Success 200 {object} resp.Body{data=model.AdminStatUsersResp}
// @Router /admin/stat/users [get]
func (h *Admin) StatUsers(c *gin.Context) {
	data, err := h.svc.StatUsers(c.Request.Context(), statDays(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// StatTraffic 流量统计（F04）
// @Summary 用户流量消耗 TopN（时间范围内）+ 节点流量分布 TopN（上报累计）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param days query int false "天数（1-365，默认 30，仅作用于用户 Top）"
// @Success 200 {object} resp.Body{data=model.AdminStatTrafficResp}
// @Router /admin/stat/traffic [get]
func (h *Admin) StatTraffic(c *gin.Context) {
	data, err := h.svc.StatTraffic(c.Request.Context(), statDays(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}
