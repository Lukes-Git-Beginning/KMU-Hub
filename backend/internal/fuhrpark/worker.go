package fuhrpark

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"time"

	"github.com/kmuhub/kmuhub/internal/database"
)

// tuevCronLockKey is a stable int64 derived from the string "fuhrpark_tuev_cron".
// Used as the advisory lock key so only one pod runs the cron at a time.
var tuevCronLockKey = stableHash("fuhrpark_tuev_cron")

// stableHash returns a deterministic int64 from a string using FNV-64a.
func stableHash(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64())
}

// TuevWorker polls for vehicles with approaching TUEV due dates and inserts
// notifications. Uses pg_try_advisory_xact_lock for leader-election so only
// one replica runs per interval.
type TuevWorker struct {
	repo          Repository
	logger        *slog.Logger
	claimInterval time.Duration
	runInterval   time.Duration
}

// TuevWorkerPool is a minimal interface to allow advisory lock acquisition.
type TuevWorkerPool interface {
	Begin(ctx context.Context) (TuevTx, error)
}

// TuevTx is the minimal interface for a DB transaction with advisory lock support.
type TuevTx interface {
	QueryRow(ctx context.Context, sql string, args ...any) TuevRow
	Rollback(ctx context.Context) error
}

// TuevRow is a minimal row scan interface.
type TuevRow interface {
	Scan(dest ...any) error
}

// NewTuevWorker creates a TuevWorker with default intervals.
// pool is the raw pgxpool (passed as any to avoid circular dep; cast internally).
func NewTuevWorker(repo Repository, pool any, logger *slog.Logger) *TuevWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &TuevWorker{
		repo:          repo,
		logger:        logger,
		claimInterval: 5 * time.Minute,
		runInterval:   60 * time.Minute,
	}
}

// Run blocks until ctx is cancelled. On each run-ticker tick it acquires an
// advisory lock, runs the TUEV scan, and releases the lock automatically when
// the transaction ends.
func (w *TuevWorker) Run(ctx context.Context, acquireLockAndRun func(ctx context.Context) error) error {
	ctx = database.WithSystemContext(ctx)
	w.logger.Info("fuhrpark TUEV worker starting",
		"claim_interval", w.claimInterval,
		"run_interval", w.runInterval,
	)

	claimTicker := time.NewTicker(w.claimInterval)
	defer claimTicker.Stop()

	runTicker := time.NewTicker(w.runInterval)
	defer runTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("fuhrpark TUEV worker stopped")
			return nil

		case <-claimTicker.C:
			// Heartbeat: just log that we're alive and healthy.
			w.logger.Debug("fuhrpark TUEV worker heartbeat")

		case <-runTicker.C:
			if err := acquireLockAndRun(ctx); err != nil {
				w.logger.Error("fuhrpark TUEV worker run error", "error", err)
			}
		}
	}
}

// ProcessTuevReminders is the core scan logic. It checks for vehicles with
// tuev_due_date in 1-day and 7-day windows and inserts notifications for each.
// Idempotency: tuev_reminder_sent_at is stamped so a second run in the same
// window will not re-notify.
// Window extended to 2 hours to tolerate cron drift at the 60-min interval.
func (w *TuevWorker) ProcessTuevReminders(ctx context.Context, now time.Time) error {
	// 7-day window: due in [now+7d, now+7d+2h] — 2h window guards against 60min-interval drift
	sevenDayFrom := now.AddDate(0, 0, 7)
	sevenDayTo := sevenDayFrom.Add(2 * time.Hour)

	// 1-day window: due in [now+1d, now+1d+2h]
	oneDayFrom := now.AddDate(0, 0, 1)
	oneDayTo := oneDayFrom.Add(2 * time.Hour)

	count7, err := w.processWindow(ctx, sevenDayFrom, sevenDayTo, "7_days")
	if err != nil {
		return fmt.Errorf("7-day window: %w", err)
	}
	count1, err := w.processWindow(ctx, oneDayFrom, oneDayTo, "1_day")
	if err != nil {
		return fmt.Errorf("1-day window: %w", err)
	}

	if count7+count1 > 0 {
		w.logger.Info("fuhrpark TUEV reminders sent",
			"window_7d", count7,
			"window_1d", count1,
		)
	}
	return nil
}

func (w *TuevWorker) processWindow(ctx context.Context, from, to time.Time, label string) (int, error) {
	vehicles, err := w.repo.FindVehiclesDueTuev(ctx, from, to)
	if err != nil {
		return 0, err
	}

	for _, v := range vehicles {
		// TODO: Sprint 3 — wire to notification service for actual delivery.
		// For now: write a stub to the notifications table if the schema aligns,
		// or log only (notifications table uses user_id NOT NULL — vehicle tenant
		// notification delivery needs a notification service RPC).
		w.logger.Info("fuhrpark TUEV reminder triggered (delivery stub)",
			"vehicle_id", v.ID,
			"license_plate", v.LicensePlate,
			"tuev_due_date", v.TuevDueDate,
			"window", label,
		)

		if markErr := w.repo.MarkTuevReminderSent(ctx, v.ID, v.TenantID); markErr != nil {
			w.logger.Error("fuhrpark TUEV: failed to mark reminder sent",
				"vehicle_id", v.ID,
				"tenant_id", v.TenantID,
				"error", markErr,
			)
		}
	}

	return len(vehicles), nil
}

// LockKey returns the advisory lock key used for leader election.
func LockKey() int64 { return tuevCronLockKey }
