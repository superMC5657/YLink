// F10 订阅模板：按客户端类型的 Go text/template 全文档模板。
// 节点列表以预渲染块变量注入（NodeBlock/Outbounds/Links），模板作者只重排外围结构、
// 无法破坏节点语法；自定义模板渲染失败由 service 层回退内置生成器。
package subscribe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// TemplateData 订阅模板渲染数据（F10）。
type TemplateData struct {
	SiteName   string // 站点名
	UserInfo   string // subscription-userinfo 头值（upload=..; download=..; total=..; expire=..）
	SpeedLimit int    // 限速 B/s（0=不限速；Clash limit-speed 值）
	NodeCount  int    // 节点数（含提示节点）
	NodeBlock  string // clash：proxies 列表块（已缩进；每行以 \n 结尾，空节点列表为空串）
	Outbounds  string // sing-box：outbounds JSON 数组（嵌入 JSON 文档用）
	Links      string // v2ray：分享链接（换行分隔，整体输出仍经 base64）
}

// builtinClashTemplate 内置 Clash 模板：渲染结果与重构前硬编码 Build 输出逐字节一致。
const builtinClashTemplate = `mixed-port: 7890
allow-lan: false
mode: rule
log-level: info
{{- if .SpeedLimit}}
limit-speed: {{.SpeedLimit}}
{{- end}}
proxies:
{{.NodeBlock}}`

// builtinSingBoxTemplate 内置 sing-box 模板：{{.Outbounds}} 为 MarshalIndent(outbounds,"  ","  ")，
// 嵌入后与重构前硬编码 Build 输出逐字节一致。
const builtinSingBoxTemplate = `{
  "log": {
    "level": "info"
  },
  "outbounds": {{.Outbounds}}
}`

// builtinV2rayTemplate 内置 v2ray 模板：链接列表整体 base64 后下发。
const builtinV2rayTemplate = `{{.Links}}`

// BuiltinTemplate 返回内置模板文本（自定义模板缺失/渲染失败时的回退基准）。
func BuiltinTemplate(format string) string {
	switch format {
	case "clash":
		return builtinClashTemplate
	case "sing-box":
		return builtinSingBoxTemplate
	case "v2ray":
		return builtinV2rayTemplate
	}
	return ""
}

// FormatNames 全部客户端类型（管理端模板列表顺序）。
var FormatNames = []string{"clash", "sing-box", "v2ray"}

// BuildTemplateData 构造指定客户端类型的模板渲染数据。
func BuildTemplateData(format string, u *User, nodes []Node, siteName, userInfo string) (*TemplateData, error) {
	data := &TemplateData{
		SiteName:  siteName,
		UserInfo:  userInfo,
		NodeCount: len(nodes),
	}
	switch format {
	case "clash":
		if u.SpeedLimit != nil && *u.SpeedLimit > 0 {
			data.SpeedLimit = *u.SpeedLimit * 1024 * 1024 / 8 // Mbps → B/s
		}
		var sb strings.Builder
		for _, n := range nodes {
			sb.WriteString(proxyYAML(n))
		}
		data.NodeBlock = sb.String()
	case "sing-box":
		obs, err := buildOutbounds(nodes)
		if err != nil {
			return nil, err
		}
		b, err := json.MarshalIndent(obs, "  ", "  ")
		if err != nil {
			return nil, err
		}
		data.Outbounds = string(b)
	case "v2ray":
		links := make([]string, 0, len(nodes))
		for _, n := range nodes {
			links = append(links, shareLink(n))
		}
		data.Links = strings.Join(links, "\n")
	default:
		return nil, fmt.Errorf("未知订阅格式: %s", format)
	}
	return data, nil
}

// RenderTemplate 用自定义模板渲染订阅内容（v2ray 输出整体 base64，与内置行为一致）。
func RenderTemplate(format, tmplText string, data *TemplateData) ([]byte, error) {
	t, err := template.New("subscription").Parse(tmplText)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	if format == "v2ray" {
		return []byte(base64.StdEncoding.EncodeToString(buf.Bytes())), nil
	}
	return buf.Bytes(), nil
}
