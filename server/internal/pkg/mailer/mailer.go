// Package mailer 封装 SMTP 邮件发送：gomail 构建报文 + 自管 net/smtp 传输。
// 传输层带绝对 deadline（默认 30s）：SMTP 挂起时 Send 在 deadline 内返回错误，
// 调用 goroutine 不因 net/smtp 无 deadline 而无限阻塞。
package mailer

import (
	"bytes"
	"crypto/tls"
	"html/template"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gomail.v2"

	"ylink-backend/internal/config"
)

// sendTimeout 单次发送的绝对 deadline（拨号后的 TLS 握手 + SMTP 会话全程共享）。
const sendTimeout = 30 * time.Second

// dialTimeout TCP 拨号超时。
const dialTimeout = 10 * time.Second

// Mailer 为邮件发送器。
type Mailer struct {
	host     string
	port     int
	username string
	password string
	fromName string
	// timeout 传输绝对 deadline（默认 sendTimeout；测试可缩短）。
	timeout time.Duration
}

func New(cfg config.SMTPConfig) *Mailer {
	return &Mailer{host: cfg.Host, port: cfg.Port, username: cfg.Username, password: cfg.Password, fromName: cfg.FromName, timeout: sendTimeout}
}

// Send 发送 HTML 邮件到指定收件人。
// gomail 的 DialAndSend 无法注入 conn deadline（包内 netDialTimeout 固定且仅覆盖
// 拨号），故报文仍由 gomail 构建、传输由本包经 SetDeadline 自管。
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

	timeout := m.timeout
	if timeout <= 0 {
		timeout = sendTimeout
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(m.host, strconv.Itoa(m.port)), dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	// 绝对 deadline：TLS 握手与后续 SMTP 读写在超时后统一返回 timeout 错误
	if derr := conn.SetDeadline(time.Now().Add(timeout)); derr != nil {
		return derr
	}
	if m.port == 465 {
		// 465 隐式 SSL/TLS（QQ/163 常用）：先包 TLS 再建 SMTP 会话
		conn = tls.Client(conn, &tls.Config{ServerName: m.host})
	}
	c, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer c.Close()
	if m.port != 465 {
		// 明文端口：服务器支持则升级 STARTTLS
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
				return err
			}
		}
	}
	if m.username != "" {
		if err := c.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return err
		}
	}
	// 信封 MAIL FROM 用 SMTP 登录账号而非收件人可见的 From 头：多数服务商
	// （QQ/163/企业邮）要求信封发件人与认证身份一致，否则拒信或 SPF/DMARC 对齐失败；
	// 可见 From 头（含 fromName 显示名）在上方 SetHeader 单独设置。
	if err := c.Mail(m.username); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := msg.WriteTo(w); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
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
