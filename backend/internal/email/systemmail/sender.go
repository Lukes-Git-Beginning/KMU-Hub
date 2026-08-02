// Package systemmail delivers transactional email through the dedicated system
// SMTP account (SYSTEM_SMTP_*), independent of any user mailbox.
//
// Services that must reach a recipient without an authenticated user behind the
// send -- dunning notices, scheduled report delivery -- wire a Sender here
// instead of hand-rolling their own SMTP client. The transport mirrors
// net/smtp.SendMail's flow (dial -> hello -> STARTTLS if offered -> auth ->
// data) but bounds both the TCP connect and the whole exchange; net/smtp offers
// neither, so an unresponsive server would otherwise block the caller forever.
//
// A Sender with an empty Host is "not configured": Send returns
// ErrNotConfigured rather than reporting a delivery that never happened.
// Callers decide whether that is fatal or a degraded mode.
package systemmail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/kmuhub/kmuhub/internal/email/send"
)

// dialTimeout bounds the TCP connect; exchangeDeadline bounds the whole SMTP
// exchange. Both are variables so tests can shrink them.
var (
	dialTimeout      = 10 * time.Second
	exchangeDeadline = 30 * time.Second
)

// ErrNotConfigured is returned by Send when no SMTP host is configured.
var ErrNotConfigured = errors.New("systemmail: system SMTP not configured")

// defaultFromName is the display name used when Config.FromName is empty.
const defaultFromName = "Cosmi"

// Config holds the system SMTP account, normally taken from config.Config's
// SystemSMTP* fields. An empty Host disables the sender.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

// Message is one outgoing mail. Attachments reuse the email/send builder types
// so callers do not need a second attachment representation.
type Message struct {
	To          []send.EmailAddress
	Subject     string
	BodyText    string
	BodyHTML    string
	Attachments []send.AttachmentData
}

// Sender delivers Messages over the configured system SMTP account.
type Sender struct {
	cfg Config
}

// New returns a Sender for cfg. cfg.Host may be empty; see Configured.
func New(cfg Config) *Sender {
	if cfg.FromName == "" {
		cfg.FromName = defaultFromName
	}
	return &Sender{cfg: cfg}
}

// Configured reports whether an SMTP host is set. Callers that must not
// silently degrade check this before wiring the sender in.
func (s *Sender) Configured() bool { return s.cfg.Host != "" }

// From returns the configured envelope sender address.
func (s *Sender) From() string { return s.cfg.From }

// Send builds the MIME message and delivers it to every recipient in msg.To.
// It returns ErrNotConfigured when no host is set -- never a silent success.
func (s *Sender) Send(ctx context.Context, msg Message) error {
	if !s.Configured() {
		return ErrNotConfigured
	}
	if len(msg.To) == 0 {
		return errors.New("systemmail: no recipients")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	raw, err := send.NewMIMEBuilder().Build(send.MIMEInput{
		From:        send.EmailAddress{Name: s.cfg.FromName, Email: s.cfg.From},
		To:          msg.To,
		Subject:     msg.Subject,
		BodyText:    msg.BodyText,
		BodyHTML:    msg.BodyHTML,
		Attachments: msg.Attachments,
	})
	if err != nil {
		return fmt.Errorf("systemmail: build mime message: %w", err)
	}

	recipients := make([]string, 0, len(msg.To))
	for _, addr := range msg.To {
		recipients = append(recipients, addr.Email)
	}

	if err := s.deliver(ctx, recipients, raw); err != nil {
		return fmt.Errorf("systemmail: send to %v: %w", recipients, err)
	}
	return nil
}

// deliver runs the SMTP exchange with a bounded dial timeout and deadline.
func (s *Sender) deliver(ctx context.Context, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	dialer := net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(exchangeDeadline)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}
