package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
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

// Generate GET /client/subscribe/{token}。
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
	// 每用户凭证兜底：迁移前注册的老用户若 uuid 为空则懒生成(正常路径注册即生成)
	if user.UUID == "" {
		user.UUID = newUUID()
		if err := s.repos.User.Update(s.db, user); err != nil {
			return nil, err
		}
	}

	gen := pickGenerator(flag, userAgent)

	subUser := &subscribe.User{
		Name:          "YLink",
		TrafficEnable: user.TransferEnable,
		U:             user.U,
		D:             user.D,
		SpeedLimit:    user.SpeedLimit,
	}
	if user.ExpiredAt != nil {
		subUser.ExpiredUnix = user.ExpiredAt.Unix()
		subUser.ExpiredText = user.ExpiredAt.Format("2006-01-02")
	}

	var nodes []subscribe.Node
	// 无订阅 → 仅提示节点
	if user.PlanID == nil {
		nodes = []subscribe.Node{subscribe.HintNode("未购买套餐，请回站选购")}
	} else {
		nodes = s.buildNodes(user)
		// 到期或流量用尽 → 注入提示节点
		now := time.Now()
		exhausted := user.TransferEnable > 0 && (user.U+user.D) >= user.TransferEnable
		if (user.ExpiredAt != nil && user.ExpiredAt.Before(now)) || exhausted {
			text := "订阅已到期，请回站续费"
			if exhausted {
				text = "流量已用尽，请回站续费"
			}
			nodes = append(nodes, subscribe.HintNode(text))
		}
	}

	content, err := gen.Build(subUser, nodes)
	if err != nil {
		return nil, err
	}

	// userinfo 缓存 30s，防客户端高频拉取打库
	ui, err := s.userInfo(ctx, user)
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

// buildNodes 组装用户套餐可见节点。
func (s *SubscribeService) buildNodes(user *model.User) []subscribe.Node {
	if user.PlanID == nil {
		return nil
	}
	plan, err := s.repos.Plan.GetByID(s.db, *user.PlanID)
	if err != nil {
		return nil
	}
	var groupIDs []int64
	if json.Unmarshal([]byte(plan.GroupIDs), &groupIDs) != nil || len(groupIDs) == 0 {
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
	nodes := make([]subscribe.Node, 0, len(servers))
	for _, srv := range servers {
		// 每用户独立凭证(契约 §15):同一节点对不同用户下发不同凭证,节点按此归因流量
		nodes = append(nodes, toNode(srv, groupName[srv.GroupID], user.UUID))
	}
	return nodes
}

// userInfo 读取/生成 subscription-userinfo（读缓存 30s）。
func (s *SubscribeService) userInfo(ctx context.Context, user *model.User) (string, error) {
	cacheKey := redispkg.Key("sub", "userinfo", user.SubToken)
	if v, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil && v != "" {
		return v, nil
	}
	expire := int64(0)
	if user.ExpiredAt != nil {
		expire = user.ExpiredAt.Unix()
	}
	ui := fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d", user.U, user.D, user.TransferEnable, expire)
	s.rdb.Set(ctx, cacheKey, ui, 30*time.Second)
	return ui, nil
}

// toNode 将 servers 行转换为订阅节点;cred 为每用户凭证(users.uuid)。
// 仅当节点 config 显式开启 per_user_credentials 时使用,否则保留 config 中的共享凭证,
// 避免存量节点 inbound 尚未配发每用户凭证时订阅刷新即断连。
func toNode(srv model.Server, groupName, cred string) subscribe.Node {
	var conf struct {
		Password           string `json:"password"`
		UUID               string `json:"uuid"`
		ID                 string `json:"id"`
		Method             string `json:"method"`
		Cipher             string `json:"cipher"`
		SNI                string `json:"sni"`
		Network            string `json:"network"`
		Security           string `json:"security"`
		Alpn               string `json:"alpn"`
		Path               string `json:"path"`
		PerUserCredentials bool   `json:"per_user_credentials"`
	}
	_ = json.Unmarshal([]byte(srv.Config), &conf)
	name := srv.Name
	if groupName != "" {
		name = groupName + " " + srv.Name
	}
	if !conf.PerUserCredentials {
		// 存量节点默认使用 config 共享凭证;升级时先配发 inbound,再开启 per_user_credentials。
		cred = conf.Password
		if cred == "" {
			cred = conf.UUID
		}
		if cred == "" {
			cred = conf.ID
		}
	}
	method := conf.Method
	if method == "" {
		method = conf.Cipher
	}
	return subscribe.Node{
		Name:     name,
		Type:     srv.Type,
		Host:     srv.Host,
		Port:     srv.Port,
		Password: cred,
		Method:   method,
		SNI:      conf.SNI,
		Network:  conf.Network,
		Security: conf.Security,
		Alpn:     conf.Alpn,
		Path:     conf.Path,
		Rate:     srv.Rate,
	}
}

// pickGenerator 按 flag 或 UA 嗅探选择生成器。
func pickGenerator(flag, ua string) subscribe.Generator {
	ua = strings.ToLower(ua)
	switch flag {
	case "clash":
		return subscribe.Clash{}
	case "sing-box":
		return subscribe.SingBox{}
	case "v2ray":
		return subscribe.V2ray{}
	}
	switch {
	case strings.Contains(ua, "clash"):
		return subscribe.Clash{}
	case strings.Contains(ua, "sing-box"):
		return subscribe.SingBox{}
	default:
		return subscribe.V2ray{}
	}
}

func contentType(gen subscribe.Generator) string {
	switch gen.Format() {
	case "clash":
		return "text/yaml; charset=utf-8"
	case "sing-box":
		return "application/json; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

func newUUID() string {
	return uuid.NewString()
}
