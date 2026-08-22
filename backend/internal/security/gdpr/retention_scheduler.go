package gdpr

// Scheduling for the retention engine (A10) -- the trigger side of A10-A13,
// which built the engine and its handlers but wired none of them to run.
//
// Two existing periodic-worker shapes in this codebase are the templates
// here, not a new pattern: IdempotencyCleanupWorker's
// pg_try_advisory_lock leader election (internal/middleware/idempotency.go)
// for the single-runner guarantee across gateway/auth replicas, and
// cmd/work's recording-cleanup goroutine for the startup-delay + ticker
// shape. RunScheduledRetention does the actual per-tick work as a plain
// function, separate from the ticker loop, so a test can call it directly
// without waiting on real time.
//
// retention_policies (and therefore a run) is per-tenant, unlike the
// idempotency cleanup which is a single cross-tenant DELETE. The scheduler
// therefore has to enumerate tenants itself and run the engine once per
// tenant, each under sysctx (RLS bypass) plus that tenant's id in context --
// the same combination testutil.WithTenantCtx + testutil.WithSystemCtx
// build for tests.

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// retentionScheduleLockKey is the pg_try_advisory_lock key so only one
// replica runs a given tick. Arbitrary but constant, distinct from
// idempotencyCleanupLockKey (0x49444D50) in internal/middleware/idempotency.go.
const retentionScheduleLockKey int64 = 0x52544E4E // "RTNN"

// RunScheduledRetention acquires the schedule lock and, if it wins, runs the
// engine once for every tenant in the given mode. The bool return is false
// both when another replica already held the lock (a normal skip) and is
// true once this call owns the tick, regardless of whether any tenant's run
// itself errored -- a broken tenant must not stop the others, so per-tenant
// errors are logged, not returned.
func RunScheduledRetention(ctx context.Context, pool *pgxpool.Pool, engine *RetentionEngine, mode RetentionMode) (bool, error) {
	sysCtx := sysctx.With(ctx)

	var locked bool
	if err := pool.QueryRow(sysCtx, `SELECT pg_try_advisory_lock($1)`, retentionScheduleLockKey).Scan(&locked); err != nil {
		return false, err
	}
	if !locked {
		return false, nil
	}
	defer func() {
		// Best-effort release: the session returns to the pool either way,
		// and a lock outliving one dropped connection is not fatal -- the
		// next tick simply skips until Postgres reclaims it on disconnect.
		_, _ = pool.Exec(sysCtx, `SELECT pg_advisory_unlock($1)`, retentionScheduleLockKey)
	}()

	tenantIDs, err := listTenantIDs(sysCtx, pool)
	if err != nil {
		return true, err
	}
	runForTenants(sysCtx, engine, mode, tenantIDs)
	return true, nil
}

// listTenantIDs enumerates every tenant. Data is single-tenant in practice
// today, but the retention engine is tenant-scoped by design (Run reads
// middleware.GetTenantID), so the scheduler must not hardcode the one
// tenant that happens to exist.
func listTenantIDs(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM tenants ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// runForTenants runs the engine once per tenant. Kept separate from its
// caller so a test can exercise it against a controlled tenant list instead
// of the shared `tenants` table other tests also seed rows into.
func runForTenants(ctx context.Context, engine *RetentionEngine, mode RetentionMode, tenantIDs []uuid.UUID) {
	for _, tenantID := range tenantIDs {
		tenantCtx := context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
		if _, err := engine.Run(tenantCtx, mode, "schedule"); err != nil {
			slog.Error("retention scheduled run failed", "tenant_id", tenantID, "error", err)
		}
	}
}

// RetentionScheduler runs RunScheduledRetention once after startupDelay,
// then every interval, until ctx is cancelled. Errors are logged, never
// propagated -- a transient failure here must not crash the process.
func RetentionScheduler(ctx context.Context, pool *pgxpool.Pool, engine *RetentionEngine, mode RetentionMode, startupDelay, interval time.Duration) {
	select {
	case <-time.After(startupDelay):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("retention scheduler starting", "interval", interval.String(), "mode", string(mode))

	runTick := func() {
		ran, err := RunScheduledRetention(ctx, pool, engine, mode)
		if err != nil {
			slog.Error("retention scheduled run errored", "error", err)
			return
		}
		if !ran {
			slog.Debug("retention scheduled run skipped, lock held elsewhere")
		}
	}

	runTick()

	for {
		select {
		case <-ticker.C:
			runTick()
		case <-ctx.Done():
			slog.Info("retention scheduler stopped")
			return
		}
	}
}
