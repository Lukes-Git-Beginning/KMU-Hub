package server

// DB-backed proof for fix-email-attachment-upload-nil-message-fk: UploadAttachment stored
// message_id = uuid.Nil, which violates email_attachments_message_id_fkey, so every upload
// failed with 500. The attachment now waits without a message and belongs to its mailbox:
// the owner downloads it, everybody else gets NotFound.

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/attachment"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// memAttachmentStore keeps uploaded bytes in memory instead of MinIO.
type memAttachmentStore struct{ keys map[string][]byte }

func (m *memAttachmentStore) Upload(_ context.Context, accountID uuid.UUID, _ uint32, filename string, r io.Reader, _ int64, _ string) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	key := "mem/" + accountID.String() + "/" + uuid.NewString() + "/" + filename
	m.keys[key] = b
	return key, nil
}

func (m *memAttachmentStore) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://files.test/" + key, nil
}

func (m *memAttachmentStore) Delete(_ context.Context, key string) error {
	delete(m.keys, key)
	return nil
}

func TestEmailGRPC_UploadAttachment_PreSendUploadBelongsToItsMailbox(t *testing.T) {
	t.Parallel()
	f := newListViewsFixture(t)
	store := &memAttachmentStore{keys: map[string][]byte{}}
	srv := NewEmailGRPCServer(
		account.NewService(account.NewPostgresRepository(f.pool), nil),
		message.NewService(f.repo, message.NewPostgresFolderRepository(f.pool)),
		nil, nil,
		attachment.NewService(attachment.NewPostgresRepository(f.pool), store),
		nil, nil, nil, nil, nil, nil, nil,
	)

	owner, other := f.user(t, f.tenantID), f.user(t, f.tenantID)
	ownerAcct := f.account(t, f.tenantID, owner, true)
	f.account(t, f.tenantID, other, true)

	up, err := srv.UploadAttachment(f.ctx(f.tenantID, owner), &emailv1.UploadAttachmentRequest{
		AccountId: ownerAcct.String(), Filename: "angebot.pdf", ContentType: "application/pdf", Content: []byte("pdf-bytes"),
	})
	require.NoError(t, err, "the upload must not hit the message foreign key")
	require.NotEmpty(t, up.Id)
	require.Equal(t, int64(9), up.SizeBytes)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "email_attachments", uuid.MustParse(up.Id)) })

	// The row waits without a message and carries its mailbox.
	var msgNull bool
	var acct uuid.UUID
	require.NoError(t, f.pool.QueryRow(testutil.WithTenantCtx(context.Background(), f.tenantID),
		`SELECT message_id IS NULL, account_id FROM email_attachments WHERE id = $1 AND tenant_id = $2`,
		up.Id, f.tenantID).Scan(&msgNull, &acct))
	require.True(t, msgNull)
	require.Equal(t, ownerAcct, acct)

	// The owner downloads it.
	dl, err := srv.GetAttachmentDownloadURL(f.ctx(f.tenantID, owner), &emailv1.GetAttachmentDownloadURLRequest{Id: up.Id})
	require.NoError(t, err)
	require.Equal(t, "angebot.pdf", dl.Filename)
	require.Equal(t, "application/pdf", dl.ContentType)
	require.Contains(t, dl.DownloadUrl, "angebot.pdf")

	// A second user in the same tenant gets NotFound, and cannot upload into the foreign mailbox.
	_, err = srv.GetAttachmentDownloadURL(f.ctx(f.tenantID, other), &emailv1.GetAttachmentDownloadURLRequest{Id: up.Id})
	require.Equal(t, codes.NotFound, status.Code(err))
	_, err = srv.UploadAttachment(f.ctx(f.tenantID, other), &emailv1.UploadAttachmentRequest{
		AccountId: ownerAcct.String(), Filename: "x.txt", Content: []byte("x"),
	})
	require.Equal(t, codes.NotFound, status.Code(err))

	// A message-bound attachment (synced mail) keeps the message-based check.
	folder := f.folder(t, f.tenantID, ownerAcct, "inbox")
	msgID := f.msg(t, f.tenantID, ownerAcct, folder, 1, false, false)
	bound, err := srv.attachmentService.CreateFromStream(
		testutil.WithTenantCtx(context.Background(), f.tenantID),
		msgID, ownerAcct, 1, "rechnung.pdf", "application/pdf", 3, bytes.NewReader([]byte("abc")), "", false, f.tenantID)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "email_attachments", bound.ID) })
	_, err = srv.GetAttachmentDownloadURL(f.ctx(f.tenantID, owner), &emailv1.GetAttachmentDownloadURLRequest{Id: bound.ID.String()})
	require.NoError(t, err)
	_, err = srv.GetAttachmentDownloadURL(f.ctx(f.tenantID, other), &emailv1.GetAttachmentDownloadURLRequest{Id: bound.ID.String()})
	require.Equal(t, codes.NotFound, status.Code(err))
}
