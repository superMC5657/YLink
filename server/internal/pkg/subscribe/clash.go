package subscribe

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Clash Clash YAML 生成器。
type Clash struct{}

func (Clash) Format() string { return "clash" }

func (Clash) Build(u *User, nodes []Node) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("mixed-port: 7890\n")
	buf.WriteString("allow-lan: false\n")
	buf.WriteString("mode: rule\n")
	buf.WriteString("log-level: info\n")
	if u.SpeedLimit != nil && *u.SpeedLimit > 0 {
		buf.WriteString(fmt.Sprintf("limit-speed: %d\n", *u.SpeedLimit*1024*1024/8)) // Mbps → B/s
	}
	buf.WriteString("proxies:\n")
	for _, n := range nodes {
		buf.WriteString(proxyYAML(n))
	}
	return buf.Bytes(), nil
}

func proxyYAML(n Node) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  - name: %s\n", quoteYAML(n.Name)))
	sb.WriteString(fmt.Sprintf("    type: %s\n", n.Type))
	sb.WriteString(fmt.Sprintf("    server: %s\n", n.Host))
	sb.WriteString(fmt.Sprintf("    port: %d\n", n.Port))
	switch n.Type {
	case "trojan":
		sb.WriteString(fmt.Sprintf("    password: %s\n", quoteYAML(n.Password)))
		if n.SNI != "" {
			sb.WriteString(fmt.Sprintf("    sni: %s\n", n.SNI))
		}
		sb.WriteString("    skip-cert-verify: true\n")
		sb.WriteString("    udp: true\n")
	case "vmess":
		sb.WriteString(fmt.Sprintf("    uuid: %s\n", n.Password))
		sb.WriteString("    alterId: 0\n")
		sb.WriteString("    cipher: auto\n")
		if n.SNI != "" {
			sb.WriteString(fmt.Sprintf("    servername: %s\n", n.SNI))
		}
		if n.Network != "" {
			sb.WriteString(fmt.Sprintf("    network: %s\n", n.Network))
		}
		if n.Path != "" {
			sb.WriteString(fmt.Sprintf("    ws-opts:\n      path: %s\n", quoteYAML(n.Path)))
		}
		if n.Security == "tls" {
			sb.WriteString("    tls: true\n")
		}
	case "vless":
		sb.WriteString(fmt.Sprintf("    uuid: %s\n", n.Password))
		if n.SNI != "" {
			sb.WriteString(fmt.Sprintf("    servername: %s\n", n.SNI))
		}
		if n.Network != "" {
			sb.WriteString(fmt.Sprintf("    network: %s\n", n.Network))
		}
		if n.Path != "" {
			sb.WriteString(fmt.Sprintf("    ws-opts:\n      path: %s\n", quoteYAML(n.Path)))
		}
		if n.Security != "" && n.Security != "none" {
			sb.WriteString(fmt.Sprintf("    tls: true\n"))
		}
	case "shadowsocks":
		sb.WriteString(fmt.Sprintf("    cipher: %s\n", n.Method))
		sb.WriteString(fmt.Sprintf("    password: %s\n", quoteYAML(n.Password)))
		sb.WriteString("    udp: true\n")
	case "hysteria2":
		sb.WriteString(fmt.Sprintf("    password: %s\n", quoteYAML(n.Password)))
		if n.SNI != "" {
			sb.WriteString(fmt.Sprintf("    sni: %s\n", n.SNI))
		}
		sb.WriteString("    skip-cert-verify: true\n")
	case "tuic":
		sb.WriteString(fmt.Sprintf("    uuid: %s\n", n.Password))
		if n.Password != "" {
			sb.WriteString(fmt.Sprintf("    password: %s\n", quoteYAML(n.Password)))
		}
		if n.SNI != "" {
			sb.WriteString(fmt.Sprintf("    sni: %s\n", n.SNI))
		}
		if n.Alpn != "" {
			sb.WriteString(fmt.Sprintf("    alpn:\n      - %s\n", n.Alpn))
		}
		sb.WriteString("    udp: true\n")
	}
	return sb.String()
}

func quoteYAML(s string) string {
	if strings.ContainsAny(s, ":#{}[],&*!|>'\"%@` ") {
		return strconv.Quote(s)
	}
	return s
}
