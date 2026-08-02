package contact

// wp-crm-core: contacts is the highest-traffic CRM table — every existing RLS
// test (rls_test.go) seeds exclusively through testutil.SeedRow (a raw INSERT
// under system context) and never calls the real Create/Update/Delete
// methods. This file closes that gap for the write surface.
//
// Each write is exercised once from a foreign-tenant ctx passing the row's
// *real* tenantID as the explicit value, so only RLS can be stopping it —
// then repeated from the owning ctx to confirm it lands.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestContactWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Contact Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Contact Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-contact-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	c := &models.Contact{
		ID:         uuid.New(),
		TenantID:   tenantOwn,
		FirstName:  "Write",
		LastName:   "Test",
		Visibility: "shared",
		CreatedBy:  userID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, c); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", c.ID, 0)

	if err := repo.Create(ctxOwn, c); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "contacts", c.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "contacts", c.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "contacts", c.ID, 0)

	// Update carries an explicit tenant_id predicate — call it from the
	// foreign ctx with the row's real tenantID so only RLS can be stopping it.
	foreign := *c
	foreign.LastName = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign, tenantOwn); err != nil {
		t.Fatalf("Update (foreign ctx): unexpected error %v", err)
	}
	got, err := repo.GetByID(ctxOwn, c.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LastName != "Test" {
		t.Fatalf("a foreign-tenant write reached the contact: last_name=%q", got.LastName)
	}
	if got.CreatedBy != userID {
		t.Fatalf("created_by did not round-trip through Create/GetByID: got %s, want %s", got.CreatedBy, userID)
	}

	foreign.LastName = "Renamed"
	if err := repo.Update(ctxOwn, &foreign, tenantOwn); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, c.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.LastName != "Renamed" {
		t.Fatalf("own-tenant write did not land: last_name=%q", got.LastName)
	}

	// Delete carries the same explicit predicate.
	if err := repo.Delete(ctxOther, c.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", c.ID, 1)

	if err := repo.Delete(ctxOwn, c.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "contacts", c.ID, 0)
}
