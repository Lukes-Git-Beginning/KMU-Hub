package message

// DB-backed tests for internal/email/message/postgres_repository.go — until now
// only the mock-repository level (service_test.go) had coverage. Every test
// mints its own tenant(s) via testutil.EnsureTenant instead of sharing
// testutil.TenantA/TenantB, since a shared tenant across parallel tests would
// race on the UNIQUE(account_id, imap_name) / UNIQUE(folder_id, uid)
// constraints these fixtures write into.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// Registered before any row-cleanup t.Cleanup calls below: Cleanup funcs
	// run LIFO, so registering the pool close first makes it run LAST — after
	// every row deletion that still needs a live pool.
	t.Cleanup(func() { pool.Close() })
	return pool
}

func seedMessageUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("msg-repo-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

func seedEmailAccount(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"tenant_id":          tenantID,
		"user_id":            userID,
		"email_address":      fmt.Sprintf("acct-%s@tenant.local", uuid.New().String()[:8]),
		"imap_host":          "imap.tenant.local",
		"smtp_host":          "smtp.tenant.local",
		"username":           "acct",
		"password_encrypted": "vault-key",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", id) })
	return id
}

func seedEmailFolder(t *testing.T, pool *pgxpool.Pool, tenantID, accountID uuid.UUID, folderType, imapName string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "email_folders", map[string]any{
		"tenant_id":   tenantID,
		"account_id":  accountID,
		"name":        imapName,
		"imap_name":   imapName,
		"folder_type": folderType,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_folders", id) })
	return id
}

// newTestMessage builds a valid EmailMessage ready for repo.Create. uid must
// be unique within folderID (UNIQUE(folder_id, uid)).
func newTestDBMessage(tenantID, accountID, folderID uuid.UUID, uid int64, date time.Time) *models.EmailMessage {
	now := time.Now().UTC()
	msgIDHeader := fmt.Sprintf("<%s@tenant.local>", uuid.New().String())
	return &models.EmailMessage{
		ID:        uuid.New(),
		TenantID:  tenantID,
		AccountID: accountID,
		FolderID:  folderID,
		UID:       uid,
		MessageID: msgIDHeader,
		FromName:  "Absender",
		FromEmail: "absender@example.com",
		ToAddresses: []models.EmailAddress{
			{Name: "Empfaenger", Email: "empfaenger@example.com"},
		},
		CcAddresses:  []models.EmailAddress{},
		BccAddresses: []models.EmailAddress{},
		Subject:      "Testbetreff",
		Preview:      "Vorschau",
		BodyText:     "Testinhalt",
		BodyHTML:     "<p>Testinhalt</p>",
		Date:         date,
		SizeBytes:    1024,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestPostgresRepository_CreateAndGetByID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenantA, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Message Repo Tenant A")
	testutil.EnsureTenant(t, pool, tenantOther, "Message Repo Tenant Other")

	userA := seedMessageUser(t, pool, tenantA)
	accountA := seedEmailAccount(t, pool, tenantA, userA)
	folderA := seedEmailFolder(t, pool, tenantA, accountA, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)

	msg := newTestDBMessage(tenantA, accountA, folderA, 1, time.Now().UTC())
	msg.ToAddresses = []models.EmailAddress{{Name: "Bob", Email: "bob@example.com"}}
	msg.CcAddresses = []models.EmailAddress{{Name: "Carol", Email: "carol@example.com"}}
	require.NoError(t, repo.Create(ctxA, msg))

	t.Run("GetByID roundtrips addresses and defaults empty LabelIDs", func(t *testing.T) {
		got, err := repo.GetByID(ctxA, msg.ID, tenantA)
		require.NoError(t, err)
		assert.Equal(t, msg.Subject, got.Subject)
		assert.Equal(t, msg.FromEmail, got.FromEmail)
		require.Len(t, got.ToAddresses, 1)
		assert.Equal(t, "bob@example.com", got.ToAddresses[0].Email)
		require.Len(t, got.CcAddresses, 1)
		assert.Equal(t, "carol@example.com", got.CcAddresses[0].Email)
		assert.Empty(t, got.BccAddresses)
		assert.Empty(t, got.LabelIDs)
	})

	t.Run("GetByID is tenant-scoped", func(t *testing.T) {
		_, err := repo.GetByID(ctxA, msg.ID, tenantOther)
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})

	t.Run("GetByID unknown id", func(t *testing.T) {
		_, err := repo.GetByID(ctxA, uuid.New(), tenantA)
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})
}

func TestPostgresRepository_ListByFolderPaginationAndSort(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo List Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folder := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	base := time.Now().UTC().Add(-time.Hour)
	oldest := newTestDBMessage(tenant, account, folder, 1, base)
	middle := newTestDBMessage(tenant, account, folder, 2, base.Add(10*time.Minute))
	newest := newTestDBMessage(tenant, account, folder, 3, base.Add(20*time.Minute))
	for _, m := range []*models.EmailMessage{oldest, middle, newest} {
		require.NoError(t, repo.Create(ctx, m))
	}

	t.Run("default DESC sort, page 1 of 2", func(t *testing.T) {
		msgs, total, err := repo.ListByFolder(ctx, folder, ListOpts{Page: 1, PerPage: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		require.Len(t, msgs, 2)
		assert.Equal(t, newest.ID, msgs[0].ID)
		assert.Equal(t, middle.ID, msgs[1].ID)
	})

	t.Run("page 2 returns the remainder", func(t *testing.T) {
		msgs, total, err := repo.ListByFolder(ctx, folder, ListOpts{Page: 2, PerPage: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		require.Len(t, msgs, 1)
		assert.Equal(t, oldest.ID, msgs[0].ID)
	})

	t.Run("ASC sort reverses order", func(t *testing.T) {
		msgs, _, err := repo.ListByFolder(ctx, folder, ListOpts{Page: 1, PerPage: 10, SortDir: "ASC"})
		require.NoError(t, err)
		require.Len(t, msgs, 3)
		assert.Equal(t, oldest.ID, msgs[0].ID)
		assert.Equal(t, newest.ID, msgs[2].ID)
	})
}

func TestPostgresRepository_ListByThread(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo Thread Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folder := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	threadID := uuid.New()
	base := time.Now().UTC().Add(-time.Hour)

	first := newTestDBMessage(tenant, account, folder, 1, base)
	first.ThreadID = &threadID
	second := newTestDBMessage(tenant, account, folder, 2, base.Add(5*time.Minute))
	second.ThreadID = &threadID
	unrelated := newTestDBMessage(tenant, account, folder, 3, base.Add(10*time.Minute))

	for _, m := range []*models.EmailMessage{first, second, unrelated} {
		require.NoError(t, repo.Create(ctx, m))
	}

	msgs, err := repo.ListByThread(ctx, threadID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, first.ID, msgs[0].ID)
	assert.Equal(t, second.ID, msgs[1].ID)
}

func TestPostgresRepository_UpdateFlags(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo Flags Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folder := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	msg := newTestDBMessage(tenant, account, folder, 1, time.Now().UTC())
	require.NoError(t, repo.Create(ctx, msg))

	trueVal := true
	require.NoError(t, repo.UpdateFlags(ctx, msg.ID, &trueVal, nil))
	got, err := repo.GetByID(ctx, msg.ID, tenant)
	require.NoError(t, err)
	assert.True(t, got.IsRead)
	assert.False(t, got.IsStarred)

	require.NoError(t, repo.UpdateFlags(ctx, msg.ID, nil, &trueVal))
	got, err = repo.GetByID(ctx, msg.ID, tenant)
	require.NoError(t, err)
	assert.True(t, got.IsRead)
	assert.True(t, got.IsStarred)

	t.Run("nil,nil is a no-op, not an error", func(t *testing.T) {
		require.NoError(t, repo.UpdateFlags(ctx, msg.ID, nil, nil))
		got, err := repo.GetByID(ctx, msg.ID, tenant)
		require.NoError(t, err)
		assert.True(t, got.IsRead)
		assert.True(t, got.IsStarred)
	})
}

func TestPostgresRepository_UpdateThreadIDMoveDeleteAndCount(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo Move Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folderA := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")
	folderB := seedEmailFolder(t, pool, tenant, account, models.FolderTypeArchive, "Archive")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	msg := newTestDBMessage(tenant, account, folderA, 1, time.Now().UTC())
	require.NoError(t, repo.Create(ctx, msg))

	newThread := uuid.New()
	require.NoError(t, repo.UpdateThreadID(ctx, msg.ID, newThread))
	got, err := repo.GetByID(ctx, msg.ID, tenant)
	require.NoError(t, err)
	require.NotNil(t, got.ThreadID)
	assert.Equal(t, newThread, *got.ThreadID)

	require.NoError(t, repo.MoveToFolder(ctx, msg.ID, folderB))
	got, err = repo.GetByID(ctx, msg.ID, tenant)
	require.NoError(t, err)
	assert.Equal(t, folderB, got.FolderID)

	t.Run("CountUnreadByFolder counts only unread", func(t *testing.T) {
		unreadInB := newTestDBMessage(tenant, account, folderB, 2, time.Now().UTC())
		require.NoError(t, repo.Create(ctx, unreadInB))
		readInB := newTestDBMessage(tenant, account, folderB, 3, time.Now().UTC())
		require.NoError(t, repo.Create(ctx, readInB))
		trueVal := true
		require.NoError(t, repo.UpdateFlags(ctx, readInB.ID, &trueVal, nil))

		count, err := repo.CountUnreadByFolder(ctx, folderB)
		require.NoError(t, err)
		// msg (moved into folderB) is still unread, plus unreadInB.
		assert.Equal(t, 2, count)
	})

	t.Run("Delete removes a single message", func(t *testing.T) {
		victim := newTestDBMessage(tenant, account, folderA, 10, time.Now().UTC())
		require.NoError(t, repo.Create(ctx, victim))
		require.NoError(t, repo.Delete(ctx, victim.ID))
		_, err := repo.GetByID(ctx, victim.ID, tenant)
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})

	t.Run("DeleteByFolder removes every message in the folder", func(t *testing.T) {
		folderC := seedEmailFolder(t, pool, tenant, account, models.FolderTypeCustom, "Sweep")
		m1 := newTestDBMessage(tenant, account, folderC, 1, time.Now().UTC())
		m2 := newTestDBMessage(tenant, account, folderC, 2, time.Now().UTC())
		require.NoError(t, repo.Create(ctx, m1))
		require.NoError(t, repo.Create(ctx, m2))

		require.NoError(t, repo.DeleteByFolder(ctx, folderC))

		_, err := repo.GetByID(ctx, m1.ID, tenant)
		assert.ErrorIs(t, err, ErrMessageNotFound)
		_, err = repo.GetByID(ctx, m2.ID, tenant)
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})
}

func TestPostgresRepository_GetHighestUID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo UID Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folder := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	t.Run("empty folder returns 0, not -1 or an error", func(t *testing.T) {
		uid, err := repo.GetHighestUID(ctx, folder)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), uid)
	})

	for _, uid := range []int64{5, 12, 3} {
		require.NoError(t, repo.Create(ctx, newTestDBMessage(tenant, account, folder, uid, time.Now().UTC())))
	}

	t.Run("returns the max uid across all messages in the folder", func(t *testing.T) {
		uid, err := repo.GetHighestUID(ctx, folder)
		require.NoError(t, err)
		assert.Equal(t, uint32(12), uid)
	})
}

func TestPostgresRepository_GetByFolderUIDAndMessageIDHeader(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo Lookup Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folder := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	older := newTestDBMessage(tenant, account, folder, 1, time.Now().UTC().Add(-time.Hour))
	older.MessageID = "<shared@tenant.local>"
	newer := newTestDBMessage(tenant, account, folder, 2, time.Now().UTC())
	newer.MessageID = "<shared@tenant.local>"
	require.NoError(t, repo.Create(ctx, older))
	require.NoError(t, repo.Create(ctx, newer))

	t.Run("GetByFolderUID found", func(t *testing.T) {
		got, err := repo.GetByFolderUID(ctx, folder, 1)
		require.NoError(t, err)
		assert.Equal(t, older.ID, got.ID)
	})

	t.Run("GetByFolderUID not found", func(t *testing.T) {
		_, err := repo.GetByFolderUID(ctx, folder, 999)
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})

	t.Run("GetByMessageIDHeader returns the most recent match", func(t *testing.T) {
		got, err := repo.GetByMessageIDHeader(ctx, "<shared@tenant.local>")
		require.NoError(t, err)
		assert.Equal(t, newer.ID, got.ID)
	})

	t.Run("GetByMessageIDHeader not found", func(t *testing.T) {
		_, err := repo.GetByMessageIDHeader(ctx, "<unknown@tenant.local>")
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})
}

func TestPostgresRepository_FindThreadByReferences(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo References Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folder := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	threadX, threadY := uuid.New(), uuid.New()

	parentInReplyTo := newTestDBMessage(tenant, account, folder, 1, time.Now().UTC())
	parentInReplyTo.MessageID = "<in-reply-to-parent@tenant.local>"
	parentInReplyTo.ThreadID = &threadX
	require.NoError(t, repo.Create(ctx, parentInReplyTo))

	refOld := newTestDBMessage(tenant, account, folder, 2, time.Now().UTC())
	refOld.MessageID = "<ref-old@tenant.local>"
	refOld.ThreadID = &threadX
	require.NoError(t, repo.Create(ctx, refOld))

	refNew := newTestDBMessage(tenant, account, folder, 3, time.Now().UTC())
	refNew.MessageID = "<ref-new@tenant.local>"
	refNew.ThreadID = &threadY
	require.NoError(t, repo.Create(ctx, refNew))

	threadlessRef := newTestDBMessage(tenant, account, folder, 4, time.Now().UTC())
	threadlessRef.MessageID = "<threadless@tenant.local>"
	require.NoError(t, repo.Create(ctx, threadlessRef))

	t.Run("InReplyTo match wins over References", func(t *testing.T) {
		got, err := repo.FindThreadByReferences(ctx, account,
			[]string{"<ref-old@tenant.local>"}, "<in-reply-to-parent@tenant.local>")
		require.NoError(t, err)
		assert.Equal(t, threadX, got)
	})

	t.Run("no InReplyTo match falls back to References, checked last element first", func(t *testing.T) {
		got, err := repo.FindThreadByReferences(ctx, account,
			[]string{"<ref-old@tenant.local>", "<ref-new@tenant.local>"}, "")
		require.NoError(t, err)
		assert.Equal(t, threadY, got, "must check references[len-1] first, i.e. the newest reference")
	})

	t.Run("a reference whose message has no thread_id is skipped, not fatal", func(t *testing.T) {
		got, err := repo.FindThreadByReferences(ctx, account,
			[]string{"<ref-old@tenant.local>", "<threadless@tenant.local>"}, "")
		require.NoError(t, err)
		assert.Equal(t, threadX, got)
	})

	t.Run("no match at all falls back to a new thread", func(t *testing.T) {
		_, err := repo.FindThreadByReferences(ctx, account, []string{"<nope@tenant.local>"}, "<also-nope@tenant.local>")
		assert.ErrorIs(t, err, ErrThreadNotFound)
	})

	t.Run("empty references and empty InReplyTo falls back to a new thread", func(t *testing.T) {
		_, err := repo.FindThreadByReferences(ctx, account, nil, "")
		assert.ErrorIs(t, err, ErrThreadNotFound)
	})
}

func TestPostgresRepository_FindBySubjectAndParticipants(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Message Repo Subject Fallback Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)
	folder := seedEmailFolder(t, pool, tenant, account, models.FolderTypeInbox, "INBOX")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	anchor := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	threadID := uuid.New()

	msg := newTestDBMessage(tenant, account, folder, 1, anchor)
	msg.ThreadID = &threadID
	msg.Subject = "RE: Angebot 4711"
	msg.FromEmail = "kunde@example.com"
	msg.ToAddresses = []models.EmailAddress{{Name: "Vertrieb", Email: "vertrieb@tenant.local"}}
	require.NoError(t, repo.Create(ctx, msg))

	threadlessMsg := newTestDBMessage(tenant, account, folder, 2, anchor)
	threadlessMsg.Subject = "RE: Angebot 4711"
	threadlessMsg.FromEmail = "kunde@example.com"
	require.NoError(t, repo.Create(ctx, threadlessMsg))

	t.Run("matches by subject substring and from_email within the date window", func(t *testing.T) {
		got, err := repo.FindBySubjectAndParticipants(ctx, account, "Angebot 4711", "kunde@example.com",
			anchor.Add(-time.Hour), anchor.Add(time.Hour))
		require.NoError(t, err)
		assert.Equal(t, msg.ID, got.ID)
	})

	t.Run("matches via to_addresses when from_email differs (reply direction)", func(t *testing.T) {
		got, err := repo.FindBySubjectAndParticipants(ctx, account, "Angebot 4711", "vertrieb@tenant.local",
			anchor.Add(-time.Hour), anchor.Add(time.Hour))
		require.NoError(t, err)
		assert.Equal(t, msg.ID, got.ID)
	})

	t.Run("a message without a thread_id is never a fallback match", func(t *testing.T) {
		_, err := repo.FindBySubjectAndParticipants(ctx, account, "Angebot 4711", "nobody@example.com",
			anchor.Add(-time.Hour), anchor.Add(time.Hour))
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})

	t.Run("outside the date window does not match", func(t *testing.T) {
		_, err := repo.FindBySubjectAndParticipants(ctx, account, "Angebot 4711", "kunde@example.com",
			anchor.Add(2*time.Hour), anchor.Add(3*time.Hour))
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})
}

func TestPostgresFolderRepository_CRUD(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := newTestPool(t)
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Folder Repo Tenant")
	user := seedMessageUser(t, pool, tenant)
	account := seedEmailAccount(t, pool, tenant, user)

	repo := NewPostgresFolderRepository(pool)
	msgRepo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	folder := &models.EmailFolder{
		ID:         uuid.New(),
		AccountID:  account,
		Name:       "Inbox",
		IMAPName:   "INBOX",
		FolderType: models.FolderTypeInbox,
		SortOrder:  1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, folder))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_folders", folder.ID) })

	otherTypeFolder := &models.EmailFolder{
		ID:         uuid.New(),
		AccountID:  account,
		Name:       "Sent",
		IMAPName:   "Sent",
		FolderType: models.FolderTypeSent,
		SortOrder:  2,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, otherTypeFolder))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_folders", otherTypeFolder.ID) })

	t.Run("GetByID found and not found", func(t *testing.T) {
		got, err := repo.GetByID(ctx, folder.ID)
		require.NoError(t, err)
		assert.Equal(t, folder.IMAPName, got.IMAPName)

		_, err = repo.GetByID(ctx, uuid.New())
		assert.ErrorIs(t, err, ErrFolderNotFound)
	})

	t.Run("GetByIMAPName found and not found", func(t *testing.T) {
		got, err := repo.GetByIMAPName(ctx, account, "INBOX")
		require.NoError(t, err)
		assert.Equal(t, folder.ID, got.ID)

		_, err = repo.GetByIMAPName(ctx, account, "does-not-exist")
		assert.ErrorIs(t, err, ErrFolderNotFound)
	})

	t.Run("ListByAccount orders by sort_order", func(t *testing.T) {
		folders, err := repo.ListByAccount(ctx, account)
		require.NoError(t, err)
		require.Len(t, folders, 2)
		assert.Equal(t, folder.ID, folders[0].ID)
		assert.Equal(t, otherTypeFolder.ID, folders[1].ID)
	})

	t.Run("GetByAccountAndType found and not found", func(t *testing.T) {
		got, err := repo.GetByAccountAndType(ctx, account, models.FolderTypeInbox)
		require.NoError(t, err)
		assert.Equal(t, folder.ID, got.ID)

		_, err = repo.GetByAccountAndType(ctx, account, models.FolderTypeSpam)
		assert.ErrorIs(t, err, ErrFolderNotFound)
	})

	t.Run("UpdateCounts and UpdateUIDValidity persist", func(t *testing.T) {
		require.NoError(t, repo.UpdateCounts(ctx, folder.ID, 42, 7))
		require.NoError(t, repo.UpdateUIDValidity(ctx, folder.ID, 99))

		got, err := repo.GetByID(ctx, folder.ID)
		require.NoError(t, err)
		assert.Equal(t, 42, got.MessageCount)
		assert.Equal(t, 7, got.UnreadCount)
		assert.Equal(t, int64(99), got.UIDValidity)
	})

	t.Run("DeleteMessagesByFolder removes every message in the folder", func(t *testing.T) {
		m1 := newTestDBMessage(tenant, account, folder.ID, 501, time.Now().UTC())
		m2 := newTestDBMessage(tenant, account, folder.ID, 502, time.Now().UTC())
		require.NoError(t, msgRepo.Create(ctx, m1))
		require.NoError(t, msgRepo.Create(ctx, m2))

		require.NoError(t, repo.DeleteMessagesByFolder(ctx, folder.ID))

		_, err := msgRepo.GetByID(ctx, m1.ID, tenant)
		assert.ErrorIs(t, err, ErrMessageNotFound)
		_, err = msgRepo.GetByID(ctx, m2.ID, tenant)
		assert.ErrorIs(t, err, ErrMessageNotFound)
	})
}
