// 订阅渲染深模块的收口缝线：规格解析、节点构建、提示收口、渲染回退、用户信息。
//
// 模块划分：本文件是唯一的深模块缝线实现；clash / sing-box / v2ray 三个
// 生成器只做格式块拼装，模板数据与回退决策一律走本文件的 Renderer interface。
package subscribe

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ServerSpec 节点规格 adapter 输入：与持久行解耦的纯数据，供模块内构建 Node。
type ServerSpec struct {
	Name      string
	Type      string
	Host      string
	Port      int
	Config    string // servers.config 原文（JSON）
	Rate      float64
	GroupName string
}

// ServerConfig servers.config 的显式结构（method / cipher 双写兼容）。
type ServerConfig struct {
	Method   string `json:"method"`
	Cipher   string `json:"cipher"`
	SNI      string `json:"sni"`
	Network  string `json:"network"`
	Security string `json:"security"`
	Alpn     string `json:"alpn"`
	Path     string `json:"path"`
}

// ParseServerConfig 显式解析节点配置：空串视为无扩展配置；
// 非空但非 JSON 时返回显式错误，由调用方记 warn 并丢弃该行，不再静默回退。
func ParseServerConfig(raw string) (ServerConfig, error) {
	var c ServerConfig
	if strings.TrimSpace(raw) == "" {
		return c, nil
	}
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return ServerConfig{}, fmt.Errorf("解析节点配置失败: %w", err)
	}
	return c, nil
}

// MethodOrCipher 加密方式回退收口于模块内：method 优先，缺失时用 cipher。
func (c ServerConfig) MethodOrCipher() string {
	if c.Method != "" {
		return c.Method
	}
	return c.Cipher
}

// NodeFromSpec 单规格转节点 adapter：凭证恒为每用户 uuid，解析失败返回显式错误。
func NodeFromSpec(sp ServerSpec, cred string) (Node, error) {
	conf, err := ParseServerConfig(sp.Config)
	if err != nil {
		return Node{}, err
	}
	name := sp.Name
	if sp.GroupName != "" {
		name = sp.GroupName + " " + sp.Name
	}
	return Node{
		Name:     name,
		Type:     sp.Type,
		Host:     sp.Host,
		Port:     sp.Port,
		Password: cred,
		Method:   conf.MethodOrCipher(),
		SNI:      conf.SNI,
		Network:  conf.Network,
		Security: conf.Security,
		Alpn:     conf.Alpn,
		Path:     conf.Path,
		Rate:     sp.Rate,
	}, nil
}

// BuildResult 节点构建结果：Nodes 为可用节点，Dropped 为配置解析失败而显式丢弃的行数。
type BuildResult struct {
	Nodes   []Node
	Dropped int
}

// BuildNodes 批量规格转节点：解析失败的行显式丢弃并计数，由调用方一次性 warn。
func BuildNodes(specs []ServerSpec, cred string) BuildResult {
	nodes := make([]Node, 0, len(specs))
	dropped := 0
	for _, sp := range specs {
		n, err := NodeFromSpec(sp, cred)
		if err != nil {
			dropped++
			continue
		}
		nodes = append(nodes, n)
	}
	return BuildResult{Nodes: nodes, Dropped: dropped}
}

// hintNode 提示收口：假 trojan 只在本模块内构造，不再经由模块外拼装后穿过渲染缝线。
func hintNode(text string) Node {
	return Node{
		Name:     "⚠ " + text,
		Type:     "trojan",
		Host:     "127.0.0.1",
		Port:     1,
		Password: "invalid-hint-node",
		Rate:     1,
		IsHint:   true,
	}
}

// NewUser 订阅用户 adapter：由调用方传入的传输量与到期值装配渲染所需 User。
func NewUser(name string, transferEnable, u, d, expireUnix int64, expireText string, speedLimit *int) *User {
	return &User{
		Name:          name,
		TrafficEnable: transferEnable,
		U:             u,
		D:             d,
		ExpiredUnix:   expireUnix,
		ExpiredText:   expireText,
		SpeedLimit:    speedLimit,
	}
}

