package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// UniProxyBaseConfig 节点轮询周期配置（秒）。
type UniProxyBaseConfig struct {
	PushInterval int `json:"push_interval"`
	PullInterval int `json:"pull_interval"`
}

// UniProxyRealitySettings Reality 专属传输层配置。
type UniProxyRealitySettings struct {
	Show        bool     `json:"show"`
	Dest        string   `json:"dest,omitempty"`
	ServerNames []string `json:"server_names,omitempty"`
	PrivateKey  string   `json:"private_key,omitempty"`
	ShortIDs    []string `json:"short_ids,omitempty"`
}

// UniProxyTLSSettings TLS 与 Reality 统一配置结构。
type UniProxyTLSSettings struct {
	ServerName      string                   `json:"server_name,omitempty"`
	ServerPort      string                   `json:"server_port,omitempty"`
	ShortID         string                   `json:"short_id,omitempty"`
	PrivateKey      string                   `json:"private_key,omitempty"`
	RealitySettings *UniProxyRealitySettings `json:"reality_settings,omitempty"`
}

// UniProxyConfigData UniProxy 下发的节点核心配置数据。
type UniProxyConfigData struct {
	ServerPort      int                  `json:"server_port"`
	Host            string               `json:"host,omitempty"`
	Network         string               `json:"network,omitempty"`
	NetworkSettings map[string]any       `json:"network_settings,omitempty"`
	TLS             int                  `json:"tls"`
	TLSSettings     *UniProxyTLSSettings `json:"tls_settings,omitempty"`
	Protocol        string               `json:"protocol,omitempty"`
	Flow            string               `json:"flow,omitempty"`
	BaseConfig      UniProxyBaseConfig   `json:"base_config"`

	// Shadowsocks 专属
	Cipher    string `json:"cipher,omitempty"`
	ServerKey string `json:"server_key,omitempty"`

	// Hysteria / Hysteria2 专属
	UpMbps   int    `json:"up_mbps,omitempty"`
	DownMbps int    `json:"down_mbps,omitempty"`
	Obfs     string `json:"obfs,omitempty"`

	// Trojan 专属
	ServerName string `json:"server_name,omitempty"`

	// 规则或路由（可选扩展）
	Routes []any `json:"routes,omitempty"`
}

// UniProxyConfigResp 节点配置响应体：
// 采用双信封设计：顶层字段供官方 V2bX 核心（反序列化到 NodeInfo/VAllssNode）直接解析；
// 同时包含 Data 字段供遵循标准 { "data": ... } 契约的客户端解析。
type UniProxyConfigResp struct {
	UniProxyConfigData
	Data *UniProxyConfigData `json:"data,omitempty"`
}

// NewUniProxyConfigResp 构建双信封配置响应。
func NewUniProxyConfigResp(data UniProxyConfigData) UniProxyConfigResp {
	dataCopy := data
	return UniProxyConfigResp{
		UniProxyConfigData: data,
		Data:               &dataCopy,
	}
}

// UniProxyUserItem 节点有效用户条目。
type UniProxyUserItem struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	SpeedLimit  int64  `json:"speed_limit"`
	DeviceLimit int    `json:"device_limit"`
}

// UniProxyUserData 用户列表包装。
type UniProxyUserData struct {
	Users []UniProxyUserItem `json:"users"`
}

// UniProxyUserResp 节点用户列表响应体：
// 采用双信封设计：顶层包含 users 字段，同时包含 data.users。
type UniProxyUserResp struct {
	Users []UniProxyUserItem `json:"users"`
	Data  *UniProxyUserData  `json:"data,omitempty"`
}

// NewUniProxyUserResp 构建双信封用户响应。
func NewUniProxyUserResp(users []UniProxyUserItem) UniProxyUserResp {
	if users == nil {
		users = []UniProxyUserItem{}
	}
	return UniProxyUserResp{
		Users: users,
		Data:  &UniProxyUserData{Users: users},
	}
}

// UniProxyPushItem V2bX / UniProxy 增量流量上报项。
type UniProxyPushItem struct {
	UserID int64 `json:"user_id"`
	U      int64 `json:"u"`
	D      int64 `json:"d"`
}

// UniProxyPushResp 节点增量流量上报响应体。
type UniProxyPushResp struct {
	Data bool `json:"data"`
}

// ParseUniProxyPushTraffic 解析 UniProxy / V2bX 节点流量上报报文。
// 支持 JSON Array 格式 [{"user_id": 1, "u": .., "d": ..}] 与 Map 格式 {"1001": [u, d]}。
func ParseUniProxyPushTraffic(body []byte) ([]UniProxyPushItem, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return []UniProxyPushItem{}, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var items []UniProxyPushItem
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("invalid json array: %w", err)
		}
		return items, nil
	}

	if strings.HasPrefix(trimmed, "{") {
		var m map[string][]int64
		if err := json.Unmarshal(body, &m); err != nil {
			return nil, fmt.Errorf("invalid json map: %w", err)
		}
		items := make([]UniProxyPushItem, 0, len(m))
		for k, v := range m {
			uid, err := strconv.ParseInt(k, 10, 64)
			if err != nil {
				continue
			}
			var u, d int64
			if len(v) > 0 {
				u = v[0]
			}
			if len(v) > 1 {
				d = v[1]
			}
			items = append(items, UniProxyPushItem{
				UserID: uid,
				U:      u,
				D:      d,
			})
		}
		return items, nil
	}

	return nil, errors.New("unexpected push traffic format: expected array or map")
}

