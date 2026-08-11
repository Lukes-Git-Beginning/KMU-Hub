package search

// DB-level tests for PostgresRepository.Search. document_files.search_vector
// is populated by a BEFORE INSERT trigger (migration 000043,
// trg_document_files_search_vector), so seeding content_text directly via
// testutil.SeedRow is enough to make plainto_tsquery matches work — no
// manual tsvector computation needed.

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedSearchTenantUserFolder(t *testing.T, pool *pgxpool.Pool, label string) (tenant, user, folder uuid.UUID) {
	t.Helper()
	tenant = uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Search Test Tenant "+label)

	user = testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         "search-" + label + "-" + uuid.New().String()[:8] + "@test.invalid",
		"password_hash": "x",
	})

	folder = testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id":  tenant,
		"name":       "Search Test Folder " + label,
		"space_type": "personal",
		"space_id":   uuid.New(),
		"created_by": user,
	})
	return tenant, user, folder
}

func seedSearchFile(t *testing.T, pool *pgxpool.Pool, tenant, folder, owner uuid.UUID, filename, contentText string, isDeleted bool) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "document_files", map[string]any{
		"tenant_id":    tenant,
		"folder_id":    folder,
		"filename":     filename,
		"mime_type":    "application/pdf",
		"file_size":    int64(1024),
		"storage_key":  "documents/search-test/" + uuid.New().String(),
		"owner_id":     owner,
		"content_text": contentText,
		"is_deleted":   isDeleted,
	})
}

func addSearchFileTag(t *testing.T, pool *pgxpool.Pool, tenant, fileID, tagID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx,
		`INSERT INTO document_file_tags (file_id, tag_id, tenant_id) VALUES ($1, $2, $3)`,
		fileID, tagID, tenant,
	); err != nil {
		t.Fatalf("seed document_file_tags: %v", err)
	}
}

func seedSearchTag(t *testing.T, pool *pgxpool.Pool, tenant, createdBy uuid.UUID, name string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "document_tags", map[string]any{
		"tenant_id":  tenant,
		"name":       name + "-" + uuid.New().String()[:8],
		"created_by": createdBy,
	})
}

func TestPostgresRepository_Search_MatchesContentAndRanks(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, user, folder := seedSearchTenantUserFolder(t, pool, "match")
	matching := seedSearchFile(t, pool, tenant, folder, user, "rechnung-2026.pdf", "Diese Rechnung enthaelt den Gesamtbetrag fuer den Kunden.", false)
	seedSearchFile(t, pool, tenant, folder, user, "vertrag.pdf", "Dieser Vertrag regelt die Zusammenarbeit der Parteien.", false)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	results, total, err := repo.Search(ctx, "Rechnung", SearchFilter{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("Search: expected 1 result, got total=%d len=%d", total, len(results))
	}
	if results[0].File.ID != matching {
		t.Fatalf("Search: expected file %s, got %s", matching, results[0].File.ID)
	}
	if results[0].Rank <= 0 {
		t.Fatalf("Search: expected a positive rank, got %f", results[0].Rank)
	}
	if !strings.Contains(results[0].Snippet, "<<") || !strings.Contains(results[0].Snippet, ">>") {
		t.Fatalf("Search: expected snippet to contain highlight markers, got %q", results[0].Snippet)
	}
}

func TestPostgresRepository_Search_NoMatchReturnsEmptyNotNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, user, folder := seedSearchTenantUserFolder(t, pool, "nomatch")
	seedSearchFile(t, pool, tenant, folder, user, "vertrag.pdf", "Dieser Vertrag regelt die Zusammenarbeit der Parteien.", false)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	results, total, err := repo.Search(ctx, "Voellig unbekannter Begriff Xyzzy", SearchFilter{})
	if err != nil {
		t.Fatalf("Search (no match): %v", err)
	}
	if total != 0 {
		t.Fatalf("Search (no match): expected total=0, got %d", total)
	}
	if results == nil {
		t.Fatal("Search (no match): expected an empty slice, got nil")
	}
	if len(results) != 0 {
		t.Fatalf("Search (no match): expected 0 results, got %d", len(results))
	}
}

