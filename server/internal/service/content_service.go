package service

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/repo"
)

// 站点配置默认值兜底
const defaultSiteName = "YLink"

type siteSettings struct {
	SiteName           string             `json:"site_name"`
	SiteLogo           string             `json:"site_logo"`
	SiteDescription    string             `json:"site_description"`
	RegisterEnabled    *bool              `json:"register_enabled"`
	InviteCodeRequired *bool              `json:"invite_code_required"`
	AppDownloads       map[string]string  `json:"app_downloads"`
	Telegram           model.TelegramInfo `json:"telegram"`
	CustomerServiceURL string             `json:"customer_service_url"`
	FreeTrafficTips    string             `json:"free_traffic_tips"`
	Languages          []string           `json:"languages"`
}

type paymentSettings struct {
	Methods []model.PaymentMethod `json:"methods"`
}

type agentSettings struct {
	RequiredValidInvites int      `json:"required_valid_invites"`
	Benefits             []string `json:"benefits"`
	Notes                []string `json:"notes"`
}

type inviteSettings struct {
	CommissionRate int `json:"commission_rate"`
}

// ContentService 站点配置、公告、知识库。
type ContentService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
	set   *SettingService
}

func NewContentService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos, set *SettingService) *ContentService {
	return &ContentService{db: db, rdb: rdb, repos: repos, set: set}
}

// SiteConfig GET /config 组装站点配置（免登录）。
func (s *ContentService) SiteConfig(ctx context.Context) (*model.SiteConfigResp, error) {
	site := siteSettings{SiteName: defaultSiteName, RegisterEnabled: boolPtr(true)}
	_ = s.set.GetJSON(ctx, "site", &site)
	if site.SiteName == "" {
		site.SiteName = defaultSiteName
	}
	reg := true
	if site.RegisterEnabled != nil {
		reg = *site.RegisterEnabled
	}
	inviteReq := false
	if site.InviteCodeRequired != nil {
		inviteReq = *site.InviteCodeRequired
	}
	if site.Languages == nil {
		site.Languages = []string{"zh-CN", "en-US"}
	}
	if site.AppDownloads == nil {
		site.AppDownloads = map[string]string{}
	}

	pay := paymentSettings{}
	_ = s.set.GetJSON(ctx, "payment", &pay)
	if pay.Methods == nil {
		pay.Methods = []model.PaymentMethod{
			{Code: "balance", Name: "余额支付", Icon: "wallet", Enabled: true},
		}
	}

	ag := agentSettings{RequiredValidInvites: 50}
	_ = s.set.GetJSON(ctx, "agent", &ag)

	inv := inviteSettings{CommissionRate: 40}
	_ = s.set.GetJSON(ctx, "invite", &inv)

	return &model.SiteConfigResp{
		SiteName:           site.SiteName,
		SiteLogo:           site.SiteLogo,
		SiteDescription:    site.SiteDescription,
		RegisterEnabled:    reg,
		InviteCodeRequired: inviteReq,
		AppDownloads:       site.AppDownloads,
		Telegram:           site.Telegram,
		CustomerServiceURL: site.CustomerServiceURL,
		FreeTrafficTips:    site.FreeTrafficTips,
		AgentPolicy: model.AgentPolicyResp{
			RequiredValidInvites: ag.RequiredValidInvites,
			CommissionRate:       inv.CommissionRate,
			Benefits:             ag.Benefits,
			Notes:                ag.Notes,
		},
		PaymentMethods: pay.Methods,
		Languages:      site.Languages,
	}, nil
}

// Notices GET /notices 公告分页。
func (s *ContentService) Notices(ctx context.Context, page, pageSize int) ([]model.Notice, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	return s.repos.Notice.ListByPage(s.db, page, pageSize)
}

// Knowledges GET /knowledges 按分类分组。
func (s *ContentService) Knowledges(ctx context.Context, language, keyword string) ([]model.KnowledgeGroup, error) {
	if language == "" {
		language = "zh-CN"
	}
	list, err := s.repos.Knowledge.List(s.db, language, keyword)
	if err != nil {
		return nil, err
	}
	// 保序分组
	var groups []model.KnowledgeGroup
	index := map[string]int{}
	for _, k := range list {
		idx, ok := index[k.Category]
		if !ok {
			groups = append(groups, model.KnowledgeGroup{Category: k.Category, Items: []model.KnowledgeItem{}})
			idx = len(groups) - 1
			index[k.Category] = idx
		}
		groups[idx].Items = append(groups[idx].Items, model.KnowledgeItem{
			ID: k.ID, Title: k.Title, UpdatedAt: k.UpdatedAt,
		})
	}
	return groups, nil
}

// KnowledgeDetail GET /knowledges/{id}。
func (s *ContentService) KnowledgeDetail(ctx context.Context, id int64) (*model.Knowledge, error) {
	k, err := s.repos.Knowledge.GetByID(s.db, id)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	return k, nil
}

// ---- JSON 辅助 ----

func boolPtr(b bool) *bool { return &b }
