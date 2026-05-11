package idempotency_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// idempotency_keys uses a composite PK (tenant_id, key) — no auto-generated id.
// SeedRow assumes RETURNING id, so we use pool.Exec directly for all DML.

func TestRLS_IdempotencyKeys_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	// idempotency_keys.user_id FK → users(id); seed a minimal user.
	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "idem-b-sees-none@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	key := "rls-test-b-sees-none-" + uuid.New().String()
	sysCtx := testutil.WithSystemCtx(context.Background())

	if _, err := pool.Exec(sysCtx,
		`INSERT INTO idempotency_keys (tenant_id, key, user_id, method, path, request_hash)
		 VALUES ($1, $2, $3, 'POST', '/api/v1/test', 'hash-b-sees-none')`,
		testutil.TenantA, key, userA,
	); err != nil {
		t.Fatalf("seed idempotency_keys: %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", testutil.TenantA, key)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	var count int
	if err := pool.QueryRow(ctxB,
		"SELECT count(*) FROM idempotency_keys WHERE key=$1", key,
	).Scan(&count); err != nil {
		t.Fatalf("count idempotency_keys as TenantB: %v", err)
	}
	if count != 0 {
		t.Fatalf("TenantB must not see TenantA idempotency_keys: got %d", count)
	}
}

func TestRLS_IdempotencyKeys_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "idem-a-sees-own@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	key := "rls-test-a-sees-own-" + uuid.New().String()
	sysCtx := testutil.WithSystemCtx(context.Background())

	if _, err := pool.Exec(sysCtx,
		`INSERT INTO idempotency_keys (tenant_id, key, user_id, method, path, request_hash)
		 VALUES ($1, $2, $3, 'POST', '/api/v1/test', 'hash-a-sees-own')`,
		testutil.TenantA, key, userA,
	); err != nil {
		t.Fatalf("seed idempotency_keys: %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", testutil.TenantA, key)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	var count int
	if err := pool.QueryRow(ctxA,
		"SELECT count(*) FROM idempotency_keys WHERE key=$1", key,
	).Scan(&count); err != nil {
		t.Fatalf("count idempotency_keys as TenantA: %v", err)
	}
	if count != 1 {
		t.Fatalf("TenantA must see own idempotency_keys: got %d", count)
	}
}

func TestRLS_IdempotencyKeys_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "idem-sys-a@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantB,
		"email":         "idem-sys-b@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	keyA := "rls-test-sys-a-" + uuid.New().String()
	keyB := "rls-test-sys-b-" + uuid.New().String()
	sysCtx := testutil.WithSystemCtx(context.Background())

	if _, err := pool.Exec(sysCtx,
		`INSERT INTO idempotency_keys (tenant_id, key, user_id, method, path, request_hash)
		 VALUES ($1, $2, $3, 'POST', '/api/v1/test', 'hash-sys-a')`,
		testutil.TenantA, keyA, userA,
	); err != nil {
		t.Fatalf("seed keyA: %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", testutil.TenantA, keyA)

	if _, err := pool.Exec(sysCtx,
		`INSERT INTO idempotency_keys (tenant_id, key, user_id, method, path, request_hash)
		 VALUES ($1, $2, $3, 'POST', '/api/v1/test', 'hash-sys-b')`,
		testutil.TenantB, keyB, userB,
	); err != nil {
		t.Fatalf("seed keyB: %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", testutil.TenantB, keyB)

	var countA, countB int
	if err := pool.QueryRow(sysCtx,
		"SELECT count(*) FROM idempotency_keys WHERE key=$1", keyA,
	).Scan(&countA); err != nil {
		t.Fatalf("count keyA as system: %v", err)
	}
	if err := pool.QueryRow(sysCtx,
		"SELECT count(*) FROM idempotency_keys WHERE key=$1", keyB,
	).Scan(&countB); err != nil {
		t.Fatalf("count keyB as system: %v", err)
	}
	if countA != 1 {
		t.Fatalf("system must see keyA: got %d", countA)
	}
	if countB != 1 {
		t.Fatalf("system must see keyB: got %d", countB)
	}
}

func TestRLS_IdempotencyKeys_SameKeyInTwoTenantsIsolated(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "idem-dup-a@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantB,
		"email":         "idem-dup-b@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	dupKey := "dup-key-" + uuid.New().String()
	sysCtx := testutil.WithSystemCtx(context.Background())

	// Insert same key for TenantA.
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO idempotency_keys (tenant_id, key, user_id, method, path, request_hash)
		 VALUES ($1, $2, $3, 'POST', '/api/v1/contacts', 'hash-dup-a')`,
		testutil.TenantA, dupKey, userA,
	); err != nil {
		t.Fatalf("seed dupKey for TenantA: %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", testutil.TenantA, dupKey)

	// Insert same key for TenantB — composite PK (tenant_id, key) must allow this.
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO idempotency_keys (tenant_id, key, user_id, method, path, request_hash)
		 VALUES ($1, $2, $3, 'POST', '/api/v1/contacts', 'hash-dup-b')`,
		testutil.TenantB, dupKey, userB,
	); err != nil {
		t.Fatalf("seed dupKey for TenantB (composite PK must allow same key in different tenants): %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", testutil.TenantB, dupKey)

	// TenantA sees exactly 1.
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	var countA int
	if err := pool.QueryRow(ctxA,
		"SELECT count(*) FROM idempotency_keys WHERE key=$1", dupKey,
	).Scan(&countA); err != nil {
		t.Fatalf("count as TenantA: %v", err)
	}
	if countA != 1 {
		t.Fatalf("TenantA must see exactly 1 row for dupKey: got %d", countA)
	}

	// TenantB sees exactly 1.
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	var countB int
	if err := pool.QueryRow(ctxB,
		"SELECT count(*) FROM idempotency_keys WHERE key=$1", dupKey,
	).Scan(&countB); err != nil {
		t.Fatalf("count as TenantB: %v", err)
	}
	if countB != 1 {
		t.Fatalf("TenantB must see exactly 1 row for dupKey: got %d", countB)
	}

	// System sees both (2 total).
	var countSys int
	if err := pool.QueryRow(sysCtx,
		"SELECT count(*) FROM idempotency_keys WHERE key=$1", dupKey,
	).Scan(&countSys); err != nil {
		t.Fatalf("count as system: %v", err)
	}
	if countSys != 2 {
		t.Fatalf("system must see 2 rows for dupKey (one per tenant): got %d", countSys)
	}
}
