package service

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/passwd"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/pkg/subscribe"
	"ylink-backend/internal/repo"
)

// SubscribeService 订阅域：当前订阅、重置、流量明细、配置下发。
type SubscribeService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	cfg   *config.Config
}

func NewSubscribeService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, cfg *config.Config) *SubscribeService {
	return &SubscribeService{db: db, rdb: rdb, repos: repos, cfg: cfg}
}

// SubscribeResp GET /user/subscribe。
type SubscribeResp struct {
	HasSubscription bool           `json:"has_subscription"`
	Plan            *SubscribePlan `json:"plan"`
	ExpiredAt       *time.Time     `json:"expired_at"`
	IsExpired       bool           `json:"is_expired"`
	ExpiredDays     int            `json:"expired_days"`
	TransferEnable  int64          `json:"transfer_enable"`
	U               int64          `json:"u"`
	D               int64          `json:"d"`
	Remaining       int64          `json:"remaining"`
	UsedPercent     int            `json:"used_percent"`
	SpeedLimit      *int           `json:"speed_limit"`
	DeviceLimit     *int           `json:"device_limit"`
	SubscribeURL    string         `json:"subscribe_url"`
}

type SubscribePlan struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// UserSubscribe 当前订阅信息。
func (s *SubscribeService) UserSubscribe(ctx context.Context, userID int64) (*SubscribeResp, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	resp := &SubscribeResp{
		HasSubscription: false,
		SubscribeURL:    s.subscribeURL(user.SubToken),
	}
	if user.PlanID != nil {
		plan, err := s.repos.Plan.GetByID(s.db, *user.PlanID)
		if err == nil {
			resp.Plan = &SubscribePlan{ID: plan.ID, Name: plan.Name}
		}
		now := time.Now()
		if user.ExpiredAt != nil {
			resp.ExpiredAt = user.ExpiredAt
			resp.IsExpired = user.ExpiredAt.Before(now)
			if resp.IsExpired {
				resp.ExpiredDays = 0
			} else {
				resp.ExpiredDays = int(user.ExpiredAt.Sub(now).Hours() / 24)
			}
		}
		resp.TransferEnable = user.TransferEnable
		resp.U = user.U
		resp.D = user.D
		resp.Remaining = user.TransferEnable - user.U - user.D
		if resp.Remaining < 0 {
			resp.Remaining = 0
		}
		if user.TransferEnable > 0 {
			resp.UsedPercent = int(float64(user.U+user.D) / float64(user.TransferEnable) * 100)
		}
		if resp.UsedPercent > 100 {
			resp.UsedPercent = 100
		}
		resp.SpeedLimit = user.SpeedLimit
		resp.DeviceLimit = user.DeviceLimit
		resp.HasSubscription = true
	}
	return resp, nil
}

