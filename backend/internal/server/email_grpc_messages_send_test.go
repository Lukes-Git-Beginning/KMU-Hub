package server

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/attachment"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/models"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// ---------------------------------------------------------------------------
// stubEmailMessageRepo implements message.Repository over an in-memory map.
// ---------------------------------------------------------------------------

type stubEmailMessageRepo struct {
	messages map[uuid.UUID]*models.EmailMessage
}

func newStubEmailMessageRepo() *stubEmailMessageRepo {
	return &stubEmailMessageRepo{messages: make(map[uuid.UUID]*models.EmailMessage)}
}

func (r *stubEmailMessageRepo) Create(_ context.Context, msg *models.EmailMessage) error {
	r.messages[msg.ID] = msg
	return nil
}

func (r *stubEmailMessageRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailMessage, error) {
	m, ok := r.messages[id]
	if !ok || m.TenantID != tenantID {
		return nil, message.ErrMessageNotFound
	}
	return m, nil
}

func (r *stubEmailMessageRepo) GetByFolderUID(_ context.Context, _ uuid.UUID, _ uint32) (*models.EmailMessage, error) {
	return nil, message.ErrMessageNotFound
}

func (r *stubEmailMessageRepo) ListByFolder(_ context.Context, folderID uuid.UUID, _ message.ListOpts) ([]*models.EmailMessage, int, error) {
	out := make([]*models.EmailMessage, 0)
	for _, m := range r.messages {
		if m.FolderID == folderID {
			out = append(out, m)
		}
	}
	return out, len(out), nil
}

