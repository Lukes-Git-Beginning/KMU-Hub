package project

// wp-work: closes the write-surface gap for projects the same way
// wp-crm-core did for CRM entities — the existing tenant_isolation_test.go/
// tenant_isolation_phase2_test.go files seed exclusively via testutil.SeedRow
// and never call the real Create/Update/Delete methods.

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

func seedProjectUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-project-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

// TestProjectWrites_LandInCallerTenant covers the core Create/Update/Delete
// write surface, following the same pattern as wp-crm-core/wp-crm-meta.
func TestProjectWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Project Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Project Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedProjectUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	proj := &models.Project{
		ID:             uuid.New(),
		TenantID:       tenantOwn,
		Name:           "Write-Test-" + uuid.New().String()[:8],
		ProjectKey:     "WP" + uuid.New().String()[:6],
		NextTaskNumber: 1,
		CreatedBy:      userOwn,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, proj); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "projects", proj.ID, 0)

	if err := repo.Create(ctxOwn, proj); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "projects", proj.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "projects", proj.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "projects", proj.ID, 0)

	// Update carries an explicit tenant_id predicate and checks RowsAffected
	// — a foreign-ctx call must come back an error, not a silent no-op.
	foreign := *proj
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetByID(ctxOwn, proj.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != proj.Name {
		t.Fatalf("a foreign-tenant write reached the project: name=%q", got.Name)
	}

	foreign.Name = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, proj.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != foreign.Name {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// Archive carries an explicit tenantID predicate and checks RowsAffected.
	if err := repo.Archive(ctxOther, proj.ID, tenantOwn); err == nil {
		t.Fatalf("Archive (foreign ctx): expected an error, got nil")
	}
	got, err = repo.GetByID(ctxOwn, proj.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ArchivedAt != nil {
		t.Fatalf("a foreign-tenant Archive call reached the project")
	}

	// Delete carries an explicit tenantID predicate.
	if err := repo.Delete(ctxOther, proj.ID, tenantOwn); err == nil {
		t.Fatalf("Delete (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "projects", proj.ID, 1)

	if err := repo.Delete(ctxOwn, proj.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "projects", proj.ID, 0)
}
