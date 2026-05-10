package vertraege

import (
	"context"
	"log/slog"
	"time"

	"github.com/kmuhub/kmuhub/internal/database"
)

// ReminderWorker polls for due reminders and marks them as sent.
// Actual delivery (email, push notification) is a Sprint 3 item.
//
// TODO: Sprint 3 — wire to notification/email service for actual delivery.
type ReminderWorker struct {
	repo         Repository
	logger       *slog.Logger
	pollInterval time.Duration
	expiryInterval time.Duration
}

// NewReminderWorker creates a ReminderWorker with default intervals.
func NewReminderWorker(repo Repository, logger *slog.Logger) *ReminderWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReminderWorker{
		repo:           repo,
		logger:         logger,
		pollInterval:   5 * time.Minute,
		expiryInterval: 60 * time.Minute,
	}
}

// Run blocks until ctx is cancelled, processing reminders and contract expiry on each tick.
func (w *ReminderWorker) Run(ctx context.Context) error {
	ctx = database.WithSystemContext(ctx)
	w.logger.Info("vertraege reminder worker starting",
		"poll_interval", w.pollInterval,
		"expiry_interval", w.expiryInterval,
	)

	reminderTicker := time.NewTicker(w.pollInterval)
	defer reminderTicker.Stop()

	expiryTicker := time.NewTicker(w.expiryInterval)
	defer expiryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("vertraege reminder worker stopped")
			return nil

		case <-reminderTicker.C:
			if err := w.processReminders(ctx); err != nil {
				w.logger.Error("vertraege reminder worker: processReminders error", "error", err)
				// Continue — do not abort on transient errors.
			}

		case <-expiryTicker.C:
			if err := w.processExpiry(ctx); err != nil {
				w.logger.Error("vertraege reminder worker: processExpiry error", "error", err)
			}
		}
	}
}

// processReminders claims all due pending reminders and marks them sent.
// Actual delivery is wired in Sprint 3.
func (w *ReminderWorker) processReminders(ctx context.Context) error {
	reminders, err := w.repo.ClaimDueReminders(ctx)
	if err != nil {
		return err
	}

	for _, rem := range reminders {
		// TODO: Sprint 3 — wire to notification/email service for actual delivery.
		w.logger.Info("vertraege reminder marked sent (delivery not yet wired)",
			"reminder_id", rem.ID,
			"contract_id", rem.ContractID,
			"type", rem.ReminderType,
			"subject", rem.Subject,
		)
	}

	if len(reminders) > 0 {
		w.logger.Info("vertraege reminder worker: batch processed", "count", len(reminders))
	}
	return nil
}

// processExpiry updates contracts whose end date has passed to status=expired.
func (w *ReminderWorker) processExpiry(ctx context.Context) error {
	n, err := w.repo.ExpireContracts(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		w.logger.Info("vertraege: contracts auto-expired", "count", n)
	}
	return nil
}
