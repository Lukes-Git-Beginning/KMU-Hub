package deal

// wp-crm-core: deals is a core CRM table — every existing RLS test
// (rls_test.go) seeds exclusively through testutil.SeedRow (a raw INSERT
// under system context) and never calls the real Create/Update/Delete
// methods. This file closes that gap for the write surface.
//
// Update, unlike contact/company, checks RowsAffected() and returns
// ErrDealNotFound on a zero-row match — so a foreign-tenant Update is
// expected to *error*, not silently no-op.
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
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestDealWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Deal Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Deal Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-deal-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantOwn,
		"name":      "Write Test Stage " + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	d := &models.Deal{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Name:      "Write Test Deal",
		Value:     decimal.NewFromInt(1000),
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, d); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "deals", d.ID, 0)

	if err := repo.Create(ctxOwn, d); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "deals", d.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "deals", d.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "deals", d.ID, 0)

	// Update checks RowsAffected() and returns ErrDealNotFound when RLS hides
	// the row from the caller's session — the row is invisible, not just
	// unmatched, so this must fail, not silently no-op.
	foreign := *d
	foreign.Name = "Hacked Deal"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); !errors.Is(err, ErrDealNotFound) {
		t.Fatalf("Update (foreign ctx): expected ErrDealNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, d.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Write Test Deal" {
		t.Fatalf("a foreign-tenant write reached the deal: name=%q", got.Name)
	}

	foreign.Name = "Renamed Deal"
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, d.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != "Renamed Deal" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	// Delete carries an explicit tenant_id predicate and does not check
	// RowsAffected — a foreign-ctx call silently matches zero rows.
	if err := repo.Delete(ctxOther, d.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "deals", d.ID, 1)

	if err := repo.Delete(ctxOwn, d.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "deals", d.ID, 0)
}
