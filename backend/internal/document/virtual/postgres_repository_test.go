package virtual

// DB-level tests for PostgresRepository. Every fixture is a fresh tenant, so
// tests are safe under t.Parallel() without cross-test collisions.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedVirtualUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("virtual-%s-%s@test.invalid", label, uuid.New().String()[:8]),
		"password_hash": "x",
		"first_name":    "Virtual",
		"last_name":     label,
	})
}

func seedVirtualChannel(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID, name string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantID,
		"name":       name,
		"created_by": createdBy,
	})
}

func addChannelMembership(t *testing.T, pool *pgxpool.Pool, tenantID, channelID, userID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx,
		`INSERT INTO channel_memberships (channel_id, user_id, tenant_id) VALUES ($1, $2, $3)`,
		channelID, userID, tenantID,
	); err != nil {
		t.Fatalf("seed channel_memberships: %v", err)
	}
}

func seedVirtualChatFile(t *testing.T, pool *pgxpool.Pool, tenantID, channelID, uploadedBy uuid.UUID, filename string, at time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "chat_files", map[string]any{
		"tenant_id":   tenantID,
		"channel_id":  channelID,
		"filename":    filename,
		"mime_type":   "application/pdf",
		"file_size":   int64(2048),
		"storage_key": "channels/" + uuid.New().String(),
		"uploaded_by": uploadedBy,
		"created_at":  at,
	})
}

func seedVirtualEmailAccount(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, displayName string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"tenant_id":          tenantID,
		"user_id":            userID,
		"email_address":      fmt.Sprintf("acct-%s@example.com", uuid.New().String()[:8]),
		"display_name":       displayName,
		"imap_host":          "imap.example.com",
		"smtp_host":          "smtp.example.com",
		"username":           "testuser",
		"password_encrypted": "enc:dummy",
	})
}

func seedVirtualEmailFolder(t *testing.T, pool *pgxpool.Pool, tenantID, accountID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "email_folders", map[string]any{
		"tenant_id":  tenantID,
		"account_id": accountID,
		"name":       "INBOX",
		"imap_name":  "INBOX",
	})
}

// seedVirtualEmailMessage inserts an email_messages row. An empty subject
// omits the column entirely (NULL) rather than inserting "" — the
// ListEmailAttachments query falls back to from_email via
// COALESCE(LEFT(subject,80), from_email), which only fires on NULL, not on
// an empty string.
func seedVirtualEmailMessage(t *testing.T, pool *pgxpool.Pool, tenantID, accountID, folderID uuid.UUID, uid int, subject string, at time.Time) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":  tenantID,
		"account_id": accountID,
		"folder_id":  folderID,
		"uid":        uid,
		"from_email": "sender@example.com",
		"date":       at,
	}
	if subject != "" {
		cols["subject"] = subject
	}
	return testutil.SeedRow(t, pool, "email_messages", cols)
}

func seedVirtualEmailAttachment(t *testing.T, pool *pgxpool.Pool, tenantID, messageID uuid.UUID, filename string, at time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "email_attachments", map[string]any{
		"tenant_id":    tenantID,
		"message_id":   messageID,
		"filename":     filename,
		"content_type": "application/pdf",
		"size_bytes":   int64(4096),
		"minio_key":    "emails/" + uuid.New().String(),
		"created_at":   at,
	})
}

func seedVirtualProject(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":        tenantID,
		"name":             "Virtual Test Project",
		"project_key":      "VF" + uuid.New().String()[:6],
		"next_task_number": 1,
		"created_by":       createdBy,
	})
}

func addProjectMember(t *testing.T, pool *pgxpool.Pool, tenantID, projectID, userID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx,
		`INSERT INTO project_members (project_id, user_id, tenant_id) VALUES ($1, $2, $3)`,
		projectID, userID, tenantID,
	); err != nil {
		t.Fatalf("seed project_members: %v", err)
	}
}

func seedVirtualTask(t *testing.T, pool *pgxpool.Pool, tenantID, projectID, createdBy uuid.UUID, taskNumber int, assigneeID *uuid.UUID) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"tenant_id":   tenantID,
		"project_id":  projectID,
		"title":       "Virtual Test Task",
		"task_number": taskNumber,
		"created_by":  createdBy,
	}
	if assigneeID != nil {
		cols["assignee_id"] = *assigneeID
	}
	return testutil.SeedRow(t, pool, "tasks", cols)
}

