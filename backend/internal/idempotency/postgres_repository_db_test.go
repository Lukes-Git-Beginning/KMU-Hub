package idempotency_test

// postgres_repository_db_test.go exercises postgresRepository against the
// real schema instead of the in-memory mockRepository in repository_test.go.
// Filed for fix-idempotency-real-sql-coverage-core (BACKLOG.yml): the value
// of this store is entirely in what Postgres actually does under
// ON CONFLICT and concurrent access, which a mock cannot answer.

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// lockKeyFromUUID derives a pg_advisory_lock key from a fresh UUID so
// parallel tests (t.Parallel()) never collide on the same session-level
// lock in the shared test database.
func lockKeyFromUUID(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[:8]))
}

func cleanupIdempotencyKey(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx, "DELETE FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", tenantID, key); err != nil {
		t.Logf("cleanup idempotency_keys tenant=%s key=%s: %v", tenantID, key, err)
	}
}

func TestPostgresReserve_FreshKey(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Reserve Fresh")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()
	key := "db-fresh-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, key)

	rec, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-fresh")
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil record for fresh reserve, got %+v", rec)
	}

	stored, err := repo.Get(ctx, tenantID, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.CompletedAt != nil {
		t.Fatalf("expected fresh reservation to have no completed_at, got %v", stored.CompletedAt)
	}
}

func TestPostgresReserve_Replay_ReturnsCachedRecord(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Reserve Replay")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()
	key := "db-replay-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, key)

	if _, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-replay"); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if err := repo.Complete(ctx, tenantID, key, 201, []byte(`{"id":"abc"}`)); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	rec, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-replay")
	if err != nil {
		t.Fatalf("Reserve (replay): %v", err)
	}
	if rec == nil {
		t.Fatal("expected replay to return the completed record, got nil")
	}
	if rec.ResponseStatus == nil || *rec.ResponseStatus != 201 {
		t.Fatalf("expected cached status 201, got %+v", rec.ResponseStatus)
	}
	// response_body is JSONB — Postgres reformats whitespace on round-trip,
	// so compare decoded content rather than raw bytes.
	var got map[string]any
	if err := json.Unmarshal(rec.ResponseBody, &got); err != nil {
		t.Fatalf("unmarshal cached body: %v", err)
	}
	if got["id"] != "abc" {
		t.Fatalf("expected cached body {\"id\":\"abc\"}, got %s", rec.ResponseBody)
	}
}

func TestPostgresReserve_Conflict_DifferentHash(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Reserve Conflict")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()
	key := "db-conflict-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, key)

	if _, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-original"); err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	_, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-different")
	if !errors.Is(err, idempotency.ErrConflict) {
		t.Fatalf("expected ErrConflict for reused key with different payload, got %v", err)
	}
}

// TestPostgresReserve_ConcurrentSameKey_NoErrorEitherSide documents a real gap
// in the Postgres-backed Reserve rather than the mock's contract: unlike
// mockRepository.Reserve in repository_test.go, which returns ErrInFlight for
// a second in-flight reservation, the real INSERT ... ON CONFLICT DO UPDATE
// path cannot distinguish "I just inserted this row" from "someone else beat
// me to it a moment ago" — both return (nil, nil) as long as completed_at is
// still unset. Two concurrent callers with the same key therefore both
// believe they own the reservation and may both proceed to run the handler.
// Already flagged as a pre-existing, out-of-scope gap at
// internal/automation/workflow/webhook.go:164 for the webhook trigger path;
// this test proves it against the shared repository directly, under real
// concurrent connections, so it is not unique to that one caller. See
// fix-idempotency-reserve-inflight-race filed at the end of BACKLOG.yml.
func TestPostgresReserve_ConcurrentSameKey_NoErrorEitherSide(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Reserve Concurrent")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()
	key := "db-concurrent-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, key)

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 2)
	recs := make([]*idempotency.Record, 2)

	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			recs[i], errs[i] = repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-concurrent")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: expected no error from concurrent Reserve (documented gap, not a hardened lock), got %v", i, err)
		}
		if recs[i] != nil {
			t.Fatalf("goroutine %d: expected nil record (neither side sees a completed response yet), got %+v", i, recs[i])
		}
	}

	// Exactly one row must exist despite two INSERTs racing the same key —
	// the composite PK still enforces that much.
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM idempotency_keys WHERE tenant_id=$1 AND key=$2", tenantID, key).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one row for the raced key, got %d", count)
	}
}