// RawServerConfigHelper 内部解析 Server.Config JSON 结构的辅助模型。
type RawServerConfigHelper struct {
	Network  string `json:"network"`
	Security string `json:"security"`
	SNI      string `json:"sni"`
	Path     string `json:"path"`
	Method   string `json:"method"`
	Cipher   string `json:"cipher"`
	Flow     string `json:"flow"`

	// Hysteria
	UpMbps   int    `json:"up_mbps"`
	DownMbps int    `json:"down_mbps"`
	Obfs     string `json:"obfs"`

	// Reality
	RealitySettings *struct {
		Show        bool     `json:"show"`
		Dest        string   `json:"dest"`
		ServerNames []string `json:"server_names"`
		PrivateKey  string   `json:"private_key"`
		ShortIDs    []string `json:"short_ids"`
	} `json:"reality_settings"`
	Dest        string   `json:"dest"`
	ServerNames []string `json:"server_names"`
	PrivateKey  string   `json:"private_key"`
	ShortIDs    []string `json:"short_ids"`

	// Intervals
	PushInterval int `json:"push_interval"`
	PullInterval int `json:"pull_interval"`

	// Custom network_settings or header
	NetworkSettings map[string]any `json:"network_settings"`
	Headers         map[string]any `json:"headers"`
}

// ParseUniProxyConfig 将 YLink Server 领域模型与 config JSON 解析为 UniProxy 配置。
func ParseUniProxyConfig(s *Server) (*UniProxyConfigData, error) {
	if s == nil {
		return nil, nil
	}

	cfg := &UniProxyConfigData{
		ServerPort: s.Port,
		Host:       s.Host,
		Protocol:   strings.ToLower(strings.TrimSpace(s.Type)),
		BaseConfig: UniProxyBaseConfig{
			PushInterval: 60,
			PullInterval: 60,
		},
	}

	var raw RawServerConfigHelper
	if strings.TrimSpace(s.Config) != "" {
		if err := json.Unmarshal([]byte(s.Config), &raw); err != nil {
			return nil, err
		}
	}

	if raw.PushInterval > 0 {
		cfg.BaseConfig.PushInterval = raw.PushInterval
	}
	if raw.PullInterval > 0 {
		cfg.BaseConfig.PullInterval = raw.PullInterval
	}

	// 协议与网络传输层处理
	cfg.Network = raw.Network
	if cfg.Network == "" {
		if cfg.Protocol == "hysteria" || cfg.Protocol == "hysteria2" {
			cfg.Network = "udp"
		} else {
			cfg.Network = "tcp"
		}
	}

	if raw.NetworkSettings != nil {
		cfg.NetworkSettings = raw.NetworkSettings
	} else if raw.Path != "" || raw.Headers != nil {
		cfg.NetworkSettings = make(map[string]any)
		if raw.Path != "" {
			cfg.NetworkSettings["path"] = raw.Path
		}
		if raw.Headers != nil {
			cfg.NetworkSettings["headers"] = raw.Headers
		}
	}

	// TLS / Reality 处理
	sec := strings.ToLower(strings.TrimSpace(raw.Security))
	isReality := sec == "reality" || raw.RealitySettings != nil || raw.PrivateKey != ""
	isTLS := sec == "tls" || cfg.Protocol == "trojan" || cfg.Protocol == "hysteria" || cfg.Protocol == "hysteria2" || isReality

	sni := raw.SNI
	if sni == "" {
		sni = s.Host
	}

	if isReality {
		cfg.TLS = 1 // V2Board 兼容契约中 reality 节点 tls 为 1 且附带 reality_settings
		reality := &UniProxyRealitySettings{
			Show:        false,
			Dest:        raw.Dest,
			ServerNames: raw.ServerNames,
			PrivateKey:  raw.PrivateKey,
			ShortIDs:    raw.ShortIDs,
		}
		if raw.RealitySettings != nil {
			reality.Show = raw.RealitySettings.Show
			if raw.RealitySettings.Dest != "" {
				reality.Dest = raw.RealitySettings.Dest
			}
			if len(raw.RealitySettings.ServerNames) > 0 {
				reality.ServerNames = raw.RealitySettings.ServerNames
			}
			if raw.RealitySettings.PrivateKey != "" {
				reality.PrivateKey = raw.RealitySettings.PrivateKey
			}
			if len(raw.RealitySettings.ShortIDs) > 0 {
				reality.ShortIDs = raw.RealitySettings.ShortIDs
			}
		}
		if len(reality.ServerNames) == 0 && sni != "" {
			reality.ServerNames = []string{sni}
		}

		shortID := ""
		if len(reality.ShortIDs) > 0 {
			shortID = reality.ShortIDs[0]
		}

		cfg.TLSSettings = &UniProxyTLSSettings{
			ServerName:      sni,
			PrivateKey:      reality.PrivateKey,
			ShortID:         shortID,
			RealitySettings: reality,
		}
	} else if isTLS {
		cfg.TLS = 1
		cfg.TLSSettings = &UniProxyTLSSettings{
			ServerName: sni,
		}
	} else {
		cfg.TLS = 0
	}

	// 协议私有字段处理
	switch cfg.Protocol {
	case "shadowsocks":
		cipher := raw.Method
		if cipher == "" {
			cipher = raw.Cipher
		}
		cfg.Cipher = cipher
	case "trojan":
		cfg.ServerName = sni
	case "hysteria", "hysteria2":
		cfg.UpMbps = raw.UpMbps
		cfg.DownMbps = raw.DownMbps
		cfg.Obfs = raw.Obfs
		cfg.ServerName = sni
	case "vless", "vmess":
		cfg.Flow = raw.Flow
	}

	return cfg, nil
}
