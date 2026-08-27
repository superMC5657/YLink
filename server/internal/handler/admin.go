package handler

import (
	"encoding/csv"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
	"ylink-backend/internal/service"
)

// Admin 管理端端点（role=admin）。
type Admin struct {
	svc *service.AdminService
}

func NewAdmin(svc *service.AdminService) *Admin { return &Admin{svc: svc} }

func (h *Admin) adminID(c *gin.Context) int64 { return middleware.UserID(c) }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: id 不合法")
		return 0, false
	}
	return id, true
}

func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return page, pageSize
}

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

// ---- 用户 ----

func (h *Admin) ListUsers(c *gin.Context) {
	page, pageSize := pageParams(c)
	keyword := c.Query("keyword")
	list, total, err := h.svc.ListUsers(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

func (h *Admin) UpdateUser(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminUpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateUser(c.Request.Context(), h.adminID(c), id, &req, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// AdjustBalance 调整用户余额（审计）
// @Summary 调整用户余额
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "用户 ID"
// @Param body body model.AdjustBalanceReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/users/{id}/balance [post]
func (h *Admin) AdjustBalance(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdjustBalanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.AdjustBalance(c.Request.Context(), h.adminID(c), id, model.YuanToFen(req.Amount), req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 用户管理增强（F05） ----

// ExportUsersCSV 用户 CSV 导出
// @Summary 用户 CSV 流式导出（按筛选，每批 500 分批写）
// @Tags 管理端
// @Security BearerAuth
// @Produce text/csv
// @Param keyword query string false "邮箱关键字"
// @Success 200 {string} string
// @Router /admin/users/export [get]
func (h *Admin) ExportUsersCSV(c *gin.Context) {
	keyword := c.Query("keyword")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="users_export.csv"`)
	w := c.Writer
	// UTF-8 BOM：Excel 直接打开不乱码
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	header := []string{
		"id", "email", "balance", "commission_balance", "plan", "expired_at",
		"transfer_bytes", "u_bytes", "d_bytes", "created_at", "inviter_email",
	}
	if err := cw.Write(header); err != nil {
		return
	}
	err := h.svc.ExportUsers(c.Request.Context(), keyword, func(rows [][]string) error {
		for _, r := range rows {
			if err := cw.Write(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		cw.Flush()
		return
	}
	// 失败时响应体已写出部分 CSV（无法再改状态码），仅记录日志
	logger.L().Error("admin export users failed", zap.Error(err))
}

// BatchUsers 批量用户操作
// @Summary 批量封禁/解封/调整余额（≤500 人，返回成功数与失败原因）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminBatchUserReq true "请求"
// @Success 200 {object} resp.Body{data=model.AdminBatchUserResp}
// @Router /admin/users/batch [post]
func (h *Admin) BatchUsers(c *gin.Context) {
	var req model.AdminBatchUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.BatchUsers(c.Request.Context(), h.adminID(c), &req, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// SendMail 管理员向用户发送邮件
// @Summary 向指定用户发邮件（≤100 人，写 mail_logs 与审计）
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminSendMailReq true "请求"
// @Success 200 {object} resp.Body{data=model.AdminSendMailResp}
// @Router /admin/users/mail [post]
func (h *Admin) SendMail(c *gin.Context) {
	var req model.AdminSendMailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.SendMail(c.Request.Context(), h.adminID(c), &req, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// ResetUserSubToken 重置用户订阅密钥
// @Summary 管理端重置用户订阅 token（旧链接立即失效，写审计）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param id path int true "用户 ID"
// @Success 200 {object} resp.Body{data=object{subscribe_url=string}}
// @Router /admin/users/{id}/sub-token/reset [post]
func (h *Admin) ResetUserSubToken(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	url, err := h.svc.ResetUserSubToken(c.Request.Context(), h.adminID(c), id, c.ClientIP())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"subscribe_url": url})
}

// ---- 审计日志（F08） ----

// ListAuditLogs 审计日志查询
// @Summary 审计日志分页查询（只读）
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Param admin_id query int false "操作人用户 ID"
// @Param action query string false "动作（如 ban_user/adjust_balance）"
// @Param target query string false "目标（如用户 ID 字符串）"
// @Param from query string false "起始日期 YYYY-MM-DD（含）"
// @Param to query string false "结束日期 YYYY-MM-DD（含）"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数（≤100）"
// @Success 200 {object} resp.Body{data=resp.Page}
// @Router /admin/audit-logs [get]
func (h *Admin) ListAuditLogs(c *gin.Context) {
	page, pageSize := pageParams(c)
	f := service.AuditLogFilter{
		Action: c.Query("action"),
		Target: c.Query("target"),
		From:   c.Query("from"),
		To:     c.Query("to"),
	}
	if v := c.Query("admin_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			resp.FailWithCode(c, 40000, "参数校验失败: admin_id 不合法")
			return
		}
		f.AdminID = &id
	}
	list, total, actions, err := h.svc.ListAuditLogs(c.Request.Context(), f, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
		"actions": actions,
	})
}

// ---- 套餐 ----

func (h *Admin) ListPlans(c *gin.Context) {
	data, err := h.svc.ListAllPlans(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreatePlan(c *gin.Context) {
	var req model.AdminPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdatePlan(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdatePlan(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeletePlan(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeletePlan(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

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

// ---- 订单 ----

func (h *Admin) ListOrders(c *gin.Context) {
	page, pageSize := pageParams(c)
	var status *int
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 && v <= 3 {
			status = &v
		}
	}
	list, total, err := h.svc.ListOrders(c.Request.Context(), status, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

// CloseOrder 关闭订单（待支付）
// @Summary 关闭订单
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param order_no path string true "订单号"
// @Param body body model.RefundReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/orders/{order_no}/close [post]
func (h *Admin) CloseOrder(c *gin.Context) {
	var req model.RefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.CloseOrder(c.Request.Context(), h.adminID(c), c.Param("order_no"), req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Refund 订单退款（佣金回滚 + 审计）
// @Summary 订单退款
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param order_no path string true "订单号"
// @Param body body model.RefundReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/orders/{order_no}/refund [post]
func (h *Admin) Refund(c *gin.Context) {
	var req model.RefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.Refund(c.Request.Context(), h.adminID(c), c.Param("order_no"), req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 优惠券 ----

func (h *Admin) ListCoupons(c *gin.Context) {
	data, err := h.svc.ListAllCoupons(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateCoupon(c *gin.Context) {
	var req model.AdminCouponReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateCoupon(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateCoupon(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminCouponReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateCoupon(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteCoupon(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteCoupon(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 内容 ----

// ListNotices 公告列表（含隐藏）
// @Summary 公告列表
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=[]model.AdminNoticeItem}
// @Router /admin/notices [get]
func (h *Admin) ListNotices(c *gin.Context) {
	data, err := h.svc.ListAllNotices(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateNotice(c *gin.Context) {
	var req model.AdminNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateNotice(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateNotice(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateNotice(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteNotice(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteNotice(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ListKnowledges 知识库列表（含隐藏）
// @Summary 知识库列表
// @Tags 管理端
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=[]model.AdminKnowledgeItem}
// @Router /admin/knowledges [get]
func (h *Admin) ListKnowledges(c *gin.Context) {
	data, err := h.svc.ListAllKnowledges(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateKnowledge(c *gin.Context) {
	var req model.AdminKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.CreateKnowledge(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateKnowledge(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.UpdateKnowledge(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteKnowledge(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteKnowledge(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 工单 ----

func (h *Admin) ListTickets(c *gin.Context) {
	page, pageSize := pageParams(c)
	list, total, err := h.svc.ListTickets(c.Request.Context(), page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

func (h *Admin) TicketDetail(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.GetTicketDetail(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) ReplyTicket(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminReplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.ReplyTicket(c.Request.Context(), h.adminID(c), id, req.Message); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) CloseTicket(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.CloseTicket(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 代理 ----

func (h *Admin) ListAgentApplies(c *gin.Context) {
	page, pageSize := pageParams(c)
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))
	list, total, err := h.svc.ListAgentApplies(c.Request.Context(), status, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
}

func (h *Admin) ApproveAgent(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminApproveReq
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.ReviewAgentApply(c.Request.Context(), h.adminID(c), id, true, req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) RejectAgent(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminApproveReq
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.ReviewAgentApply(c.Request.Context(), h.adminID(c), id, false, req.Remark, c.ClientIP()); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 佣金 ----

func (h *Admin) ListCommissions(c *gin.Context) {
	page, pageSize := pageParams(c)
	var status *int
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = &v
		}
	}
	list, total, err := h.svc.ListCommissions(c.Request.Context(), status, page, pageSize)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.PageOK(c, list, total, page, pageSize)
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

// ---- 设置 ----

func (h *Admin) ListSettings(c *gin.Context) {
	data, err := h.svc.ListSettings(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

// SaveSetting 保存站点设置
// @Summary 保存站点设置
// @Tags 管理端
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.AdminSettingsReq true "请求"
// @Success 200 {object} resp.Body
// @Router /admin/settings [put]
func (h *Admin) SaveSetting(c *gin.Context) {
	var req model.AdminSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.SaveSetting(c.Request.Context(), req.Key, req.Value); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
