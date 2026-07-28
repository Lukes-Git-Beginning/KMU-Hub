package tag

// wp-crm-meta: like the other rls_test.go files in this package, the existing
// isolation test seeds exclusively through testutil.SeedRow and never calls
// the real Create/Update/Delete methods. This file closes that gap for the
// write surface.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTagWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Tag Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Tag Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	tg := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantOwn,
		Name:       "Write-Test-" + uuid.New().String()[:8],
		Color:      "#3d8abf",
		EntityType: models.EntityType("contact"),
		CreatedAt:  time.Now().UTC(),
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, tg); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tags", tg.ID, 0)

	if err := repo.Create(ctxOwn, tg); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tags", tg.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "tags", tg.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "tags", tg.ID, 0)

	// Update reads its tenant predicate off the passed-in struct — call it
	// from the foreign ctx with the row's real tenantID so only RLS can be
	// stopping it.
	foreign := *tg
	foreign.Name = "Hacked"
	if err := repo.Update(ctxOther, &foreign); err != nil {
		t.Fatalf("Update (foreign ctx): unexpected error %v", err)
	}
	got, err := repo.GetByID(ctxOwn, tg.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != tg.Name {
		t.Fatalf("a foreign-tenant write reached the tag: name=%q", got.Name)
	}

	foreign.Name = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, tg.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != foreign.Name {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// Delete carries an explicit tenantID predicate.
	if err := repo.Delete(ctxOther, tg.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tags", tg.ID, 1)

	if err := repo.Delete(ctxOwn, tg.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tags", tg.ID, 0)
}

// TestTagNames_UniquePerTenantNotGlobally is a regression test for a
// pre-existing bug found while writing the write-surface test above:
// idx_tags_entity_name (migration 000006) was never re-scoped by tenant_id
// when tenant_id was retrofitted onto tags (migration 000106), so a tag name
// was unique per entity_type GLOBALLY — the second tenant to create e.g. a
// "VIP" contact tag got a raw unique-violation 500. Fixed in migration
// 000255 (tenant_id, entity_type, LOWER(name)).
func TestTagNames_UniquePerTenantNotGlobally(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Tag Unique Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Tag Unique Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	name := "VIP-" + uuid.New().String()[:8]

	tagA := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantA,
		Name:       name,
		Color:      "#3d8abf",
		EntityType: models.EntityType("contact"),
		CreatedAt:  time.Now().UTC(),
	}
	if err := repo.Create(ctxA, tagA); err != nil {
		t.Fatalf("Create tenant A: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tags", tagA.ID)

	tagB := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantB,
		Name:       name,
		Color:      "#bf3d3d",
		EntityType: models.EntityType("contact"),
		CreatedAt:  time.Now().UTC(),
	}
	if err := repo.Create(ctxB, tagB); err != nil {
		t.Fatalf("Create tenant B with the same name: %v (idx_tags_entity_name is not tenant-scoped)", err)
	}
	defer testutil.CleanupRow(t, pool, "tags", tagB.ID)
}
