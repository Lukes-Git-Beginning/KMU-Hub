package send

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/models"
)

// AccountProvider retrieves decrypted SMTP credentials.
type AccountProvider interface {
	GetDecryptedCredentials(ctx context.Context, accountID uuid.UUID, tenantID uuid.UUID) (*Credentials, error)
}

// Credentials holds decrypted SMTP connection credentials.
type Credentials struct {
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	UseSSL   bool
}

// MessageCreator stores sent messages locally.
type MessageCreator interface {
	Create(ctx context.Context, msg *models.EmailMessage) error
}

// SignatureProvider retrieves email signatures scoped to a tenant.
type SignatureProvider interface {
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error)
	GetDefault(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error)
}

// SendInput contains the data needed to compose and send a new email.
type SendInput struct {
	AccountID uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	// ContactID, when set, triggers a consent pre-check for ChannelEmail.
	// Leave nil for system emails or transactional messages that do not
	// require marketing consent (e.g. password reset, invoice delivery).
	ContactID   *uuid.UUID
	From        EmailAddress
	To          []EmailAddress
	CC          []EmailAddress
	BCC         []EmailAddress
	Subject     string
	BodyHTML    string
	BodyText    string
	SignatureID *uuid.UUID
	Attachments []AttachmentData
}

// ReplyInput contains the data needed to reply to an email.
type ReplyInput struct {
	AccountID     uuid.UUID
	TenantID      uuid.UUID
	UserID        uuid.UUID
	OriginalMsgID uuid.UUID
	OriginalMsg   *models.EmailMessage
	From          EmailAddress
	To            []EmailAddress
	CC            []EmailAddress
	BCC           []EmailAddress
	BodyHTML      string
	BodyText      string
	SignatureID   *uuid.UUID
}

// ForwardInput contains the data needed to forward an email.
type ForwardInput struct {
	AccountID   uuid.UUID
	TenantID    uuid.UUID
	UserID      uuid.UUID
	OriginalMsg *models.EmailMessage
	From        EmailAddress
	To          []EmailAddress
	CC          []EmailAddress
	BCC         []EmailAddress
	BodyHTML    string
	BodyText    string
	SignatureID *uuid.UUID
	Attachments []AttachmentData
}

// DraftInput contains the data needed to save a draft.
type DraftInput struct {
	AccountID uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	From      EmailAddress
	To        []EmailAddress
	CC        []EmailAddress
	BCC       []EmailAddress
	Subject   string
	BodyHTML  string
	BodyText  string
	SignatureID *uuid.UUID
}

// Service handles email sending and composition.
type Service struct {
	accountProvider   AccountProvider
	messageCreator    MessageCreator
	signatureProvider SignatureProvider
	mimeBuilder       *MIMEBuilder
	// consentAsserter, when non-nil, enforces active consent before sending.
	// It is nil-safe: if not injected, consent checks are skipped (development
	// / legacy path). Production wiring must always inject an Asserter.
	consentAsserter   *consent.Asserter
}

// NewService creates a new send service.
func NewService(
	accountProvider AccountProvider,
	messageCreator MessageCreator,
	signatureProvider SignatureProvider,
) *Service {
	return &Service{
		accountProvider:   accountProvider,
		messageCreator:    messageCreator,
		signatureProvider: signatureProvider,
		mimeBuilder:       NewMIMEBuilder(),
	}
}

// NewServiceWithConsent creates a send service with a consent asserter wired in.
// This is the production constructor. The asserter is mandatory in production
// to satisfy the Rigorosum P0 requirement.
func NewServiceWithConsent(
	accountProvider AccountProvider,
	messageCreator MessageCreator,
	signatureProvider SignatureProvider,
	asserter *consent.Asserter,
) *Service {
	svc := NewService(accountProvider, messageCreator, signatureProvider)
	svc.consentAsserter = asserter
	return svc
}

