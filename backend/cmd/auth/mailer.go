package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"
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

	msg := m.buildPasswordResetMessage(toEmail, toName, resetLink,
		randomToken(), messageID(m.from), time.Now())

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

// randomToken returns hex bytes for a MIME boundary or a Message-ID local part.
func randomToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// A boundary only has to be absent from the body, not be unguessable.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// messageID builds an RFC 5322 Message-ID from the sender's domain. Mail without
// one scores worse with spam filters regardless of SPF and DKIM.
func messageID(from string) string {
	domain := "localhost"
	if at := strings.LastIndex(from, "@"); at >= 0 && at+1 < len(from) {
		domain = strings.Trim(from[at+1:], "<> ")
	}
	return "<" + randomToken() + "@" + domain + ">"
}

// buildPasswordResetMessage assembles the multipart/alternative mail. The
// boundary, Message-ID and timestamp are parameters so a test can assert on a
// fixed message.
//
// The address the link belongs to is named in both parts on purpose: anyone with
// more than one account cannot otherwise tell which password this link sets, and
// then resets one account while trying to sign in to another. The address is
// only ever shown to the mailbox that already received the mail.
func (m *systemMailer) buildPasswordResetMessage(toEmail, toName, resetLink, boundary, msgID string, now time.Time) []byte {
	const subject = "Passwort für dein Cosmi-Konto zurücksetzen"

	plain := fmt.Sprintf("Hallo %s,\r\n\r\n"+
		"für dein Cosmi-Konto %s wurde ein neues Passwort angefordert.\r\n\r\n"+
		"Über diesen Link kannst du es festlegen. Er gilt eine Stunde und lässt sich\r\n"+
		"nur einmal verwenden:\r\n\r\n"+
		"%s\r\n\r\n"+
		"Kam die Anfrage nicht von dir, kannst du diese Nachricht ignorieren — dein\r\n"+
		"Passwort bleibt dann unverändert.\r\n\r\n"+
		"Cosmi — von Zentria\r\n",
		toName, toEmail, resetLink,
	)

	// Inline styles and no images: mail clients strip <style> and @font-face, and
	// a remote image would be a tracking pixel we promise not to send.
	htmlBody := fmt.Sprintf(`<!doctype html>
<html lang="de">
<head><meta charset="utf-8"><meta name="color-scheme" content="light"></head>
<body style="margin:0;padding:24px;background:#f4f4f6;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;color:#18181b;line-height:1.5;">
  <div style="max-width:30rem;margin:0 auto;background:#ffffff;border:1px solid #e4e4e7;border-radius:14px;padding:32px;">
    <p style="margin:0 0 22px;font-size:11px;font-weight:600;letter-spacing:0.13em;text-transform:uppercase;color:#1e7e74;">Cosmi</p>
    <h1 style="margin:0 0 10px;font-size:20px;font-weight:600;">Neues Passwort festlegen</h1>
    <p style="margin:0 0 20px;color:#71717a;font-size:15px;">Hallo %s, für dein Cosmi-Konto <strong style="color:#18181b;font-weight:600;">%s</strong> wurde ein neues Passwort angefordert.</p>
    <a href="%s" style="display:inline-block;padding:12px 22px;background:#1e7e74;color:#ffffff;font-size:15px;font-weight:600;text-decoration:none;border-radius:9px;">Passwort festlegen</a>
    <p style="margin:20px 0 0;color:#71717a;font-size:13px;">Der Link gilt eine Stunde und lässt sich nur einmal verwenden. Falls die Schaltfläche nicht funktioniert:</p>
    <p style="margin:6px 0 0;font-size:13px;word-break:break-all;"><a href="%s" style="color:#1e7e74;">%s</a></p>
    <p style="margin:22px 0 0;padding-top:18px;border-top:1px solid #e4e4e7;color:#71717a;font-size:13px;">Kam die Anfrage nicht von dir, kannst du diese Nachricht ignorieren — dein Passwort bleibt unverändert.</p>
    <p style="margin:18px 0 0;color:#71717a;font-size:12px;">Cosmi — von Zentria</p>
  </div>
</body>
</html>
`, html.EscapeString(toName), html.EscapeString(toEmail),
		html.EscapeString(resetLink), html.EscapeString(resetLink), html.EscapeString(resetLink))

	var buf bytes.Buffer
	buf.WriteString("From: " + m.from + "\r\n")
	buf.WriteString("To: " + toEmail + "\r\n")
	// Umlauts in a header must be RFC 2047 encoded or older clients show mojibake.
	buf.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n")
	buf.WriteString("Date: " + now.Format(time.RFC1123Z) + "\r\n")
	buf.WriteString("Message-ID: " + msgID + "\r\n")
	// Keeps out-of-office replies from bouncing back at a no-reply sender (RFC 3834).
	buf.WriteString("Auto-Submitted: auto-generated\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")

	for _, part := range []struct{ contentType, body string }{
		{"text/plain; charset=UTF-8", plain},
		{"text/html; charset=UTF-8", htmlBody},
	} {
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: " + part.contentType + "\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		qp := quotedprintable.NewWriter(&buf)
		_, _ = qp.Write([]byte(part.body))
		_ = qp.Close()
		buf.WriteString("\r\n")
	}
	buf.WriteString("--" + boundary + "--\r\n")

	return buf.Bytes()
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
