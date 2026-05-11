package consent

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// ─────────────────────────────────────────────────────────────────────────────
// consent_records  (FK: contact_id → contacts, nullable)
// ─────────────────────────────────────────────────────────────────────────────

func TestRLS_ConsentRecords_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	// Seed a user (needed as created_by FK on contacts)
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "consent-b@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Seed a contact for TenantA
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "ConsentB",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	// Seed a consent_record for TenantA
	consentID := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    testutil.TenantA,
		"contact_id":   contactID,
		"consent_type": "marketing_email",
		"granted":      true,
		"legal_basis":  "consent",
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentID)

	// TenantB must not see TenantA's record
	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "consent_records", consentID, 0)
}

func TestRLS_ConsentRecords_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "consent-own@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "ConsentOwn",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	consentID := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    testutil.TenantA,
		"contact_id":   contactID,
		"consent_type": "newsletter",
		"granted":      false,
		"legal_basis":  "consent",
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "consent_records", consentID, 1)
}

func TestRLS_ConsentRecords_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "consent-sys@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "ConsentSys",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	consentID := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    testutil.TenantA,
		"contact_id":   contactID,
		"consent_type": "data_processing",
		"granted":      true,
		"legal_basis":  "legitimate_interest",
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentID)

	sysCtx := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, sysCtx, "consent_records", consentID, 1)
}

func TestRLS_ConsentRecords_CrossTenantUpdateBlockedByRLS(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "consent-update-b@rls.test",
		"password_hash": "$2a$10$placeholder",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "RLS",
		"last_name":  "ConsentUpdateB",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	consentID := testutil.SeedRow(t, pool, "consent_records", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    testutil.TenantA,
		"contact_id":   contactID,
		"consent_type": "profiling",
		"granted":      true,
		"legal_basis":  "consent",
	})
	defer testutil.CleanupRow(t, pool, "consent_records", consentID)

	// TenantB must not be able to update TenantA's consent record
	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		`UPDATE consent_records SET granted = false WHERE id = $1`,
		0, consentID,
	)
}
