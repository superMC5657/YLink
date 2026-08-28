package model

import "time"

// ---- 站点配置 GET /config ----

type SiteConfigResp struct {
	SiteName           string            `json:"site_name"`
	SiteLogo           string            `json:"site_logo"`
	SiteDescription    string            `json:"site_description"`
	PrimaryColor       string            `json:"primary_color"`  // 品牌主色(Hex,空=默认,F19)
	BackgroundUrl      string            `json:"background_url"` // 品牌背景图 URL(空=默认,F19)
	RegisterEnabled    bool              `json:"register_enabled"`
	InviteCodeRequired bool              `json:"invite_code_required"`
	AppDownloads       map[string]string `json:"app_downloads"`
	Telegram           TelegramInfo      `json:"telegram"`
	CustomerServiceURL string            `json:"customer_service_url"`
	FreeTrafficTips    string            `json:"free_traffic_tips"`
	AgentPolicy        AgentPolicyResp   `json:"agent_policy"`
	PaymentMethods     []PaymentMethod   `json:"payment_methods"`
	Languages          []string          `json:"languages"`
}

type TelegramInfo struct {
	GroupURL string `json:"group_url"`
	BotURL   string `json:"bot_url"`
}

type AgentPolicyResp struct {
	RequiredValidInvites int      `json:"required_valid_invites"`
	CommissionRate       int      `json:"commission_rate"`
	Benefits             []string `json:"benefits"`
	Notes                []string `json:"notes"`
}

type PaymentMethod struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Enabled bool   `json:"enabled"`
}

// ---- 知识库 ----

type KnowledgeItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

type KnowledgeGroup struct {
	Category string          `json:"category"`
	Items    []KnowledgeItem `json:"items"`
}

// ---- 节点 ----

type ServerResp struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Rate   float64  `json:"rate"`
	Status int      `json:"status"`
	Tags   []string `json:"tags"`
}

type ServerGroupResp struct {
	Group   string       `json:"group"`
	Servers []ServerResp `json:"servers"`
}
