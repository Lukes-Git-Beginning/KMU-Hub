package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
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

	if err := smtp.SendMail(addr, auth, m.from, []string{toEmail}, msg); err != nil {
		return fmt.Errorf("system mailer: send to %s: %w", toEmail, err)
	}

	slog.Info("password reset email sent", "to", toEmail)
	return nil
}