func TestPostgresRepository_Search_ExcludesSoftDeletedFiles(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, user, folder := seedSearchTenantUserFolder(t, pool, "deleted")
	active := seedSearchFile(t, pool, tenant, folder, user, "active.pdf", "Formular fuer den Zeiterfassungsantrag.", false)
	seedSearchFile(t, pool, tenant, folder, user, "deleted.pdf", "Formular fuer den Zeiterfassungsantrag.", true)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	results, total, err := repo.Search(ctx, "Formular", SearchFilter{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("Search: expected only the non-deleted file, got total=%d len=%d", total, len(results))
	}
	if results[0].File.ID != active {
		t.Fatalf("Search: expected active file %s, got %s", active, results[0].File.ID)
	}
}

func TestPostgresRepository_Search_FilterByFolderID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, user, folderA := seedSearchTenantUserFolder(t, pool, "folder-a")
	folderB := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id":  tenant,
		"name":       "Second Folder",
		"space_type": "personal",
		"space_id":   uuid.New(),
		"created_by": user,
	})

	fileA := seedSearchFile(t, pool, tenant, folderA, user, "a.pdf", "Protokoll der Teambesprechung.", false)
	seedSearchFile(t, pool, tenant, folderB, user, "b.pdf", "Protokoll der Teambesprechung.", false)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	results, total, err := repo.Search(ctx, "Protokoll", SearchFilter{FolderID: &folderA})
	if err != nil {
		t.Fatalf("Search (folder filter): %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("Search (folder filter): expected 1 result, got total=%d len=%d", total, len(results))
	}
	if results[0].File.ID != fileA {
		t.Fatalf("Search (folder filter): expected file %s, got %s", fileA, results[0].File.ID)
	}
}

func TestPostgresRepository_Search_FilterByOwnerID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, ownerA, folder := seedSearchTenantUserFolder(t, pool, "owner-a")
	ownerB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         "search-owner-b-" + uuid.New().String()[:8] + "@test.invalid",
		"password_hash": "x",
	})

	fileA := seedSearchFile(t, pool, tenant, folder, ownerA, "a.pdf", "Angebot fuer den Kunden erstellen.", false)
	seedSearchFile(t, pool, tenant, folder, ownerB, "b.pdf", "Angebot fuer den Kunden erstellen.", false)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	results, total, err := repo.Search(ctx, "Angebot", SearchFilter{OwnerID: &ownerA})
	if err != nil {
		t.Fatalf("Search (owner filter): %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("Search (owner filter): expected 1 result, got total=%d len=%d", total, len(results))
	}
	if results[0].File.ID != fileA {
		t.Fatalf("Search (owner filter): expected file %s, got %s", fileA, results[0].File.ID)
	}
}

func TestPostgresRepository_Search_FilterByTagIDs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, user, folder := seedSearchTenantUserFolder(t, pool, "tags")
	tagged := seedSearchFile(t, pool, tenant, folder, user, "tagged.pdf", "Lieferschein fuer die Bestellung.", false)
	seedSearchFile(t, pool, tenant, folder, user, "untagged.pdf", "Lieferschein fuer die Bestellung.", false)

	tag := seedSearchTag(t, pool, tenant, user, "invoice")
	addSearchFileTag(t, pool, tenant, tagged, tag)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	results, total, err := repo.Search(ctx, "Lieferschein", SearchFilter{TagIDs: []uuid.UUID{tag}})
	if err != nil {
		t.Fatalf("Search (tag filter): %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("Search (tag filter): expected 1 result, got total=%d len=%d", total, len(results))
	}
	if results[0].File.ID != tagged {
		t.Fatalf("Search (tag filter): expected file %s, got %s", tagged, results[0].File.ID)
	}
}
