package file

// DB-level tests for the core document_files CRUD and document_file_versions
// paths (Create, GetByID, List, Update, SoftDelete, versioning). Every other
// test in this package that exercises these methods goes through
// MockRepository — this file is the only one that proves the real SQL: filter
// composition in List, the tag-membership subquery, the sort switch, and that
// Update/SoftDelete correctly report ErrFileNotFound via RowsAffected == 0.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type fileFixture struct {
	tenant uuid.UUID
	user   uuid.UUID
	folder uuid.UUID
}

func seedFileFixture(t *testing.T, pool *pgxpool.Pool, name string) fileFixture {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "document file test "+name)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "docfile-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Doc", "last_name": "Filer",
	})

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "File Test Folder",
		"space_type": "personal", "space_id": uuid.New(), "created_by": userID,
	})

	return fileFixture{tenant: tenantID, user: userID, folder: folderID}
}

func newTestFile(fx fileFixture, filename string) *models.DocumentFile {
	return &models.DocumentFile{
		ID: uuid.New(), TenantID: fx.tenant, FolderID: fx.folder,
		Filename: filename, MimeType: "application/pdf", FileSize: 1024,
		StorageKey: "documents/db-test/" + uuid.NewString(), OwnerID: fx.user,
		CurrentVersion: 1,
	}
}

func TestPostgresRepository_CreateAndGetByID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "create-get")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "invoice.pdf")
	require.NoError(t, repo.Create(ctx, file))

	got, err := repo.GetByID(ctx, file.ID, fx.tenant)
	require.NoError(t, err)
	assert.Equal(t, file.Filename, got.Filename)
	assert.Equal(t, file.FolderID, got.FolderID)
	assert.Equal(t, file.OwnerID, got.OwnerID)
	assert.False(t, got.IsDeleted)
	assert.Nil(t, got.DeletedAt)
}

func TestPostgresRepository_GetByID_WrongTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "get-wrong-tenant")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "secret.pdf")
	require.NoError(t, repo.Create(ctx, file))

	otherTenant := uuid.New()
	got, err := repo.GetByID(ctx, file.ID, otherTenant)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestPostgresRepository_List_DefaultExcludesDeleted(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "list-exclude-deleted")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	active := newTestFile(fx, "active.pdf")
	require.NoError(t, repo.Create(ctx, active))
	deleted := newTestFile(fx, "deleted.pdf")
	require.NoError(t, repo.Create(ctx, deleted))
	require.NoError(t, repo.SoftDelete(ctx, deleted.ID, fx.tenant))

	files, total, err := repo.List(ctx, ListFilter{TenantID: fx.tenant})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, files, 1)
	assert.Equal(t, active.ID, files[0].ID)
}

func TestPostgresRepository_List_IsDeletedFilter_ShowsOnlyDeleted(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "list-only-deleted")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	active := newTestFile(fx, "active.pdf")
	require.NoError(t, repo.Create(ctx, active))
	deleted := newTestFile(fx, "deleted.pdf")
	require.NoError(t, repo.Create(ctx, deleted))
	require.NoError(t, repo.SoftDelete(ctx, deleted.ID, fx.tenant))

	isDeleted := true
	files, total, err := repo.List(ctx, ListFilter{TenantID: fx.tenant, IsDeleted: &isDeleted})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, files, 1)
	assert.Equal(t, deleted.ID, files[0].ID)
}

func TestPostgresRepository_List_FolderAndOwnerAndFavoriteFilters(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "list-filters")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	otherFolder := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"id": uuid.New(), "tenant_id": fx.tenant, "name": "Other Folder",
		"space_type": "personal", "space_id": uuid.New(), "created_by": fx.user,
	})

	inFolder := newTestFile(fx, "in-folder.pdf")
	require.NoError(t, repo.Create(ctx, inFolder))
	favorite := *inFolder
	favorite.ID = uuid.New()
	favorite.Filename = "favorite.pdf"
	favorite.IsFavorite = true
	require.NoError(t, repo.Create(ctx, &favorite))

	elsewhere := newTestFile(fx, "elsewhere.pdf")
	elsewhere.FolderID = otherFolder
	require.NoError(t, repo.Create(ctx, elsewhere))

	byFolder, total, err := repo.List(ctx, ListFilter{TenantID: fx.tenant, FolderID: &fx.folder})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, byFolder, 2)

	isFav := true
	byFavorite, total, err := repo.List(ctx, ListFilter{TenantID: fx.tenant, IsFavorite: &isFav})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, byFavorite, 1)
	assert.Equal(t, favorite.ID, byFavorite[0].ID)

	byOwner, total, err := repo.List(ctx, ListFilter{TenantID: fx.tenant, OwnerID: &fx.user})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, byOwner, 3)
}

