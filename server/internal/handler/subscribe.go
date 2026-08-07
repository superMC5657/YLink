package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"nanocloud/internal/middleware"
	"nanocloud/internal/pkg/errs"
	"nanocloud/internal/pkg/resp"
	"nanocloud/internal/pkg/validate"
	"nanocloud/internal/service"
)

// Subscribe 订阅端点。
type Subscribe struct {
	svc *service.SubscribeService
}

func NewSubscribe(svc *service.SubscribeService) *Subscribe { return &Subscribe{svc: svc} }

// UserSubscribe GET /user/subscribe
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

// Reset POST /user/subscribe/reset
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

// TrafficLogs GET /user/traffic-logs?from=&to=（范围最大 90 天）
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

// ClientSubscribe GET /client/subscribe/{token}（代理客户端直连，不走 envelope）
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
