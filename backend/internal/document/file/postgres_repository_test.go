package file

// DB-level tests for document_file_activity (Migration 000264): the JOIN-based
// actor_name resolution, tenant scoping, and the append-only trigger that
// blocks UPDATE/DELETE at the database level (same pattern as audit_log,
// mig 000222). Mock-repository tests in service_test.go already cover which
// service methods record which action — this file proves the SQL layer.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// activityFixture mints a fresh tenant + user + folder + file per test so
// seeds never collide under t.Parallel(). None of the four rows are
// registered for testutil.CleanupRow: once the file carries
// document_file_activity children, the FK's ON DELETE CASCADE would issue a
// DELETE against them too, which the append-only trigger rejects — cascading
// up, the folder and user become undeletable as well. CleanupRow only logs on
// failure, so registering it here would just add noise on every run. Same
// tradeoff the audit_log tests already accept (rows accumulate in the local/
// CI-ephemeral dev DB, never in production).
type activityFixture struct {
	tenant uuid.UUID
	user   uuid.UUID
	file   uuid.UUID
}

func seedActivityFixture(t *testing.T, pool *pgxpool.Pool, name string) activityFixture {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "document activity test "+name)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "docact-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Doc", "last_name": "Tester",
	})

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Activity Test Folder",
		"space_type": "personal", "space_id": uuid.New(), "created_by": userID,
	})

	fileID := testutil.SeedRow(t, pool, "document_files", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "folder_id": folderID,
		"filename": "test.pdf", "mime_type": "application/pdf", "file_size": 1024,
		"storage_key": "documents/activity-test/" + uuid.NewString(), "owner_id": userID,
	})

	return activityFixture{tenant: tenantID, user: userID, file: fileID}
}

func TestPostgresRepository_CreateAndListActivity(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "create-list")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	older := &models.DocumentFileActivity{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		Action: models.DocumentActivityUploaded, ActorID: fx.user,
		CreatedAt: time.Now().Add(-time.Hour),
	}
	newer := &models.DocumentFileActivity{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		Action: models.DocumentActivityRenamed, ActorID: fx.user, Detail: "renamed.pdf",
		CreatedAt: time.Now(),
	}
	if err := repo.CreateActivity(ctx, older); err != nil {
		t.Fatalf("CreateActivity (older): %v", err)
	}
	if err := repo.CreateActivity(ctx, newer); err != nil {
		t.Fatalf("CreateActivity (newer): %v", err)
	}

	activities, err := repo.ListActivity(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("ListActivity returned %d entries, want 2", len(activities))
	}
	if activities[0].ID != newer.ID || activities[1].ID != older.ID {
		t.Fatalf("ListActivity order = [%s, %s], want newest-first [%s, %s]",
			activities[0].Action, activities[1].Action, models.DocumentActivityRenamed, models.DocumentActivityUploaded)
	}
	if activities[0].Detail != "renamed.pdf" {
		t.Errorf("detail = %q, want %q", activities[0].Detail, "renamed.pdf")
	}
	if activities[0].ActorName != "Doc Tester" {
		t.Errorf("actor_name = %q, want %q (JOIN over users.first_name/last_name)", activities[0].ActorName, "Doc Tester")
	}
}

func TestPostgresRepository_ListActivity_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "tenant-isolation")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	if err := repo.CreateActivity(ctx, &models.DocumentFileActivity{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		Action: models.DocumentActivityUploaded, ActorID: fx.user, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateActivity: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document activity test other tenant")

	activities, err := repo.ListActivity(ctx, fx.file, otherTenant)
	if err != nil {
		t.Fatalf("ListActivity (foreign tenant): %v", err)
	}
	if len(activities) != 0 {
		t.Fatalf("ListActivity under a foreign tenant_id returned %d entries, want 0", len(activities))
	}
}

// TestDocumentFileActivity_AppendOnly proves the DB-level trigger (mig 000264)
// blocks UPDATE and DELETE, matching the audit_log precedent (mig 000222) —
// this is the "append-only durchgesetzt" requirement, not just a Go-level
// convention that a future direct-SQL script could bypass.
func TestDocumentFileActivity_AppendOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedActivityFixture(t, pool, "append-only")
	ctx := testutil.WithSystemCtx(context.Background())

	activityID := testutil.SeedRow(t, pool, "document_file_activity", map[string]any{
		"id": uuid.New(), "tenant_id": fx.tenant, "file_id": fx.file,
		"action": models.DocumentActivityUploaded, "actor_id": fx.user,
	})

	_, updateErr := pool.Exec(ctx, "UPDATE document_file_activity SET detail = $1 WHERE id = $2", "tampered", activityID)
	if updateErr == nil {
		t.Fatal("UPDATE on document_file_activity succeeded, want append-only trigger to reject it")
	}

	_, deleteErr := pool.Exec(ctx, "DELETE FROM document_file_activity WHERE id = $1", activityID)
	if deleteErr == nil {
		t.Fatal("DELETE on document_file_activity succeeded, want append-only trigger to reject it")
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM document_file_activity WHERE id = $1", activityID).Scan(&count); err != nil {
		t.Fatalf("verify row survived: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count after rejected UPDATE/DELETE = %d, want 1 (row must be untouched)", count)
	}
}
