package savedfilter

// wp-crm-meta: closes the write-surface gap for saved_filters the same way
// wp-crm-core did for the core CRM entities — the package had no RLS test at
// all beforehand.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSavedFilterWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "SavedFilter Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "SavedFilter Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("savedfilter-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	f := &models.SavedFilter{
		ID:         uuid.New(),
		TenantID:   tenantOwn,
		Name:       "Write-Test-" + uuid.New().String()[:8],
		EntityType: models.EntityType("contact"),
		FilterJSON: `{"status":"open"}`,
		IsDefault:  false,
		CreatedBy:  userID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, f); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "saved_filters", f.ID, 0)

	if err := repo.Create(ctxOwn, f); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "saved_filters", f.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "saved_filters", f.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "saved_filters", f.ID, 0)

	// Update carries an explicit tenant_id predicate (filter.TenantID) and
	// treats zero affected rows as not-found — from a foreign session, RLS
	// makes the row invisible so the predicate never matches.
	foreign := *f
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); !errors.Is(err, ErrFilterNotFound) {
		t.Fatalf("Update (foreign ctx): expected ErrFilterNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, f.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != f.Name {
		t.Fatalf("a foreign-tenant write reached the saved filter: name=%q", got.Name)
	}

	foreign.Name = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, f.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != foreign.Name {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// Delete carries the same explicit predicate + not-found-on-zero-rows.
	if err := repo.Delete(ctxOther, f.ID, tenantOwn); !errors.Is(err, ErrFilterNotFound) {
		t.Fatalf("Delete (foreign ctx): expected ErrFilterNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "saved_filters", f.ID, 1)

	if err := repo.Delete(ctxOwn, f.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "saved_filters", f.ID, 0)
}
