// Package notify sends operator alert emails over SMTP.
package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

// SMTPConfig holds connection details for one outgoing mailbox. Port 465 is
// treated as implicit TLS (SMTPS); any other port dials plain and upgrades
// with STARTTLS when the server advertises it, which covers the common
// self-host cases (587 for Gmail/Outlook/most providers, 25 for a local relay).
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       string
}

const dialTimeout = 10 * time.Second

type SMTPClient struct {
	cfg SMTPConfig
}

func NewSMTPClient(cfg SMTPConfig) *SMTPClient {
	return &SMTPClient{cfg: cfg}
}

func (c *SMTPClient) NotifySenderMessage(ctx context.Context, message domain.ChatMessage) error {
	subject := fmt.Sprintf("Kick Logs: %s yazdi (#%s)", message.SenderUsername, message.ChannelSlug)
	body := fmt.Sprintf(
		"Kanal: %s\nKullanici: %s\nZaman: %s\nMesaj: %s\n",
		message.ChannelSlug,
		message.SenderUsername,
		message.MessageCreatedAt.UTC().Format(time.RFC3339),
		message.Content,
	)
	return c.send(subject, body)
}

func (c *SMTPClient) send(subject string, body string) error {
	addr := net.JoinHostPort(c.cfg.Host, strconv.Itoa(c.cfg.Port))
	message := buildMIMEMessage(c.cfg.From, c.cfg.To, subject, body)

	var auth smtp.Auth
	if c.cfg.Username != "" {
		auth = smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Host)
	}

	if c.cfg.Port == 465 {
		return sendImplicitTLS(addr, c.cfg.Host, auth, c.cfg.From, c.cfg.To, message)
	}
	return sendSTARTTLS(addr, c.cfg.Host, auth, c.cfg.From, c.cfg.To, message)
}

func sendSTARTTLS(addr string, host string, auth smtp.Auth, from string, to string, message []byte) error {
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("dial smtp %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp handshake %s: %w", addr, err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("starttls %s: %w", addr, err)
		}
	}
	return deliver(client, auth, from, to, message)
}

func sendImplicitTLS(addr string, host string, auth smtp.Auth, from string, to string, message []byte) error {
	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("dial smtps %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp handshake %s: %w", addr, err)
	}
	defer client.Close()
	return deliver(client, auth, from, to, message)
}

func deliver(client *smtp.Client, auth smtp.Auth, from string, to string, message []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}
	return client.Quit()
}

func buildMIMEMessage(from string, to string, subject string, body string) []byte {
	var builder strings.Builder
	builder.WriteString("From: " + from + "\r\n")
	builder.WriteString("To: " + to + "\r\n")
	builder.WriteString("Subject: " + subject + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(body)
	return []byte(builder.String())
}
