package attachment_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/email/attachment"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// newPending builds a pre-send upload: bound to a mailbox but to no message yet.
func newPending(box mailbox, filename string) *models.EmailAttachment {
	return &models.EmailAttachment{
		ID: uuid.New(), TenantID: box.tenant, AccountID: box.account,
		Filename: filename, ContentType: "application/pdf", SizeBytes: 10,
		MinIOKey: "email-attachments/pending/" + filename, CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
}

func createAttachment(ctx context.Context, t *testing.T, repo *attachment.PostgresRepository, att *models.EmailAttachment) {
	t.Helper()
	if err := repo.Create(ctx, att); err != nil {
		t.Fatalf("Create %s: %v", att.Filename, err)
	}
	id := att.ID
	t.Cleanup(func() { testutil.CleanupRow(t, testutil.PoolFromEnv(t), "email_attachments", id) })
}

func TestRepository_PendingUploadHasNoMessageUntilAttached(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	box := seedMailbox(t, pool, "pending-null")
	ctx := testutil.WithTenantCtx(context.Background(), box.tenant)

	att := newPending(box, "vorab.pdf")
	createAttachment(ctx, t, repo, att)

	got, err := repo.GetByID(ctx, att.ID, box.tenant)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	// message_id NULL is read back as the nil UUID; the account stays.
	if got.MessageID != uuid.Nil || got.AccountID != box.account {
		t.Fatalf("pending upload message/account = %s/%s, want nil/%s", got.MessageID, got.AccountID, box.account)
	}
}

func TestRepository_AttachToMessageBindsOnlyUnboundOwnRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "attach-own")
	other := seedMailbox(t, pool, "attach-other")
	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)

	first := newPending(mine, "eins.pdf")
	second := newPending(mine, "zwei.pdf")
	untouched := newPending(mine, "drei.pdf")
	foreign := newPending(other, "fremd.pdf")
	for _, a := range []*models.EmailAttachment{first, second, untouched} {
		createAttachment(ctxMine, t, repo, a)
	}
	createAttachment(ctxOther, t, repo, foreign)

	// A foreign tenant's row named in the same list must not move (filter + RLS).
	if err := repo.AttachToMessage(ctxMine, []uuid.UUID{first.ID, second.ID, foreign.ID}, mine.message, mine.tenant); err != nil {
		t.Fatalf("AttachToMessage: %v", err)
	}

	for _, a := range []*models.EmailAttachment{first, second} {
		got, err := repo.GetByID(ctxMine, a.ID, mine.tenant)
		if err != nil || got.MessageID != mine.message {
			t.Fatalf("%s message = %v, %v, want %s", a.Filename, got, err, mine.message)
		}
		if got.AccountID != mine.account || got.MinIOKey != a.MinIOKey {
			t.Fatalf("%s lost its account/key on attach: %s/%s", a.Filename, got.AccountID, got.MinIOKey)
		}
	}
	got, err := repo.GetByID(ctxMine, untouched.ID, mine.tenant)
	if err != nil || got.MessageID != uuid.Nil {
		t.Fatalf("row not in the id list got message %v, %v, want nil", got, err)
	}
	foreignGot, err := repo.GetByID(ctxOther, foreign.ID, other.tenant)
	if err != nil || foreignGot.MessageID != uuid.Nil {
		t.Fatalf("foreign tenant's pending row = %v, %v, want untouched", foreignGot, err)
	}

	// A row that already belongs to a message keeps it: a second attach is a no-op.
	if err := repo.AttachToMessage(ctxMine, []uuid.UUID{first.ID}, uuid.New(), mine.tenant); err != nil {
		t.Fatalf("second AttachToMessage must be a no-op, got %v", err)
	}
	if got, err := repo.GetByID(ctxMine, first.ID, mine.tenant); err != nil || got.MessageID != mine.message {
		t.Fatalf("rebound row message = %v, %v, want still %s", got, err, mine.message)
	}
}

func TestRepository_AttachToMessageIgnoresForeignTenantArgument(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "attach-arg")
	other := seedMailbox(t, pool, "attach-arg-other")
	ctx := testutil.WithTenantCtx(context.Background(), mine.tenant)

	att := newPending(mine, "arg.pdf")
	createAttachment(ctx, t, repo, att)

	// Own context, foreign tenant argument: the tenant predicate alone must hold.
	if err := repo.AttachToMessage(ctx, []uuid.UUID{att.ID}, other.message, other.tenant); err != nil {
		t.Fatalf("AttachToMessage: %v", err)
	}
	got, err := repo.GetByID(ctx, att.ID, mine.tenant)
	if err != nil || got.MessageID != uuid.Nil {
		t.Fatalf("row bound through a foreign tenant argument: %v, %v", got, err)
	}
}

func TestRepository_ErrorPathsWithCancelledContext(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	box := seedMailbox(t, pool, "cancelled")
	ctx, cancel := context.WithCancel(testutil.WithTenantCtx(context.Background(), box.tenant))
	cancel()

	att := newAttachment(box, "x.pdf", "k/x.pdf", time.Now().UTC())
	if err := repo.Create(ctx, att); err == nil || !strings.Contains(err.Error(), "insert attachment") {
		t.Fatalf("Create = %v, want wrapped insert error", err)
	}
	if _, err := repo.GetByMessage(ctx, box.message, box.tenant); err == nil {
		t.Fatal("GetByMessage with a cancelled context succeeded")
	}
	// A transport error is not "not found": callers must not map it to 404.
	if _, err := repo.GetByID(ctx, att.ID, box.tenant); err == nil || errors.Is(err, attachment.ErrAttachmentNotFound) {
		t.Fatalf("GetByID = %v, want a non-NotFound error", err)
	}
	if _, err := repo.GetMinIOKeyByID(ctx, att.ID, box.tenant); err == nil || errors.Is(err, attachment.ErrAttachmentNotFound) {
		t.Fatalf("GetMinIOKeyByID = %v, want a non-NotFound error", err)
	}
	if err := repo.AttachToMessage(ctx, []uuid.UUID{att.ID}, box.message, box.tenant); err == nil || !strings.Contains(err.Error(), "attach to message") {
		t.Fatalf("AttachToMessage = %v, want wrapped error", err)
	}
}

