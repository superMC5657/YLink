package mailer

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ylink-backend/internal/config"
)

// 缝测试：Send 的传输 deadline。最怕的失败=SMTP 挂起（接受连接不回话）时
// net/smtp 无 deadline 无限阻塞，调用 goroutine 泄漏。

// fakeSMTP 最小 SMTP 会话服务器：应答 EHLO/AUTH/MAIL/RCPT/DATA/QUIT，DATA 正文入通道。
type fakeSMTP struct {
	ln   net.Listener
	data chan string
}

func newFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	f := &fakeSMTP{ln: ln, data: make(chan string, 1)}
	go f.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return f
}

func (f *fakeSMTP) port() int { return f.ln.Addr().(*net.TCPAddr).Port }

func (f *fakeSMTP) serve() {
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

func (f *fakeSMTP) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
	write("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			write("250-fake")
			write("250 AUTH PLAIN")
		case strings.HasPrefix(cmd, "AUTH"):
			write("235 ok")
		case strings.HasPrefix(cmd, "DATA"):
			write("354 go")
			var sb strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(l) == "." {
					select {
					case f.data <- sb.String():
					default:
					}
					write("250 ok")
					break
				}
				sb.WriteString(l)
			}
		case strings.HasPrefix(cmd, "QUIT"):
			write("221 bye")
			return
		default: // MAIL / RCPT 等
			write("250 ok")
		}
	}
}

// 全链路：拨号 → EHLO → AUTH → MAIL/RCPT → DATA（gomail 构建的报文落地）→ QUIT。
func TestSendViaFakeSMTP(t *testing.T) {
	f := newFakeSMTP(t)
	m := New(config.SMTPConfig{Host: "127.0.0.1", Port: f.port(), Username: "u@x.com", Password: "p", FromName: "YLink"})
	require.NoError(t, m.Send("to@x.com", "receipt", "<b>paid</b>"))
	select {
	case body := <-f.data:
		assert.Contains(t, body, "To: to@x.com")
		assert.Contains(t, body, "Subject: receipt")
		assert.Contains(t, body, "<b>paid</b>")
	default:
		t.Fatal("DATA 阶段未收到邮件正文")
	}
}

// SMTP 挂起（接受连接不回 banner）：Send 在 deadline 内返回错误，不无限阻塞。
func TestSendHangServerReturnsByDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close() // 接受但保持静默：smtp.NewClient 等 220 banner 挂起
		}
	}()
	m := New(config.SMTPConfig{Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Username: "u", Password: "p"})
	m.timeout = 300 * time.Millisecond

	start := time.Now()
	err = m.Send("to@x.com", "s", "b")
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second, "必须在 deadline 附近返回，而非无限阻塞")
}
