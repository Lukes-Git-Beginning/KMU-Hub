package activity

// wp-crm-core: activities is a core CRM table — every existing RLS test
// (rls_test.go) seeds exclusively through testutil.SeedRow (a raw INSERT
// under system context) and never calls the real Create/Update/Delete
// methods. This file closes that gap for the write surface.
//
// Update, like deal.Update, checks RowsAffected() and returns
// ErrActivityNotFound on a zero-row match — so a foreign-tenant Update is
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

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestActivityWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Activity Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CRM Activity Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("crm-activity-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	a := &models.Activity{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		ActivityType: models.ActivityTypeNote,
		Subject:      "Write Test Activity",
		CreatedBy:    userID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, a); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "activities", a.ID, 0)

	if err := repo.Create(ctxOwn, a); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "activities", a.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "activities", a.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "activities", a.ID, 0)

	// Update checks RowsAffected() and returns ErrActivityNotFound when RLS
	// hides the row from the caller's session — the row is invisible, not
	// just unmatched, so this must fail, not silently no-op.
	foreign := *a
	foreign.Subject = "Hacked Activity"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); !errors.Is(err, ErrActivityNotFound) {
		t.Fatalf("Update (foreign ctx): expected ErrActivityNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, a.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Subject != "Write Test Activity" {
		t.Fatalf("a foreign-tenant write reached the activity: subject=%q", got.Subject)
	}

	foreign.Subject = "Renamed Activity"
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, a.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Subject != "Renamed Activity" {
		t.Fatalf("own-tenant write did not land: subject=%q", got.Subject)
	}

	// Delete carries an explicit tenant_id predicate and does not check
	// RowsAffected — a foreign-ctx call silently matches zero rows.
	if err := repo.Delete(ctxOther, a.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "activities", a.ID, 1)

	if err := repo.Delete(ctxOwn, a.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "activities", a.ID, 0)
}
