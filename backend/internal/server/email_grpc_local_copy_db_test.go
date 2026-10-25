package server

// DB-backed proof for fix-email-sent-copy-and-draft-have-no-folder: the send path
// stored its local copy with folder_id = uuid.Nil and uid = 0, so the INSERT always
// failed on email_messages_folder_id_fkey. SaveDraft answered 500 for every user, and
// a sent mail never reached the Sent folder. The copy now lands in the account's
// Sent/Drafts folder with a negative UID, and the IMAP sync adopts it instead of
// storing the server's copy a second time.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

type storedCopy struct {
	folderID  uuid.UUID
	uid       int64
	isDraft   bool
	messageID string
}

// storedCopy reads the row behind a returned message ID; ok is false when none exists.
func (r *sendAttachmentsRig) storedCopy(t *testing.T, id string) (storedCopy, bool) {
	t.Helper()
	var c storedCopy
	err := r.pool.QueryRow(testutil.WithTenantCtx(context.Background(), r.tenantID),
		`SELECT folder_id, uid, is_draft, message_id FROM email_messages WHERE id = $1 AND tenant_id = $2`,
		id, r.tenantID).Scan(&c.folderID, &c.uid, &c.isDraft, &c.messageID)
	if err != nil {
		return storedCopy{}, false
	}
	return c, true
}

func TestEmailGRPC_SaveDraft_StoresTheDraftInTheDraftsFolder(t *testing.T) {
	t.Parallel()
	r := newSendAttachmentsRig(t)

	draft, err := r.srv.SaveDraft(r.ctx(r.tenantID, r.owner), &emailv1.SaveDraftRequest{
		AccountId: r.ownerAcct.String(), To: []*emailv1.EmailAddress{{Email: "kunde@example.com"}},
		Subject: "Entwurf", BodyText: "noch nicht fertig",
	})
	require.NoError(t, err)

	c, ok := r.storedCopy(t, draft.Message.Id)
	require.True(t, ok, "the draft must be stored")
	require.Equal(t, r.drafts, c.folderID, "a draft belongs in the Drafts folder")
	require.True(t, c.isDraft)
	require.Less(t, c.uid, int64(0), "a local copy carries a negative UID")
	require.Empty(t, r.smtp.mails(), "saving a draft does not send")
}

func TestEmailGRPC_SendEmail_StoresEachMailInTheSentFolder(t *testing.T) {
	t.Parallel()
	r := newSendAttachmentsRig(t)
	ctx := r.ctx(r.tenantID, r.owner)
	doSend := func(subject string) string {
		resp, err := r.srv.SendEmail(ctx, &emailv1.SendEmailRequest{
			AccountId: r.ownerAcct.String(), To: []*emailv1.EmailAddress{{Email: "kunde@example.com"}},
			Subject: subject, BodyText: "hallo",
		})
		require.NoError(t, err)
		return resp.Message.Id
	}

	first, second := doSend("eins"), doSend("zwei")
	require.Len(t, r.smtp.mails(), 2)
	a, ok := r.storedCopy(t, first)
	require.True(t, ok, "the first sent mail must be stored")
	b, ok := r.storedCopy(t, second)
	require.True(t, ok, "a second send must not collide on (folder_id, uid)")
	require.Equal(t, r.folder, a.folderID, "a sent mail belongs in the Sent folder")
	require.Equal(t, r.folder, b.folderID)
	require.False(t, a.isDraft)
	require.NotEqual(t, a.uid, b.uid)
	require.Less(t, a.uid, int64(0))
	require.Less(t, b.uid, int64(0))
}

func TestEmailGRPC_SendWithoutFolders_SendsAndDraftIsRefused(t *testing.T) {
	t.Parallel()
	r := newSendAttachmentsRig(t)
	// A mailbox that was never synced has no folders at all.
	user := r.user(t, r.tenantID)
	acct := r.account(t, r.tenantID, user, true)
	r.provider.creds[acct] = r.provider.creds[r.ownerAcct]
	ctx := r.ctx(r.tenantID, user)

	resp, err := r.srv.SendEmail(ctx, &emailv1.SendEmailRequest{
		AccountId: acct.String(), To: []*emailv1.EmailAddress{{Email: "kunde@example.com"}}, Subject: "s",
	})
	require.NoError(t, err, "a missing Sent folder must not fail a mail that went out")
	require.Len(t, r.smtp.mails(), 1)
	_, stored := r.storedCopy(t, resp.Message.Id)
	require.False(t, stored, "without a Sent folder there is nowhere to keep the copy")

	_, err = r.srv.SaveDraft(ctx, &emailv1.SaveDraftRequest{AccountId: acct.String(), Subject: "Entwurf"})
	require.Equal(t, codes.NotFound, status.Code(err), "without a Drafts folder the draft cannot be kept")
}

func TestEmailSync_AdoptsTheLocalCopyOfASentMail(t *testing.T) {
	t.Parallel()
	r := newSendAttachmentsRig(t)
	tctx := testutil.WithTenantCtx(context.Background(), r.tenantID)
	msgSvc := message.NewService(r.repo, message.NewPostgresFolderRepository(r.pool))

	resp, err := r.srv.SendEmail(r.ctx(r.tenantID, r.owner), &emailv1.SendEmailRequest{
		AccountId: r.ownerAcct.String(), To: []*emailv1.EmailAddress{{Email: "kunde@example.com"}}, Subject: "Angebot",
	})
	require.NoError(t, err)
	local, ok := r.storedCopy(t, resp.Message.Id)
	require.True(t, ok)

	highest, err := msgSvc.GetHighestUID(tctx, r.folder)
	require.NoError(t, err)
	require.Zero(t, highest, "a local copy must not move the delta sync past the server's mails")

	// The server hands back the same mail; the IMAP envelope drops the angle brackets.
	now := time.Now().UTC()
	synced := &models.EmailMessage{
		ID: uuid.New(), TenantID: r.tenantID, AccountID: r.ownerAcct, FolderID: r.folder, UID: 77,
		MessageID: strings.Trim(local.messageID, "<>"), FromEmail: "acct@tenant.local", Subject: "Angebot",
		ToAddresses: []models.EmailAddress{}, CcAddresses: []models.EmailAddress{}, BccAddresses: []models.EmailAddress{},
		Date: now, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, msgSvc.CreateSynced(tctx, synced))

	var rows int
	require.NoError(t, r.pool.QueryRow(tctx,
		`SELECT count(*) FROM email_messages WHERE folder_id = $1 AND btrim(message_id, '<>') = $2`,
		r.folder, synced.MessageID).Scan(&rows))
	require.Equal(t, 1, rows, "the synced mail must not be stored a second time")
	adopted, ok := r.storedCopy(t, resp.Message.Id)
	require.True(t, ok)
	require.Equal(t, int64(77), adopted.uid, "the local copy takes over the server's UID")

	highest, err = msgSvc.GetHighestUID(tctx, r.folder)
	require.NoError(t, err)
	require.Equal(t, uint32(77), highest)

	// A mail the app did not send is stored normally.
	other := *synced
	other.ID, other.UID, other.MessageID = uuid.New(), 78, uuid.NewString()+"@elsewhere.example"
	require.NoError(t, msgSvc.CreateSynced(tctx, &other))
	_, ok = r.storedCopy(t, other.ID.String())
	require.True(t, ok)
}