func (r *stubEmailMessageRepo) ListByThread(_ context.Context, threadID uuid.UUID) ([]*models.EmailMessage, error) {
	out := make([]*models.EmailMessage, 0)
	for _, m := range r.messages {
		if m.ThreadID != nil && *m.ThreadID == threadID {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *stubEmailMessageRepo) Search(_ context.Context, accountID uuid.UUID, query string, _ message.ListOpts) ([]*models.EmailMessage, int, error) {
	out := make([]*models.EmailMessage, 0)
	for _, m := range r.messages {
		if m.AccountID == accountID && strings.Contains(m.Subject, query) {
			out = append(out, m)
		}
	}
	return out, len(out), nil
}

func (r *stubEmailMessageRepo) UpdateFlags(_ context.Context, id uuid.UUID, isRead, isStarred *bool) error {
	m, ok := r.messages[id]
	if !ok {
		return message.ErrMessageNotFound
	}
	if isRead != nil {
		m.IsRead = *isRead
	}
	if isStarred != nil {
		m.IsStarred = *isStarred
	}
	return nil
}

func (r *stubEmailMessageRepo) UpdateThreadID(_ context.Context, id, threadID uuid.UUID) error {
	m, ok := r.messages[id]
	if !ok {
		return message.ErrMessageNotFound
	}
	m.ThreadID = &threadID
	return nil
}

func (r *stubEmailMessageRepo) MoveToFolder(_ context.Context, id, folderID uuid.UUID) error {
	m, ok := r.messages[id]
	if !ok {
		return message.ErrMessageNotFound
	}
	m.FolderID = folderID
	return nil
}

func (r *stubEmailMessageRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.messages[id]; !ok {
		return message.ErrMessageNotFound
	}
	delete(r.messages, id)
	return nil
}

func (r *stubEmailMessageRepo) DeleteByFolder(_ context.Context, folderID uuid.UUID) error {
	for id, m := range r.messages {
		if m.FolderID == folderID {
			delete(r.messages, id)
		}
	}
	return nil
}

func (r *stubEmailMessageRepo) GetHighestUID(_ context.Context, _ uuid.UUID) (uint32, error) {
	return 0, nil
}

func (r *stubEmailMessageRepo) GetByMessageIDHeader(_ context.Context, _ string) (*models.EmailMessage, error) {
	return nil, message.ErrMessageNotFound
}

func (r *stubEmailMessageRepo) FindThreadByReferences(_ context.Context, _ uuid.UUID, _ []string, _ string) (uuid.UUID, error) {
	return uuid.Nil, message.ErrThreadNotFound
}

func (r *stubEmailMessageRepo) CountUnreadByFolder(_ context.Context, folderID uuid.UUID) (int, error) {
	n := 0
	for _, m := range r.messages {
		if m.FolderID == folderID && !m.IsRead {
			n++
		}
	}
	return n, nil
}

func (r *stubEmailMessageRepo) FindBySubjectAndParticipants(_ context.Context, _ uuid.UUID, _ string, _ string, _, _ interface{}) (*models.EmailMessage, error) {
	return nil, message.ErrMessageNotFound
}

// ---------------------------------------------------------------------------
// stubAttachmentRepo implements attachment.Repository over an in-memory map.
// stubObjectStore implements attachment.ObjectStore without touching MinIO.
// ---------------------------------------------------------------------------

type stubAttachmentRepo struct {
	attachments map[uuid.UUID]*models.EmailAttachment
}

func newStubAttachmentRepo() *stubAttachmentRepo {
	return &stubAttachmentRepo{attachments: make(map[uuid.UUID]*models.EmailAttachment)}
}

func (r *stubAttachmentRepo) Create(_ context.Context, att *models.EmailAttachment) error {
	r.attachments[att.ID] = att
	return nil
}

func (r *stubAttachmentRepo) GetByMessage(_ context.Context, messageID, tenantID uuid.UUID) ([]*models.EmailAttachment, error) {
	out := make([]*models.EmailAttachment, 0)
	for _, a := range r.attachments {
		if a.MessageID == messageID && a.TenantID == tenantID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *stubAttachmentRepo) GetMinIOKeyByID(_ context.Context, id, tenantID uuid.UUID) (string, error) {
	a, ok := r.attachments[id]
	if !ok || a.TenantID != tenantID {
		return "", attachment.ErrAttachmentNotFound
	}
	return a.MinIOKey, nil
}

type stubObjectStore struct{}

func (stubObjectStore) Upload(_ context.Context, _ uuid.UUID, _ uint32, filename string, _ io.Reader, _ int64, _ string) (string, error) {
	return "minio/" + filename, nil
}

func (stubObjectStore) GetPresignedURL(_ context.Context, minioKey string, _ time.Duration) (string, error) {
	return "https://minio.local/" + minioKey, nil
}

func (stubObjectStore) Delete(_ context.Context, _ string) error {
	return nil
}

// ---------------------------------------------------------------------------
// stubLinkRepo implements EmailContactLinkRepository over in-memory maps.
// ---------------------------------------------------------------------------

type stubLinkRepo struct {
	links           map[uuid.UUID]*models.EmailContactLink
	contactMessages map[uuid.UUID][]*models.EmailMessage
}

func newStubLinkRepo() *stubLinkRepo {
	return &stubLinkRepo{
		links:           make(map[uuid.UUID]*models.EmailContactLink),
		contactMessages: make(map[uuid.UUID][]*models.EmailMessage),
	}
}

func (r *stubLinkRepo) Create(_ context.Context, link *models.EmailContactLink) error {
	r.links[link.ID] = link
	return nil
}

func (r *stubLinkRepo) GetByMessageID(_ context.Context, messageID, tenantID uuid.UUID) ([]*models.EmailContactLink, error) {
	out := make([]*models.EmailContactLink, 0)
	for _, l := range r.links {
		if l.MessageID == messageID && l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *stubLinkRepo) GetByContactID(_ context.Context, contactID, tenantID uuid.UUID, _, _ int) ([]*models.EmailMessage, int, error) {
	out := make([]*models.EmailMessage, 0)
	for _, m := range r.contactMessages[contactID] {
		if m.TenantID == tenantID {
			out = append(out, m)
		}
	}
	return out, len(out), nil
}

func (r *stubLinkRepo) Delete(_ context.Context, messageID, contactID, tenantID uuid.UUID) error {
	for id, l := range r.links {
		if l.MessageID == messageID && l.ContactID == contactID && l.TenantID == tenantID {
			delete(r.links, id)
			return nil
		}
	}
	return errNoSuchLink
}

var errNoSuchLink = &linkNotFoundError{}

type linkNotFoundError struct{}

func (*linkNotFoundError) Error() string { return "email contact link not found" }

// ---------------------------------------------------------------------------
// stubSendAccountProvider implements send.AccountProvider.
// ---------------------------------------------------------------------------

type stubSendAccountProvider struct {
	creds map[uuid.UUID]*send.Credentials
}

func newStubSendAccountProvider() *stubSendAccountProvider {
	return &stubSendAccountProvider{creds: make(map[uuid.UUID]*send.Credentials)}
}

func (p *stubSendAccountProvider) GetDecryptedCredentials(_ context.Context, accountID, _ uuid.UUID) (*send.Credentials, error) {
	c, ok := p.creds[accountID]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return c, nil
}

// ---------------------------------------------------------------------------
// fake SMTP server -- accepts a connection and speaks just enough SMTP for
// net/smtp's client to complete a send: no STARTTLS advertised (so the client
// skips TLS entirely), AUTH PLAIN accepted unconditionally, MAIL/RCPT/DATA all
// acknowledged. This is what lets SendEmail/ReplyEmail/ForwardEmail exercise
// the real MIME build + SMTP dial instead of stopping at credential lookup.
// ---------------------------------------------------------------------------

func startFakeSMTPServer(t *testing.T) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go serveFakeSMTP(conn)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

func serveFakeSMTP(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }

	write("220 fake.smtp ESMTP")
	inData := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				inData = false
				write("250 2.0.0 OK queued")
			}
			continue
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			write("250 fake.smtp")
		case strings.HasPrefix(upper, "AUTH"):
			write("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM"):
			write("250 2.1.0 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			write("250 2.1.5 OK")
		case strings.HasPrefix(upper, "DATA"):
			inData = true
			write("354 Go ahead")
		case strings.HasPrefix(upper, "QUIT"):
			write("221 2.0.0 Bye")
			return
		default:
			write("500 unrecognized command")
		}
	}
}

// ---------------------------------------------------------------------------
// Server + fixture wiring
// ---------------------------------------------------------------------------

type testEmailMessagesFixture struct {
	srv        *EmailGRPCServer
	msgRepo    *stubEmailMessageRepo
	folderRepo *stubFolderRepo
	accounts   *stubSendAccountProvider
	attRepo    *stubAttachmentRepo
	linkRepo   *stubLinkRepo
}

// newEmailMessagesFixture wires an EmailGRPCServer for the message/send/
// attachment/contact-linking RPCs. accountService, signatureService, syncEngine,
// contactService, companyService, ruleService, labelService and templateService
// are all nil because no RPC under test reaches them.
func newEmailMessagesFixture() *testEmailMessagesFixture {
	msgRepo := newStubEmailMessageRepo()
	folderRepo := newStubFolderRepo()
	accounts := newStubSendAccountProvider()
	attRepo := newStubAttachmentRepo()
	linkRepo := newStubLinkRepo()

	msgSvc := message.NewService(msgRepo, folderRepo)
	sendSvc := send.NewService(accounts, msgSvc, nil)
	attSvc := attachment.NewService(attRepo, stubObjectStore{})

	srv := NewEmailGRPCServer(nil, msgSvc, sendSvc, nil, attSvc, nil, linkRepo, nil, nil, nil, nil, nil)

	return &testEmailMessagesFixture{
		srv:        srv,
		msgRepo:    msgRepo,
		folderRepo: folderRepo,
		accounts:   accounts,
		attRepo:    attRepo,
		linkRepo:   linkRepo,
	}
}

func seedEmailMessage(repo *stubEmailMessageRepo, tenantID, accountID, folderID uuid.UUID) *models.EmailMessage {
	m := &models.EmailMessage{
		ID:        uuid.New(),
		TenantID:  tenantID,
		AccountID: accountID,
		FolderID:  folderID,
		FromName:  "Sender",
		FromEmail: "sender@example.com",
		Subject:   "Test subject",
		Date:      time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	repo.messages[m.ID] = m
	return m
}

// ---------------------------------------------------------------------------
// Message operations
// ---------------------------------------------------------------------------

func TestListMessages_InvalidFolderID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ListMessages(context.Background(), &emailv1.ListMessagesRequest{FolderId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListMessages_ByFolder(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, folderID := uuid.New(), uuid.New()
	seedEmailMessage(f.msgRepo, tenantID, uuid.New(), folderID)

	resp, err := f.srv.ListMessages(context.Background(), &emailv1.ListMessagesRequest{FolderId: folderID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Messages, 1)
	require.EqualValues(t, 1, resp.Total)
}

func TestListMessages_Empty(t *testing.T) {
	f := newEmailMessagesFixture()
	resp, err := f.srv.ListMessages(context.Background(), &emailv1.ListMessagesRequest{FolderId: uuid.NewString()})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.Messages, "wire shape must be [] not null for an empty list")
	require.Empty(t, resp.Messages)
}

func TestListMessages_SearchUsesFolderAccount(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, accountID, folderID := uuid.New(), uuid.New(), uuid.New()
	f.folderRepo.folders[folderID] = &models.EmailFolder{ID: folderID, AccountID: accountID, Name: "INBOX", IMAPName: "INBOX"}
	m := seedEmailMessage(f.msgRepo, tenantID, accountID, folderID)
	m.Subject = "Invoice 2026-08"

	resp, err := f.srv.ListMessages(context.Background(), &emailv1.ListMessagesRequest{FolderId: folderID.String(), Search: "Invoice"})
	requireGRPCOK(t, err)
	require.Len(t, resp.Messages, 1)
}

func TestListMessages_SearchFolderNotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ListMessages(context.Background(), &emailv1.ListMessagesRequest{FolderId: uuid.NewString(), Search: "x"})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetMessage_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetMessage(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetMessageRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetMessage_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetMessage(context.Background(), &emailv1.GetMessageRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetMessage_NotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetMessage(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetMessageRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetMessage(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	att := &models.EmailAttachment{ID: uuid.New(), TenantID: tenantID, MessageID: m.ID, Filename: "a.txt", ContentType: "text/plain", SizeBytes: 3}
	f.attRepo.attachments[att.ID] = att

	resp, err := f.srv.GetMessage(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.GetMessageRequest{Id: m.ID.String()})
	requireGRPCOK(t, err)
	require.Equal(t, m.Subject, resp.Message.Subject)
	require.Len(t, resp.Message.Attachments, 1)
	require.Equal(t, "a.txt", resp.Message.Attachments[0].Filename)
}

func TestGetThreadMessages_InvalidThreadID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetThreadMessages(context.Background(), &emailv1.GetThreadMessagesRequest{ThreadId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetThreadMessages(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, threadID := uuid.New(), uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	m.ThreadID = &threadID

	resp, err := f.srv.GetThreadMessages(context.Background(), &emailv1.GetThreadMessagesRequest{ThreadId: threadID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Messages, 1)
}

func TestMarkRead_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.MarkRead(context.Background(), &emailv1.MarkReadRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMarkRead_NotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.MarkRead(context.Background(), &emailv1.MarkReadRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestMarkRead(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	m.IsRead = false

	_, err := f.srv.MarkRead(context.Background(), &emailv1.MarkReadRequest{Id: m.ID.String()})
	requireGRPCOK(t, err)
	require.True(t, f.msgRepo.messages[m.ID].IsRead)
}

func TestMarkUnread_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.MarkUnread(context.Background(), &emailv1.MarkUnreadRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMarkUnread(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	m.IsRead = true

	_, err := f.srv.MarkUnread(context.Background(), &emailv1.MarkUnreadRequest{Id: m.ID.String()})
	requireGRPCOK(t, err)
	require.False(t, f.msgRepo.messages[m.ID].IsRead)
}

func TestToggleStar_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ToggleStar(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.ToggleStarRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestToggleStar_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ToggleStar(context.Background(), &emailv1.ToggleStarRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestToggleStar(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	m.IsStarred = false

	resp, err := f.srv.ToggleStar(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.ToggleStarRequest{Id: m.ID.String()})
	requireGRPCOK(t, err)
	require.True(t, resp.IsStarred)

	resp, err = f.srv.ToggleStar(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.ToggleStarRequest{Id: m.ID.String()})
	requireGRPCOK(t, err)
	require.False(t, resp.IsStarred)
}

func TestMoveToFolder_InvalidMessageID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.MoveToFolder(context.Background(), &emailv1.MoveToFolderRequest{MessageId: "not-a-uuid", TargetFolderId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMoveToFolder_InvalidTargetFolderID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.MoveToFolder(context.Background(), &emailv1.MoveToFolderRequest{MessageId: uuid.NewString(), TargetFolderId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMoveToFolder_NotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.MoveToFolder(context.Background(), &emailv1.MoveToFolderRequest{MessageId: uuid.NewString(), TargetFolderId: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestMoveToFolder(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	targetFolder := uuid.New()

	_, err := f.srv.MoveToFolder(context.Background(), &emailv1.MoveToFolderRequest{
		MessageId:      m.ID.String(),
		TargetFolderId: targetFolder.String(),
	})
	requireGRPCOK(t, err)
	require.Equal(t, targetFolder, f.msgRepo.messages[m.ID].FolderID)
}

func TestDeleteMessage_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.DeleteMessage(context.Background(), &emailv1.DeleteMessageRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteMessage_NotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.DeleteMessage(context.Background(), &emailv1.DeleteMessageRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteMessage(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())

	_, err := f.srv.DeleteMessage(context.Background(), &emailv1.DeleteMessageRequest{Id: m.ID.String()})
	requireGRPCOK(t, err)
	require.NotContains(t, f.msgRepo.messages, m.ID)
}

func TestBulkMessageAction_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.BulkMessageAction(context.Background(), &emailv1.BulkMessageActionRequest{Ids: []string{uuid.NewString()}, Action: "read"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestBulkMessageAction_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.BulkMessageAction(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.BulkMessageActionRequest{
		Ids:    []string{"not-a-uuid"},
		Action: "read",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestBulkMessageAction_UnknownAction(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())

	_, err := f.srv.BulkMessageAction(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.BulkMessageActionRequest{
		Ids:    []string{m.ID.String()},
		Action: "not-a-real-action",
	})
	requireGRPCCode(t, err, codes.OutOfRange)
}

func TestBulkMessageAction_MoveRequiresTarget(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())

	_, err := f.srv.BulkMessageAction(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.BulkMessageActionRequest{
		Ids:    []string{m.ID.String()},
		Action: "move",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestBulkMessageAction_PartialFailureSkipsMissing proves the documented
// behavior of message.Service.BulkAction: an id that doesn't exist (or belongs
// to another tenant) is silently skipped and not counted in Affected, but does
// not fail the rest of the batch.
func TestBulkMessageAction_PartialFailureSkipsMissing(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m1 := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	m2 := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	missing := uuid.New() // syntactically valid, never seeded

	resp, err := f.srv.BulkMessageAction(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.BulkMessageActionRequest{
		Ids:    []string{m1.ID.String(), m2.ID.String(), missing.String()},
		Action: "read",
	})
	requireGRPCOK(t, err)
	require.EqualValues(t, 2, resp.Affected, "the missing id must be skipped, not fail the whole batch")
	require.True(t, f.msgRepo.messages[m1.ID].IsRead)
	require.True(t, f.msgRepo.messages[m2.ID].IsRead)
}

func TestSetReadFlag_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.SetReadFlag(context.Background(), &emailv1.SetReadFlagRequest{MessageId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetReadFlag_NotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.SetReadFlag(context.Background(), &emailv1.SetReadFlagRequest{MessageId: uuid.NewString(), IsRead: true})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSetReadFlag_MarksRead(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	m.IsRead = false

	_, err := f.srv.SetReadFlag(context.Background(), &emailv1.SetReadFlagRequest{MessageId: m.ID.String(), IsRead: true})
	requireGRPCOK(t, err)
	require.True(t, f.msgRepo.messages[m.ID].IsRead)
}

func TestSetReadFlag_MarksUnread(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	m := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	m.IsRead = true

	_, err := f.srv.SetReadFlag(context.Background(), &emailv1.SetReadFlagRequest{MessageId: m.ID.String(), IsRead: false})
	requireGRPCOK(t, err)
	require.False(t, f.msgRepo.messages[m.ID].IsRead)
}

// ---------------------------------------------------------------------------
// Send/compose operations
// ---------------------------------------------------------------------------

func TestSendEmail_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.SendEmail(context.Background(), &emailv1.SendEmailRequest{AccountId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSendEmail_InvalidAccountID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.SendEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.SendEmailRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSendEmail_InvalidContactID(t *testing.T) {
	f := newEmailMessagesFixture()
	badContact := "not-a-uuid"
	_, err := f.srv.SendEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.SendEmailRequest{
		AccountId: uuid.NewString(),
		To:        []*emailv1.EmailAddress{{Email: "to@example.com"}},
		ContactId: &badContact,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSendEmail_AccountNotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.SendEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.SendEmailRequest{
		AccountId: uuid.NewString(),
		To:        []*emailv1.EmailAddress{{Email: "to@example.com"}},
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSendEmail(t *testing.T) {
	f := newEmailMessagesFixture()
	host, port := startFakeSMTPServer(t)
	tenantID, accountID := uuid.New(), uuid.New()
	f.accounts.creds[accountID] = &send.Credentials{SMTPHost: host, SMTPPort: port, Username: "user", Password: "pass"}

	resp, err := f.srv.SendEmail(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.SendEmailRequest{
		AccountId: accountID.String(),
		To:        []*emailv1.EmailAddress{{Name: "Bob", Email: "bob@example.com"}},
		Subject:   "Hello",
		BodyText:  "Hi Bob",
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Hello", resp.Message.Subject)
	require.NotEmpty(t, f.msgRepo.messages, "the sent message must be persisted locally")
}

func TestSaveDraft_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.SaveDraft(context.Background(), &emailv1.SaveDraftRequest{AccountId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSaveDraft_InvalidAccountID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.SaveDraft(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.SaveDraftRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSaveDraft(t *testing.T) {
	f := newEmailMessagesFixture()
	resp, err := f.srv.SaveDraft(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.SaveDraftRequest{
		AccountId: uuid.NewString(),
		Subject:   "Draft",
		To:        []*emailv1.EmailAddress{{Email: "to@example.com"}},
	})
	requireGRPCOK(t, err)
	require.True(t, resp.Message.IsDraft)
	require.NotEmpty(t, f.msgRepo.messages)
}

func TestReplyEmail_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ReplyEmail(context.Background(), &emailv1.ReplyEmailRequest{AccountId: uuid.NewString(), OriginalMessageId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestReplyEmail_InvalidAccountID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ReplyEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.ReplyEmailRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestReplyEmail_InvalidOriginalMessageID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ReplyEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.ReplyEmailRequest{
		AccountId:         uuid.NewString(),
		OriginalMessageId: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestReplyEmail_OriginalMessageNotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ReplyEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.ReplyEmailRequest{
		AccountId:         uuid.NewString(),
		OriginalMessageId: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestReplyEmail(t *testing.T) {
	f := newEmailMessagesFixture()
	host, port := startFakeSMTPServer(t)
	tenantID, accountID := uuid.New(), uuid.New()
	f.accounts.creds[accountID] = &send.Credentials{SMTPHost: host, SMTPPort: port, Username: "u", Password: "p"}
	orig := seedEmailMessage(f.msgRepo, tenantID, accountID, uuid.New())
	orig.FromEmail = "orig@example.com"
	orig.Subject = "Original subject"

	resp, err := f.srv.ReplyEmail(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.ReplyEmailRequest{
		AccountId:         accountID.String(),
		OriginalMessageId: orig.ID.String(),
		BodyText:          "reply body",
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Re: Original subject", resp.Message.Subject)
}

func TestReplyEmail_ReplyAll(t *testing.T) {
	f := newEmailMessagesFixture()
	host, port := startFakeSMTPServer(t)
	tenantID, accountID := uuid.New(), uuid.New()
	f.accounts.creds[accountID] = &send.Credentials{SMTPHost: host, SMTPPort: port, Username: "u", Password: "p"}
	orig := seedEmailMessage(f.msgRepo, tenantID, accountID, uuid.New())
	orig.FromEmail = "orig@example.com"
	orig.ToAddresses = []models.EmailAddress{{Email: "orig@example.com"}, {Email: "other@example.com"}}

	resp, err := f.srv.ReplyEmail(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.ReplyEmailRequest{
		AccountId:         accountID.String(),
		OriginalMessageId: orig.ID.String(),
		BodyText:          "reply all body",
		ReplyAll:          true,
	})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.Message)
}

func TestForwardEmail_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ForwardEmail(context.Background(), &emailv1.ForwardEmailRequest{AccountId: uuid.NewString(), OriginalMessageId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestForwardEmail_InvalidAccountID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ForwardEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.ForwardEmailRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestForwardEmail_InvalidOriginalMessageID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ForwardEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.ForwardEmailRequest{
		AccountId:         uuid.NewString(),
		OriginalMessageId: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestForwardEmail_OriginalMessageNotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.ForwardEmail(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.ForwardEmailRequest{
		AccountId:         uuid.NewString(),
		OriginalMessageId: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestForwardEmail(t *testing.T) {
	f := newEmailMessagesFixture()
	host, port := startFakeSMTPServer(t)
	tenantID, accountID := uuid.New(), uuid.New()
	f.accounts.creds[accountID] = &send.Credentials{SMTPHost: host, SMTPPort: port, Username: "u", Password: "p"}
	orig := seedEmailMessage(f.msgRepo, tenantID, accountID, uuid.New())
	orig.Subject = "Original"

	resp, err := f.srv.ForwardEmail(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.ForwardEmailRequest{
		AccountId:         accountID.String(),
		OriginalMessageId: orig.ID.String(),
		To:                []*emailv1.EmailAddress{{Email: "fwd@example.com"}},
		BodyText:          "fwd body",
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Fwd: Original", resp.Message.Subject)
}

// ---------------------------------------------------------------------------
// Attachment operations
// ---------------------------------------------------------------------------

func TestUploadAttachment_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.UploadAttachment(context.Background(), &emailv1.UploadAttachmentRequest{AccountId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUploadAttachment_InvalidAccountID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.UploadAttachment(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.UploadAttachmentRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUploadAttachment(t *testing.T) {
	f := newEmailMessagesFixture()
	resp, err := f.srv.UploadAttachment(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.UploadAttachmentRequest{
		AccountId:   uuid.NewString(),
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Content:     []byte("fake-pdf-bytes"),
	})
	requireGRPCOK(t, err)
	require.NotEmpty(t, resp.Id)
	require.EqualValues(t, len("fake-pdf-bytes"), resp.SizeBytes)
	require.Contains(t, resp.MinioKey, "invoice.pdf")
}

func TestGetAttachmentDownloadURL_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetAttachmentDownloadURL(context.Background(), &emailv1.GetAttachmentDownloadURLRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAttachmentDownloadURL_InvalidID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetAttachmentDownloadURL(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetAttachmentDownloadURLRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAttachmentDownloadURL_NotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetAttachmentDownloadURL(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetAttachmentDownloadURLRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

// TestGetAttachmentDownloadURL_MetadataEmptyForRealMessageAttachment documents
// a real bug (see JOURNAL.md and the new fix-email-attachment-download-metadata-
// wrong-message-id backlog unit): the handler looks up filename/content_type/
// size_bytes via GetByMessage(ctx, uuid.Nil, tenantID) -- a hardcoded uuid.Nil,
// not the attachment's own MessageID (email_grpc.go:1131). For any attachment
// that belongs to a real message this lookup finds nothing, so the metadata
// fields stay empty even though the presigned URL itself resolves correctly.
func TestGetAttachmentDownloadURL_MetadataEmptyForRealMessageAttachment(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	att := &models.EmailAttachment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		MessageID:   uuid.New(), // a real message, not the pre-send uuid.Nil bucket
		Filename:    "contract.pdf",
		ContentType: "application/pdf",
		SizeBytes:   1234,
		MinIOKey:    "minio/contract.pdf",
	}
	f.attRepo.attachments[att.ID] = att

	resp, err := f.srv.GetAttachmentDownloadURL(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.GetAttachmentDownloadURLRequest{Id: att.ID.String()})
	requireGRPCOK(t, err)
	require.NotEmpty(t, resp.DownloadUrl, "the presigned URL itself is looked up correctly, by attachment ID")
	require.Empty(t, resp.Filename, "known bug: metadata lookup uses uuid.Nil instead of att.MessageID")
	require.Empty(t, resp.ContentType)
	require.Zero(t, resp.SizeBytes)
}

func TestGetAttachmentDownloadURL_MetadataPresentForPreSendAttachment(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	att := &models.EmailAttachment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		MessageID:   uuid.Nil, // matches UploadAttachment's pre-send convention
		Filename:    "contract.pdf",
		ContentType: "application/pdf",
		SizeBytes:   1234,
		MinIOKey:    "minio/contract.pdf",
	}
	f.attRepo.attachments[att.ID] = att

	resp, err := f.srv.GetAttachmentDownloadURL(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.GetAttachmentDownloadURLRequest{Id: att.ID.String()})
	requireGRPCOK(t, err)
	require.Equal(t, "contract.pdf", resp.Filename)
}

// ---------------------------------------------------------------------------
// CRM linking operations
// ---------------------------------------------------------------------------

func TestGetEmailContactLinks_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetEmailContactLinks(context.Background(), &emailv1.GetEmailContactLinksRequest{MessageId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetEmailContactLinks_InvalidMessageID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetEmailContactLinks(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetEmailContactLinksRequest{MessageId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetEmailContactLinks(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, msgID, contactID := uuid.New(), uuid.New(), uuid.New()
	link := &models.EmailContactLink{ID: uuid.New(), TenantID: tenantID, MessageID: msgID, ContactID: contactID, LinkType: models.LinkTypeManual, CreatedAt: time.Now()}
	f.linkRepo.links[link.ID] = link

	resp, err := f.srv.GetEmailContactLinks(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.GetEmailContactLinksRequest{MessageId: msgID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Links, 1)
}

func TestGetEmailContactLinks_Empty(t *testing.T) {
	f := newEmailMessagesFixture()
	resp, err := f.srv.GetEmailContactLinks(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetEmailContactLinksRequest{MessageId: uuid.NewString()})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.Links, "wire shape must be [] not null for an empty list")
	require.Empty(t, resp.Links)
}

func TestLinkEmailToContact_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.LinkEmailToContact(context.Background(), &emailv1.LinkEmailToContactRequest{MessageId: uuid.NewString(), ContactId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLinkEmailToContact_InvalidMessageID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.LinkEmailToContact(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.LinkEmailToContactRequest{MessageId: "not-a-uuid", ContactId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLinkEmailToContact_InvalidContactID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.LinkEmailToContact(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.LinkEmailToContactRequest{MessageId: uuid.NewString(), ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestLinkEmailToContact(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID := uuid.New()
	resp, err := f.srv.LinkEmailToContact(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.LinkEmailToContactRequest{
		MessageId: uuid.NewString(),
		ContactId: uuid.NewString(),
	})
	requireGRPCOK(t, err)
	require.Equal(t, models.LinkTypeManual, resp.Link.LinkType, "empty link_type must default to manual")
}

func TestUnlinkEmailFromContact_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.UnlinkEmailFromContact(context.Background(), &emailv1.UnlinkEmailFromContactRequest{MessageId: uuid.NewString(), ContactId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUnlinkEmailFromContact_InvalidMessageID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.UnlinkEmailFromContact(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.UnlinkEmailFromContactRequest{MessageId: "not-a-uuid", ContactId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUnlinkEmailFromContact_InvalidContactID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.UnlinkEmailFromContact(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.UnlinkEmailFromContactRequest{MessageId: uuid.NewString(), ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUnlinkEmailFromContact_NotFound(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.UnlinkEmailFromContact(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.UnlinkEmailFromContactRequest{
		MessageId: uuid.NewString(),
		ContactId: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.Internal) // stub error isn't a mapEmailError sentinel; falls through to the default branch
}

func TestUnlinkEmailFromContact(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, msgID, contactID := uuid.New(), uuid.New(), uuid.New()
	link := &models.EmailContactLink{ID: uuid.New(), TenantID: tenantID, MessageID: msgID, ContactID: contactID, LinkType: models.LinkTypeManual, CreatedAt: time.Now()}
	f.linkRepo.links[link.ID] = link

	_, err := f.srv.UnlinkEmailFromContact(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.UnlinkEmailFromContactRequest{
		MessageId: msgID.String(),
		ContactId: contactID.String(),
	})
	requireGRPCOK(t, err)
	require.Empty(t, f.linkRepo.links)
}

func TestGetContactEmails_MissingTenant(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetContactEmails(context.Background(), &emailv1.GetContactEmailsRequest{ContactId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetContactEmails_InvalidContactID(t *testing.T) {
	f := newEmailMessagesFixture()
	_, err := f.srv.GetContactEmails(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetContactEmailsRequest{ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetContactEmails(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, contactID := uuid.New(), uuid.New()
	msg := seedEmailMessage(f.msgRepo, tenantID, uuid.New(), uuid.New())
	f.linkRepo.contactMessages[contactID] = []*models.EmailMessage{msg}

	resp, err := f.srv.GetContactEmails(ctxWithActorAndTenant(uuid.New(), tenantID), &emailv1.GetContactEmailsRequest{ContactId: contactID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Messages, 1)
	require.EqualValues(t, 1, resp.Total)
}

func TestGetContactEmails_Empty(t *testing.T) {
	f := newEmailMessagesFixture()
	resp, err := f.srv.GetContactEmails(ctxWithActorAndTenant(uuid.New(), uuid.New()), &emailv1.GetContactEmailsRequest{ContactId: uuid.NewString()})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.Messages, "wire shape must be [] not null for an empty list")
	require.Empty(t, resp.Messages)
}
