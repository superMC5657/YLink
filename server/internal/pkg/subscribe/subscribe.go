// Package subscribe 订阅配置生成器：Clash YAML / sing-box JSON / v2ray base64 分享链接。
package subscribe

// Node 订阅节点（脱敏后仅含生成配置所需字段）。
type Node struct {
	Name     string // 节点名（含分组前缀）
	Type     string // trojan / vmess / vless / shadowsocks / hysteria2 / tuic
	Host     string
	Port     int
	Password string // trojan/vless/tuic/hysteria2 密码或 UUID
	Method   string // shadowsocks 加密方式
	Cipher   string // 备用加密字段
	SNI      string
	Network  string // ws/grpc/h2/...
	Security string // tls/reality/...
	Alpn     string
	Path     string
	Rate     float64 // 流量倍率
}

// User 生成订阅所需的用户信息（不含敏感字段）。
type User struct {
	Name          string
	TrafficEnable int64 // 总流量（字节）
	U             int64 // 已用上行
	D             int64 // 已用下行
	ExpiredUnix   int64 // 到期 unix 秒，0=无
	ExpiredText   string
	SpeedLimit    *int
}

// Generator 生成器接口。
type Generator interface {
	Format() string // clash / sing-box / v2ray
	Build(u *User, nodes []Node) ([]byte, error)
}

// HintNode 构造「到期/流量耗尽提示节点」（引导回站续费）。
func HintNode(text string) Node {
	return Node{
		Name:     "⚠ " + text,
		Type:     "trojan",
		Host:     "127.0.0.1",
		Port:     1,
		Password: "invalid-hint-node",
		Rate:     1,
	}
}
