package folder

// c-cov-document-folder-repo: postgres_repository.go had zero coverage at the
// repository-implementation level -- service_test.go only exercises the mock
// Repository, and tenant_isolation_phase2_test.go seeds rows via
// testutil.SeedRow without ever calling a real repository method. This file
// closes that gap following the pattern from internal/document/tag/
// tenant_write_test.go: real repository calls, a foreign-tenant ctx probing
// with the victim's real tenantID (so only RLS, not the WHERE clause, can be
// stopping it), then the same call from the owning ctx.

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

// seedFolderUser seeds a user row. The caller must defer its cleanup
// immediately (before any tenant cleanup already deferred) so LIFO deferred
// execution deletes the user before the tenant it references.
func seedFolderUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("docfolder-%s-%s@tenant.local", label, uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
}

func TestFolderCRUD_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Folder CRUD Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Folder CRUD Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedFolderUser(t, pool, tenantOwn, "crud")
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	f := &models.DocumentFolder{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Name:      "CRUD Test Folder",
		SpaceType: models.FolderSpaceTeam,
		SpaceID:   uuid.New(),
		Icon:      "folder",
		CreatedBy: userOwn,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session -- only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, f); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "document_folders", f.ID, 0)

	if err := repo.Create(ctxOwn, f); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "document_folders", f.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "document_folders", f.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "document_folders", f.ID, 0)

	// GetByID: foreign ctx passing the victim's real tenantID must not see the row.
	if _, err := repo.GetByID(ctxOther, tenantOwn, f.ID); !errors.Is(err, ErrFolderNotFound) {
		t.Fatalf("GetByID (foreign ctx): expected ErrFolderNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, tenantOwn, f.ID)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != "CRUD Test Folder" || got.FileCount != 0 {
		t.Fatalf("GetByID (own ctx): unexpected folder %+v", got)
	}

	// Update: foreign ctx with the real tenantID must be a no-op (RLS scopes
	// the UPDATE's WHERE to rows visible to tenantOther -> RowsAffected == 0).
	newName := "Hacked Name"
	if err := repo.Update(ctxOther, tenantOwn, f.ID, UpdateInput{Name: &newName}); !errors.Is(err, ErrFolderNotFound) {
		t.Fatalf("Update (foreign ctx): expected ErrFolderNotFound, got %v", err)
	}
	got, err = repo.GetByID(sysCtx, tenantOwn, f.ID)
	if err != nil {
		t.Fatalf("GetByID (sys ctx, after foreign update attempt): %v", err)
	}
	if got.Name != "CRUD Test Folder" {
		t.Fatalf("a foreign-tenant update reached the folder: name=%q", got.Name)
	}

	renamed := "Renamed Folder"
	newIcon := "star"
	if err := repo.Update(ctxOwn, tenantOwn, f.ID, UpdateInput{Name: &renamed, Icon: &newIcon}); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, tenantOwn, f.ID)
	if err != nil {
		t.Fatalf("GetByID (own ctx, after update): %v", err)
	}
	if got.Name != renamed || got.Icon != newIcon {
		t.Fatalf("own-tenant update did not land: %+v", got)
	}

	// Update with no fields set is a documented no-op, not an error.
	if err := repo.Update(ctxOwn, tenantOwn, f.ID, UpdateInput{}); err != nil {
		t.Fatalf("Update (no fields): expected nil, got %v", err)
	}

	// Delete: foreign ctx with the real tenantID must not remove the row.
	if err := repo.Delete(ctxOther, tenantOwn, f.ID); !errors.Is(err, ErrFolderNotFound) {
		t.Fatalf("Delete (foreign ctx): expected ErrFolderNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "document_folders", f.ID, 1)

	if err := repo.Delete(ctxOwn, tenantOwn, f.ID); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "document_folders", f.ID, 0)
	if _, err := repo.GetByID(sysCtx, tenantOwn, f.ID); !errors.Is(err, ErrFolderNotFound) {
		t.Fatalf("GetByID after delete: expected ErrFolderNotFound, got %v", err)
	}
}

