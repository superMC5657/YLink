package subscribe

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// V2ray base64 分享链接列表生成器（每行一个链接，整体 base64）。
type V2ray struct{}

func (V2ray) Format() string { return "v2ray" }

func (V2ray) Build(u *User, nodes []Node) ([]byte, error) {
	links := make([]string, 0, len(nodes))
	for _, n := range nodes {
		links = append(links, shareLink(n))
	}
	raw := strings.Join(links, "\n")
	return []byte(base64.StdEncoding.EncodeToString([]byte(raw))), nil
}

// shareLink 单节点分享链接。
func shareLink(n Node) string {
	name := url.QueryEscape(n.Name)
	switch n.Type {
	case "trojan":
		q := url.Values{}
		if n.SNI != "" {
			q.Set("sni", n.SNI)
		}
		q.Set("allowInsecure", "1")
		u := "trojan://" + n.Password + "@" + n.Host + ":" + fmt.Sprint(n.Port)
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
		return u + "#" + name
	case "vmess":
		vm := map[string]any{
			"v":    "2",
			"ps":   n.Name,
			"add":  n.Host,
			"port": fmt.Sprint(n.Port),
			"id":   n.Password,
			"aid":  "0",
			"net":  n.Network,
			"tls":  n.Security,
			"host": n.SNI,
		}
		b, _ := json.Marshal(vm)
		return "vmess://" + base64.StdEncoding.EncodeToString(b)
	case "vless":
		q := url.Values{}
		if n.Network != "" {
			q.Set("type", n.Network)
			q.Set("encryption", "none")
		}
		if n.SNI != "" {
			q.Set("sni", n.SNI)
		}
		if n.Security != "" {
			q.Set("security", n.Security)
		}
		if n.Path != "" {
			q.Set("path", n.Path)
		}
		u := "vless://" + n.Password + "@" + n.Host + ":" + fmt.Sprint(n.Port)
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
		return u + "#" + name
	case "shadowsocks":
		method := n.Method
		if method == "" {
			method = "aes-256-gcm"
		}
		userinfo := base64.StdEncoding.EncodeToString([]byte(method + ":" + n.Password))
		return "ss://" + userinfo + "@" + n.Host + ":" + fmt.Sprint(n.Port) + "#" + name
	case "hysteria2":
		q := url.Values{}
		if n.SNI != "" {
			q.Set("sni", n.SNI)
		}
		q.Set("insecure", "1")
		u := "hysteria2://" + n.Password + "@" + n.Host + ":" + fmt.Sprint(n.Port)
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
		return u + "#" + name
	case "tuic":
		q := url.Values{}
		if n.SNI != "" {
			q.Set("sni", n.SNI)
		}
		q.Set("congestion_control", "bbr")
		u := "tuic://" + n.Password + "@" + n.Host + ":" + fmt.Sprint(n.Port)
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
		return u + "#" + name
	}
	return ""
}
