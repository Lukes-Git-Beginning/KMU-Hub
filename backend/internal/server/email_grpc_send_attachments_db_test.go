package server

// DB-backed proof for fix-email-send-ignores-attachment-ids: attachment_ids of
// SendEmail, ReplyEmail, ForwardEmail and SaveDraft were never read, so an uploaded
// file never reached the mail. The bytes now travel in the MIME message, and the row
// is bound to the stored copy of the mail (in the account's Sent or Drafts folder).

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/attachment"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// capturingSMTP is a fake SMTP server that keeps the DATA of every mail it accepts.
type capturingSMTP struct {
	mu   sync.Mutex
	data []string
}

func (c *capturingSMTP) mails() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.data...)
}

func startCapturingSMTP(t *testing.T) (*capturingSMTP, string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	capture := &capturingSMTP{}
	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer conn.Close()
				reader := bufio.NewReader(conn)
				write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
				write("220 fake.smtp ESMTP")
				var body strings.Builder
				inData := false
				for {
					line, readErr := reader.ReadString('\n')
					if readErr != nil {
						return
					}
					line = strings.TrimRight(line, "\r\n")
					if inData {
						if line == "." {
							inData = false
							capture.mu.Lock()
							capture.data = append(capture.data, body.String())
							capture.mu.Unlock()
							body.Reset()
							write("250 2.0.0 OK queued")
							continue
						}
						body.WriteString(line + "\n")
						continue
					}
					switch upper := strings.ToUpper(line); {
					case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
						write("250 fake.smtp")
					case strings.HasPrefix(upper, "AUTH"):
						write("235 2.7.0 ok")
					case strings.HasPrefix(upper, "MAIL FROM"), strings.HasPrefix(upper, "RCPT TO"):
						write("250 OK")
					case strings.HasPrefix(upper, "DATA"):
						inData = true
						write("354 Go ahead")
					case strings.HasPrefix(upper, "QUIT"):
						write("221 Bye")
						return
					default:
						write("500 unrecognized command")
					}
				}
			}()
		}
	}()
	addr := ln.Addr().(*net.TCPAddr)
	return capture, "127.0.0.1", addr.Port
}

type sendAttachmentsRig struct {
	*listViewsFixture
	srv       *EmailGRPCServer
	smtp      *capturingSMTP
	owner     uuid.UUID
	ownerAcct uuid.UUID
	folder    uuid.UUID // the owner's Sent folder
	drafts    uuid.UUID
	provider  *stubSendAccountProvider
}

func newSendAttachmentsRig(t *testing.T) *sendAttachmentsRig {
	t.Helper()
	f := newListViewsFixture(t)
	capture, host, port := startCapturingSMTP(t)

	owner := f.user(t, f.tenantID)
	ownerAcct := f.account(t, f.tenantID, owner, true)
	folder := f.folder(t, f.tenantID, ownerAcct, models.FolderTypeSent)
	drafts := f.folder(t, f.tenantID, ownerAcct, models.FolderTypeDrafts)

	provider := newStubSendAccountProvider()
	provider.creds[ownerAcct] = &send.Credentials{SMTPHost: host, SMTPPort: port, Username: "acct", Password: "pw"}

	folders := message.NewPostgresFolderRepository(f.pool)
	msgSvc := message.NewService(f.repo, folders)
	srv := NewEmailGRPCServer(
		account.NewService(account.NewPostgresRepository(f.pool), nil),
		msgSvc,
		send.NewService(provider, msgSvc, nil),
		nil,
		attachment.NewService(attachment.NewPostgresRepository(f.pool), &memAttachmentStore{keys: map[string][]byte{}}),
		nil, nil, nil, nil, nil, nil, nil,
	)
	return &sendAttachmentsRig{listViewsFixture: f, srv: srv, smtp: capture, owner: owner, ownerAcct: ownerAcct, folder: folder, drafts: drafts, provider: provider}
}

func (r *sendAttachmentsRig) upload(t *testing.T, user, acct uuid.UUID, filename, content string) string {
	t.Helper()
	up, err := r.srv.UploadAttachment(r.ctx(r.tenantID, user), &emailv1.UploadAttachmentRequest{
		AccountId: acct.String(), Filename: filename, ContentType: "application/pdf", Content: []byte(content),
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, r.pool, "email_attachments", uuid.MustParse(up.Id)) })
	return up.Id
}

// boundTo reads the message an attachment row hangs on ("" while it waits).
func (r *sendAttachmentsRig) boundTo(t *testing.T, attachmentID string) string {
	t.Helper()
	var msgID *uuid.UUID
	require.NoError(t, r.pool.QueryRow(testutil.WithTenantCtx(context.Background(), r.tenantID),
		`SELECT message_id FROM email_attachments WHERE id = $1 AND tenant_id = $2`, attachmentID, r.tenantID).Scan(&msgID))
	if msgID == nil {
		return ""
	}
	return msgID.String()
}

