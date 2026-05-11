package contact

// RLS integration tests for the contacts table (Migration 000120, tenant_isolation policy).
// Run with: DATABASE_URL='postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable' go test -count=1 -short ./internal/crm/contact

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRLS_Contacts_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-c-a-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "Alice",
		"last_name":  "RLS-Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "contacts", rowID, 0)
}

func TestRLS_Contacts_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-c-own-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "Bob",
		"last_name":  "RLS-Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", rowID)

	ctxA := testutil.WithTenantCtx(ctx, testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "contacts", rowID, 1)
}

func TestRLS_Contacts_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-c-sys-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "Carol",
		"last_name":  "RLS-Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", rowID)

	ctxSys := testutil.WithSystemCtx(ctx)
	testutil.AssertRowCount(t, pool, ctxSys, "contacts", rowID, 1)
}

func TestRLS_Contacts_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	ctx := context.Background()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("rls-c-upd-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	rowID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"first_name": "Dave",
		"last_name":  "RLS-Test",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", rowID)

	ctxB := testutil.WithTenantCtx(ctx, testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE contacts SET notes = 'hacked' WHERE id = $1", 0, rowID)
}
