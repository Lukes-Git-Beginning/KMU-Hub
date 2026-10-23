package gdpr

// Sweeper for export requests whose generation never finished. ApproveExport
// starts generation in a detached goroutine, and a failed run only flips the
// row back to "approved" -- nothing picks either state up again. A restart
// (deploy, OOM) between approval and the stored result would otherwise leave
// the Art. 15 request on "processing" for good, and ExecuteExportAsync itself
// refuses anything but "approved". Same shape as the retention scheduler:
// advisory lock on one held connection, plain per-tick function, thin loop.

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// exportSweepLockKey is distinct from retentionScheduleLockKey and
// idempotencyCleanupLockKey (0x49444D50).
const exportSweepLockKey int64 = 0x45585053 // "EXPS"

// ExportLease is how long a request may sit in approved/processing, measured
// from the last generation start (or the approval, if none ran), before the
// sweeper treats the run as dead. Generation takes seconds; half an hour is
// generous without letting a request wait long against the one-month deadline.
const ExportLease = 30 * time.Minute

type staleExport struct {
	id       uuid.UUID
	tenantID uuid.UUID
}

// claimStaleExports resets every request past the lease to "approved" and
// stamps the lease clock anew in the same statement, so a second replica (or
// the next tick) does not pick the same row while this one executes it.
// FOR UPDATE SKIP LOCKED keeps two concurrent claimers from sharing a row.
func claimStaleExports(ctx context.Context, pool *pgxpool.Pool, lease time.Duration) ([]staleExport, error) {
	rows, err := pool.Query(ctx,
		`UPDATE gdpr_export_requests
		 SET status = 'approved', processing_started_at = NOW()
		 WHERE id IN (
		     SELECT id FROM gdpr_export_requests
		     WHERE status IN ('approved', 'processing')
		       AND COALESCE(processing_started_at, reviewed_at, requested_at) < NOW() - make_interval(secs => $1)
		     ORDER BY requested_at
		     LIMIT 20
		     FOR UPDATE SKIP LOCKED)
		 RETURNING id, tenant_id`, lease.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]staleExport, 0)
	for rows.Next() {
		var e staleExport
		if scanErr := rows.Scan(&e.id, &e.tenantID); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// RunExportSweep claims stale requests under the sweep lock and re-runs each
// one in its own tenant context. The bool is false when another replica holds
// the lock (a normal skip). A request that fails again is logged and left on
// "approved" with a fresh lease, so it is retried one lease later.
//
// lean: no attempt cap -- a permanently failing export is retried every lease
// (30 min); add an attempts column + "failed" status when that gets noisy.
func RunExportSweep(ctx context.Context, pool *pgxpool.Pool, svc *Service, lease time.Duration) (bool, error) {
	return withScheduleLock(sysctx.With(ctx), pool, exportSweepLockKey, func(lockedCtx context.Context) error {
		stale, err := claimStaleExports(lockedCtx, pool, lease)
		if err != nil {
			return err
		}
		for _, e := range stale {
			// Generation runs as the tenant, not as system: the same context
			// ApproveExport hands its goroutine.
			tenantCtx := context.WithValue(ctx, middleware.TenantIDKey, e.tenantID.String())
			slog.Warn("gdpr: re-running stalled export", "request_id", e.id, "tenant_id", e.tenantID)
			if execErr := svc.ExecuteExportAsync(tenantCtx, e.id); execErr != nil {
				slog.Error("gdpr: stalled export retry failed", "request_id", e.id, "error", execErr)
			}
		}
		return nil
	})
}

// ExportSweeper runs RunExportSweep once after startupDelay, then every
// interval, until ctx is cancelled. Errors are logged, never propagated.
func ExportSweeper(ctx context.Context, pool *pgxpool.Pool, svc *Service, lease, startupDelay, interval time.Duration) {
	select {
	case <-time.After(startupDelay):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("gdpr export sweeper starting", "interval", interval.String(), "lease", lease.String())

	for {
		if _, err := RunExportSweep(ctx, pool, svc, lease); err != nil {
			slog.Error("gdpr export sweep errored", "error", err)
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			slog.Info("gdpr export sweeper stopped")
			return
		}
	}
}
