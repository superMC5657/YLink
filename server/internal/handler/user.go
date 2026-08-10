package handler

import (
	"github.com/gin-gonic/gin"

	"ylink/internal/middleware"
	"ylink/internal/model"
	"ylink/internal/pkg/resp"
	"ylink/internal/pkg/validate"
	"ylink/internal/service"
)

// User 用户端点。
type User struct {
	svc *service.UserService
}

func NewUser(svc *service.UserService) *User { return &User{svc: svc} }

// Stat 用户信息与仪表板统计
// @Summary 用户仪表板统计
// @Tags 用户
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=model.UserStatResp}
// @Router /user/stat [get]
func (h *User) Stat(c *gin.Context) {
	data, err := h.svc.Stat(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// UpdateProfile 更新通知设置
// @Summary 更新通知设置
// @Tags 用户
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.UpdateProfileReq true "请求"
// @Success 200 {object} resp.Body{data=model.UserProfileResp}
// @Router /user/profile [put]
func (h *User) UpdateProfile(c *gin.Context) {
	var req model.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.UpdateProfile(c.Request.Context(), middleware.UserID(c), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Tags 用户
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.ChangePasswordReq true "请求"
// @Success 200 {object} resp.Body
// @Failure 401 {object} resp.Body
// @Router /user/password/change [post]
func (h *User) ChangePassword(c *gin.Context) {
	var req model.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), middleware.UserID(c), c.GetString("jti"), &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
