package vertraege

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/models"
	notificationevent "github.com/kmuhub/kmuhub/internal/notification/event"
)

// ReminderWorker polls for due reminders and marks them as sent.
// It emits a notification event for each due reminder via EventEmitter.
type ReminderWorker struct {
	repo           Repository
	emitter        EventEmitter // optional — no-op when nil
	logger         *slog.Logger
	pollInterval   time.Duration
	expiryInterval time.Duration
}

// NewReminderWorker creates a ReminderWorker with default intervals.
// emitter may be nil; in that case reminders are still claimed and marked sent,
// but no notification event is dispatched (useful in tests).
func NewReminderWorker(repo Repository, emitter EventEmitter, logger *slog.Logger) *ReminderWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReminderWorker{
		repo:           repo,
		emitter:        emitter,
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

// processReminders claims all due pending reminders, emits a notification event
// for each one, and logs the result. Idempotency is guaranteed by ClaimDueReminders,
// which atomically sets status=sent; a second run in the same window will find no
// pending rows and return an empty slice.
func (w *ReminderWorker) processReminders(ctx context.Context) error {
	reminders, err := w.repo.ClaimDueReminders(ctx)
	if err != nil {
		return err
	}

	for _, rem := range reminders {
		if w.emitter != nil {
			if emitErr := w.emitter.EmitReminderEvent(ctx, w.buildEventPayload(rem)); emitErr != nil {
				// Log and continue — reminder is already marked sent in the DB.
				w.logger.Error("vertraege reminder: failed to emit notification event",
					"reminder_id", rem.ID,
					"contract_id", rem.ContractID,
					"error", emitErr,
				)
			} else {
				w.logger.Info("vertraege reminder delivered",
					"reminder_id", rem.ID,
					"contract_id", rem.ContractID,
					"type", rem.ReminderType,
					"subject", rem.Subject,
				)
			}
		} else {
			w.logger.Info("vertraege reminder marked sent (no emitter configured)",
				"reminder_id", rem.ID,
				"contract_id", rem.ContractID,
				"type", rem.ReminderType,
				"subject", rem.Subject,
			)
		}
	}

	if len(reminders) > 0 {
		w.logger.Info("vertraege reminder worker: batch processed", "count", len(reminders))
	}
	return nil
}

// buildEventPayload constructs the notification event payload for a reminder.
// TargetUserIDs is empty here because reminders are tenant-scoped system alerts;
// the notification service will fan-out to users with the vertraege module enabled.
func (w *ReminderWorker) buildEventPayload(rem *ContractReminder) models.EventPayload {
	resourceID := rem.ContractID.String()
	deepLink := fmt.Sprintf("/vertraege/%s", rem.ContractID)

	return models.EventPayload{
		Type:       notificationevent.EventVertraegeReminderDue,
		TenantID:   rem.TenantID,
		Priority:   models.PriorityNormal,
		ModuleID:   notificationevent.ModuleVertraege,
		ResourceID: resourceID,
		Title:      rem.Subject,
		Body:       rem.Message,
		DeepLink:   deepLink,
		GroupKey:   fmt.Sprintf("vertraege.reminder.%s", rem.ContractID),
		Timestamp:  time.Now(),
	}
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