func TestPostgresRepository_List_SortByNameAscending(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "list-sort")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	zeta := newTestFile(fx, "zeta.pdf")
	require.NoError(t, repo.Create(ctx, zeta))
	alpha := newTestFile(fx, "alpha.pdf")
	require.NoError(t, repo.Create(ctx, alpha))

	files, _, err := repo.List(ctx, ListFilter{TenantID: fx.tenant, SortField: "name", SortDir: "asc"})
	require.NoError(t, err)
	require.Len(t, files, 2)
	assert.Equal(t, "alpha.pdf", files[0].Filename)
	assert.Equal(t, "zeta.pdf", files[1].Filename)
}

func TestPostgresRepository_List_TagFilter_RequiresAllTags(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "list-tags")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	tagged := newTestFile(fx, "tagged.pdf")
	require.NoError(t, repo.Create(ctx, tagged))
	untagged := newTestFile(fx, "untagged.pdf")
	require.NoError(t, repo.Create(ctx, untagged))

	tagA := testutil.SeedRow(t, pool, "document_tags", map[string]any{
		"id": uuid.New(), "tenant_id": fx.tenant, "name": "tag-a-" + uuid.NewString(), "color": "#fff",
		"created_by": fx.user,
	})
	tagB := testutil.SeedRow(t, pool, "document_tags", map[string]any{
		"id": uuid.New(), "tenant_id": fx.tenant, "name": "tag-b-" + uuid.NewString(), "color": "#000",
		"created_by": fx.user,
	})
	// document_file_tags has a composite PK (file_id, tag_id), no id column —
	// testutil.SeedRow always does RETURNING id, so a raw insert is required.
	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx, `INSERT INTO document_file_tags (file_id, tag_id, tenant_id) VALUES ($1, $2, $3)`,
		tagged.ID, tagA, fx.tenant)
	require.NoError(t, err)
	_, err = pool.Exec(sysCtx, `INSERT INTO document_file_tags (file_id, tag_id, tenant_id) VALUES ($1, $2, $3)`,
		tagged.ID, tagB, fx.tenant)
	require.NoError(t, err)

	// Only the file carrying BOTH tags must match — the HAVING COUNT(DISTINCT
	// tag_id) = $n clause is what proves "all", not "any".
	files, total, err := repo.List(ctx, ListFilter{TenantID: fx.tenant, TagIDs: []uuid.UUID{tagA, tagB}})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, files, 1)
	assert.Equal(t, tagged.ID, files[0].ID)
}

func TestPostgresRepository_Update_RenameAndMove(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "update")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "old-name.pdf")
	require.NoError(t, repo.Create(ctx, file))

	newFolder := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"id": uuid.New(), "tenant_id": fx.tenant, "name": "Target Folder",
		"space_type": "personal", "space_id": uuid.New(), "created_by": fx.user,
	})
	newName := "new-name.pdf"
	require.NoError(t, repo.Update(ctx, file.ID, fx.tenant, UpdateInput{Filename: &newName, FolderID: &newFolder}))

	got, err := repo.GetByID(ctx, file.ID, fx.tenant)
	require.NoError(t, err)
	assert.Equal(t, newName, got.Filename)
	assert.Equal(t, newFolder, got.FolderID)
}

func TestPostgresRepository_Update_WrongTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "update-wrong-tenant")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "name.pdf")
	require.NoError(t, repo.Create(ctx, file))

	newName := "hijacked.pdf"
	err := repo.Update(ctx, file.ID, uuid.New(), UpdateInput{Filename: &newName})
	assert.ErrorIs(t, err, ErrFileNotFound)

	got, getErr := repo.GetByID(ctx, file.ID, fx.tenant)
	require.NoError(t, getErr)
	assert.Equal(t, "name.pdf", got.Filename, "cross-tenant update must not have applied")
}

func TestPostgresRepository_Update_NoFieldsIsNoop(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "update-noop")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "unchanged.pdf")
	require.NoError(t, repo.Create(ctx, file))

	assert.NoError(t, repo.Update(ctx, file.ID, fx.tenant, UpdateInput{}))
}

func TestPostgresRepository_SoftDelete_SetsFlagAndTimestamp(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "soft-delete")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "to-delete.pdf")
	require.NoError(t, repo.Create(ctx, file))
	require.NoError(t, repo.SoftDelete(ctx, file.ID, fx.tenant))

	got, err := repo.GetByID(ctx, file.ID, fx.tenant)
	require.NoError(t, err)
	assert.True(t, got.IsDeleted)
	require.NotNil(t, got.DeletedAt)
}