// ResetSubscribe POST /user/subscribe/reset 重置订阅 token（旧链接立即失效）。
func (s *SubscribeService) ResetSubscribe(ctx context.Context, userID int64, password string) (*string, error) {
	user, err := s.repos.User.GetByID(s.db, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	if !passwd.Verify(user.PasswordHash, password) {
		return nil, errs.ErrBadCredentials
	}
	newToken := newUUID()
	oldToken := user.SubToken
	user.SubToken = newToken
	if err := s.repos.User.Update(s.db, user); err != nil {
		return nil, err
	}
	// 清除旧 token 缓存（userinfo / 限流计数）
	s.rdb.Del(ctx, redispkg.Key("sub", "userinfo", oldToken))
	s.rdb.Del(ctx, redispkg.Key("sub", "rl", oldToken))
	url := s.subscribeURL(newToken)
	return &url, nil
}

func (s *SubscribeService) subscribeURL(token string) string {
	return s.cfg.App.BaseURL + "/api/v1/" + s.cfg.Security.SubscribePathOrDefault() + "/subscribe/" + token
}

// TrafficLogs GET /user/traffic-logs。
func (s *SubscribeService) TrafficLogs(ctx context.Context, userID int64, from, to string) ([]model.TrafficLog, error) {
	list, err := s.repos.Traffic.ListByRange(s.db, userID, from, to)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ---- 订阅下发 ----

// GenerateResult 下发结果。
type GenerateResult struct {
	Content     []byte
	ContentType string
	UserInfo    string // subscription-userinfo 头值
	Filename    string
}

// Generate GET /client/subscribe/{token}：深模块缝线唯一入口，
// 格式分辨、节点构建、提示收口、渲染回退、用户信息均收敛于 subscribe 模块内。
func (s *SubscribeService) Generate(ctx context.Context, token, flag, userAgent string) (*GenerateResult, error) {
	// 独立限流：10 次/分钟/token
	rlKey := redispkg.Key("sub", "rl", token)
	n, _ := s.rdb.Incr(ctx, rlKey).Result()
	if n == 1 {
		s.rdb.Expire(ctx, rlKey, time.Minute)
	}
	if n > 10 {
		return nil, &errs.Error{Code: 42900, Message: "请求过于频繁", HTTP: 429}
	}

	user, err := s.repos.User.GetBySubToken(s.db, token)
	if err != nil {
		return nil, errs.ErrUnauthorized
	}
	if user.IsBanned {
		// 契约 §15：token 无效/用户封禁 → 401 纯文本
		return nil, errs.ErrUnauthorized
	}
	gen := pickGenerator(flag, userAgent)

	var expireUnix int64
	var expireText string
	if user.ExpiredAt != nil {
		expireUnix = user.ExpiredAt.Unix()
		expireText = user.ExpiredAt.Format("2006-01-02")
	}
	subUser := subscribe.NewUser("YLink", user.TransferEnable, user.U, user.D, expireUnix, expireText, user.SpeedLimit)

	var nodes []subscribe.Node
	var hints []string
	// 无订阅 → 仅提示文案，由模块内收口为提示节点
	if user.PlanID == nil {
		hints = []string{"未购买套餐，请回站选购"}
	} else {
		nodes = s.buildNodes(user)
		// 到期或流量用尽 → 注入提示文案
		now := time.Now()
		exhausted := user.TransferEnable > 0 && (user.U+user.D) >= user.TransferEnable
		if (user.ExpiredAt != nil && user.ExpiredAt.Before(now)) || exhausted {
			text := "订阅已到期，请回站续费"
			if exhausted {
				text = "流量已用尽，请回站续费"
			}
			hints = append(hints, text)
		}
	}

	// userinfo 缓存 30s，防客户端高频拉取打库（先于内容生成：F10 模板变量需要）
	ui, err := s.userInfo(ctx, user)
	if err != nil {
		return nil, err
	}
	// F10：自定义订阅模板优先，缺失/渲染失败回退内置模块渲染（验收：模板错误不导致订阅 500）
	content, err := s.renderWithHints(gen, subUser, nodes, hints, s.cfg.App.Name, ui)
	if err != nil {
		return nil, err
	}
	return &GenerateResult{
		Content:     content,
		ContentType: contentType(gen),
		UserInfo:    ui,
		Filename:    "YLink",
	}, nil
}

// renderSubscription 历史缝线 adapter：兼容既有调用，内部经由模块 Renderer 单缝线回退。
func (s *SubscribeService) renderSubscription(gen subscribe.Generator, u *subscribe.User, nodes []subscribe.Node, siteName, userInfo string) ([]byte, error) {
	return s.renderWithHints(gen, u, nodes, nil, siteName, userInfo)
}

// renderWithHints 渲染 adapter：取自定义模板文本后经由模块 Renderer 一次调用完成，
// 回退只在模块内发生一次，adapter 侧对 fellBack 做一次 warn。
func (s *SubscribeService) renderWithHints(gen subscribe.Generator, u *subscribe.User, nodes []subscribe.Node, hints []string, siteName, userInfo string) ([]byte, error) {
	format := gen.Format()
	custom, err := (repo.SubscriptionTemplateRepo{}).Get(s.db, format)
	customContent := ""
	if err == nil && custom != nil {
		customContent = custom.Content
	}
	out, fellBack, rerr := subscribe.Render(subscribe.RenderInput{
		Format:         format,
		User:           u,
		Nodes:          nodes,
		Hints:          hints,
		SiteName:       siteName,
		UserInfo:       userInfo,
		CustomTemplate: customContent,
	})
	if rerr != nil {
		return nil, rerr
	}
	if fellBack {
		logger.L().Warn("自定义订阅模板渲染失败，回退内置模块渲染", zapS("format", format))
	}
	return out, nil
}

// buildNodes 组装用户套餐可见节点 adapter：取库行后经由模块 BuildNodes 转换，
// 配置解析失败的行显式丢弃并记 warn，不再静默下发空加密方式。
func (s *SubscribeService) buildNodes(user *model.User) []subscribe.Node {
	if user.PlanID == nil {
		return nil
	}
	plan, err := s.repos.Plan.GetByID(s.db, *user.PlanID)
	if err != nil {
		return nil
	}
	var groupIDs []int64
	if err := json.Unmarshal([]byte(plan.GroupIDs), &groupIDs); err != nil || len(groupIDs) == 0 {
		logger.L().Warn("套餐分组解析失败，返回空节点", zapS("plan", plan.Name), zapE(err))
		return nil
	}
	servers, err := s.repos.Server.ListByGroupIDs(s.db, groupIDs)
	if err != nil {
		return nil
	}
	groups, _ := s.repos.Server.ListGroups(s.db)
	groupName := map[int64]string{}
	for _, g := range groups {
		groupName[g.ID] = g.Name
	}
	specs := make([]subscribe.ServerSpec, 0, len(servers))
	for _, srv := range servers {
		specs = append(specs, subscribe.ServerSpec{
			Name:      srv.Name,
			Type:      srv.Type,
			Host:      srv.Host,
			Port:      srv.Port,
			Config:    srv.Config,
			Rate:      srv.Rate,
			GroupName: groupName[srv.GroupID],
		})
	}
	// 每用户独立凭证(契约 §15):同一节点对不同用户下发不同凭证,节点按此归因流量
	res := subscribe.BuildNodes(specs, user.UUID)
	if res.Dropped > 0 {
		logger.L().Warn("节点配置解析失败，已显式丢弃", zapS("dropped", strconv.Itoa(res.Dropped)))
	}
	return res.Nodes
}

// userInfo 读取/生成 subscription-userinfo（读缓存 30s），装配收口于模块内。
func (s *SubscribeService) userInfo(ctx context.Context, user *model.User) (string, error) {
	cacheKey := redispkg.Key("sub", "userinfo", user.SubToken)
	if v, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil && v != "" {
		return v, nil
	}
	expire := int64(0)
	if user.ExpiredAt != nil {
		expire = user.ExpiredAt.Unix()
	}
	ui := subscribe.FormatUserInfo(user.U, user.D, user.TransferEnable, expire)
	s.rdb.Set(ctx, cacheKey, ui, 30*time.Second)
	return ui, nil
}

// toNode 历史缝线 adapter：将 servers 行转换为订阅节点;cred 为每用户凭证(users.uuid)。
// 解析失败时返回空加密方式的显式回退；新链路经由模块 BuildNodes 显式丢弃，不再走此静默路径。
func toNode(srv model.Server, groupName, cred string) subscribe.Node {
	n, err := subscribe.NodeFromSpec(subscribe.ServerSpec{
		Name:      srv.Name,
		Type:      srv.Type,
		Host:      srv.Host,
		Port:      srv.Port,
		Config:    srv.Config,
		Rate:      srv.Rate,
		GroupName: groupName,
	}, cred)
	if err != nil {
		name := srv.Name
		if groupName != "" {
			name = groupName + " " + srv.Name
		}
		return subscribe.Node{
			Name:     name,
			Type:     srv.Type,
			Host:     srv.Host,
			Port:     srv.Port,
			Password: cred,
			Rate:     srv.Rate,
		}
	}
	return n
}

// pickGenerator 按 flag 或 UA 嗅探选择生成器 adapter，收口于模块 ResolveFormat。
func pickGenerator(flag, ua string) subscribe.Generator {
	return subscribe.GeneratorFor(subscribe.ResolveFormat(flag, ua))
}

func contentType(gen subscribe.Generator) string {
	return subscribe.ContentTypeFor(gen.Format())
}

func newUUID() string {
	return uuid.NewString()
}