func TestFolderList_TenantScopedPagination(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Folder List Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Folder List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedFolderUser(t, pool, tenantOwn, "list")
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	spaceID := uuid.New()
	root := &models.DocumentFolder{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Root", SpaceType: models.FolderSpaceTeam,
		SpaceID: spaceID, Icon: "folder", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, root); err != nil {
		t.Fatalf("Create root: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "document_folders", root.ID)

	childA := &models.DocumentFolder{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Child A", SpaceType: models.FolderSpaceTeam,
		SpaceID: spaceID, ParentID: &root.ID, Icon: "folder", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, childA); err != nil {
		t.Fatalf("Create childA: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "document_folders", childA.ID)

	childB := &models.DocumentFolder{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Child B", SpaceType: models.FolderSpaceTeam,
		SpaceID: spaceID, ParentID: &root.ID, Icon: "folder", CreatedBy: userOwn, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctxOwn, childB); err != nil {
		t.Fatalf("Create childB: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "document_folders", childB.ID)

	// ParentID filter: the two children, not the root.
	folders, total, err := repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, ParentID: &root.ID})
	if err != nil {
		t.Fatalf("List (children): %v", err)
	}
	if total != 2 || len(folders) != 2 {
		t.Fatalf("List (children): expected total=2 len=2, got total=%d len=%d", total, len(folders))
	}

	// No ParentID but a SpaceType filter: root folders only.
	spaceType := models.FolderSpaceTeam
	folders, total, err = repo.List(ctxOwn, ListFilter{TenantID: tenantOwn, SpaceType: &spaceType})
	if err != nil {
		t.Fatalf("List (roots): %v", err)
	}
	if total != 1 || len(folders) != 1 || folders[0].ID != root.ID {
		t.Fatalf("List (roots): expected exactly root folder, got total=%d folders=%+v", total, folders)
	}

	// Foreign ctx passing the victim's real tenantID and ParentID must see
	// nothing -- RLS, not the explicit WHERE predicate, is what must stop it.
	folders, total, err = repo.List(ctxOther, ListFilter{TenantID: tenantOwn, ParentID: &root.ID})
	if err != nil {
		t.Fatalf("List (foreign ctx): %v", err)
	}
	if total != 0 || len(folders) != 0 {
		t.Fatalf("List (foreign ctx): expected total=0 len=0, got total=%d len=%d", total, len(folders))
	}
}

func TestGetPath_FourLevelHierarchy(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Folder Path Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedFolderUser(t, pool, tenantOwn, "path")
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	rootID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Level0-Root", "space_type": "team",
		"space_id": uuid.New(), "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", rootID)

	level1ID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Level1", "space_type": "team",
		"space_id": uuid.New(), "parent_id": rootID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", level1ID)

	level2ID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Level2", "space_type": "team",
		"space_id": uuid.New(), "parent_id": level1ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", level2ID)

	leafID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Level3-Leaf", "space_type": "team",
		"space_id": uuid.New(), "parent_id": level2ID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", leafID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	segments, err := repo.GetPath(ctxOwn, leafID)
	if err != nil {
		t.Fatalf("GetPath: %v", err)
	}
	if len(segments) != 4 {
		t.Fatalf("GetPath: expected 4 segments, got %d (%+v)", len(segments), segments)
	}
	wantOrder := []string{"Level0-Root", "Level1", "Level2", "Level3-Leaf"}
	wantIDs := []uuid.UUID{rootID, level1ID, level2ID, leafID}
	for i, seg := range segments {
		if seg.Name != wantOrder[i] || seg.ID != wantIDs[i] {
			t.Fatalf("GetPath segment %d: expected (%s, %s), got (%s, %s)", i, wantIDs[i], wantOrder[i], seg.ID, seg.Name)
		}
	}

	if _, err := repo.GetPath(ctxOwn, uuid.New()); !errors.Is(err, ErrFolderNotFound) {
		t.Fatalf("GetPath (unknown id): expected ErrFolderNotFound, got %v", err)
	}
}

func TestIsDescendant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Folder Descendant Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedFolderUser(t, pool, tenantOwn, "descendant")
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	rootID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Root", "space_type": "team",
		"space_id": uuid.New(), "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", rootID)

	childID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Child", "space_type": "team",
		"space_id": uuid.New(), "parent_id": rootID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", childID)

	grandchildID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Grandchild", "space_type": "team",
		"space_id": uuid.New(), "parent_id": childID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", grandchildID)

	unrelatedID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Unrelated", "space_type": "team",
		"space_id": uuid.New(), "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", unrelatedID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	cases := []struct {
		name     string
		folderID uuid.UUID
		ancestor uuid.UUID
		want     bool
	}{
		{"direct child", childID, rootID, true},
		{"transitive grandchild", grandchildID, rootID, true},
		{"reverse is not a descendant", rootID, grandchildID, false},
		{"unrelated folder", unrelatedID, rootID, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.IsDescendant(ctxOwn, tc.folderID, tc.ancestor)
			if err != nil {
				t.Fatalf("IsDescendant: %v", err)
			}
			if got != tc.want {
				t.Fatalf("IsDescendant(%s, %s): expected %v, got %v", tc.folderID, tc.ancestor, tc.want, got)
			}
		})
	}
}

