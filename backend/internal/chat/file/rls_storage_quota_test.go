package file_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// storage_quotas is unique on tenant_id since migration 000275, so these
// tests seed their own tenants instead of sharing testutil.TenantA/TenantB: a
// parallel test writing a quota row for a shared tenant would collide on that
// index. Cleanups run as `defer` in the test, not via t.Cleanup — these tests
// close the pool with `defer pool.Close()`, and t.Cleanup runs after every
// defer, so a cleanup registered there would hit a closed pool and fail
// silently, leaking a tenant per run.

func seedQuotaTenants(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantOne, tenantTwo := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOne, "Storage Quota Tenant One")
	testutil.EnsureTenant(t, pool, tenantTwo, "Storage Quota Tenant Two")
	return tenantOne, tenantTwo
}

// TestRLS_StorageQuotas_ForeignTenantSeesNothing is the standard isolation
// check for the tenant_isolation policy migration 000275 added.
func TestRLS_StorageQuotas_ForeignTenantSeesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne, tenantTwo := seedQuotaTenants(t, pool)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)
	defer testutil.CleanupRow(t, pool, "tenants", tenantTwo)

	quotaID := testutil.SeedRow(t, pool, "storage_quotas", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  tenantOne,
		"max_bytes":  int64(1 << 30),
		"used_bytes": int64(0),
	})
	defer testutil.CleanupRow(t, pool, "storage_quotas", quotaID)

	ctxOne := testutil.WithTenantCtx(context.Background(), tenantOne)
	testutil.AssertRowCount(t, pool, ctxOne, "storage_quotas", quotaID, 1)

	ctxTwo := testutil.WithTenantCtx(context.Background(), tenantTwo)
	testutil.AssertRowCount(t, pool, ctxTwo, "storage_quotas", quotaID, 0)
}

// TestStorageQuota_TenantsHoldIndependentCounters is the bug 000275 exists
// for. IncrementUsedBytes/DecrementUsedBytes ran without a WHERE clause
// against a single global row, so one tenant's upload consumed every other
// tenant's budget. Both tenants must now keep their own counter.
func TestStorageQuota_TenantsHoldIndependentCounters(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne, tenantTwo := seedQuotaTenants(t, pool)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)
	defer testutil.CleanupRow(t, pool, "tenants", tenantTwo)

	repo := file.NewPostgresRepository(pool)
	ctxOne := testutil.WithTenantCtx(context.Background(), tenantOne)
	ctxTwo := testutil.WithTenantCtx(context.Background(), tenantTwo)

	if err := repo.IncrementUsedBytes(ctxOne, tenantOne, 1000); err != nil {
		t.Fatalf("increment tenant one: %v", err)
	}
	defer cleanupStorageQuota(t, pool, tenantOne)

	if err := repo.IncrementUsedBytes(ctxTwo, tenantTwo, 5000); err != nil {
		t.Fatalf("increment tenant two: %v", err)
	}
	defer cleanupStorageQuota(t, pool, tenantTwo)

	if err := repo.IncrementUsedBytes(ctxOne, tenantOne, 500); err != nil {
		t.Fatalf("second increment tenant one: %v", err)
	}
	if err := repo.DecrementUsedBytes(ctxTwo, tenantTwo, 2000); err != nil {
		t.Fatalf("decrement tenant two: %v", err)
	}

	quotaOne, err := repo.GetStorageQuota(ctxOne, tenantOne)
	if err != nil {
		t.Fatalf("read back tenant one: %v", err)
	}
	if quotaOne.UsedBytes != 1500 {
		t.Fatalf("tenant one used_bytes: want 1500, got %d — the other tenant's write leaked", quotaOne.UsedBytes)
	}

	quotaTwo, err := repo.GetStorageQuota(ctxTwo, tenantTwo)
	if err != nil {
		t.Fatalf("read back tenant two: %v", err)
	}
	if quotaTwo.UsedBytes != 3000 {
		t.Fatalf("tenant two used_bytes: want 3000, got %d", quotaTwo.UsedBytes)
	}
}

// TestStorageQuota_IncrementUpsertsForFreshTenant covers the reason
// IncrementUsedBytes upserts: tenants created after 000275 hold no row, and a
// plain UPDATE would have reported success while changing nothing.
func TestStorageQuota_IncrementUpsertsForFreshTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	fresh := uuid.New()
	testutil.EnsureTenant(t, pool, fresh, "Storage Quota Fresh Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", fresh)

	repo := file.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), fresh)

	if _, err := repo.GetStorageQuota(ctx, fresh); !errors.Is(err, file.ErrQuotaNotFound) {
		t.Fatalf("fresh tenant: want ErrQuotaNotFound, got %v", err)
	}

	if err := repo.IncrementUsedBytes(ctx, fresh, 2048); err != nil {
		t.Fatalf("first increment for fresh tenant: %v", err)
	}
	defer cleanupStorageQuota(t, pool, fresh)

	quota, err := repo.GetStorageQuota(ctx, fresh)
	if err != nil {
		t.Fatalf("read back fresh tenant: %v", err)
	}
	if quota.UsedBytes != 2048 {
		t.Fatalf("fresh tenant used_bytes: want 2048, got %d", quota.UsedBytes)
	}
	if quota.MaxBytes != file.DefaultMaxQuotaBytes {
		t.Fatalf("fresh tenant max_bytes: want column default %d, got %d", file.DefaultMaxQuotaBytes, quota.MaxBytes)
	}
}

// cleanupStorageQuota removes a tenant's quota row; the table is keyed on
// tenant_id, not on an id the test knows in advance.
func cleanupStorageQuota(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx, `DELETE FROM storage_quotas WHERE tenant_id = $1`, tenantID); err != nil {
		t.Logf("cleanup storage_quotas tenant=%s: %v", tenantID, err)
	}
}