func seedVirtualTaskFile(t *testing.T, pool *pgxpool.Pool, tenantID, taskID, uploadedBy uuid.UUID, filename string, at time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "task_files", map[string]any{
		"tenant_id":   tenantID,
		"task_id":     taskID,
		"filename":    filename,
		"mime_type":   "application/pdf",
		"file_size":   int64(1024),
		"storage_key": "tasks/" + uuid.New().String(),
		"uploaded_by": uploadedBy,
		"created_at":  at,
	})
}

// --- ListChatFiles ---

func TestPostgresRepository_ListChatFiles_EmptyForNonMember(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual Chat Non-Member Tenant")
	owner := seedVirtualUser(t, pool, tenant, "owner")
	outsider := seedVirtualUser(t, pool, tenant, "outsider")
	channel := seedVirtualChannel(t, pool, tenant, owner, "private-channel")
	addChannelMembership(t, pool, tenant, channel, owner)
	seedVirtualChatFile(t, pool, tenant, channel, owner, "report.pdf", time.Now().UTC())

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListChatFiles(ctx, outsider, 20, 0)
	if err != nil {
		t.Fatalf("ListChatFiles (non-member): unexpected error %v", err)
	}
	if total != 0 || len(files) != 0 {
		t.Fatalf("ListChatFiles (non-member): expected 0 files, got total=%d len=%d", total, len(files))
	}
}

func TestPostgresRepository_ListChatFiles_UploadedByName(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual Chat Uploaded By Name Tenant")
	member := seedVirtualUser(t, pool, tenant, "member")
	channel := seedVirtualChannel(t, pool, tenant, member, "member-channel")
	addChannelMembership(t, pool, tenant, channel, member)
	seedVirtualChatFile(t, pool, tenant, channel, member, "report.pdf", time.Now().UTC())

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListChatFiles(ctx, member, 20, 0)
	if err != nil {
		t.Fatalf("ListChatFiles: unexpected error %v", err)
	}
	if total != 1 || len(files) != 1 {
		t.Fatalf("ListChatFiles: expected 1 file, got total=%d len=%d", total, len(files))
	}
	if files[0].UploadedByName != "Virtual member" {
		t.Fatalf("ListChatFiles: uploaded_by_name=%q, want %q", files[0].UploadedByName, "Virtual member")
	}
}

// --- ListEmailAttachments ---

func TestPostgresRepository_ListEmailAttachments_HappyPathAndIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual Email Tenant")

	userOwn := seedVirtualUser(t, pool, tenant, "own")
	accountOwn := seedVirtualEmailAccount(t, pool, tenant, userOwn, "Own Mailbox")
	folderOwn := seedVirtualEmailFolder(t, pool, tenant, accountOwn)

	older := time.Now().UTC().Add(-time.Hour)
	newer := time.Now().UTC()
	msgOlder := seedVirtualEmailMessage(t, pool, tenant, accountOwn, folderOwn, 1, "Older subject that is long enough to matter for truncation checks", older)
	msgNewer := seedVirtualEmailMessage(t, pool, tenant, accountOwn, folderOwn, 2, "", newer)
	attOlder := seedVirtualEmailAttachment(t, pool, tenant, msgOlder, "old.pdf", older)
	attNewer := seedVirtualEmailAttachment(t, pool, tenant, msgNewer, "new.pdf", newer)

	// A different user's mailbox in the same tenant must not leak in.
	userOther := seedVirtualUser(t, pool, tenant, "other")
	accountOther := seedVirtualEmailAccount(t, pool, tenant, userOther, "Other Mailbox")
	folderOther := seedVirtualEmailFolder(t, pool, tenant, accountOther)
	msgOther := seedVirtualEmailMessage(t, pool, tenant, accountOther, folderOther, 1, "Other user's subject", newer)
	seedVirtualEmailAttachment(t, pool, tenant, msgOther, "other.pdf", newer)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListEmailAttachments(ctx, userOwn, 20, 0)
	if err != nil {
		t.Fatalf("ListEmailAttachments: %v", err)
	}
	if total != 2 || len(files) != 2 {
		t.Fatalf("ListEmailAttachments: expected 2 files, got total=%d len=%d", total, len(files))
	}
	// ORDER BY ea.created_at DESC.
	if files[0].ID != attNewer || files[1].ID != attOlder {
		t.Fatalf("ListEmailAttachments: expected newest-first order [%s,%s], got [%s,%s]", attNewer, attOlder, files[0].ID, files[1].ID)
	}
	newest := files[0]
	if newest.SourceType != "email" {
		t.Fatalf("ListEmailAttachments: source_type=%q, want email", newest.SourceType)
	}
	if newest.SourceID != msgNewer {
		t.Fatalf("ListEmailAttachments: source_id=%s, want message id %s", newest.SourceID, msgNewer)
	}
	// Empty subject falls back to from_email per COALESCE(LEFT(subject,80), from_email).
	if newest.SourceName != "sender@example.com" {
		t.Fatalf("ListEmailAttachments: source_name=%q, want fallback to from_email", newest.SourceName)
	}
	if newest.UploadedBy != userOwn {
		t.Fatalf("ListEmailAttachments: uploaded_by=%s, want %s", newest.UploadedBy, userOwn)
	}
	if newest.UploadedByName != "Own Mailbox" {
		t.Fatalf("ListEmailAttachments: uploaded_by_name=%q, want display_name", newest.UploadedByName)
	}

	oldest := files[1]
	wantSubject := "Older subject that is long enough to matter for truncation checks"
	if oldest.SourceName != wantSubject {
		t.Fatalf("ListEmailAttachments: source_name=%q, want subject %q (LEFT(subject,80) of a %d-char string)", oldest.SourceName, wantSubject, len(wantSubject))
	}
}