// FormatUserInfo subscription-userinfo 头值装配收口于模块内。
func FormatUserInfo(upload, download, total, expire int64) string {
	return fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d", upload, download, total, expire)
}

// ResolveFormat 格式分辨收口于模块内：flag 优先，缺失时按 UA 嗅探，默认 v2ray。
func ResolveFormat(flag, ua string) string {
	switch flag {
	case "clash":
		return "clash"
	case "sing-box":
		return "sing-box"
	case "v2ray":
		return "v2ray"
	}
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "clash"):
		return "clash"
	case strings.Contains(ua, "sing-box"):
		return "sing-box"
	default:
		return "v2ray"
	}
}

// GeneratorFor 按格式取生成器 adapter（历史缝线保留，渲染经由 Renderer 时无需调用方分支）。
func GeneratorFor(format string) Generator {
	switch format {
	case "clash":
		return Clash{}
	case "sing-box":
		return SingBox{}
	default:
		return V2ray{}
	}
}

// ContentTypeFor 按格式取下发 Content-Type，收口于模块内。
func ContentTypeFor(format string) string {
	switch format {
	case "clash":
		return "text/yaml; charset=utf-8"
	case "sing-box":
		return "application/json; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

// RenderInput 渲染输入：Hints 为提示文案，由模块内转为提示节点；
// CustomTemplate 为自定义模板原文，空串表示无自定义。
type RenderInput struct {
	Format         string
	User           *User
	Nodes          []Node
	Hints          []string
	SiteName       string
	UserInfo       string
	CustomTemplate string
}

// Renderer 深模块的窄 interface：一次调用承载订阅生成，
// 自定义模板缺失或渲染失败时在模块内回退内置模板，调用方只做一次 warn。
type Renderer interface {
	Render(in RenderInput) (out []byte, fellBack bool, err error)
}

type moduleRenderer struct{}

// DefaultRenderer 模块默认 Renderer 实现，供 adapter 直接复用。
var DefaultRenderer Renderer = moduleRenderer{}

// Render 包级 adapter，委托 DefaultRenderer。
func Render(in RenderInput) ([]byte, bool, error) {
	return DefaultRenderer.Render(in)
}

// Render 按 RenderInput 生成订阅字节：模板数据只构造一次，
// 自定义渲染失败则回退内置模板并置 fellBack，内置失败则返回显式错误。
func (moduleRenderer) Render(in RenderInput) ([]byte, bool, error) {
	nodes := make([]Node, 0, len(in.Nodes)+len(in.Hints))
	nodes = append(nodes, in.Nodes...)
	for _, h := range in.Hints {
		if strings.TrimSpace(h) == "" {
			continue
		}
		nodes = append(nodes, hintNode(h))
	}
	data, err := BuildTemplateData(in.Format, in.User, nodes, in.SiteName, in.UserInfo)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(in.CustomTemplate) == "" {
		out, err := RenderTemplate(in.Format, BuiltinTemplate(in.Format), data)
		return out, false, err
	}
	if out, err := RenderTemplate(in.Format, in.CustomTemplate, data); err == nil {
		return out, false, nil
	} else {
		fb, ferr := RenderTemplate(in.Format, BuiltinTemplate(in.Format), data)
		if ferr != nil {
			return nil, false, ferr
		}
		return fb, true, nil
	}
}

// RenderBuiltin 内置渲染 adapter：三生成器 Build 共用同一缝线，消除三处重复的数据构造。
func RenderBuiltin(format string, u *User, nodes []Node) ([]byte, error) {
	data, err := BuildTemplateData(format, u, nodes, "", "")
	if err != nil {
		return nil, err
	}
	return RenderTemplate(format, BuiltinTemplate(format), data)
}
