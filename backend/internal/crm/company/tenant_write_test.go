package company

// wp-crm-core: companies is a core CRM table — every existing RLS test
// (rls_test.go) seeds exclusively through testutil.SeedRow (a raw INSERT
// under system context) and never calls the real Create/Update/Delete
// methods. This file closes that gap for the write surface.
//
// Each write is exercised once from a foreign-tenant ctx passing the row's
// *real* tenantID as the explicit value, so only RLS can be stopping it —
// then repeated from the owning ctx to confirm it lands.

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

func TestCompanyWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Company Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	c := &models.Company{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Name:      "Write Test GmbH",
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, c); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "companies", c.ID, 0)

	if err := repo.Create(ctxOwn, c); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", c.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "companies", c.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "companies", c.ID, 0)

	// Update carries an explicit tenant_id predicate — call it from the
	// foreign ctx with the row's real tenantID so only RLS can be stopping it.
	foreign := *c
	foreign.Name = "Hacked GmbH"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign, tenantOwn); err != nil {
		t.Fatalf("Update (foreign ctx): unexpected error %v", err)
	}
	got, err := repo.GetByID(ctxOwn, c.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Write Test GmbH" {
		t.Fatalf("a foreign-tenant write reached the company: name=%q", got.Name)
	}

	foreign.Name = "Renamed GmbH"
	if err := repo.Update(ctxOwn, &foreign, tenantOwn); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, c.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != "Renamed GmbH" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// Delete carries the same explicit predicate.
	if err := repo.Delete(ctxOther, c.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "companies", c.ID, 1)

	if err := repo.Delete(ctxOwn, c.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "companies", c.ID, 0)
}

// TestCompanyGetByName_CaseInsensitiveIgnoresMergedAndOtherTenant exercises the SQL
// behind Service.FindOrCreateByName (contact import): case-insensitive exact match,
// merged companies excluded, and tenant isolation on a raw repository call.
func TestCompanyGetByName_CaseInsensitiveIgnoresMergedAndOtherTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Company GetByName Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Company GetByName Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-company-getbyname-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	active := &models.Company{
		ID: uuid.New(), TenantID: tenantOwn, Name: "GetByName Active GmbH",
		CreatedBy: userID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := repo.Create(ctxOwn, active); err != nil {
		t.Fatalf("Create active: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", active.ID)

	// A merged duplicate with the same name must not shadow the active company.
	// merged_into_id references companies(id), so it points at the active row itself.
	merged := &models.Company{
		ID: uuid.New(), TenantID: tenantOwn, Name: "GetByName Merged GmbH",
		CreatedBy: userID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := repo.Create(ctxOwn, merged); err != nil {
		t.Fatalf("Create merged: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "companies", merged.ID)
	if _, err := pool.Exec(ctxOwn, `UPDATE companies SET merged_into_id = $1 WHERE id = $2`, active.ID, merged.ID); err != nil {
		t.Fatalf("mark merged: %v", err)
	}

	// GetByName is case-insensitive but, like GetByID, takes an already-normalized
	// value — trimming is the Service's job (see FindOrCreateByName / TestService_
	// FindOrCreateByName_FindsCaseInsensitive for that layer).
	got, err := repo.GetByName(ctxOwn, tenantOwn, "getbyname active gmbh")
	if err != nil {
		t.Fatalf("GetByName (case-insensitive): %v", err)
	}
	if got.ID != active.ID {
		t.Fatalf("expected active company %s, got %s", active.ID, got.ID)
	}

	if _, err := repo.GetByName(ctxOwn, tenantOwn, "GetByName Merged GmbH"); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound for a merged company, got %v", err)
	}

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	if _, err := repo.GetByName(ctxOther, tenantOther, "GetByName Active GmbH"); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound across tenants, got %v", err)
	}

	names, err := repo.GetNamesByIDs(ctxOwn, []uuid.UUID{active.ID, merged.ID}, tenantOwn)
	if err != nil {
		t.Fatalf("GetNamesByIDs: %v", err)
	}
	if names[active.ID] != "GetByName Active GmbH" {
		t.Fatalf("expected active company name in batch result, got %q", names[active.ID])
	}
	otherNames, err := repo.GetNamesByIDs(ctxOther, []uuid.UUID{active.ID}, tenantOther)
	if err != nil {
		t.Fatalf("GetNamesByIDs (foreign tenant): %v", err)
	}
	if _, leaked := otherNames[active.ID]; leaked {
		t.Fatal("GetNamesByIDs leaked a company across tenants")
	}
}
