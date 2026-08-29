package admin

import (
	"encoding/csv"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
	"ylink-backend/internal/service"
)

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