func TestPostgresGet_ReservedButNeverCompleted(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Get Uncompleted")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()
	key := "db-uncompleted-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, key)

	if _, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-uncompleted"); err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	rec, err := repo.Get(ctx, tenantID, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.CompletedAt != nil {
		t.Fatalf("expected CompletedAt nil for a reserved-but-never-completed key, got %v", rec.CompletedAt)
	}
	if rec.ResponseStatus != nil {
		t.Fatalf("expected no response_status before Complete, got %v", *rec.ResponseStatus)
	}
	if !rec.ExpiresAt.After(rec.CreatedAt) {
		t.Fatalf("expected expires_at after created_at, got created=%v expires=%v", rec.CreatedAt, rec.ExpiresAt)
	}
}

func TestPostgresGet_UnknownKey_ReturnsErrNoRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Get Unknown")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	_, err := repo.Get(ctx, tenantID, "does-not-exist-"+uuid.New().String())
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows for unknown key, got %v", err)
	}
}

func TestPostgresComplete_StoresResponse(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Complete Stores")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()
	key := "db-complete-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, key)

	if _, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-complete"); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if err := repo.Complete(ctx, tenantID, key, 200, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	rec, err := repo.Get(ctx, tenantID, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.CompletedAt == nil {
		t.Fatal("expected CompletedAt set after Complete")
	}
	if rec.ResponseStatus == nil || *rec.ResponseStatus != 200 {
		t.Fatalf("expected response_status 200, got %+v", rec.ResponseStatus)
	}
}

func TestPostgresComplete_UnknownKey_ReturnsErrKeyMissing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Complete Missing")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	err := repo.Complete(ctx, tenantID, "does-not-exist-"+uuid.New().String(), 200, []byte(`{}`))
	if !errors.Is(err, idempotency.ErrKeyMissing) {
		t.Fatalf("expected ErrKeyMissing for a key that was never reserved, got %v", err)
	}
}

// TestPostgresComplete_CrossTenant_ReturnsErrKeyMissing proves the
// tenant-scoped WHERE clause in Complete() — not just RLS — refuses to
// complete another tenant's reservation, per the composite-PK cross-tenant
// defense documented on Repository.Complete.
func TestPostgresComplete_CrossTenant_ReturnsErrKeyMissing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Idempotency Complete Cross A")
	testutil.EnsureTenant(t, pool, tenantB, "Idempotency Complete Cross B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := idempotency.NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	userID := uuid.New()
	key := "db-cross-tenant-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantA, key)

	if _, err := repo.Reserve(ctxA, key, tenantA, userID, "POST", "/api/v1/test", "hash-cross"); err != nil {
		t.Fatalf("Reserve as tenant A: %v", err)
	}

	err := repo.Complete(ctxB, tenantB, key, 200, []byte(`{}`))
	if !errors.Is(err, idempotency.ErrKeyMissing) {
		t.Fatalf("expected ErrKeyMissing when tenant B completes tenant A's key, got %v", err)
	}
}

func TestPostgresCleanup_DeletesOnlyExpired(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Cleanup Expiry")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()

	freshKey := "db-cleanup-fresh-" + uuid.New().String()
	expiredKey := "db-cleanup-expired-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, freshKey)
	defer cleanupIdempotencyKey(t, pool, tenantID, expiredKey)

	if _, err := repo.Reserve(ctx, freshKey, tenantID, userID, "POST", "/api/v1/test", "hash-fresh"); err != nil {
		t.Fatalf("Reserve fresh: %v", err)
	}
	if _, err := repo.Reserve(ctx, expiredKey, tenantID, userID, "POST", "/api/v1/test", "hash-expired"); err != nil {
		t.Fatalf("Reserve expired: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		"UPDATE idempotency_keys SET expires_at = NOW() - INTERVAL '1 hour' WHERE tenant_id=$1 AND key=$2",
		tenantID, expiredKey); err != nil {
		t.Fatalf("force-expire: %v", err)
	}

	if _, err := repo.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if _, err := repo.Get(ctx, tenantID, freshKey); err != nil {
		t.Fatalf("expected fresh (non-expired) reservation to survive Cleanup, got %v", err)
	}
	if _, err := repo.Get(ctx, tenantID, expiredKey); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected expired reservation removed by Cleanup, got %v", err)
	}
}

