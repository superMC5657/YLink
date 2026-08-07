package subscribe

import (
	"encoding/json"
)

// SingBox sing-box JSON 生成器。
type SingBox struct{}

func (SingBox) Format() string { return "sing-box" }

type outbound struct {
	Type       string           `json:"type"`
	Tag        string           `json:"tag"`
	Server     string           `json:"server"`
	ServerPort int              `json:"server_port"`
	Password   string           `json:"password,omitempty"`
	UUID       string           `json:"uuid,omitempty"`
	Method     string           `json:"method,omitempty"`
	TLS        *tlsConfig       `json:"tls,omitempty"`
	Transport  *transportConfig `json:"transport,omitempty"`
	UDP        bool             `json:"udp"`
}

type tlsConfig struct {
	Enabled    bool     `json:"enabled"`
	ServerName string   `json:"server_name,omitempty"`
	Insecure   bool     `json:"insecure"`
	ALPN       []string `json:"alpn,omitempty"`
}

type transportConfig struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

func (SingBox) Build(u *User, nodes []Node) ([]byte, error) {
	outbounds := make([]outbound, 0, len(nodes))
	for _, n := range nodes {
		ob := outbound{Type: n.Type, Tag: n.Name, Server: n.Host, ServerPort: n.Port, UDP: true}
		switch n.Type {
		case "trojan", "hysteria2":
			ob.Password = n.Password
			if n.SNI != "" {
				ob.TLS = &tlsConfig{Enabled: true, ServerName: n.SNI, Insecure: true}
			}
		case "vmess", "vless":
			ob.UUID = n.Password
			if n.SNI != "" || n.Security == "tls" {
				ob.TLS = &tlsConfig{Enabled: true, ServerName: n.SNI, Insecure: true}
				if n.Alpn != "" {
					ob.TLS.ALPN = []string{n.Alpn}
				}
			}
			if n.Network != "" {
				ob.Transport = &transportConfig{Type: n.Network, Path: n.Path}
			}
		case "shadowsocks":
			ob.Method = n.Method
			ob.Password = n.Password
		case "tuic":
			ob.UUID = n.Password
			if n.Password != "" {
				ob.Password = n.Password
			}
			if n.SNI != "" {
				ob.TLS = &tlsConfig{Enabled: true, ServerName: n.SNI, Insecure: true}
			}
		}
		outbounds = append(outbounds, ob)
	}
	doc := map[string]any{
		"log":       map[string]any{"level": "info"},
		"outbounds": outbounds,
	}
	return json.MarshalIndent(doc, "", "  ")
}
