package handler

import (
	"github.com/gin-gonic/gin"

	"nanocloud/internal/middleware"
	"nanocloud/internal/model"
	"nanocloud/internal/pkg/resp"
	"nanocloud/internal/pkg/validate"
	"nanocloud/internal/service"
)

// User 用户端点。
type User struct {
	svc *service.UserService
}

func NewUser(svc *service.UserService) *User { return &User{svc: svc} }

// Stat GET /user/stat
func (h *User) Stat(c *gin.Context) {
	data, err := h.svc.Stat(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// UpdateProfile PUT /user/profile
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

// ChangePassword POST /user/password/change
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
