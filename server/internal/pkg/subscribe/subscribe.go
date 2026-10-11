// Package subscribe 订阅渲染深模块：节点构建 / 渲染 / 用户信息集中一处。
//
// 深模块的唯一窄缝是 Renderer interface：一次 Render 调用承载订阅生成，
// 内部收敛格式分辨、节点规格解析、提示收口与模板回退。adapter 侧只保留
// 薄封装（取库行、取缓存、取自定义模板文本），不做渲染决策。
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
	IsHint   bool    // 提示节点标记：由模块内收口构造，渲染侧显式可辨
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

// Generator 生成器接口（历史缝线：保留以兼容既有调用，新增调用走 Renderer）。
type Generator interface {
	Format() string // clash / sing-box / v2ray
	Build(u *User, nodes []Node) ([]byte, error)
}

// HintNode 构造提示节点（历史缝线：保留以兼容既有调用；
// 新链路改传 hints 字符串，由模块内 hintNode 收口构造，不再经由调用方拼假节点）。
func HintNode(text string) Node {
	return hintNode(text)
}
