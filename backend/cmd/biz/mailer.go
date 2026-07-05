package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"time"

	"github.com/kmuhub/kmuhub/internal/email/send"
)

// smtpDialTimeout bounds the TCP connect; smtpExchangeDeadline bounds the whole
// SMTP exchange. net/smtp.SendMail offers neither, so an unresponsive SMTP
// server would otherwise block the dunning-send request indefinitely.
const (
	smtpDialTimeout      = 10 * time.Second
	smtpExchangeDeadline = 30 * time.Second
)

// systemMailer implements dunning.NoticeSender using a dedicated system SMTP
// account. It mirrors cmd/auth/mailer.go's systemMailer (same dial/STARTTLS
// flow), but builds the message via email/send's MIMEBuilder instead of a
// hand-rolled string — the auth mailer's manual MIME has no attachment
// support, and the dunning notice must carry the PDF as an attachment.
// All fields are optional: if host is empty the mailer logs instead of
// sending (safe for dev/CI environments without SMTP configured).
type systemMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// SendDunningNotice implements dunning.NoticeSender.
func (m *systemMailer) SendDunningNotice(_ context.Context, toEmail, toName string, pdfBytes []byte, filename, subject string, level int) error {
	if m.host == "" {
		slog.Info("system SMTP not configured — dunning notice email suppressed",
			"to", toEmail,
			"filename", filename,
			"level", level,
		)
		return nil
	}

	body := dunningNoticeBody(toName, level)

	msg, err := send.NewMIMEBuilder().Build(send.MIMEInput{
		From:     send.EmailAddress{Name: "Cosmi", Email: m.from},
		To:       []send.EmailAddress{{Name: toName, Email: toEmail}},
		Subject:  subject,
		BodyText: body,
		BodyHTML: "<p>" + body + "</p>",
		Attachments: []send.AttachmentData{
			{
				Filename:    filename,
				ContentType: "application/pdf",
				Data:        bytes.NewReader(pdfBytes),
				Size:        int64(len(pdfBytes)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("build dunning notice mime message: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	if err := m.sendMail(addr, auth, []string{toEmail}, msg); err != nil {
		return fmt.Errorf("system mailer: send dunning notice to %s: %w", toEmail, err)
	}

	slog.Info("dunning notice email sent", "to", toEmail, "level", level, "filename", filename)
	return nil
}

// dunningNoticeBody returns a level-appropriate German body text (friendly ->
// formal -> urgent), matching the PDF's own tone escalation
// (see pdf.buildDunningBody).
func dunningNoticeBody(toName string, level int) string {
	greeting := fmt.Sprintf("Hallo %s,", toName)
	switch level {
	case 1:
		return greeting + "\r\n\r\nanbei erhalten Sie eine freundliche Zahlungserinnerung zu einer offenen Rechnung. Bitte pruefen Sie den Vorgang.\r\n\r\n-- Ihr Cosmi-Team"
	case 2:
		return greeting + "\r\n\r\nanbei erhalten Sie die 1. Mahnung zu einer offenen Rechnung. Bitte begleichen Sie den Betrag zeitnah.\r\n\r\n-- Ihr Cosmi-Team"
	default:
		return greeting + "\r\n\r\nanbei erhalten Sie die 2./letzte Mahnung zu einer offenen Rechnung. Bitte begleichen Sie den Betrag umgehend, um weitere Schritte zu vermeiden.\r\n\r\n-- Ihr Cosmi-Team"
	}
}

// sendMail delivers a message over STARTTLS (or plain) with a bounded dial
// timeout and exchange deadline. Mirrors net/smtp.SendMail's flow
// (hello -> STARTTLS if offered -> auth -> data), same as cmd/auth/mailer.go.
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
