package attachment_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/email/attachment"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// mailbox mirrors the fixture chain in internal/email/label's tests: tenant ->
// user -> account -> folder -> message. Tenants are minted per test because
// email_accounts has UNIQUE(user_id) and shared constants collide.
type mailbox struct {
	tenant  uuid.UUID
	message uuid.UUID
	account uuid.UUID
}

func seedMailbox(t *testing.T, pool *pgxpool.Pool, name string) mailbox {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "email attachment test "+name)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "att-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Att", "last_name": "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
		"email_address": "att-" + name + "@test.local",
		"imap_host":     "imap.test.local", "smtp_host": "smtp.test.local",
		"username": "att-" + name, "password_encrypted": "x",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", accountID) })

	folderID := testutil.SeedRow(t, pool, "email_folders", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID,
		"name": "Posteingang", "imap_name": "inbox", "folder_type": "inbox",
	})

	messageID := testutil.SeedRow(t, pool, "email_messages", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "account_id": accountID, "folder_id": folderID,
		"uid": 1, "from_name": "Sender", "from_email": "sender@example.de",
		"subject": "Rechnung", "date": time.Now().UTC(),
	})

	return mailbox{tenant: tenantID, message: messageID, account: accountID}
}

func newAttachment(box mailbox, filename, key string, createdAt time.Time) *models.EmailAttachment {
	return &models.EmailAttachment{
		ID: uuid.New(), TenantID: box.tenant, MessageID: box.message,
		Filename: filename, ContentType: "application/pdf", SizeBytes: 2048,
		MinIOKey: key, ContentID: "cid-" + filename, IsInline: true, CreatedAt: createdAt,
	}
}

func TestRepository_CreateRoundTripsEveryColumn(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	box := seedMailbox(t, pool, "roundtrip")
	ctx := testutil.WithTenantCtx(context.Background(), box.tenant)

	want := newAttachment(box, "rechnung.pdf", "email-attachments/acc/1/rechnung.pdf", time.Now().UTC().Truncate(time.Microsecond))
	if err := repo.Create(ctx, want); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_attachments", want.ID) })

	got, err := repo.GetByID(ctx, want.ID, box.tenant)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != want.ID || got.TenantID != box.tenant || got.MessageID != box.message {
		t.Fatalf("identity columns = %s/%s/%s, want %s/%s/%s",
			got.ID, got.TenantID, got.MessageID, want.ID, box.tenant, box.message)
	}
	if got.Filename != want.Filename || got.ContentType != want.ContentType || got.SizeBytes != want.SizeBytes {
		t.Fatalf("file columns = %q/%q/%d, want %q/%q/%d",
			got.Filename, got.ContentType, got.SizeBytes, want.Filename, want.ContentType, want.SizeBytes)
	}
	if got.MinIOKey != want.MinIOKey || got.ContentID != want.ContentID || !got.IsInline {
		t.Fatalf("storage columns = %q/%q/inline=%v, want %q/%q/inline=true",
			got.MinIOKey, got.ContentID, got.IsInline, want.MinIOKey, want.ContentID)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
}

func TestRepository_CreateForUnknownMessageFails(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	box := seedMailbox(t, pool, "orphan")
	ctx := testutil.WithTenantCtx(context.Background(), box.tenant)

	orphan := newAttachment(box, "orphan.pdf", "email-attachments/acc/9/orphan.pdf", time.Now().UTC())
	orphan.MessageID = uuid.New()
	err := repo.Create(ctx, orphan)
	if err == nil {
		testutil.CleanupRow(t, pool, "email_attachments", orphan.ID)
		t.Fatal("Create with a message that does not exist succeeded, want FK error")
	}
}

func TestRepository_GetByMessageIsTenantScopedAndOrdered(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "list-own")
	other := seedMailbox(t, pool, "list-other")
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)

	base := time.Now().UTC().Truncate(time.Microsecond)
	// Written newest first: the ORDER BY created_at ASC must reorder them.
	newer := newAttachment(mine, "zweite.pdf", "k/zweite.pdf", base.Add(time.Minute))
	older := newAttachment(mine, "erste.pdf", "k/erste.pdf", base)
	for _, att := range []*models.EmailAttachment{newer, older} {
		if err := repo.Create(ctxMine, att); err != nil {
			t.Fatalf("Create %s: %v", att.Filename, err)
		}
		id := att.ID
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_attachments", id) })
	}

	got, err := repo.GetByMessage(ctxMine, mine.message, mine.tenant)
	if err != nil {
		t.Fatalf("GetByMessage own: %v", err)
	}
	if len(got) != 2 || got[0].ID != older.ID || got[1].ID != newer.ID {
		t.Fatalf("own list = %d rows, want [erste, zweite] ordered by created_at", len(got))
	}

	// Same message ID, foreign tenant: neither the explicit tenant filter nor
	// RLS may let the attachments through.
	foreign, err := repo.GetByMessage(ctxOther, mine.message, other.tenant)
	if err != nil {
		t.Fatalf("GetByMessage foreign: %v", err)
	}
	if len(foreign) != 0 {
		t.Fatalf("foreign tenant sees %d attachments of someone else's message, want 0", len(foreign))
	}

	// The explicit tenant argument is a second line of defence on its own:
	// own context but a foreign tenant ID must also come back empty.
	mismatch, err := repo.GetByMessage(ctxMine, mine.message, other.tenant)
	if err != nil {
		t.Fatalf("GetByMessage mismatched tenant arg: %v", err)
	}
	if len(mismatch) != 0 {
		t.Fatalf("tenant argument ignored: %d rows for a foreign tenant ID, want 0", len(mismatch))
	}
}

