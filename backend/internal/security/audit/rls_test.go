package audit_test

import (
	"context"
	"fmt"
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

func TestRLS_AuditLog_CrossTenantUpdateAffectsZeroRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	id := testutil.SeedRow(t, pool, "audit_log", map[string]any{
		"tenant_id":     testutil.TenantA,
		"action":        fmt.Sprintf("rls-test-cross-update-%s", uuid.New()),
		"result":        "success",
		"entry_hash":    "aabbcc",
		"previous_hash": "",
	})
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE audit_log SET result='failure' WHERE id = $1", 0, id)
}
