package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"time"
)

// smtpDialTimeout bounds the TCP connect; smtpExchangeDeadline bounds the whole
// SMTP exchange. net/smtp.SendMail offers neither, so an unresponsive SMTP
// server would otherwise block the password-reset request handler indefinitely.
const (
	smtpDialTimeout      = 10 * time.Second
	smtpExchangeDeadline = 30 * time.Second
)

// systemMailer implements auth.Mailer using a dedicated system SMTP account.
// All fields are optional: if host is empty the mailer logs instead of sending
// (safe for dev/CI environments without SMTP configured).
type systemMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func (m *systemMailer) SendPasswordResetEmail(_ context.Context, toEmail, toName, resetLink string) error {
	if m.host == "" {
		slog.Info("system SMTP not configured — password reset email suppressed",
			"to", toEmail,
			"reset_link", resetLink,
		)
		return nil
	}

	subject := "Reset your Cosmi password"
	body := fmt.Sprintf("Hello %s,\r\n\r\n"+
		"We received a request to reset your password.\r\n\r\n"+
		"Click the link below to choose a new password (valid for 1 hour):\r\n\r\n"+
		"%s\r\n\r\n"+
		"If you did not request this, you can safely ignore this email.\r\n\r\n"+
		"— The Cosmi Team\r\n",
		toName, resetLink,
	)

	msg := []byte(
		"From: " + m.from + "\r\n" +
			"To: " + toEmail + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	if err := m.sendMail(addr, auth, []string{toEmail}, msg); err != nil {
		return fmt.Errorf("system mailer: send to %s: %w", toEmail, err)
	}

	slog.Info("password reset email sent", "to", toEmail)
	return nil
}

// sendMail delivers a message over STARTTLS (or plain) with a bounded dial
// timeout and exchange deadline. It mirrors net/smtp.SendMail's flow
// (hello → STARTTLS if offered → auth → data) but cannot block forever on an
// unresponsive server.
func (m *systemMailer) sendMail(addr string, auth smtp.Auth, to []string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, smtpDialTimeout)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(smtpExchangeDeadline))

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(m.from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	return client.Quit()
}