func TestRepository_GetByMessageWithoutAttachmentsIsEmptyNotError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	box := seedMailbox(t, pool, "empty")
	ctx := testutil.WithTenantCtx(context.Background(), box.tenant)

	got, err := repo.GetByMessage(ctx, box.message, box.tenant)
	if err != nil {
		t.Fatalf("GetByMessage: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d attachments, want 0", len(got))
	}
}

func TestRepository_LookupsByIDAreTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "id-own")
	other := seedMailbox(t, pool, "id-other")
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)

	att := newAttachment(mine, "geheim.pdf", "email-attachments/acc/7/geheim.pdf", time.Now().UTC())
	if err := repo.Create(ctxMine, att); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_attachments", att.ID) })

	key, err := repo.GetMinIOKeyByID(ctxMine, att.ID, mine.tenant)
	if err != nil || key != att.MinIOKey {
		t.Fatalf("GetMinIOKeyByID own = %q, %v, want %q", key, err, att.MinIOKey)
	}

	// RLS: foreign context with the foreign tenant ID.
	if _, err := repo.GetByID(ctxOther, att.ID, other.tenant); !errors.Is(err, attachment.ErrAttachmentNotFound) {
		t.Fatalf("GetByID across tenants: %v, want ErrAttachmentNotFound", err)
	}
	if key, err := repo.GetMinIOKeyByID(ctxOther, att.ID, other.tenant); !errors.Is(err, attachment.ErrAttachmentNotFound) || key != "" {
		t.Fatalf("GetMinIOKeyByID across tenants = %q, %v, want empty key + ErrAttachmentNotFound", key, err)
	}

	// Explicit tenant argument: own context, foreign tenant ID.
	if _, err := repo.GetByID(ctxMine, att.ID, other.tenant); !errors.Is(err, attachment.ErrAttachmentNotFound) {
		t.Fatalf("GetByID with foreign tenant arg: %v, want ErrAttachmentNotFound", err)
	}
	if _, err := repo.GetMinIOKeyByID(ctxMine, att.ID, other.tenant); !errors.Is(err, attachment.ErrAttachmentNotFound) {
		t.Fatalf("GetMinIOKeyByID with foreign tenant arg: %v, want ErrAttachmentNotFound", err)
	}

	// Unknown ID.
	if _, err := repo.GetByID(ctxMine, uuid.New(), mine.tenant); !errors.Is(err, attachment.ErrAttachmentNotFound) {
		t.Fatalf("GetByID unknown: %v, want ErrAttachmentNotFound", err)
	}
}

func TestService_GetByIDAndDownloadURLOverRealRepository(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	box := seedMailbox(t, pool, "service")
	ctx := testutil.WithTenantCtx(context.Background(), box.tenant)
	svc := attachment.NewService(attachment.NewPostgresRepository(pool), urlStore{})

	att := newAttachment(box, "angebot.pdf", "email-attachments/acc/3/angebot.pdf", time.Now().UTC())
	if err := attachment.NewPostgresRepository(pool).Create(ctx, att); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_attachments", att.ID) })

	got, err := svc.GetByID(ctx, att.ID, box.tenant)
	if err != nil || got.Filename != "angebot.pdf" {
		t.Fatalf("GetByID = %+v, %v", got, err)
	}

	url, err := svc.GetDownloadURL(ctx, att.ID, box.tenant)
	if err != nil || url != "signed:"+att.MinIOKey {
		t.Fatalf("GetDownloadURL = %q, %v, want signed:%s", url, err, att.MinIOKey)
	}

	// A tenant that does not own the attachment never reaches the store.
	if _, err := svc.GetDownloadURL(ctx, att.ID, uuid.New()); !errors.Is(err, attachment.ErrAttachmentNotFound) {
		t.Fatalf("GetDownloadURL foreign tenant: %v, want ErrAttachmentNotFound", err)
	}
}

// urlStore is an ObjectStore that only answers presign requests, echoing the
// key it was asked for so a test can see which key reached the store.
type urlStore struct{}

func (urlStore) Upload(context.Context, uuid.UUID, uint32, string, io.Reader, int64, string) (string, error) {
	return "", errors.New("not used")
}

func (urlStore) Download(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not used")
}

func (urlStore) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "signed:" + key, nil
}

func (urlStore) Delete(context.Context, string) error { return nil }
