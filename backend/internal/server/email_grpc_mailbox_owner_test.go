package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/models"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// testMailboxOwner holds every account of stubMailboxAccounts unless the
// account is listed as foreign.
var testMailboxOwner = uuid.MustParse("0b0b0b0b-0000-4000-8000-000000000001")

// mailboxCtx is a request from testMailboxOwner in tenantID.
func mailboxCtx(tenantID uuid.UUID) context.Context {
	return ctxWithActorAndTenant(testMailboxOwner, tenantID)
}

// stubMailboxAccounts answers the mailbox owner checks: every account belongs
// to testMailboxOwner except the ones in foreign, which belong to someone else.
// list feeds ListByUserAndTenant with the owner's account IDs.
type stubMailboxAccounts struct {
	foreign map[uuid.UUID]bool
	list    func() []uuid.UUID
}

func (s *stubMailboxAccounts) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailAccount, error) {
	owner := testMailboxOwner
	if s.foreign[id] {
		owner = uuid.New()
	}
	return &models.EmailAccount{ID: id, TenantID: tenantID, UserID: owner}, nil
}

func (s *stubMailboxAccounts) ListByUserAndTenant(_ context.Context, userID, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	if userID != testMailboxOwner || s.list == nil {
		return []*models.EmailAccount{}, nil
	}
	out := []*models.EmailAccount{}
	for _, id := range s.list() {
		if !s.foreign[id] {
			out = append(out, &models.EmailAccount{ID: id, TenantID: tenantID, UserID: testMailboxOwner})
		}
	}
	return out, nil
}