func TestPostgresRepository_ListEmailAttachments_EmptyForUserWithNoAccount(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual Email Empty Tenant")
	userNoAccount := seedVirtualUser(t, pool, tenant, "no-account")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListEmailAttachments(ctx, userNoAccount, 20, 0)
	if err != nil {
		t.Fatalf("ListEmailAttachments (no account): unexpected error %v", err)
	}
	if total != 0 || len(files) != 0 {
		t.Fatalf("ListEmailAttachments (no account): expected 0 files, got total=%d len=%d", total, len(files))
	}
}

// --- ListTaskFiles ---

func TestPostgresRepository_ListTaskFiles_EmptyForNoAccess(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual Task No Access Tenant")
	owner := seedVirtualUser(t, pool, tenant, "task-owner")
	outsider := seedVirtualUser(t, pool, tenant, "task-outsider")
	project := seedVirtualProject(t, pool, tenant, owner)
	task := seedVirtualTask(t, pool, tenant, project, owner, 1, &owner)
	seedVirtualTaskFile(t, pool, tenant, task, owner, "spec.pdf", time.Now().UTC())

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListTaskFiles(ctx, outsider, 20, 0)
	if err != nil {
		t.Fatalf("ListTaskFiles (no access): unexpected error %v", err)
	}
	if total != 0 || len(files) != 0 {
		t.Fatalf("ListTaskFiles (no access): expected 0 files, got total=%d len=%d", total, len(files))
	}
}

func TestPostgresRepository_ListTaskFiles_UploadedByNameViaProjectMembership(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual Task Uploaded By Name Tenant")
	owner := seedVirtualUser(t, pool, tenant, "task-owner")
	member := seedVirtualUser(t, pool, tenant, "task-member")
	project := seedVirtualProject(t, pool, tenant, owner)
	addProjectMember(t, pool, tenant, project, member)
	// Access via project membership, not direct assignment.
	task := seedVirtualTask(t, pool, tenant, project, owner, 1, nil)
	seedVirtualTaskFile(t, pool, tenant, task, owner, "spec.pdf", time.Now().UTC())

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListTaskFiles(ctx, member, 20, 0)
	if err != nil {
		t.Fatalf("ListTaskFiles: unexpected error %v", err)
	}
	if total != 1 || len(files) != 1 {
		t.Fatalf("ListTaskFiles: expected 1 file, got total=%d len=%d", total, len(files))
	}
	if files[0].UploadedByName != "Virtual task-owner" {
		t.Fatalf("ListTaskFiles: uploaded_by_name=%q, want %q", files[0].UploadedByName, "Virtual task-owner")
	}
}

// --- ListAll ---

func TestPostgresRepository_ListAll_UnknownSourceTypeReturnsEmpty(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual ListAll Unknown Source Tenant")
	user := seedVirtualUser(t, pool, tenant, "unknown-source")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListAll(ctx, user, "not-a-real-source", 20, 0)
	if err != nil {
		t.Fatalf("ListAll (unknown source): unexpected error %v", err)
	}
	if total != 0 || len(files) != 0 {
		t.Fatalf("ListAll (unknown source): expected 0 files, got total=%d len=%d", total, len(files))
	}
}

