package send

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// recordingSMTP is a fake SMTP server that records the envelope, the AUTH
// payload and the DATA section of the one connection it serves.
type recordingSMTP struct {
	addr string

	mu       sync.Mutex
	mailFrom string
	rcpts    []string
	auth     string
	data     string
	quit     bool

	// authReply, when set, answers AUTH with this line instead of 235.
	authReply string
	// startTLS makes the server advertise STARTTLS and, once asked, hang up
	// instead of completing the handshake.
	startTLS bool
	// noAuth leaves AUTH out of the EHLO answer and refuses it when tried.
	noAuth bool
}

func startRecordingSMTP(t *testing.T, cfg *recordingSMTP) *recordingSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	cfg.addr = ln.Addr().String()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		_ = tp.PrintfLine("220 fake.smtp ready")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			fields := strings.Fields(line)
			verb := ""
			if len(fields) > 0 {
				verb = strings.ToUpper(fields[0])
			}
			switch verb {
			case "EHLO":
				_ = tp.PrintfLine("250-fake.smtp")
				if cfg.startTLS {
					_ = tp.PrintfLine("250-STARTTLS")
				}
				if !cfg.noAuth {
					_ = tp.PrintfLine("250-AUTH PLAIN")
				}
				_ = tp.PrintfLine("250 8BITMIME")
			case "STARTTLS":
				_ = tp.PrintfLine("220 go ahead")
				return // no handshake follows: the client's TLS setup must fail
			case "AUTH":
				if cfg.noAuth {
					_ = tp.PrintfLine("504 5.5.1 authentication not supported")
					continue
				}
				if cfg.authReply != "" {
					_ = tp.PrintfLine("%s", cfg.authReply)
					continue
				}
				raw := ""
				if len(fields) > 2 {
					raw = fields[2]
				}
				dec, _ := base64.StdEncoding.DecodeString(raw)
				cfg.mu.Lock()
				cfg.auth = string(dec)
				cfg.mu.Unlock()
				_ = tp.PrintfLine("235 2.7.0 accepted")
			case "MAIL":
				cfg.mu.Lock()
				cfg.mailFrom = line
				cfg.mu.Unlock()
				_ = tp.PrintfLine("250 ok")
			case "RCPT":
				cfg.mu.Lock()
				cfg.rcpts = append(cfg.rcpts, line)
				cfg.mu.Unlock()
				_ = tp.PrintfLine("250 ok")
			case "DATA":
				_ = tp.PrintfLine("354 go ahead")
				var sb strings.Builder
				for {
					dline, derr := tp.ReadLine()
					if derr != nil || dline == "." {
						break
					}
					sb.WriteString(dline)
					sb.WriteString("\n")
				}
				cfg.mu.Lock()
				cfg.data = sb.String()
				cfg.mu.Unlock()
				_ = tp.PrintfLine("250 ok")
			case "QUIT":
				cfg.mu.Lock()
				cfg.quit = true
				cfg.mu.Unlock()
				_ = tp.PrintfLine("221 bye")
				return
			default:
				_ = tp.PrintfLine("250 ok")
			}
		}
	}()
	return cfg
}

func (r *recordingSMTP) creds(t *testing.T) *Credentials {
	t.Helper()
	host, portStr, err := net.SplitHostPort(r.addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return &Credentials{SMTPHost: host, SMTPPort: port, Username: "smtp-user", Password: "smtp-pass"}
}

// folderCreator records the folder type next to each stored message and can be
// told to fail.
type folderCreator struct {
	messages []*models.EmailMessage
	folders  []string
	err      error
}

func (f *folderCreator) CreateLocalCopy(_ context.Context, msg *models.EmailMessage, folderType string) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, msg)
	f.folders = append(f.folders, folderType)
	return nil
}

type stubSignatures struct {
	html string
	err  error
	seen uuid.UUID
}

func (s *stubSignatures) GetByID(_ context.Context, _ uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	s.seen = tenantID
	if s.err != nil {
		return nil, s.err
	}
	return &models.EmailSignature{HTMLContent: s.html}, nil
}