// Send composes and sends a new email via SMTP.
func (s *Service) Send(ctx context.Context, input SendInput) (*models.EmailMessage, error) {
	// Validate recipients
	if len(input.To) == 0 {
		return nil, ErrInvalidRecipient
	}

	// Consent pre-check: block marketing email when no active consent exists.
	// Only triggered when both a ContactID and an Asserter are present.
	if s.consentAsserter != nil && input.ContactID != nil {
		if err := s.consentAsserter.Assert(ctx, *input.ContactID, consent.ChannelEmail); err != nil {
			return nil, err
		}
	}

	// Get SMTP credentials scoped to tenant
	creds, err := s.accountProvider.GetDecryptedCredentials(ctx, input.AccountID, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMTP credentials: %w", err)
	}

	// Append signature if requested
	bodyHTML := input.BodyHTML
	if input.SignatureID != nil {
		bodyHTML = s.appendSignature(ctx, bodyHTML, *input.SignatureID, input.TenantID)
	}

	// Build MIME message
	messageID := fmt.Sprintf("<%s@kmuhub>", uuid.New().String())
	mimeInput := MIMEInput{
		From:        input.From,
		To:          input.To,
		CC:          input.CC,
		BCC:         input.BCC,
		Subject:     input.Subject,
		BodyHTML:    bodyHTML,
		BodyText:    input.BodyText,
		Attachments: input.Attachments,
		MessageID:   messageID,
	}

	rawMsg, err := s.mimeBuilder.Build(mimeInput)
	if err != nil {
		return nil, fmt.Errorf("failed to build MIME message: %w", err)
	}

	// Send via SMTP
	if err := s.sendSMTP(creds, input.From.Email, collectRecipients(input.To, input.CC, input.BCC), rawMsg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	// Save copy locally
	msg := &models.EmailMessage{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		AccountID:      input.AccountID,
		MessageID:      messageID,
		FromName:       input.From.Name,
		FromEmail:      input.From.Email,
		ToAddresses:    toModelAddresses(input.To),
		CcAddresses:    toModelAddresses(input.CC),
		BccAddresses:   toModelAddresses(input.BCC),
		Subject:        input.Subject,
		BodyHTML:       bodyHTML,
		BodyText:       input.BodyText,
		IsRead:         true,
		HasAttachments: len(input.Attachments) > 0,
		Date:           time.Now().UTC(),
		SizeBytes:      int64(len(rawMsg)),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.messageCreator.Create(ctx, msg); err != nil {
		slog.Warn("failed to save sent message locally", "error", err)
	}

	slog.Info("email sent",
		"message_id", messageID,
		"to", formatRecipients(input.To),
	)

	return msg, nil
}

// Reply composes and sends a reply to an existing message.
func (s *Service) Reply(ctx context.Context, input ReplyInput) (*models.EmailMessage, error) {
	if input.OriginalMsg == nil {
		return nil, fmt.Errorf("original message is required for reply")
	}

	subject := input.OriginalMsg.Subject
	if !hasReplyPrefix(subject) {
		subject = "Re: " + subject
	}

	return s.Send(ctx, SendInput{
		AccountID:   input.AccountID,
		TenantID:    input.TenantID,
		UserID:      input.UserID,
		From:        input.From,
		To:          input.To,
		CC:          input.CC,
		BCC:         input.BCC,
		Subject:     subject,
		BodyHTML:    input.BodyHTML,
		BodyText:    input.BodyText,
		SignatureID: input.SignatureID,
	})
}

// ReplyAll composes and sends a reply-all including all original recipients.
func (s *Service) ReplyAll(ctx context.Context, input ReplyInput) (*models.EmailMessage, error) {
	if input.OriginalMsg == nil {
		return nil, fmt.Errorf("original message is required for reply-all")
	}

	// Include original To and CC, excluding the sender
	for _, addr := range input.OriginalMsg.ToAddresses {
		if addr.Email != input.From.Email {
			input.CC = append(input.CC, EmailAddress{Name: addr.Name, Email: addr.Email})
		}
	}
	for _, addr := range input.OriginalMsg.CcAddresses {
		if addr.Email != input.From.Email {
			input.CC = append(input.CC, EmailAddress{Name: addr.Name, Email: addr.Email})
		}
	}

	return s.Reply(ctx, input)
}

// Forward composes and sends a forwarded message.
func (s *Service) Forward(ctx context.Context, input ForwardInput) (*models.EmailMessage, error) {
	if input.OriginalMsg == nil {
		return nil, fmt.Errorf("original message is required for forward")
	}

	subject := input.OriginalMsg.Subject
	if !hasForwardPrefix(subject) {
		subject = "Fwd: " + subject
	}

	return s.Send(ctx, SendInput{
		AccountID:   input.AccountID,
		TenantID:    input.TenantID,
		UserID:      input.UserID,
		From:        input.From,
		To:          input.To,
		CC:          input.CC,
		BCC:         input.BCC,
		Subject:     subject,
		BodyHTML:    input.BodyHTML,
		BodyText:    input.BodyText,
		SignatureID: input.SignatureID,
		Attachments: input.Attachments,
	})
}

// SaveDraft saves a message as a draft locally.
func (s *Service) SaveDraft(ctx context.Context, input DraftInput) (*models.EmailMessage, error) {
	bodyHTML := input.BodyHTML
	if input.SignatureID != nil {
		bodyHTML = s.appendSignature(ctx, bodyHTML, *input.SignatureID, input.TenantID)
	}

	msg := &models.EmailMessage{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		AccountID:    input.AccountID,
		MessageID:    fmt.Sprintf("<%s@kmuhub>", uuid.New().String()),
		FromName:     input.From.Name,
		FromEmail:    input.From.Email,
		ToAddresses:  toModelAddresses(input.To),
		CcAddresses:  toModelAddresses(input.CC),
		BccAddresses: toModelAddresses(input.BCC),
		Subject:      input.Subject,
		BodyHTML:     bodyHTML,
		BodyText:     input.BodyText,
		IsDraft:      true,
		Date:         time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.messageCreator.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to save draft: %w", err)
	}

	slog.Info("draft saved", "message_id", msg.ID)
	return msg, nil
}

// smtpDialTimeout bounds the TCP connect to the SMTP server. An unreachable
// server fails fast instead of blocking the calling goroutine indefinitely.
//
// smtpExchangeDeadline bounds the entire SMTP exchange (handshake, auth, data)
// once connected, so a server that stalls mid-conversation cannot leak the
// goroutine. net/smtp itself offers no per-operation timeout.
//
// Both are vars (not consts) only so tests can shrink them; production never
// reassigns them.
var (
	smtpDialTimeout      = 10 * time.Second
	smtpExchangeDeadline = 30 * time.Second
)

// sendSMTP sends raw email data via SMTP. Both transports are bounded by a dial
// timeout and an exchange deadline; net/smtp's SendMail offers neither, which
// would let an unresponsive SMTP server block the calling goroutine forever.
func (s *Service) sendSMTP(creds *Credentials, from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", creds.SMTPHost, creds.SMTPPort)
	auth := smtp.PlainAuth("", creds.Username, creds.Password, creds.SMTPHost)

	if creds.SMTPPort == 465 {
		// Direct TLS connection (SMTPS)
		return s.sendSMTPTLS(addr, auth, creds.SMTPHost, from, to, msg)
	}

	// Port 587: STARTTLS
	return s.sendSMTPStartTLS(addr, auth, creds.SMTPHost, from, to, msg)
}

// sendSMTPStartTLS sends via STARTTLS (port 587) with bounded dial + exchange.
// It mirrors net/smtp.SendMail (hello → STARTTLS if offered → auth → data) but
// adds a net.DialTimeout and a connection deadline.
func (s *Service) sendSMTPStartTLS(addr string, auth smtp.Auth, host, from string, to []string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, smtpDialTimeout)
	if err != nil {
		return fmt.Errorf("SMTP dial failed: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(smtpExchangeDeadline))

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	return s.smtpDeliver(client, auth, from, to, msg)
}

// sendSMTPTLS sends via direct TLS connection (port 465) with bounded dial +
// exchange.
func (s *Service) sendSMTPTLS(addr string, auth smtp.Auth, host, from string, to []string, msg []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: smtpDialTimeout}, "tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(smtpExchangeDeadline))

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	return s.smtpDeliver(client, auth, from, to, msg)
}

// smtpDeliver runs the auth + envelope + data phase shared by both transports.
func (s *Service) smtpDeliver(client *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("%w: %v", ErrSMTPAuthFailed, err)
		}
	}

	if err := client.Mail(from); err != nil {
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

// appendSignature fetches a signature scoped to a tenant and appends it to the HTML body.
func (s *Service) appendSignature(ctx context.Context, bodyHTML string, signatureID uuid.UUID, tenantID uuid.UUID) string {
	if s.signatureProvider == nil {
		return bodyHTML
	}
	sig, err := s.signatureProvider.GetByID(ctx, signatureID, tenantID)
	if err != nil {
		slog.Warn("failed to load signature", "signature_id", signatureID, "tenant_id", tenantID, "error", err)
		return bodyHTML
	}
	return bodyHTML + "<br/><br/>--<br/>" + sig.HTMLContent
}

// collectRecipients merges To, CC, and BCC into a flat list of email addresses.
func collectRecipients(to, cc, bcc []EmailAddress) []string {
	seen := make(map[string]bool)
	var result []string
	for _, lists := range [][]EmailAddress{to, cc, bcc} {
		for _, addr := range lists {
			if !seen[addr.Email] {
				seen[addr.Email] = true
				result = append(result, addr.Email)
			}
		}
	}
	return result
}

// toModelAddresses converts send EmailAddress to model EmailAddress.
func toModelAddresses(addrs []EmailAddress) []models.EmailAddress {
	if len(addrs) == 0 {
		return nil
	}
	result := make([]models.EmailAddress, 0, len(addrs))
	for _, a := range addrs {
		result = append(result, models.EmailAddress{Name: a.Name, Email: a.Email})
	}
	return result
}

// formatRecipients returns a comma-separated list of recipient emails.
func formatRecipients(addrs []EmailAddress) string {
	emails := make([]string, 0, len(addrs))
	for _, a := range addrs {
		emails = append(emails, a.Email)
	}
	return strings.Join(emails, ", ")
}

// hasReplyPrefix checks if a subject already has a reply prefix.
func hasReplyPrefix(subject string) bool {
	lower := strings.ToLower(strings.TrimSpace(subject))
	return strings.HasPrefix(lower, "re:") || strings.HasPrefix(lower, "aw:")
}

// hasForwardPrefix checks if a subject already has a forward prefix.
func hasForwardPrefix(subject string) bool {
	lower := strings.ToLower(strings.TrimSpace(subject))
	return strings.HasPrefix(lower, "fwd:") || strings.HasPrefix(lower, "fw:") || strings.HasPrefix(lower, "wg:")
}