func TestService_PendingForSendOverRealRepository(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	mine := seedMailbox(t, pool, "send-own")
	other := seedMailbox(t, pool, "send-other")
	ctx := testutil.WithTenantCtx(context.Background(), mine.tenant)
	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)
	svc := attachment.NewService(repo, urlStore{})

	a := newPending(mine, "a.pdf")
	b := newPending(mine, "b.pdf")
	otherMailbox := newPending(other, "fremdes-postfach.pdf")
	sent := newAttachment(mine, "schon-gesendet.pdf", "k/gesendet.pdf", time.Now().UTC())
	sent.AccountID = mine.account
	createAttachment(ctx, t, repo, a)
	createAttachment(ctx, t, repo, b)
	createAttachment(ctx, t, repo, sent)
	createAttachment(ctxOther, t, repo, otherMailbox)

	// Order is kept and a repeated ID collapses to one entry.
	got, err := svc.PendingForSend(ctx, []uuid.UUID{b.ID, a.ID, b.ID}, mine.account, mine.tenant)
	if err != nil || len(got) != 2 || got[0].ID != b.ID || got[1].ID != a.ID {
		t.Fatalf("PendingForSend = %v, %v, want [b a]", got, err)
	}

	empty, err := svc.PendingForSend(ctx, nil, mine.account, mine.tenant)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("PendingForSend(nil) = %#v, %v, want empty non-nil slice", empty, err)
	}

	cases := map[string]struct {
		id      uuid.UUID
		account uuid.UUID
		tenant  uuid.UUID
	}{
		"unknown id":           {uuid.New(), mine.account, mine.tenant},
		"already on a message": {sent.ID, mine.account, mine.tenant},
		"other mailbox":        {a.ID, uuid.New(), mine.tenant},
		"foreign tenant row":   {otherMailbox.ID, other.account, mine.tenant},
		"foreign tenant args":  {otherMailbox.ID, other.account, other.tenant},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.PendingForSend(ctx, []uuid.UUID{tc.id}, tc.account, tc.tenant)
			if !errors.Is(err, attachment.ErrAttachmentNotFound) {
				t.Fatalf("PendingForSend = %v, want ErrAttachmentNotFound", err)
			}
		})
	}

	// One bad ID fails the whole batch, even behind a valid one.
	if res, err := svc.PendingForSend(ctx, []uuid.UUID{a.ID, uuid.New()}, mine.account, mine.tenant); !errors.Is(err, attachment.ErrAttachmentNotFound) || res != nil {
		t.Fatalf("mixed batch = %v, %v, want nil + ErrAttachmentNotFound", res, err)
	}
}

func TestService_AttachToMessageOverRealRepository(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := attachment.NewPostgresRepository(pool)
	box := seedMailbox(t, pool, "svc-attach")
	ctx := testutil.WithTenantCtx(context.Background(), box.tenant)
	svc := attachment.NewService(repo, urlStore{})

	a := newPending(box, "a.pdf")
	b := newPending(box, "b.pdf")
	createAttachment(ctx, t, repo, a)
	createAttachment(ctx, t, repo, b)

	if err := svc.AttachToMessage(ctx, nil, box.message, box.tenant); err != nil {
		t.Fatalf("AttachToMessage(nil): %v", err)
	}
	if got, _ := repo.GetByID(ctx, a.ID, box.tenant); got.MessageID != uuid.Nil {
		t.Fatalf("empty attach list bound a row to %s", got.MessageID)
	}

	if err := svc.AttachToMessage(ctx, []*models.EmailAttachment{a, b}, box.message, box.tenant); err != nil {
		t.Fatalf("AttachToMessage: %v", err)
	}
	list, err := svc.GetByMessage(ctx, box.message, box.tenant)
	if err != nil || len(list) != 2 {
		t.Fatalf("GetByMessage after attach = %d rows, %v, want 2", len(list), err)
	}
}

// bytesStore serves bytes derived from the key for Open.
type bytesStore struct{ urlStore }

func (bytesStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	if key == "missing" {
		return nil, errors.New("no such key")
	}
	return io.NopCloser(strings.NewReader("inhalt:" + key)), nil
}

func TestService_OpenStreamsTheStoredKey(t *testing.T) {
	svc := attachment.NewService(nil, bytesStore{})

	rc, err := svc.Open(context.Background(), &models.EmailAttachment{MinIOKey: "email-attachments/a/1/x.pdf"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = rc.Close() }()
	body, _ := io.ReadAll(rc)
	if string(body) != "inhalt:email-attachments/a/1/x.pdf" {
		t.Fatalf("Open read %q, want the bytes of the stored key", body)
	}

	if _, err := svc.Open(context.Background(), &models.EmailAttachment{MinIOKey: "missing"}); err == nil {
		t.Fatal("Open must surface the store's error")
	}
}
