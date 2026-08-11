package file

// Covers the read methods on PostgresRepository that tenant_write_test.go
// does not exercise: GetFileByID, ListChannelFiles, GetFilesByMessageIDs
// and the channel/membership helper reads (IsChannelMember,
// IsChannelArchived, GetChannelRole). GetStorageQuota/IncrementUsedBytes/
// DecrementUsedBytes are already DB-tested against a real tenant pair in
// rls_storage_quota_test.go and are intentionally not repeated here.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type fileReadFixture struct {
	tenant  uuid.UUID
	user    uuid.UUID
	channel uuid.UUID
}

func newFileReadFixture(t *testing.T, pool *pgxpool.Pool) fileReadFixture {
	t.Helper()
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Chat File Read Tenant")

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("chat-file-read-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Filip",
		"last_name":     "Uploader",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user) })

	channel := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenant,
		"name":       "chat-file-read-test",
		"created_by": user,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channel) })

	return fileReadFixture{tenant: tenant, user: user, channel: channel}
}

func seedChatFile(t *testing.T, pool *pgxpool.Pool, tenantID, channelID, uploadedBy uuid.UUID, messageID *uuid.UUID, filename string, isDeleted bool, at time.Time) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":   tenantID,
		"channel_id":  channelID,
		"filename":    filename,
		"mime_type":   "application/pdf",
		"file_size":   int64(1024),
		"storage_key": fmt.Sprintf("channels/%s/%s", channelID, filename),
		"uploaded_by": uploadedBy,
		"is_deleted":  isDeleted,
		"created_at":  at,
	}
	if messageID != nil {
		cols["message_id"] = *messageID
	}
	return testutil.SeedRow(t, pool, "chat_files", cols)
}

func seedFileMessage(t *testing.T, pool *pgxpool.Pool, tenantID, channelID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantID,
		"channel_id": channelID,
		"content":    "message with file",
		"lang":       "german",
		"created_by": createdBy,
	})
}

func TestPostgresRepository_GetFileByID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newFileReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	fileID := seedChatFile(t, pool, fx.tenant, fx.channel, fx.user, nil, "report.pdf", false, time.Now().UTC())
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", fileID) })

	got, err := repo.GetFileByID(ctxOwn, fileID)
	if err != nil {
		t.Fatalf("GetFileByID: %v", err)
	}
	if got.Filename != "report.pdf" {
		t.Fatalf("GetFileByID: expected filename %q, got %q", "report.pdf", got.Filename)
	}
	if got.ChannelID != fx.channel {
		t.Fatalf("GetFileByID: expected channel %s, got %s", fx.channel, got.ChannelID)
	}

	if _, err := repo.GetFileByID(ctxOwn, uuid.New()); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("GetFileByID nonexistent: expected ErrFileNotFound, got %v", err)
	}
}

func TestPostgresRepository_ListChannelFiles(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newFileReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	now := time.Now().UTC()
	older := seedChatFile(t, pool, fx.tenant, fx.channel, fx.user, nil, "older.pdf", false, now.Add(-time.Hour))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", older) })
	newer := seedChatFile(t, pool, fx.tenant, fx.channel, fx.user, nil, "newer.pdf", false, now)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", newer) })
	deleted := seedChatFile(t, pool, fx.tenant, fx.channel, fx.user, nil, "deleted.pdf", true, now)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", deleted) })

	files, total, err := repo.ListChannelFiles(ctxOwn, fx.channel, 50, 0)
	if err != nil {
		t.Fatalf("ListChannelFiles: %v", err)
	}
	if total != 2 {
		t.Fatalf("ListChannelFiles: expected total 2 (soft-deleted excluded), got %d", total)
	}
	if len(files) != 2 {
		t.Fatalf("ListChannelFiles: expected 2 files, got %d", len(files))
	}
	if files[0].Filename != "newer.pdf" || files[1].Filename != "older.pdf" {
		t.Fatalf("ListChannelFiles: expected newest-first order, got %q then %q", files[0].Filename, files[1].Filename)
	}
	if files[0].UploaderFirstName == "" {
		t.Fatalf("ListChannelFiles: expected uploader name to be joined in")
	}

	empty, emptyTotal, err := repo.ListChannelFiles(ctxOwn, uuid.New(), 50, 0)
	if err != nil {
		t.Fatalf("ListChannelFiles (empty channel): %v", err)
	}
	if emptyTotal != 0 || len(empty) != 0 {
		t.Fatalf("ListChannelFiles (empty channel): expected no files, got total=%d len=%d", emptyTotal, len(empty))
	}
}

