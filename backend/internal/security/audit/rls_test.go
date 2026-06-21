package audit_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRLS_AuditLog_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	id := testutil.SeedRow(t, pool, "audit_log", map[string]any{
		"tenant_id":     testutil.TenantA,
		"action":        fmt.Sprintf("rls-test-b-sees-none-%s", uuid.New()),
		"result":        "success",
		"entry_hash":    "aabbcc",
		"previous_hash": "",
	})
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "audit_log", id, 0)
}

func TestRLS_AuditLog_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")

	id := testutil.SeedRow(t, pool, "audit_log", map[string]any{
		"tenant_id":     testutil.TenantA,
		"action":        fmt.Sprintf("rls-test-a-sees-own-%s", uuid.New()),
		"result":        "success",
		"entry_hash":    "aabbcc",
		"previous_hash": "",
	})
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "audit_log", id, 1)
}

func TestRLS_AuditLog_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	idA := testutil.SeedRow(t, pool, "audit_log", map[string]any{
		"tenant_id":     testutil.TenantA,
		"action":        fmt.Sprintf("rls-test-sys-a-%s", uuid.New()),
		"result":        "success",
		"entry_hash":    "aabbcc",
		"previous_hash": "",
	})
	defer testutil.CleanupRow(t, pool, "audit_log", idA)

	idB := testutil.SeedRow(t, pool, "audit_log", map[string]any{
		"tenant_id":     testutil.TenantB,
		"action":        fmt.Sprintf("rls-test-sys-b-%s", uuid.New()),
		"result":        "success",
		"entry_hash":    "ddeeff",
		"previous_hash": "",
	})
	defer testutil.CleanupRow(t, pool, "audit_log", idB)

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "audit_log", idA, 1)
	testutil.AssertRowCount(t, pool, sysCtx, "audit_log", idB, 1)
}

// TestRLS_AuditLog_AppendOnlyTriggerBlocksMutation verifies that the
// append-only trigger introduced in migration 000222 (GoBD/§257/§147 AO)
// blocks any UPDATE on audit_log rows with an explicit exception, even for the
// owning tenant. The old cross-tenant RLS isolation (UPDATE affects 0 rows for
// a foreign tenant) is still valid but is subsumed by the stronger guarantee:
// audit_log is immutable at the DB level regardless of who attempts the write.
func TestRLS_AuditLog_AppendOnlyTriggerBlocksMutation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")

	id := testutil.SeedRow(t, pool, "audit_log", map[string]any{
		"tenant_id":     testutil.TenantA,
		"action":        fmt.Sprintf("rls-test-append-only-%s", uuid.New()),
		"result":        "success",
		"entry_hash":    "aabbcc",
		"previous_hash": "",
	})
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	// Attempt to mutate the row as TenantA (the owner) — the trigger must
	// block this with the "audit_log is append-only" exception.
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	_, err := pool.Exec(ctxA, "UPDATE audit_log SET result='failure' WHERE id = $1", id)
	if err == nil {
		t.Fatal("expected append-only trigger to block UPDATE on audit_log, but got no error")
	}
	if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("expected 'append-only' in error message, got: %v", err)
	}
}
