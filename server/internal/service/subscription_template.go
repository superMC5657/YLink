package service

import (
	"context"
	"encoding/base64"
	"time"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/subscribe"
)

// ---- 订阅模板管理（F10） ----

// subTemplateMeta 客户端类型的模板元信息（变量清单与说明，前端展示用）。
var subTemplateMeta = map[string]struct {
	Variables []string
	Remark    string
}{
	"clash": {
		Variables: []string{"{{.SiteName}}", "{{.UserInfo}}", "{{.SpeedLimit}}", "{{.NodeCount}}", "{{.NodeBlock}}"},
		Remark:    "Clash YAML 全文档模板；{{.NodeBlock}} 为预渲染 proxies 节点块（防语法破坏），{{.SpeedLimit}} 为限速 B/s（0=不限）",
	},
	"sing-box": {
		Variables: []string{"{{.SiteName}}", "{{.UserInfo}}", "{{.NodeCount}}", "{{.Outbounds}}"},
		Remark:    "sing-box JSON 全文档模板；{{.Outbounds}} 为预渲染 outbounds JSON 数组",
	},
	"v2ray": {
		Variables: []string{"{{.SiteName}}", "{{.UserInfo}}", "{{.NodeCount}}", "{{.Links}}"},
		Remark:    "v2ray 分享链接模板（{{.Links}} 为换行分隔链接）；渲染结果整体 base64 后下发",
	},
}

// sampleSubNodes 模板校验/预览用示例节点（1 个正常节点 + 1 个提示节点）。
func sampleSubNodes() []subscribe.Node {
	return []subscribe.Node{
		{Name: "示例 香港01", Type: "trojan", Host: "1.2.3.4", Port: 443, Password: "sample-password", SNI: "hk.example.com", Rate: 1},
		subscribe.HintNode("订阅已到期，请回站续费"),
	}
}

// sampleSubUser 模板校验/预览用示例用户（100Mbps 限速）。
func sampleSubUser() *subscribe.User {
	limit := 100
	return &subscribe.User{Name: "YLink", TrafficEnable: 100 << 30, U: 10 << 30, D: 20 << 30, SpeedLimit: &limit}
}

// ListSubscriptionTemplates GET /admin/subscription-templates：内置模板全量 + 自定义覆盖合并。
func (s *AdminService) ListSubscriptionTemplates(ctx context.Context) ([]model.AdminSubscriptionTemplateItem, error) {
	out := make([]model.AdminSubscriptionTemplateItem, 0, len(subscribe.FormatNames))
	for _, name := range subscribe.FormatNames {
		meta := subTemplateMeta[name]
		item := model.AdminSubscriptionTemplateItem{
			Name: name, Content: subscribe.BuiltinTemplate(name), IsCustom: false,
			Variables: meta.Variables, Remark: meta.Remark,
		}
		if st, err := s.repos.SubscriptionTemplate.Get(s.db, name); err == nil {
			item.Content, item.IsCustom = st.Content, true
			item.UpdatedAt = &st.UpdatedAt
		}
		out = append(out, item)
	}
	return out, nil
}

// SaveSubscriptionTemplate PUT /admin/subscription-templates/{name}：保存自定义模板
// （保存前用示例数据渲染校验，防语法错误导致订阅下发异常；渲染失败另有内置生成器回退兜底）。
func (s *AdminService) SaveSubscriptionTemplate(ctx context.Context, adminID int64, name, content, ip string) error {
	if _, ok := subTemplateMeta[name]; !ok {
		return errs.ErrNotFound
	}
	data, err := subscribe.BuildTemplateData(name, sampleSubUser(), sampleSubNodes(), s.cfg.App.Name, "upload=0; download=0; total=0; expire=0")
	if err == nil {
		_, err = subscribe.RenderTemplate(name, content, data)
	}
	if err != nil {
		return errs.New(40000, "模板语法错误，无法渲染: "+err.Error())
	}
	st := &model.SubscriptionTemplate{Name: name, Content: content, UpdatedAt: time.Now()}
	if err := s.repos.SubscriptionTemplate.Upsert(s.db, st); err != nil {
		return err
	}
	return s.audit(s.db, adminID, "edit_subscription_template", name, ip, map[string]any{})
}

// ResetSubscriptionTemplate DELETE /admin/subscription-templates/{name}：删除自定义行，恢复内置生成器。
func (s *AdminService) ResetSubscriptionTemplate(ctx context.Context, adminID int64, name, ip string) error {
	if _, ok := subTemplateMeta[name]; !ok {
		return errs.ErrNotFound
	}
	if err := s.repos.SubscriptionTemplate.Delete(s.db, name); err != nil {
		return err
	}
	return s.audit(s.db, adminID, "reset_subscription_template", name, ip, map[string]any{})
}

// PreviewSubscriptionTemplate POST /admin/subscription-templates/{name}/preview：按当前模板
// （自定义或内置）用示例数据渲染；v2ray 返回 base64 前的模板渲染结果，便于阅读校对。
func (s *AdminService) PreviewSubscriptionTemplate(ctx context.Context, name string) (string, error) {
	if _, ok := subTemplateMeta[name]; !ok {
		return "", errs.ErrNotFound
	}
	content := subscribe.BuiltinTemplate(name)
	if st, err := s.repos.SubscriptionTemplate.Get(s.db, name); err == nil {
		content = st.Content
	}
	data, err := subscribe.BuildTemplateData(name, sampleSubUser(), sampleSubNodes(), s.cfg.App.Name, "upload=0; download=0; total=0; expire=0")
	if err != nil {
		return "", err
	}
	out, err := subscribe.RenderTemplate(name, content, data)
	if err != nil {
		// 自定义模板已损坏：预览按内置模板渲染并提示，与下发回退行为一致
		fallback, ferr := subscribe.RenderTemplate(name, subscribe.BuiltinTemplate(name), data)
		if ferr != nil {
			return "", err
		}
		return string(fallback), nil
	}
	if name == "v2ray" {
		// base64 解回可读文本（base64.StdEncoding 必然可解码）
		if raw, derr := base64.StdEncoding.DecodeString(string(out)); derr == nil {
			return string(raw), nil
		}
	}
	return string(out), nil
}
