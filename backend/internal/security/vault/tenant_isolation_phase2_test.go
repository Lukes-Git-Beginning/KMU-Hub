package vault

// Cross-tenant isolation tests for vault_secrets (Migration 000124).
// vault_secrets.tenant_id is NOT NULL after backfill + SET NOT NULL in mig 000124.
// RLS policy: standard tenant_isolation.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_VaultSecrets(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	secretID := testutil.SeedRow(t, pool, "vault_secrets", map[string]any{
		"tenant_id":        testutil.TenantA,
		"key_name":         "smtp_password_" + uuid.New().String()[:8],
		"encrypted_value":  "enc:abc123",
	})
	defer testutil.CleanupRow(t, pool, "vault_secrets", secretID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "vault_secrets", secretID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "vault_secrets", secretID, 0)
}
