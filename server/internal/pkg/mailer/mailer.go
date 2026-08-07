// Package mailer 封装 SMTP 邮件发送（gomail）。
package mailer

import (
	"bytes"
	"html/template"
	"strings"

	"gopkg.in/gomail.v2"

	"nanocloud/internal/config"
)

// Mailer 为邮件发送器。
type Mailer struct {
	host     string
	port     int
	username string
	password string
	fromName string
}

func New(cfg config.SMTPConfig) *Mailer {
	return &Mailer{host: cfg.Host, port: cfg.Port, username: cfg.Username, password: cfg.Password, fromName: cfg.FromName}
}

// Send 发送 HTML 邮件到指定收件人。
func (m *Mailer) Send(to, subject, htmlBody string) error {
	msg := gomail.NewMessage()
	from := m.fromName
	if from == "" {
		from = m.username
	}
	msg.SetHeader("From", msg.FormatAddress(m.username, from))
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlBody)
	d := gomail.NewDialer(m.host, m.port, m.username, m.password)
	return d.DialAndSend(msg)
}

// Render 以站点模板渲染邮件正文（支持 {site_name} {code} 等变量）。
func Render(tpl, siteName string, vars map[string]string) (string, error) {
	var buf bytes.Buffer
	t, err := template.New("mail").Parse(tpl)
	if err != nil {
		return "", err
	}
	data := map[string]string{"site_name": siteName}
	for k, v := range vars {
		data[k] = v
	}
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WrapTpl 生成品牌头 + 内容 + 页脚的通用模板。
func WrapTpl() string {
	return `<!DOCTYPE html><html><body style="margin:0;background:#f5f6f8;font-family:-apple-system,Segoe UI,Roboto,sans-serif">
<div style="max-width:520px;margin:24px auto;background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 2px 12px rgba(0,0,0,.06)">
<div style="padding:24px;background:#4f46e5;color:#fff"><b style="font-size:18px">{{.site_name}}</b></div>
<div style="padding:24px;color:#334155;line-height:1.7">{{.body}}</div>
<div style="padding:16px 24px;border-top:1px solid #eee;color:#94a3b8;font-size:12px">本邮件由 {{.site_name}} 系统自动发送，请勿直接回复。</div>
</div></body></html>`
}

// Template 返回通用正文包裹（Body 变量为 .body）。
func Template(body string) string {
	tpl := WrapTpl()
	return strings.Replace(tpl, "{{.body}}", body, 1)
}