func (s *stubSignatures) GetDefault(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.EmailSignature, error) {
	return nil, errors.New("not used")
}

func newFlowService(t *testing.T, smtp *recordingSMTP, creator MessageCreator, sigs SignatureProvider) *Service {
	t.Helper()
	return NewService(&stubAccountProvider{creds: smtp.creds(t)}, creator, sigs).WithDialGuard(loopbackGuard(t))
}

func TestSend_DeliversEnvelopeAuthAndStoresSentCopy(t *testing.T) {
	smtp := startRecordingSMTP(t, &recordingSMTP{})
	creator := &folderCreator{}
	sigs := &stubSignatures{html: "<b>Mit freundlichen Grüßen</b>"}
	svc := newFlowService(t, smtp, creator, sigs)

	tenantID := uuid.New()
	sigID := uuid.New()
	msg, err := svc.Send(context.Background(), SendInput{
		AccountID:   uuid.New(),
		TenantID:    tenantID,
		From:        EmailAddress{Name: "Alice", Email: "alice@example.com"},
		To:          []EmailAddress{{Name: "Bob", Email: "bob@example.com"}},
		CC:          []EmailAddress{{Email: "carol@example.com"}, {Email: "bob@example.com"}},
		BCC:         []EmailAddress{{Email: "secret@example.com"}},
		Subject:     "Angebot",
		BodyHTML:    "<p>Hallo</p>",
		BodyText:    "Hallo",
		SignatureID: &sigID,
	})
	require.NoError(t, err)

	smtp.mu.Lock()
	defer smtp.mu.Unlock()
	assert.Contains(t, smtp.mailFrom, "alice@example.com")
	require.Len(t, smtp.rcpts, 3, "the duplicate bob@ must be sent once")
	assert.Contains(t, smtp.rcpts[0], "bob@example.com")
	assert.Contains(t, smtp.rcpts[1], "carol@example.com")
	assert.Contains(t, smtp.rcpts[2], "secret@example.com")
	assert.Equal(t, "\x00smtp-user\x00smtp-pass", smtp.auth)
	assert.True(t, smtp.quit, "the session must end with QUIT")

	assert.Contains(t, smtp.data, "Subject: Angebot")
	assert.Contains(t, smtp.data, "Cc: carol@example.com")
	assert.NotContains(t, strings.ToLower(smtp.data), "bcc:", "the blind copy must not appear in the headers")
	assert.NotContains(t, smtp.data, "secret@example.com")

	assert.Equal(t, tenantID, sigs.seen, "the signature is looked up in the sender's tenant")
	assert.Contains(t, msg.BodyHTML, "<p>Hallo</p><br/><br/>--<br/><b>Mit freundlichen Grüßen</b>")

	require.Len(t, creator.messages, 1)
	assert.Equal(t, []string{models.FolderTypeSent}, creator.folders)
	stored := creator.messages[0]
	assert.Same(t, msg, stored)
	assert.Equal(t, tenantID, stored.TenantID)
	assert.True(t, stored.IsRead)
	assert.False(t, stored.IsDraft)
	assert.False(t, stored.HasAttachments)
	assert.Equal(t, "alice@example.com", stored.FromEmail)
	require.Len(t, stored.BccAddresses, 1)
	assert.Equal(t, "secret@example.com", stored.BccAddresses[0].Email)
	assert.Greater(t, stored.SizeBytes, int64(0))
	assert.True(t, strings.HasPrefix(stored.MessageID, "<") && strings.Contains(stored.MessageID, "@example.com"))
}

func TestSend_LocalCopyFailureDoesNotFailTheSend(t *testing.T) {
	smtp := startRecordingSMTP(t, &recordingSMTP{})
	creator := &folderCreator{err: errors.New("no sent folder")}
	svc := newFlowService(t, smtp, creator, nil)

	msg, err := svc.Send(context.Background(), SendInput{
		From: EmailAddress{Email: "alice@example.com"},
		To:   []EmailAddress{{Email: "bob@example.com"}},
	})
	require.NoError(t, err, "the mail is out; a missing Sent folder must not turn that into an error")
	assert.NotNil(t, msg)
	assert.Empty(t, creator.messages)
}

