package file

// DB-level tests for document_entity_links tenant scoping. CreateEntityLink
// left tenant_id unset from migration 000114 (tenant_id added, NOT NULL) up
// to this change -- every real INSERT would have violated the NOT NULL
// constraint. Mock-repository tests in service_test.go cover the Service
// contract against a fake; this file proves the SQL layer actually persists
// and filters by tenant_id.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_CreateAndListEntityLinks(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "entity-link-create-list")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	link := &models.DocumentEntityLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		EntityType: "contact", EntityID: uuid.New(), LinkedBy: fx.user,
	}
	if err := repo.CreateEntityLink(ctx, link); err != nil {
		t.Fatalf("CreateEntityLink: %v", err)
	}

	links, err := repo.ListEntityLinks(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListEntityLinks: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("ListEntityLinks returned %d entries, want 1", len(links))
	}
	if links[0].ID != link.ID || links[0].TenantID != fx.tenant || links[0].EntityType != "contact" {
		t.Errorf("ListEntityLinks[0] = %+v, want ID=%s TenantID=%s EntityType=contact", links[0], link.ID, fx.tenant)
	}
}

func TestPostgresRepository_ListEntityLinks_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "entity-link-tenant-isolation")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	if err := repo.CreateEntityLink(ctx, &models.DocumentEntityLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		EntityType: "contact", EntityID: uuid.New(), LinkedBy: fx.user,
	}); err != nil {
		t.Fatalf("CreateEntityLink: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document entity link test other tenant")

	links, err := repo.ListEntityLinks(ctx, fx.file, otherTenant)
	if err != nil {
		t.Fatalf("ListEntityLinks (foreign tenant): %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("ListEntityLinks under a foreign tenant_id returned %d entries, want 0", len(links))
	}
}

func TestPostgresRepository_ListFilesByEntity(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "list-files-by-entity")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)
	contactID := uuid.New()

	if err := repo.CreateEntityLink(ctx, &models.DocumentEntityLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		EntityType: "contact", EntityID: contactID, LinkedBy: fx.user,
	}); err != nil {
		t.Fatalf("CreateEntityLink: %v", err)
	}

	files, err := repo.ListFilesByEntity(ctx, "contact", contactID, fx.tenant)
	if err != nil {
		t.Fatalf("ListFilesByEntity: %v", err)
	}
	if len(files) != 1 || files[0].ID != fx.file {
		t.Fatalf("ListFilesByEntity = %+v, want exactly file %s", files, fx.file)
	}
}

func TestPostgresRepository_ListFilesByEntity_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "list-files-by-entity-tenant-isolation")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)
	contactID := uuid.New()

	if err := repo.CreateEntityLink(ctx, &models.DocumentEntityLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		EntityType: "contact", EntityID: contactID, LinkedBy: fx.user,
	}); err != nil {
		t.Fatalf("CreateEntityLink: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document list-files-by-entity test other tenant")

	files, err := repo.ListFilesByEntity(ctx, "contact", contactID, otherTenant)
	if err != nil {
		t.Fatalf("ListFilesByEntity (foreign tenant): %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("ListFilesByEntity under a foreign tenant_id returned %d entries, want 0", len(files))
	}
}

func TestPostgresRepository_DeleteEntityLink(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "entity-link-delete")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	link := &models.DocumentEntityLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		EntityType: "contact", EntityID: uuid.New(), LinkedBy: fx.user,
	}
	if err := repo.CreateEntityLink(ctx, link); err != nil {
		t.Fatalf("CreateEntityLink: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document entity link test delete wrong tenant")
	if err := repo.DeleteEntityLink(ctx, link.ID, otherTenant); err != ErrFileNotFound {
		t.Fatalf("DeleteEntityLink under a foreign tenant_id = %v, want ErrFileNotFound", err)
	}
	if links, _ := repo.ListEntityLinks(ctx, fx.file, fx.tenant); len(links) != 1 {
		t.Fatalf("link survived foreign-tenant delete attempt: got %d rows, want 1", len(links))
	}

	if err := repo.DeleteEntityLink(ctx, link.ID, fx.tenant); err != nil {
		t.Fatalf("DeleteEntityLink: %v", err)
	}
	links, err := repo.ListEntityLinks(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListEntityLinks after delete: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("ListEntityLinks after delete returned %d entries, want 0", len(links))
	}
}