// TestMailbox_ForeignAccountIsInvisible pins the owner boundary of
// fix-email-module-for-members: a mailbox belongs to one user, and every RPC
// that reaches its content answers NotFound for anyone else — the same answer
// as for an ID that does not exist.
func TestMailbox_ForeignAccountIsInvisible(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, foreignAcct, foreignFolder := uuid.New(), uuid.New(), uuid.New()
	f.mailbox.foreign[foreignAcct] = true
	f.folderRepo.folders[foreignFolder] = &models.EmailFolder{ID: foreignFolder, AccountID: foreignAcct, Name: "INBOX", IMAPName: "INBOX"}
	theirs := seedEmailMessage(f.msgRepo, tenantID, foreignAcct, foreignFolder)
	threadID := uuid.New()
	theirs.ThreadID = &threadID
	att := &models.EmailAttachment{ID: uuid.New(), TenantID: tenantID, MessageID: theirs.ID, Filename: "a.txt"}
	f.attRepo.attachments[att.ID] = att
	ctx := mailboxCtx(tenantID)
	id := theirs.ID.String()

	notFound := map[string]func() error{
		"get message": func() error { _, err := f.srv.GetMessage(ctx, &emailv1.GetMessageRequest{Id: id}); return err },
		"list folder": func() error {
			_, err := f.srv.ListMessages(ctx, &emailv1.ListMessagesRequest{FolderId: foreignFolder.String()})
			return err
		},
		"filtered folder": func() error {
			_, err := f.srv.ListMessages(ctx, &emailv1.ListMessagesRequest{FolderId: foreignFolder.String(), Filter: "unread"})
			return err
		},
		"mark read": func() error { _, err := f.srv.MarkRead(ctx, &emailv1.MarkReadRequest{Id: id}); return err },
		"mark unread": func() error {
			_, err := f.srv.MarkUnread(ctx, &emailv1.MarkUnreadRequest{Id: id})
			return err
		},
		"star":   func() error { _, err := f.srv.ToggleStar(ctx, &emailv1.ToggleStarRequest{Id: id}); return err },
		"delete": func() error { _, err := f.srv.DeleteMessage(ctx, &emailv1.DeleteMessageRequest{Id: id}); return err },
		"read flag": func() error {
			_, err := f.srv.SetReadFlag(ctx, &emailv1.SetReadFlagRequest{MessageId: id, IsRead: true})
			return err
		},
		"move": func() error {
			_, err := f.srv.MoveToFolder(ctx, &emailv1.MoveToFolderRequest{MessageId: id, TargetFolderId: foreignFolder.String()})
			return err
		},
		"reply to theirs": func() error {
			_, err := f.srv.ReplyEmail(ctx, &emailv1.ReplyEmailRequest{AccountId: uuid.NewString(), OriginalMessageId: id})
			return err
		},
		"send from theirs": func() error {
			_, err := f.srv.SendEmail(ctx, &emailv1.SendEmailRequest{AccountId: foreignAcct.String(), To: []*emailv1.EmailAddress{{Email: "x@example.com"}}})
			return err
		},
		"draft in theirs": func() error {
			_, err := f.srv.SaveDraft(ctx, &emailv1.SaveDraftRequest{AccountId: foreignAcct.String()})
			return err
		},
		"contact links": func() error {
			_, err := f.srv.GetEmailContactLinks(ctx, &emailv1.GetEmailContactLinksRequest{MessageId: id})
			return err
		},
		"link to contact": func() error {
			_, err := f.srv.LinkEmailToContact(ctx, &emailv1.LinkEmailToContactRequest{MessageId: id, ContactId: uuid.NewString()})
			return err
		},
		"download attachment": func() error {
			_, err := f.srv.GetAttachmentDownloadURL(ctx, &emailv1.GetAttachmentDownloadURLRequest{Id: att.ID.String()})
			return err
		},
		"upload into theirs": func() error {
			_, err := f.srv.UploadAttachment(ctx, &emailv1.UploadAttachmentRequest{AccountId: foreignAcct.String(), Filename: "a.txt", Content: []byte("x")})
			return err
		},
	}
	for name, call := range notFound {
		t.Run(name, func(t *testing.T) { requireGRPCCode(t, call(), codes.NotFound) })
	}

	t.Run("thread hides their messages", func(t *testing.T) {
		resp, err := f.srv.GetThreadMessages(ctx, &emailv1.GetThreadMessagesRequest{ThreadId: threadID.String()})
		requireGRPCOK(t, err)
		require.Empty(t, resp.Messages)
	})
	t.Run("bulk skips their messages", func(t *testing.T) {
		resp, err := f.srv.BulkMessageAction(ctx, &emailv1.BulkMessageActionRequest{Ids: []string{id}, Action: "read"})
		requireGRPCOK(t, err)
		require.Zero(t, resp.Affected)
		require.False(t, theirs.IsRead, "a foreign message must stay untouched")
	})
	t.Run("no caller is no mailbox", func(t *testing.T) {
		_, err := f.srv.GetMessage(ctxWithTenant(tenantID), &emailv1.GetMessageRequest{Id: id})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
}

// TestMailbox_AutomationSendsWithoutCaller keeps the one internal caller
// working: the automation worker sends from the account its workflow names and
// carries no user, so SendEmail skips the owner check only then.
func TestMailbox_AutomationSendsWithoutCaller(t *testing.T) {
	f := newEmailMessagesFixture()
	tenantID, acct := uuid.New(), uuid.New()
	f.mailbox.foreign[acct] = true
	host, port := startFakeSMTPServer(t)
	f.accounts.creds[acct] = &send.Credentials{SMTPHost: host, SMTPPort: port, Username: "user", Password: "pass"}

	_, err := f.srv.SendEmail(ctxWithTenant(tenantID), &emailv1.SendEmailRequest{AccountId: acct.String(), To: []*emailv1.EmailAddress{{Email: "x@example.com"}}, Subject: "s"})
	require.NoError(t, err, "an internal send must not run the owner check, even for an account another user holds")

	_, err = f.srv.SendEmail(mailboxCtx(tenantID), &emailv1.SendEmailRequest{AccountId: acct.String(), To: []*emailv1.EmailAddress{{Email: "x@example.com"}}, Subject: "s"})
	requireGRPCCode(t, err, codes.NotFound)
}
