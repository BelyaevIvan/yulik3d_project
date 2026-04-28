// Package mail — низкоуровневая отправка через SMTP.
//
// Поддерживает:
//   - implicit TLS (SSL on connect, порт 465 у Mail.ru/VK)
//   - STARTTLS (порт 587)
//
// Без сторонних зависимостей — стандартные net/smtp + crypto/tls.
package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPConfig — параметры подключения.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	UseSSL   bool // true для 465, false для 587 (STARTTLS)
}

// FromAddress — адрес «От» для писем.
type FromAddress struct {
	Name  string
	Email string
}

// Format возвращает значение заголовка From: "Name" <email>.
// Имя кодируется в RFC 2047 для поддержки кириллицы.
func (f FromAddress) Format() string {
	if f.Name == "" {
		return f.Email
	}
	encoded := mime.QEncoding.Encode("utf-8", f.Name)
	return fmt.Sprintf("%s <%s>", encoded, f.Email)
}

// Sender — отправляет письма через настроенный SMTP.
type Sender struct {
	cfg  SMTPConfig
	from FromAddress
}

// NewSender создаёт отправителя. Не открывает соединение — оно создаётся на каждый Send.
func NewSender(cfg SMTPConfig, from FromAddress) *Sender {
	return &Sender{cfg: cfg, from: from}
}

// Configured возвращает true, если в конфиге заполнены обязательные поля.
// При false Send вернёт ошибку — это позволяет в dev-окружении запускать бэк
// без SMTP, а письма складывать только в логи воркера.
func (s *Sender) Configured() bool {
	return s.cfg.Host != "" && s.cfg.User != "" && s.cfg.Password != "" && s.from.Email != ""
}

// Message — то, что отдаётся в Send. Тело — пары html/plain.
type Message struct {
	To       string // один получатель достаточно для нашего юзкейса
	Subject  string
	HTMLBody string
	TextBody string
}

// Send — отправляет письмо. Возвращает ошибку при сетевом сбое или отказе SMTP.
// Каркас MIME — multipart/alternative (text/plain + text/html).
func (s *Sender) Send(m Message) error {
	if !s.Configured() {
		return errors.New("smtp not configured")
	}
	if m.To == "" {
		return errors.New("empty recipient")
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)

	dialer := &net.Dialer{Timeout: 15 * time.Second}
	var (
		conn net.Conn
		err  error
	)
	if s.cfg.UseSSL {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.cfg.Host})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Quit() //nolint:errcheck

	if !s.cfg.UseSSL {
		if err := c.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(s.from.Email); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(m.To); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := wc.Write([]byte(s.buildMIME(m))); err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return nil
}

func (s *Sender) buildMIME(m Message) string {
	subject := mime.QEncoding.Encode("utf-8", m.Subject)
	boundary := fmt.Sprintf("yulik3d-%d", time.Now().UnixNano())

	var b strings.Builder
	b.WriteString("From: " + s.from.Format() + "\r\n")
	b.WriteString("To: " + m.To + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
	b.WriteString("\r\n")

	// plain
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(m.TextBody)
	b.WriteString("\r\n")

	// html
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(m.HTMLBody)
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "--\r\n")
	return b.String()
}
