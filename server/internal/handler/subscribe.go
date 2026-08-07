package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"ylink/internal/middleware"
	"ylink/internal/pkg/errs"
	"ylink/internal/pkg/resp"
	"ylink/internal/pkg/validate"
	"ylink/internal/service"
)

// Subscribe 订阅端点。
type Subscribe struct {
	svc *service.SubscribeService
}

func NewSubscribe(svc *service.SubscribeService) *Subscribe { return &Subscribe{svc: svc} }

// UserSubscribe 当前订阅信息
// @Summary 当前订阅信息
// @Tags 订阅
// @Security BearerAuth
// @Produce json
// @Success 200 {object} resp.Body{data=service.SubscribeResp}
// @Router /user/subscribe [get]
func (h *Subscribe) UserSubscribe(c *gin.Context) {
	data, err := h.svc.UserSubscribe(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

type resetReq struct {
	Password string `json:"password" binding:"required"`
}

// Reset 重置订阅信息
// @Summary 重置订阅信息
// @Tags 订阅
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object{password=string} true "请求"
// @Success 200 {object} resp.Body{data=object{subscribe_url=string}}
// @Router /user/subscribe/reset [post]
func (h *Subscribe) Reset(c *gin.Context) {
	var req resetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.FailWithCode(c, 40000, "参数校验失败: "+validate.Messages(err))
		return
	}
	url, err := h.svc.ResetSubscribe(c.Request.Context(), middleware.UserID(c), req.Password)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"subscribe_url": *url})
}

// TrafficLogs 流量明细
// @Summary 流量明细
// @Tags 订阅
// @Security BearerAuth
// @Produce json
// @Param from query string true "开始日期"
// @Param to query string true "结束日期"
// @Success 200 {object} resp.Body{data=object{list=[]model.TrafficLog}}
// @Router /user/traffic-logs [get]
func (h *Subscribe) TrafficLogs(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	if !validDate(from) || !validDate(to) {
		resp.FailWithCode(c, 40000, "参数校验失败: from/to 需为 YYYY-MM-DD")
		return
	}
	f, _ := time.Parse("2006-01-02", from)
	t, _ := time.Parse("2006-01-02", to)
	if t.Before(f) || t.Sub(f) > 90*24*time.Hour {
		resp.FailWithCode(c, 40000, "参数校验失败: 查询范围最大 90 天")
		return
	}
	list, err := h.svc.TrafficLogs(c.Request.Context(), middleware.UserID(c), from, to)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// ClientSubscribe 订阅下发（代理客户端直连，不走 envelope）
// @Summary 订阅下发
// @Tags 订阅
// @Produce plain
// @Param token path string true "订阅 token"
// @Param flag query string false "clash/sing-box/v2ray"
// @Success 200 {string} string "配置正文"
// @Router /client/subscribe/{token} [get]
func (h *Subscribe) ClientSubscribe(c *gin.Context) {
	token := c.Param("token")
	flag := c.Query("flag")
	ua := c.GetHeader("User-Agent")
	result, err := h.svc.Generate(c.Request.Context(), token, flag, ua)
	if err != nil {
		e := errs.From(err)
		c.String(e.HTTP, e.Message)
		return
	}
	c.Header("subscription-userinfo", result.UserInfo)
	c.Header("profile-update-interval", "24")
	c.Header("Content-Disposition", `attachment; filename="`+result.Filename+`"`)
	c.Data(200, result.ContentType, result.Content)
}