// TestPostgresCleanupWithLock_SkipsWhenLockHeld proves the non-blocking half
// of the contract: pg_try_advisory_lock never blocks, so a second replica
// holding the same lockKey on its own connection must make CleanupWithLock
// return (0, nil) immediately instead of deleting anything.
func TestPostgresCleanupWithLock_SkipsWhenLockHeld(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctx := context.Background()
	lockKey := lockKeyFromUUID(uuid.New())

	holder, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire holder connection: %v", err)
	}
	defer holder.Release()

	var locked bool
	if err := holder.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&locked); err != nil {
		t.Fatalf("holder pg_try_advisory_lock: %v", err)
	}
	if !locked {
		t.Fatal("expected the holder connection to acquire the lock first")
	}
	defer func() {
		_, _ = holder.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
	}()

	repo := idempotency.NewPostgresRepository(pool)
	deleted, err := repo.CleanupWithLock(ctx, lockKey)
	if err != nil {
		t.Fatalf("CleanupWithLock: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected CleanupWithLock to skip while another session holds the lock, got %d deleted", deleted)
	}
}

// TestPostgresCleanupWithLock_AcquiresAndReleases proves the other half:
// once free, CleanupWithLock acquires the lock, deletes expired rows, and —
// critically — actually releases the lock afterwards. pool.QueryRow and
// pool.Exec each acquire a connection independently and are not guaranteed
// to reuse the same physical session, so the acquire and the release inside
// CleanupWithLock could silently run on different sessions; this test
// verifies from a second, independently acquired connection that the lock
// is genuinely free again, not leaked on whichever connection acquired it.
func TestPostgresCleanupWithLock_AcquiresAndReleases(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency CleanupWithLock Free")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := idempotency.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	userID := uuid.New()
	key := "db-lock-free-" + uuid.New().String()
	defer cleanupIdempotencyKey(t, pool, tenantID, key)

	if _, err := repo.Reserve(ctx, key, tenantID, userID, "POST", "/api/v1/test", "hash-lock-free"); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		"UPDATE idempotency_keys SET expires_at = NOW() - INTERVAL '1 hour' WHERE tenant_id=$1 AND key=$2",
		tenantID, key); err != nil {
		t.Fatalf("force-expire: %v", err)
	}

	lockKey := lockKeyFromUUID(uuid.New())
	bg := context.Background()
	// CleanupWithLock's DELETE goes through the same RLS policies as any
	// other write; production wraps it with sysctx (middleware/idempotency.go
	// cleanupCtx) because the worker has no per-request tenant. Without that,
	// the DELETE below matches zero rows, not because nothing is expired but
	// because RLS admits nothing.
	deleted, err := repo.CleanupWithLock(testutil.WithSystemCtx(bg), lockKey)
	if err != nil {
		t.Fatalf("CleanupWithLock: %v", err)
	}
	if deleted < 1 {
		t.Fatalf("expected the expired reservation to be deleted, got %d", deleted)
	}
	if _, err := repo.Get(ctx, tenantID, key); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected the expired reservation removed, got %v", err)
	}

	verifier, err := pool.Acquire(bg)
	if err != nil {
		t.Fatalf("acquire verifier connection: %v", err)
	}
	defer verifier.Release()

	var stillFree bool
	if err := verifier.QueryRow(bg, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&stillFree); err != nil {
		t.Fatalf("verifier pg_try_advisory_lock: %v", err)
	}
	if !stillFree {
		t.Fatalf("advisory lock %d is still held after CleanupWithLock returned -- "+
			"acquire and release ran on different pooled connections, leaking the lock", lockKey)
	}
	_, _ = verifier.Exec(bg, "SELECT pg_advisory_unlock($1)", lockKey)
}