func TestPostgresRepository_ListAll_DelegatesEmailSourceType(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual ListAll Email Delegate Tenant")
	user := seedVirtualUser(t, pool, tenant, "delegate")
	account := seedVirtualEmailAccount(t, pool, tenant, user, "Delegate Mailbox")
	folder := seedVirtualEmailFolder(t, pool, tenant, account)
	msg := seedVirtualEmailMessage(t, pool, tenant, account, folder, 1, "Delegate subject", time.Now().UTC())
	seedVirtualEmailAttachment(t, pool, tenant, msg, "delegate.pdf", time.Now().UTC())

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	direct, directTotal, err := repo.ListEmailAttachments(ctx, user, 20, 0)
	if err != nil {
		t.Fatalf("ListEmailAttachments: %v", err)
	}

	viaAll, allTotal, err := repo.ListAll(ctx, user, "email", 20, 0)
	if err != nil {
		t.Fatalf("ListAll (source=email): %v", err)
	}
	if allTotal != directTotal || len(viaAll) != len(direct) {
		t.Fatalf("ListAll (source=email) did not match direct ListEmailAttachments: total %d vs %d, len %d vs %d", allTotal, directTotal, len(viaAll), len(direct))
	}
	if len(viaAll) != 1 || viaAll[0].ID != direct[0].ID {
		t.Fatalf("ListAll (source=email): unexpected result set %+v", viaAll)
	}
}

func TestPostgresRepository_ListAll_EmptyAcrossAllSources(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual ListAll Empty Tenant")
	user := seedVirtualUser(t, pool, tenant, "no-access-anywhere")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListAll(ctx, user, "", 20, 0)
	if err != nil {
		t.Fatalf("ListAll (empty union, real happy path): unexpected error %v", err)
	}
	if total != 0 || len(files) != 0 {
		t.Fatalf("ListAll (empty union): expected 0 files, got total=%d len=%d", total, len(files))
	}
}

func TestPostgresRepository_ListAll_UnionAcrossAllSourcesWithEmailOnlyAccess(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual ListAll Union Tenant")
	user := seedVirtualUser(t, pool, tenant, "union")
	account := seedVirtualEmailAccount(t, pool, tenant, user, "Union Mailbox")
	folder := seedVirtualEmailFolder(t, pool, tenant, account)
	msg := seedVirtualEmailMessage(t, pool, tenant, account, folder, 1, "Union subject", time.Now().UTC())
	seedVirtualEmailAttachment(t, pool, tenant, msg, "union.pdf", time.Now().UTC())

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	// The user has ZERO chat/task access, only one email attachment — but the
	// union count is > 0, so the combined SELECT runs and its chat/task
	// branches must not fail even though no row comes from them.
	files, total, err := repo.ListAll(ctx, user, "", 20, 0)
	if err != nil {
		t.Fatalf("ListAll (source=\"\"): unexpected error %v", err)
	}
	if total != 1 || len(files) != 1 {
		t.Fatalf("ListAll (source=\"\"): expected 1 file, got total=%d len=%d", total, len(files))
	}
	if files[0].SourceType != "email" {
		t.Fatalf("ListAll (source=\"\"): source_type=%q, want email", files[0].SourceType)
	}
	if files[0].UploadedByName != "Union Mailbox" {
		t.Fatalf("ListAll (source=\"\"): uploaded_by_name=%q, want %q", files[0].UploadedByName, "Union Mailbox")
	}
}

// TestPostgresRepository_ListChatFiles_UploadedByNameFallsBackToEmail covers
// the case first_name/last_name are both blank (columns are NOT NULL DEFAULT
// '') — uploaded_by_name must fall back to the uploader's email instead of
// rendering a bare space.
func TestPostgresRepository_ListChatFiles_UploadedByNameFallsBackToEmail(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Virtual Chat Email Fallback Tenant")
	email := fmt.Sprintf("nameless-%s@test.invalid", uuid.New().String()[:8])
	member := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         email,
		"password_hash": "x",
		"first_name":    "",
		"last_name":     "",
	})
	channel := seedVirtualChannel(t, pool, tenant, member, "nameless-channel")
	addChannelMembership(t, pool, tenant, channel, member)
	seedVirtualChatFile(t, pool, tenant, channel, member, "report.pdf", time.Now().UTC())

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	files, total, err := repo.ListChatFiles(ctx, member, 20, 0)
	if err != nil {
		t.Fatalf("ListChatFiles: unexpected error %v", err)
	}
	if total != 1 || len(files) != 1 {
		t.Fatalf("ListChatFiles: expected 1 file, got total=%d len=%d", total, len(files))
	}
	if files[0].UploadedByName != email {
		t.Fatalf("ListChatFiles: uploaded_by_name=%q, want fallback to email %q", files[0].UploadedByName, email)
	}
}