func TestEmailGRPC_SendEmail_AttachmentIDsReachTheMailAndBindToIt(t *testing.T) {
	t.Parallel()
	r := newSendAttachmentsRig(t)
	ctx := r.ctx(r.tenantID, r.owner)
	attID := r.upload(t, r.owner, r.ownerAcct, "angebot.pdf", "pdf-bytes")

	resp, err := r.srv.SendEmail(ctx, &emailv1.SendEmailRequest{
		AccountId: r.ownerAcct.String(), To: []*emailv1.EmailAddress{{Email: "kunde@example.com"}},
		Subject: "Angebot", BodyText: "anbei", AttachmentIds: []string{attID, attID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, r.pool, "email_messages", uuid.MustParse(resp.Message.Id)) })

	mails := r.smtp.mails()
	require.Len(t, mails, 1)
	require.Contains(t, mails[0], `filename="angebot.pdf"`, "the attachment must be part of the MIME message")
	require.Contains(t, mails[0], "cGRmLWJ5dGVz", "the stored bytes (base64 of pdf-bytes) must be in the MIME message")
	require.Equal(t, 1, strings.Count(mails[0], `filename="angebot.pdf"`), "a repeated id is attached once")
	require.Contains(t, mails[0], "From: ", "the sender is the mailbox the mail leaves from")
	require.Contains(t, mails[0], "@tenant.local", "the From address comes from the account")
	require.True(t, resp.Message.HasAttachments)
	require.Equal(t, resp.Message.Id, r.boundTo(t, attID), "the row must hang on the stored copy of the mail")
}

func TestEmailGRPC_SendEmail_ForeignOrBoundAttachmentSendsNothing(t *testing.T) {
	t.Parallel()
	r := newSendAttachmentsRig(t)
	other := r.user(t, r.tenantID)
	otherAcct := r.account(t, r.tenantID, other, true)
	ctx := r.ctx(r.tenantID, r.owner)
	doSend := func(ids ...string) error {
		_, err := r.srv.SendEmail(ctx, &emailv1.SendEmailRequest{
			AccountId: r.ownerAcct.String(), To: []*emailv1.EmailAddress{{Email: "kunde@example.com"}},
			Subject: "s", AttachmentIds: ids,
		})
		return err
	}

	// An upload of a colleague's mailbox is not the caller's to send.
	foreign := r.upload(t, other, otherAcct, "geheim.pdf", "secret")
	require.Equal(t, codes.NotFound, status.Code(doSend(foreign)))
	// Neither is an ID that does not exist, nor a malformed one.
	require.Equal(t, codes.NotFound, status.Code(doSend(uuid.NewString())))
	require.Equal(t, codes.InvalidArgument, status.Code(doSend("not-a-uuid")))
	// A bound attachment (a synced mail's) cannot be attached to a new mail.
	msgID := r.msg(t, r.tenantID, r.ownerAcct, r.folder, 900, false, false)
	bound, err := r.srv.attachmentService.CreateFromStream(testutil.WithTenantCtx(context.Background(), r.tenantID),
		msgID, r.ownerAcct, 900, "rechnung.pdf", "application/pdf", 1, strings.NewReader("x"), "", false, r.tenantID)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, r.pool, "email_attachments", bound.ID) })
	require.Equal(t, codes.NotFound, status.Code(doSend(bound.ID.String())))

	require.Empty(t, r.smtp.mails(), "a rejected attachment id must stop the send before SMTP")
	require.Empty(t, r.boundTo(t, foreign), "the colleague's upload stays unbound")
}

func TestEmailGRPC_ForwardAndDraft_AttachmentIDsAreHonoured(t *testing.T) {
	t.Parallel()
	r := newSendAttachmentsRig(t)
	ctx := r.ctx(r.tenantID, r.owner)

	// Forward carries the upload in the MIME message.
	orig := r.msg(t, r.tenantID, r.ownerAcct, r.folder, 901, false, false)
	fwdAtt := r.upload(t, r.owner, r.ownerAcct, "weiter.pdf", "fwd-bytes")
	fwd, err := r.srv.ForwardEmail(ctx, &emailv1.ForwardEmailRequest{
		AccountId: r.ownerAcct.String(), OriginalMessageId: orig.String(),
		To: []*emailv1.EmailAddress{{Email: "x@example.com"}}, BodyText: "fyi", AttachmentIds: []string{fwdAtt},
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, r.pool, "email_messages", uuid.MustParse(fwd.Message.Id)) })
	mails := r.smtp.mails()
	require.Len(t, mails, 1)
	require.Contains(t, mails[0], `filename="weiter.pdf"`)
	require.Equal(t, fwd.Message.Id, r.boundTo(t, fwdAtt))

	// Reply carries it too.
	repAtt := r.upload(t, r.owner, r.ownerAcct, "antwort.pdf", "rep-bytes")
	rep, err := r.srv.ReplyEmail(ctx, &emailv1.ReplyEmailRequest{
		AccountId: r.ownerAcct.String(), OriginalMessageId: orig.String(), BodyText: "danke", AttachmentIds: []string{repAtt},
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, r.pool, "email_messages", uuid.MustParse(rep.Message.Id)) })
	mails = r.smtp.mails()
	require.Len(t, mails, 2)
	require.Contains(t, mails[1], `filename="antwort.pdf"`)
	require.Equal(t, rep.Message.Id, r.boundTo(t, repAtt))

	// A draft sends nothing but keeps the upload.
	draftAtt := r.upload(t, r.owner, r.ownerAcct, "entwurf.pdf", "draft-bytes")
	draft, err := r.srv.SaveDraft(ctx, &emailv1.SaveDraftRequest{
		AccountId: r.ownerAcct.String(), Subject: "Entwurf", AttachmentIds: []string{draftAtt},
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, r.pool, "email_messages", uuid.MustParse(draft.Message.Id)) })
	require.Len(t, r.smtp.mails(), 2, "saving a draft does not send")
	require.True(t, draft.Message.HasAttachments)
	require.Equal(t, draft.Message.Id, r.boundTo(t, draftAtt))
}