func TestSend_RejectsEmptyRecipientsAndPropagatesCredentialErrors(t *testing.T) {
	creator := &folderCreator{}
	svc := NewService(&errAccountProvider{}, creator, nil)

	_, err := svc.Send(context.Background(), SendInput{From: EmailAddress{Email: "a@example.com"}})
	assert.ErrorIs(t, err, ErrInvalidRecipient)

	_, err = svc.Send(context.Background(), SendInput{
		From: EmailAddress{Email: "a@example.com"},
		To:   []EmailAddress{{Email: "b@example.com"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get SMTP credentials")
	assert.Empty(t, creator.messages)
}

func TestSend_AuthRejectionIsReportedAndNothingIsStored(t *testing.T) {
	smtp := startRecordingSMTP(t, &recordingSMTP{authReply: "535 5.7.8 bad credentials"})
	creator := &folderCreator{}
	svc := newFlowService(t, smtp, creator, nil)

	_, err := svc.Send(context.Background(), SendInput{
		From: EmailAddress{Email: "alice@example.com"},
		To:   []EmailAddress{{Email: "bob@example.com"}},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSendFailed)
	assert.Contains(t, err.Error(), ErrSMTPAuthFailed.Error())
	assert.Empty(t, creator.messages)
	smtp.mu.Lock()
	defer smtp.mu.Unlock()
	assert.Empty(t, smtp.rcpts, "no envelope may be sent after a failed login")
}

func TestSend_ServerWithoutAuthExtensionFailsTheLogin(t *testing.T) {
	smtp := startRecordingSMTP(t, &recordingSMTP{noAuth: true})
	svc := newFlowService(t, smtp, &folderCreator{}, nil)

	_, err := svc.Send(context.Background(), SendInput{
		From: EmailAddress{Email: "alice@example.com"},
		To:   []EmailAddress{{Email: "bob@example.com"}},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSendFailed)
	assert.Contains(t, err.Error(), ErrSMTPAuthFailed.Error())
}

func TestSend_StartTLSHandshakeFailureIsReported(t *testing.T) {
	smtp := startRecordingSMTP(t, &recordingSMTP{startTLS: true})
	creator := &folderCreator{}
	svc := newFlowService(t, smtp, creator, nil)

	_, err := svc.Send(context.Background(), SendInput{
		From: EmailAddress{Email: "alice@example.com"},
		To:   []EmailAddress{{Email: "bob@example.com"}},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSendFailed)
	assert.Contains(t, err.Error(), "STARTTLS failed")
	assert.Empty(t, creator.messages)
}

func TestSend_DirectTLSPortRefusesAPlainServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		_, _ = c.Write([]byte("220 plain smtp, no tls\r\n"))
		c.Close()
	}()

	svc := &Service{dialGuard: loopbackGuard(t)}
	err = svc.sendSMTPTLS(ln.Addr().String(), nil, "127.0.0.1", "a@example.com", []string{"b@example.com"}, []byte("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS dial failed")
}

func TestSend_UnbuildableAttachmentFailsBeforeAnySMTP(t *testing.T) {
	creator := &folderCreator{}
	svc := NewService(&stubAccountProvider{creds: &Credentials{SMTPHost: "127.0.0.1", SMTPPort: 1}}, creator, nil)

	_, err := svc.Send(context.Background(), SendInput{
		From:    EmailAddress{Email: "a@example.com"},
		To:      []EmailAddress{{Email: "b@example.com"}},
		Subject: "x",
		Attachments: []AttachmentData{{
			Filename: "broken.bin", ContentType: "application/octet-stream",
			Data: errReader{},
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build MIME message")
	assert.Empty(t, creator.messages)
}

func TestSend_AttachmentsMarkTheStoredCopy(t *testing.T) {
	smtp := startRecordingSMTP(t, &recordingSMTP{})
	creator := &folderCreator{}
	svc := newFlowService(t, smtp, creator, nil)

	msg, err := svc.Send(context.Background(), SendInput{
		From:    EmailAddress{Email: "alice@example.com"},
		To:      []EmailAddress{{Email: "bob@example.com"}},
		Subject: "Mit Anhang",
		Attachments: []AttachmentData{{
			Filename: "a.txt", ContentType: "text/plain", Data: strings.NewReader("hello"),
		}},
	})
	require.NoError(t, err)
	assert.True(t, msg.HasAttachments)
	smtp.mu.Lock()
	defer smtp.mu.Unlock()
	assert.Contains(t, smtp.data, `filename="a.txt"`)
}

func TestReply_PrefixesSubjectOnceAndRequiresTheOriginal(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    string
	}{
		{"adds the prefix", "Angebot", "Subject: Re: Angebot"},
		{"keeps an English prefix", "RE: Angebot", "Subject: RE: Angebot"},
		{"keeps a German prefix", "AW: Angebot", "Subject: AW: Angebot"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smtp := startRecordingSMTP(t, &recordingSMTP{})
			creator := &folderCreator{}
			svc := newFlowService(t, smtp, creator, nil)

			msg, err := svc.Reply(context.Background(), ReplyInput{
				TenantID:    uuid.New(),
				OriginalMsg: &models.EmailMessage{Subject: tt.subject},
				From:        EmailAddress{Email: "alice@example.com"},
				To:          []EmailAddress{{Email: "bob@example.com"}},
				BodyText:    "ok",
			})
			require.NoError(t, err)
			assert.Equal(t, strings.TrimPrefix(tt.want, "Subject: "), msg.Subject)
			smtp.mu.Lock()
			defer smtp.mu.Unlock()
			assert.Contains(t, smtp.data, tt.want)
		})
	}

	svc := NewService(nil, &folderCreator{}, nil)
	_, err := svc.Reply(context.Background(), ReplyInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "original message is required for reply")
}

func TestReplyAll_CopiesOriginalRecipientsExceptTheSender(t *testing.T) {
	smtp := startRecordingSMTP(t, &recordingSMTP{})
	creator := &folderCreator{}
	svc := newFlowService(t, smtp, creator, nil)

	msg, err := svc.ReplyAll(context.Background(), ReplyInput{
		OriginalMsg: &models.EmailMessage{
			Subject: "Plan",
			ToAddresses: []models.EmailAddress{
				{Name: "Alice", Email: "alice@example.com"}, // the replier, must be dropped
				{Name: "Dan", Email: "dan@example.com"},
			},
			CcAddresses: []models.EmailAddress{
				{Email: "eve@example.com"},
				{Email: "alice@example.com"},
			},
		},
		From: EmailAddress{Email: "alice@example.com"},
		To:   []EmailAddress{{Email: "bob@example.com"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "Re: Plan", msg.Subject)

	var ccs []string
	for _, a := range msg.CcAddresses {
		ccs = append(ccs, a.Email)
	}
	assert.Equal(t, []string{"dan@example.com", "eve@example.com"}, ccs)
	assert.Equal(t, "Dan", msg.CcAddresses[0].Name)

	smtp.mu.Lock()
	defer smtp.mu.Unlock()
	require.Len(t, smtp.rcpts, 3)
	assert.Contains(t, smtp.rcpts[0], "bob@example.com")

	_, err = svc.ReplyAll(context.Background(), ReplyInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "original message is required for reply-all")
}

func TestForward_PrefixesSubjectOnceAndRequiresTheOriginal(t *testing.T) {
	tests := []struct {
		subject string
		want    string
	}{
		{"Angebot", "Fwd: Angebot"},
		{"FW: Angebot", "FW: Angebot"},
		{"WG: Angebot", "WG: Angebot"},
	}
	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			smtp := startRecordingSMTP(t, &recordingSMTP{})
			svc := newFlowService(t, smtp, &folderCreator{}, nil)

			msg, err := svc.Forward(context.Background(), ForwardInput{
				OriginalMsg: &models.EmailMessage{Subject: tt.subject},
				From:        EmailAddress{Email: "alice@example.com"},
				To:          []EmailAddress{{Email: "bob@example.com"}},
			})
			require.NoError(t, err)
			assert.Equal(t, tt.want, msg.Subject)
			smtp.mu.Lock()
			defer smtp.mu.Unlock()
			assert.Contains(t, smtp.data, "Subject: "+tt.want)
		})
	}

	svc := NewService(nil, &folderCreator{}, nil)
	_, err := svc.Forward(context.Background(), ForwardInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "original message is required for forward")
}

func TestSaveDraft_StoresInDraftsFolderWithSignatureAndFlags(t *testing.T) {
	creator := &folderCreator{}
	sigs := &stubSignatures{html: "<i>sig</i>"}
	svc := NewService(nil, creator, sigs)

	tenantID := uuid.New()
	sigID := uuid.New()
	msg, err := svc.SaveDraft(context.Background(), DraftInput{
		TenantID:       tenantID,
		AccountID:      uuid.New(),
		From:           EmailAddress{Name: "Alice", Email: "alice@example.com"},
		To:             []EmailAddress{{Email: "bob@example.com"}},
		CC:             []EmailAddress{{Email: "carol@example.com"}},
		BCC:            []EmailAddress{{Email: "dan@example.com"}},
		Subject:        "Entwurf",
		BodyHTML:       "<p>x</p>",
		SignatureID:    &sigID,
		HasAttachments: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{models.FolderTypeDrafts}, creator.folders)
	assert.True(t, msg.IsDraft)
	assert.True(t, msg.HasAttachments)
	assert.Equal(t, "Alice", msg.FromName)
	assert.Equal(t, "<p>x</p><br/><br/>--<br/><i>sig</i>", msg.BodyHTML)
	assert.Equal(t, tenantID, sigs.seen)
	require.Len(t, msg.CcAddresses, 1)
	require.Len(t, msg.BccAddresses, 1)
	assert.NotEqual(t, uuid.Nil, msg.ID)
}

func TestSaveDraft_StoreFailureIsReturned(t *testing.T) {
	svc := NewService(nil, &folderCreator{err: errors.New("db down")}, nil)

	msg, err := svc.SaveDraft(context.Background(), DraftInput{From: EmailAddress{Email: "a@example.com"}})
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "failed to save draft")
	assert.Contains(t, err.Error(), "db down")
}

func TestAppendSignature_FallsBackToTheBodyWhenNoSignatureCanBeLoaded(t *testing.T) {
	ctx := context.Background()
	id, tenant := uuid.New(), uuid.New()

	svc := NewService(nil, nil, nil)
	svc.signatureProvider = nil
	assert.Equal(t, "<p>x</p>", svc.appendSignature(ctx, "<p>x</p>", id, tenant), "no provider wired")

	svc = NewService(nil, nil, &stubSignatures{err: errors.New("not found")})
	assert.Equal(t, "<p>x</p>", svc.appendSignature(ctx, "<p>x</p>", id, tenant), "lookup error keeps the body")

	svc = NewService(nil, nil, &stubSignatures{html: "S"})
	assert.Equal(t, "<p>x</p><br/><br/>--<br/>S", svc.appendSignature(ctx, "<p>x</p>", id, tenant))
}

func TestFormatRecipients(t *testing.T) {
	assert.Equal(t, "", formatRecipients(nil))
	assert.Equal(t, "a@ex.com, b@ex.com", formatRecipients([]EmailAddress{
		{Name: "A", Email: "a@ex.com"}, {Email: "b@ex.com"},
	}))
}

func TestToModelAddresses_EmptyIsNil(t *testing.T) {
	assert.Nil(t, toModelAddresses(nil))
	assert.Nil(t, toModelAddresses([]EmailAddress{}))
}
