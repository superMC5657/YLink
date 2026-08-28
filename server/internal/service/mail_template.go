package service

import (
	"context"
	"sort"
	"time"

	"gorm.io/gorm"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	"ylink-backend/internal/pkg/mailer"
	"ylink-backend/internal/repo"
)

// ---- 邮件模板（F11） ----

// mailTemplateDef 内置邮件模板定义（subject/body 为 Go template 语法；
// body 为正文片段，发送时统一经 mailer.Template 品牌外壳包裹）。
type mailTemplateDef struct {
	Subject      string
	Body         string
	Placeholders []string
	Remark       string
}

// builtinMailTemplates 内置模板注册表：name 与发送侧 renderMailTemplate 调用点一一对应。
// 占位符统一 {{.xxx}}，{{.site_name}} 由 mailer.Render 注入、全部模板可用。
var builtinMailTemplates = map[string]mailTemplateDef{
	"captcha": {
		Subject:      "[{{.site_name}}] 邮箱验证码",
		Body:         `您的验证码是 <b style="font-size:22px;color:#4f46e5">{{.code}}</b>，10 分钟内有效。若非本人操作请忽略本邮件。`,
		Placeholders: []string{"{{.site_name}}", "{{.code}}"},
		Remark:       "注册/找回密码验证码邮件",
	},
	"expire_remind": {
		Subject:      "[{{.site_name}}] 订阅到期提醒",
		Body:         `您的订阅将于 <b>{{.expire_date}}</b> 到期，请及时续费以免影响使用。`,
		Placeholders: []string{"{{.site_name}}", "{{.expire_date}}"},
		Remark:       "到期前 3 天与 1 天各发送一次（每日 10:00）",
	},
	"traffic_remind": {
		Subject:      "[{{.site_name}}] 流量使用提醒",
		Body:         `您的流量已使用 <b>{{.percent}}%</b>，请注意剩余流量。`,
		Placeholders: []string{"{{.site_name}}", "{{.percent}}"},
		Remark:       "用量 ≥80% 时发送（每日 10:00）",
	},
}

// renderMail 渲染 subject 与品牌外壳包裹后的正文。
func renderMail(subjectTpl, bodyTpl, siteName string, vars map[string]string) (string, string, error) {
	subject, err := mailer.Render(subjectTpl, siteName, vars)
	if err != nil {
		return "", "", err
	}
	body, err := mailer.Render(mailer.Template(bodyTpl), siteName, vars)
	if err != nil {
		return "", "", err
	}
	return subject, body, nil
}

// renderMailTemplate 渲染系统邮件：自定义模板（mail_templates 表）优先；
// 自定义渲染失败回退内置文案（spec F11 验收要点：模板缺失/错误不阻断发送）。
// 返回 subject 与已包裹品牌外壳的 HTML 正文。
func renderMailTemplate(db *gorm.DB, siteName, name string, vars map[string]string) (string, string, error) {
	def, ok := builtinMailTemplates[name]
	if !ok {
		return "", "", errs.New(40000, "未知邮件模板")
	}
	subjectTpl, bodyTpl := def.Subject, def.Body
	custom := false
	if mt, err := (repo.MailTemplateRepo{}).Get(db, name); err == nil {
		subjectTpl, bodyTpl, custom = mt.Subject, mt.Body, true
	}
	subject, body, err := renderMail(subjectTpl, bodyTpl, siteName, vars)
	if err != nil && custom {
		subject, body, err = renderMail(def.Subject, def.Body, siteName, vars)
	}
	return subject, body, err
}

// sampleTemplateVars 测试发送用的示例占位符值。
func sampleTemplateVars(name string) map[string]string {
	vars := map[string]string{}
	switch name {
	case "captcha":
		vars["code"] = "123456"
	case "expire_remind":
		vars["expire_date"] = time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	case "traffic_remind":
		vars["percent"] = "85"
	}
	return vars
}

// ---- 管理端 · 邮件模板 CRUD ----

// ListMailTemplates GET /admin/mail-templates：内置模板全量 + 自定义覆盖合并。
func (s *AdminService) ListMailTemplates(ctx context.Context) ([]model.AdminMailTemplateItem, error) {
	names := make([]string, 0, len(builtinMailTemplates))
	for name := range builtinMailTemplates {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]model.AdminMailTemplateItem, 0, len(names))
	for _, name := range names {
		def := builtinMailTemplates[name]
		item := model.AdminMailTemplateItem{
			Name: name, Subject: def.Subject, Body: def.Body, IsCustom: false,
			Placeholders: def.Placeholders, Remark: def.Remark,
		}
		if mt, err := s.repos.MailTemplate.Get(s.db, name); err == nil {
			item.Subject, item.Body, item.IsCustom = mt.Subject, mt.Body, true
			item.UpdatedAt = &mt.UpdatedAt
		}
		out = append(out, item)
	}
	return out, nil
}

// SaveMailTemplate PUT /admin/mail-templates/{name}：保存自定义模板（保存前校验模板可解析，
// 防语法错误导致发送失败；渲染失败另有内置文案回退兜底）。
func (s *AdminService) SaveMailTemplate(ctx context.Context, adminID int64, name string, req *model.AdminMailTemplateReq, ip string) error {
	def, ok := builtinMailTemplates[name]
	if !ok {
		return errs.ErrNotFound
	}
	if _, _, err := renderMail(req.Subject, req.Body, s.cfg.App.Name, sampleTemplateVars(name)); err != nil {
		return errs.New(40000, "模板语法错误，无法解析: "+err.Error())
	}
	mt := &model.MailTemplate{Name: name, Subject: req.Subject, Body: req.Body, UpdatedAt: time.Now()}
	if err := s.repos.MailTemplate.Upsert(s.db, mt); err != nil {
		return err
	}
	return s.audit(s.db, adminID, "edit_mail_template", name, ip, map[string]any{
		"placeholders": def.Placeholders,
	})
}

// ResetMailTemplate DELETE /admin/mail-templates/{name}：删除自定义行，恢复内置默认文案。
func (s *AdminService) ResetMailTemplate(ctx context.Context, adminID int64, name, ip string) error {
	if _, ok := builtinMailTemplates[name]; !ok {
		return errs.ErrNotFound
	}
	if err := s.repos.MailTemplate.Delete(s.db, name); err != nil {
		return err
	}
	return s.audit(s.db, adminID, "reset_mail_template", name, ip, map[string]any{})
}

// TestMailTemplate POST /admin/mail-templates/{name}/test：按当前模板（自定义或默认）渲染
// 示例内容并走真实 SMTP 发送，失败原因原样返回供前端展示。
func (s *AdminService) TestMailTemplate(ctx context.Context, adminID int64, name, toEmail, ip string) error {
	if _, ok := builtinMailTemplates[name]; !ok {
		return errs.ErrNotFound
	}
	subject, body, err := renderMailTemplate(s.db, s.cfg.App.Name, name, sampleTemplateVars(name))
	if err != nil {
		return err
	}
	if s.ml == nil {
		return errs.New(40000, "邮件服务未配置，无法测试发送")
	}
	sendErr := s.ml.Send(toEmail, subject, body)
	detail := map[string]any{"to": toEmail}
	if sendErr != nil {
		detail["error"] = sendErr.Error()
	}
	_ = s.audit(s.db, adminID, "test_mail_template", name, ip, detail)
	return sendErr
}
