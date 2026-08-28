package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
	"ylink-backend/internal/pkg/validate"
	"ylink-backend/internal/service"
)

// Auth 认证端点。
type Auth struct {
	svc *service.AuthService
}

func NewAuth(svc *service.AuthService) *Auth { return &Auth{svc: svc} }

// ctxWithIP 将客户端 IP 与 User-Agent 注入上下文（供验证码 IP 限频与会话元数据）。
func ctxWithIP(c *gin.Context) context.Context {
	ctx := context.WithValue(c.Request.Context(), "client_ip", c.ClientIP())
	return context.WithValue(ctx, "user_agent", c.Request.UserAgent())
}

// CaptchaEmail 发送邮箱验证码（免登录）
// @Summary 发送邮箱验证码
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body model.CaptchaReq true "请求"
// @Success 200 {object} resp.Body{data=model.CaptchaResp}
// @Failure 400 {object} resp.Body
// @Router /captcha/email [post]
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

// Register 注册（免登录）
// @Summary 注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body model.AuthRegisterReq true "请求"
// @Success 200 {object} resp.Body{data=model.TokenResp}
// @Failure 400 {object} resp.Body
// @Router /auth/register [post]
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

// Login 登录（免登录）
// @Summary 登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body model.AuthLoginReq true "请求"
// @Success 200 {object} resp.Body{data=model.TokenResp}
// @Failure 401 {object} resp.Body
// @Router /auth/login [post]
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

// Refresh 刷新令牌（免登录，body 鉴权）
// @Summary 刷新令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body model.RefreshReq true "请求"
// @Success 200 {object} resp.Body{data=model.TokenResp}
// @Failure 401 {object} resp.Body
// @Router /auth/refresh [post]
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

// Forgot 找回密码（免登录）
// @Summary 找回密码
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body model.ForgotReq true "请求"
// @Success 200 {object} resp.Body
// @Failure 400 {object} resp.Body
// @Router /auth/forgot [post]
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

// Logout 退出登录（需鉴权）
// @Summary 退出登录
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body
// @Router /auth/logout [post]
func (h *Auth) Logout(c *gin.Context) {
	if err := h.svc.Logout(c.Request.Context(), middleware.UserID(c), c.GetString("jti")); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
