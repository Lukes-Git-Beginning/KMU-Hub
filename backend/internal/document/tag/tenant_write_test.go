package tag

// wp-document-wiki: the existing tenant_isolation_test.go / tenant_isolation_phase2_test.go
// in this package only exercise the mock repository or SeedRow fixtures — neither calls
// the real PostgresRepository write methods. This file closes that gap for the write
// surface, following the pattern established in internal/wiki/tenant_write_test.go.

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

func TestDocumentTagWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Doctag Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Doctag Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("doctag-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Write-Test-Folder",
		"space_type": "team",
		"space_id":   uuid.New(),
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", folderID)

	fileID := testutil.SeedRow(t, pool, "document_files", map[string]any{
		"tenant_id":   tenantOwn,
		"folder_id":   folderID,
		"filename":    "write-test.pdf",
		"mime_type":   "application/pdf",
		"file_size":   1024,
		"storage_key": "documents/" + uuid.New().String() + ".pdf",
		"owner_id":    userID,
	})
	defer testutil.CleanupRow(t, pool, "document_files", fileID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	tag := &models.DocumentTag{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Name:      "Write-Test-" + uuid.New().String()[:8],
		CreatedBy: userID,
		CreatedAt: now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, tag); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "document_tags", tag.ID, 0)

	if err := repo.Create(ctxOwn, tag); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "document_tags", tag.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "document_tags", tag.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "document_tags", tag.ID, 0)

	// List, scoped both by the ctx's RLS session and the explicit tenantID
	// predicate — a foreign session asking for the victim's own tenantID must
	// still see nothing.
	tags, err := repo.List(ctxOther, tenantOwn)
	if err != nil {
		t.Fatalf("List (foreign ctx): %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("List (foreign ctx): expected 0 tags visible, got %d", len(tags))
	}
	tags, err = repo.List(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("List (own ctx): %v", err)
	}
	found := false
	for _, tg := range tags {
		if tg.ID == tag.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("List (own ctx): expected to find tag %s, got %d tags", tag.ID, len(tags))
	}

	// --- TagFile / UntagFile -------------------------------------------------

	// A foreign session tagging the victim's real file with the victim's real
	// tenantID must be refused by RLS's WITH CHECK on the insert.
	if err := repo.TagFile(ctxOther, tenantOwn, fileID, tag.ID); err == nil {
		t.Fatalf("TagFile (foreign ctx): expected an RLS error, got nil")
	}
	assertFileTagCount(t, pool, sysCtx, fileID, tag.ID, 0)

	if err := repo.TagFile(ctxOwn, tenantOwn, fileID, tag.ID); err != nil {
		t.Fatalf("TagFile (own ctx): %v", err)
	}
	assertFileTagCount(t, pool, sysCtx, fileID, tag.ID, 1)
	assertFileTagCount(t, pool, ctxOther, fileID, tag.ID, 0)

	if err := repo.UntagFile(ctxOther, tenantOwn, fileID, tag.ID); err != nil {
		t.Fatalf("UntagFile (foreign ctx): %v", err)
	}
	// RLS scopes the DELETE to rows visible to tenantOther, so a foreign
	// session's UntagFile is a no-op — the association must survive.
	assertFileTagCount(t, pool, sysCtx, fileID, tag.ID, 1)

	if err := repo.UntagFile(ctxOwn, tenantOwn, fileID, tag.ID); err != nil {
		t.Fatalf("UntagFile (own ctx): %v", err)
	}
	assertFileTagCount(t, pool, sysCtx, fileID, tag.ID, 0)

	// --- Delete ---------------------------------------------------------------

	if err := repo.Delete(ctxOther, tenantOwn, tag.ID); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("Delete (foreign ctx): expected ErrTagNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "document_tags", tag.ID, 1)

	if err := repo.Delete(ctxOwn, tenantOwn, tag.ID); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "document_tags", tag.ID, 0)
}

// assertFileTagCount checks document_file_tags row presence under ctx's RLS
// visibility. The table has a composite (file_id, tag_id) primary key, not a
// single UUID id column, so testutil.AssertRowCount does not apply directly.
func assertFileTagCount(t *testing.T, pool *pgxpool.Pool, ctx context.Context, fileID, tagID uuid.UUID, expected int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM document_file_tags WHERE file_id = $1 AND tag_id = $2`,
		fileID, tagID,
	).Scan(&count); err != nil {
		t.Fatalf("count document_file_tags file_id=%s tag_id=%s: %v", fileID, tagID, err)
	}
	if count != expected {
		t.Fatalf("document_file_tags file_id=%s tag_id=%s: expected %d row(s), got %d", fileID, tagID, expected, count)
	}
}