func TestGetChildren_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Folder Children Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Folder Children Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedFolderUser(t, pool, tenantOwn, "children")
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	rootID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Root", "space_type": "team",
		"space_id": uuid.New(), "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", rootID)

	childID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Child", "space_type": "team",
		"space_id": uuid.New(), "parent_id": rootID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", childID)

	grandchildID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "Grandchild", "space_type": "team",
		"space_id": uuid.New(), "parent_id": childID, "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", grandchildID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	children, err := repo.GetChildren(ctxOwn, tenantOwn, rootID)
	if err != nil {
		t.Fatalf("GetChildren (own ctx): %v", err)
	}
	if len(children) != 1 || children[0].ID != childID {
		t.Fatalf("GetChildren (own ctx): expected exactly [Child], got %+v", children)
	}

	// Foreign ctx with the victim's real tenantID and parentID must see nothing.
	children, err = repo.GetChildren(ctxOther, tenantOwn, rootID)
	if err != nil {
		t.Fatalf("GetChildren (foreign ctx): %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("GetChildren (foreign ctx): expected 0 children, got %d", len(children))
	}
}

func TestCountFiles(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Folder CountFiles Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedFolderUser(t, pool, tenantOwn, "countfiles")
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id": tenantOwn, "name": "CountFiles Folder", "space_type": "team",
		"space_id": uuid.New(), "created_by": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", folderID)

	activeFileID := testutil.SeedRow(t, pool, "document_files", map[string]any{
		"tenant_id": tenantOwn, "folder_id": folderID, "filename": "active.pdf",
		"mime_type": "application/pdf", "file_size": 512,
		"storage_key": "documents/" + uuid.New().String() + ".pdf", "owner_id": userOwn,
	})
	defer testutil.CleanupRow(t, pool, "document_files", activeFileID)

	deletedFileID := testutil.SeedRow(t, pool, "document_files", map[string]any{
		"tenant_id": tenantOwn, "folder_id": folderID, "filename": "deleted.pdf",
		"mime_type": "application/pdf", "file_size": 512,
		"storage_key": "documents/" + uuid.New().String() + ".pdf", "owner_id": userOwn,
		"is_deleted": true,
	})
	defer testutil.CleanupRow(t, pool, "document_files", deletedFileID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	count, err := repo.CountFiles(ctxOwn, folderID)
	if err != nil {
		t.Fatalf("CountFiles: %v", err)
	}
	if count != 1 {
		t.Fatalf("CountFiles: expected 1 (deleted file excluded), got %d", count)
	}
}