func TestPostgresRepository_SoftDelete_AlreadyDeleted_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "soft-delete-twice")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "delete-twice.pdf")
	require.NoError(t, repo.Create(ctx, file))
	require.NoError(t, repo.SoftDelete(ctx, file.ID, fx.tenant))

	// The `AND NOT is_deleted` guard means a second SoftDelete affects zero
	// rows and must surface as ErrFileNotFound, same as a bad id.
	err := repo.SoftDelete(ctx, file.ID, fx.tenant)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

// --- Versioning ---

func TestPostgresRepository_CreateAndListVersions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "versions-list")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "versioned.pdf")
	require.NoError(t, repo.Create(ctx, file))

	v1 := &models.DocumentFileVersion{
		ID: uuid.New(), TenantID: fx.tenant, FileID: file.ID, VersionNumber: 1,
		StorageKey: file.StorageKey, FileSize: 1024, CreatedBy: fx.user,
	}
	require.NoError(t, repo.CreateVersion(ctx, v1))
	v2 := &models.DocumentFileVersion{
		ID: uuid.New(), TenantID: fx.tenant, FileID: file.ID, VersionNumber: 2,
		StorageKey: "documents/db-test/" + uuid.NewString(), FileSize: 2048, CreatedBy: fx.user,
	}
	require.NoError(t, repo.CreateVersion(ctx, v2))

	versions, err := repo.ListVersions(ctx, file.ID)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, 2, versions[0].VersionNumber, "must be newest-first")
	assert.Equal(t, 1, versions[1].VersionNumber)
}

func TestPostgresRepository_GetVersion_ByNumber(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "version-by-number")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "versioned.pdf")
	require.NoError(t, repo.Create(ctx, file))
	v1 := &models.DocumentFileVersion{
		ID: uuid.New(), TenantID: fx.tenant, FileID: file.ID, VersionNumber: 1,
		StorageKey: file.StorageKey, FileSize: 1024, CreatedBy: fx.user,
	}
	require.NoError(t, repo.CreateVersion(ctx, v1))

	got, err := repo.GetVersion(ctx, file.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, v1.ID, got.ID)

	_, err = repo.GetVersion(ctx, file.ID, 99)
	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func TestPostgresRepository_GetVersionByID_ScopedToFileAndTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "version-by-id")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "versioned.pdf")
	require.NoError(t, repo.Create(ctx, file))
	otherFile := newTestFile(fx, "other.pdf")
	require.NoError(t, repo.Create(ctx, otherFile))

	v1 := &models.DocumentFileVersion{
		ID: uuid.New(), TenantID: fx.tenant, FileID: file.ID, VersionNumber: 1,
		StorageKey: file.StorageKey, FileSize: 1024, CreatedBy: fx.user,
	}
	require.NoError(t, repo.CreateVersion(ctx, v1))

	got, err := repo.GetVersionByID(ctx, file.ID, v1.ID, fx.tenant)
	require.NoError(t, err)
	assert.Equal(t, v1.ID, got.ID)

	// A version of file A must not resolve through file B's id — a caller
	// naming its own file id plus a foreign version id must not reach it.
	_, err = repo.GetVersionByID(ctx, otherFile.ID, v1.ID, fx.tenant)
	assert.ErrorIs(t, err, ErrVersionNotFound)

	_, err = repo.GetVersionByID(ctx, file.ID, v1.ID, uuid.New())
	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func TestPostgresRepository_UpdateCurrentVersion(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "update-current-version")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "versioned.pdf")
	require.NoError(t, repo.Create(ctx, file))

	require.NoError(t, repo.UpdateCurrentVersion(ctx, file.ID, 3))

	got, err := repo.GetByID(ctx, file.ID, fx.tenant)
	require.NoError(t, err)
	assert.Equal(t, 3, got.CurrentVersion)
}

func TestPostgresRepository_UpdateCurrentVersion_UnknownFile_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)

	err := repo.UpdateCurrentVersion(context.Background(), uuid.New(), 2)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestPostgresRepository_UpdateSearchContent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedFileFixture(t, pool, "search-content")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	file := newTestFile(fx, "searchable.pdf")
	require.NoError(t, repo.Create(ctx, file))

	require.NoError(t, repo.UpdateSearchContent(ctx, file.ID, "extracted pdf body text"))

	got, err := repo.GetByID(ctx, file.ID, fx.tenant)
	require.NoError(t, err)
	require.NotNil(t, got.ContentText)
	assert.Equal(t, "extracted pdf body text", *got.ContentText)
}
