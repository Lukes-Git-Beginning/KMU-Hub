package resource

// wp-work: closes the write-surface gap for bookable resources the same way
// wp-crm-core did for CRM entities. Every write path here already carried an
// explicit tenant_id predicate AND checked RowsAffected before this unit —
// this test is pure coverage, no bug found.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedResourceUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-resource-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

// TestResourceWrites_LandInCallerTenant covers the core Create/Update/Delete
// write surface, following the same pattern as wp-crm-core/wp-crm-meta.
// Delete is a soft delete (is_active = false), not a row removal.
func TestResourceWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Resource Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Resource Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedResourceUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	res := &models.Resource{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Name:         "Write-Test-" + uuid.New().String()[:8],
		ResourceType: models.ResourceTypeRoom,
		IsActive:     true,
		CreatedBy:    userOwn,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, res); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "resources", res.ID, 0)

	if err := repo.Create(ctxOwn, res); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "resources", res.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "resources", res.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "resources", res.ID, 0)

	// Update carries an explicit tenant_id predicate and checks RowsAffected
	// — a foreign-ctx call must come back an error, not a silent no-op.
	foreign := *res
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByID(ctxOwn, res.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != res.Name {
		t.Fatalf("a foreign-tenant write reached the resource: name=%q", got.Name)
	}

	foreign.Name = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, res.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != foreign.Name {
		t.Fatalf("own-tenant write did not land: name=%v", got.Name)
	}

	// Delete (soft delete) carries an explicit tenant_id predicate and checks
	// RowsAffected.
	if err := repo.Delete(ctxOther, res.ID, tenantOwn); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	got, err = repo.GetByID(ctxOwn, res.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID after foreign delete: %v", err)
	}
	if !got.IsActive {
		t.Fatalf("a foreign-tenant delete reached the resource: is_active=%v", got.IsActive)
	}

	if err := repo.Delete(ctxOwn, res.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, res.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID after own delete: %v", err)
	}
	if got.IsActive {
		t.Fatalf("own-tenant delete did not land: is_active=%v", got.IsActive)
	}
}