func TestPostgresRepository_GetFilesByMessageIDs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newFileReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	msgWithFile := seedFileMessage(t, pool, fx.tenant, fx.channel, fx.user)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", msgWithFile) })
	msgWithoutFile := seedFileMessage(t, pool, fx.tenant, fx.channel, fx.user)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", msgWithoutFile) })

	attached := seedChatFile(t, pool, fx.tenant, fx.channel, fx.user, &msgWithFile, "attached.pdf", false, time.Now().UTC())
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", attached) })
	deletedAttached := seedChatFile(t, pool, fx.tenant, fx.channel, fx.user, &msgWithFile, "deleted-attached.pdf", true, time.Now().UTC())
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", deletedAttached) })

	result, err := repo.GetFilesByMessageIDs(ctxOwn, []uuid.UUID{msgWithFile, msgWithoutFile})
	if err != nil {
		t.Fatalf("GetFilesByMessageIDs: %v", err)
	}
	files, ok := result[msgWithFile]
	if !ok || len(files) != 1 {
		t.Fatalf("GetFilesByMessageIDs: expected exactly 1 non-deleted file for msgWithFile, got %v", files)
	}
	if files[0].Filename != "attached.pdf" {
		t.Fatalf("GetFilesByMessageIDs: expected attached.pdf, got %q", files[0].Filename)
	}
	if _, ok := result[msgWithoutFile]; ok {
		t.Fatalf("GetFilesByMessageIDs: expected no entry for msgWithoutFile")
	}

	empty, err := repo.GetFilesByMessageIDs(ctxOwn, nil)
	if err != nil {
		t.Fatalf("GetFilesByMessageIDs (empty input): %v", err)
	}
	if empty != nil {
		t.Fatalf("GetFilesByMessageIDs (empty input): expected nil map, got %v", empty)
	}
}

func TestPostgresRepository_ChannelMembershipHelpers(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newFileReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	sysCtx := testutil.WithSystemCtx(context.Background())
	repo := NewPostgresRepository(pool)

	// Not a member yet.
	isMember, err := repo.IsChannelMember(ctxOwn, fx.channel, fx.user)
	if err != nil {
		t.Fatalf("IsChannelMember (before join): %v", err)
	}
	if isMember {
		t.Fatalf("IsChannelMember: expected false before a membership row exists")
	}
	if _, err := repo.GetChannelRole(ctxOwn, fx.channel, fx.user); !errors.Is(err, ErrNotChannelMember) {
		t.Fatalf("GetChannelRole (before join): expected ErrNotChannelMember, got %v", err)
	}

	if _, err := pool.Exec(sysCtx,
		`INSERT INTO channel_memberships (channel_id, user_id, tenant_id, role, joined_at) VALUES ($1, $2, $3, $4, $5)`,
		fx.channel, fx.user, fx.tenant, models.ChannelRoleAdmin, time.Now().UTC(),
	); err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	isMember, err = repo.IsChannelMember(ctxOwn, fx.channel, fx.user)
	if err != nil {
		t.Fatalf("IsChannelMember (after join): %v", err)
	}
	if !isMember {
		t.Fatalf("IsChannelMember: expected true after a membership row exists")
	}

	role, err := repo.GetChannelRole(ctxOwn, fx.channel, fx.user)
	if err != nil {
		t.Fatalf("GetChannelRole: %v", err)
	}
	if role != models.ChannelRoleAdmin {
		t.Fatalf("GetChannelRole: expected %s, got %s", models.ChannelRoleAdmin, role)
	}

	archived, err := repo.IsChannelArchived(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("IsChannelArchived: %v", err)
	}
	if archived {
		t.Fatalf("IsChannelArchived: expected false for a fresh channel")
	}

	if _, err := pool.Exec(sysCtx, `UPDATE channels SET is_archived = TRUE WHERE id = $1`, fx.channel); err != nil {
		t.Fatalf("archive channel: %v", err)
	}
	archived, err = repo.IsChannelArchived(ctxOwn, fx.channel)
	if err != nil {
		t.Fatalf("IsChannelArchived (after archive): %v", err)
	}
	if !archived {
		t.Fatalf("IsChannelArchived: expected true after archiving")
	}

	if _, err := repo.IsChannelArchived(ctxOwn, uuid.New()); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("IsChannelArchived (nonexistent channel): expected ErrFileNotFound, got %v", err)
	}
}
