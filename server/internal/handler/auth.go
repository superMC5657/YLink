package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"nanocloud/internal/middleware"
	"nanocloud/internal/model"
	"nanocloud/internal/pkg/resp"
	"nanocloud/internal/pkg/validate"
	"nanocloud/internal/service"
)

// Auth 认证端点。
type Auth struct {
	svc *service.AuthService
}

func NewAuth(svc *service.AuthService) *Auth { return &Auth{svc: svc} }

// ctxWithIP 将客户端 IP 注入上下文（供验证码 IP 限频）。
func ctxWithIP(c *gin.Context) context.Context {
	return context.WithValue(c.Request.Context(), "client_ip", c.ClientIP())
}

// CaptchaEmail POST /captcha/email
func (h *Auth) CaptchaEmail(c *gin.Context) {
	var req model.CaptchaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.SendCaptcha(ctxWithIP(c), req.Email, req.Type)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OKWithMessage(c, "发送成功", data)
}

// Register POST /auth/register
func (h *Auth) Register(c *gin.Context) {
	var req model.AuthRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.Register(ctxWithIP(c), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Login POST /auth/login
func (h *Auth) Login(c *gin.Context) {
	var req model.AuthLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.Login(ctxWithIP(c), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Refresh POST /auth/refresh
func (h *Auth) Refresh(c *gin.Context) {
	var req model.RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	data, err := h.svc.Refresh(ctxWithIP(c), req.RefreshToken)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

// Forgot POST /auth/forgot
func (h *Auth) Forgot(c *gin.Context) {
	var req model.ForgotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	if err := h.svc.Forgot(ctxWithIP(c), &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// Logout POST /auth/logout
func (h *Auth) Logout(c *gin.Context) {
	if err := h.svc.Logout(c.Request.Context(), middleware.UserID(c), c.GetString("jti")); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
