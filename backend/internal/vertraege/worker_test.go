package vertraege

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// mockEventEmitter
// ============================================================================

type mockEventEmitter struct {
	calls atomic.Int32
	last  models.EventPayload
}

func (m *mockEventEmitter) EmitReminderEvent(_ context.Context, p models.EventPayload) error {
	m.calls.Add(1)
	m.last = p
	return nil
}

// ============================================================================
// ReminderWorker Tests
// ============================================================================

func TestReminderWorker_ProcessReminders_MarksDueAsSent(t *testing.T) {
	repo := newMockRepository()

	tenantID := uuid.New()
	contractID := uuid.New()

	// One reminder that is past due
	pastDue := &ContractReminder{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ContractID:   contractID,
		RemindAt:     time.Now().Add(-1 * time.Hour),
		ReminderType: ReminderTypeRenewal,
		Subject:      "Renewal",
		Status:       ReminderStatusPending,
		CreatedAt:    time.Now(),
	}
	// One reminder that is in the future
	future := &ContractReminder{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ContractID:   contractID,
		RemindAt:     time.Now().Add(24 * time.Hour),
		ReminderType: ReminderTypeExpiry,
		Subject:      "Expiry",
		Status:       ReminderStatusPending,
		CreatedAt:    time.Now(),
	}
	repo.reminders[pastDue.ID] = pastDue
	repo.reminders[future.ID] = future

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := NewReminderWorker(repo, nil, logger)

	err := w.processReminders(context.Background())
	require.NoError(t, err)

	assert.Equal(t, ReminderStatusSent, repo.reminders[pastDue.ID].Status)
	assert.NotNil(t, repo.reminders[pastDue.ID].SentAt)
	assert.Equal(t, ReminderStatusPending, repo.reminders[future.ID].Status)
	assert.Nil(t, repo.reminders[future.ID].SentAt)
}

func TestReminderWorker_ProcessReminders_NoOp_WhenNoneDue(t *testing.T) {
	repo := newMockRepository()

	tenantID := uuid.New()
	contractID := uuid.New()
	future := &ContractReminder{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ContractID: contractID,
		RemindAt:   time.Now().Add(7 * 24 * time.Hour),
		Status:     ReminderStatusPending,
		CreatedAt:  time.Now(),
	}
	repo.reminders[future.ID] = future

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := NewReminderWorker(repo, nil, logger)

	err := w.processReminders(context.Background())
	require.NoError(t, err)

	assert.Equal(t, ReminderStatusPending, repo.reminders[future.ID].Status)
}

func TestReminderWorker_ProcessExpiry_LogsCount(t *testing.T) {
	repo := newMockRepository()
	repo.expiredCount = 3 // simulate 3 contracts expired

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := NewReminderWorker(repo, nil, logger)

	err := w.processExpiry(context.Background())
	require.NoError(t, err)
}

func TestReminderWorker_Run_StopsOnContextCancel(t *testing.T) {
	repo := newMockRepository()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := &ReminderWorker{
		repo:           repo,
		logger:         logger,
		pollInterval:   100 * time.Millisecond,
		expiryInterval: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)
	assert.NoError(t, err)
}

func TestReminderWorker_AlreadySentReminder_NotReprocessed(t *testing.T) {
	repo := newMockRepository()

	tenantID := uuid.New()
	contractID := uuid.New()
	sent := &ContractReminder{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ContractID: contractID,
		RemindAt:   time.Now().Add(-2 * time.Hour),
		Status:     ReminderStatusSent, // already sent
		CreatedAt:  time.Now(),
	}
	repo.reminders[sent.ID] = sent

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := NewReminderWorker(repo, nil, logger)

	err := w.processReminders(context.Background())
	require.NoError(t, err)

	// Status should remain Sent (not re-sent)
	assert.Equal(t, ReminderStatusSent, repo.reminders[sent.ID].Status)
}

// TestReminderWorker_EmitsEventForDueReminder verifies that exactly one
// notification event is emitted per due reminder, with correct field values.
func TestReminderWorker_EmitsEventForDueReminder(t *testing.T) {
	repo := newMockRepository()
	emitter := &mockEventEmitter{}

	tenantID := uuid.New()
	contractID := uuid.New()
	reminderID := uuid.New()

	due := &ContractReminder{
		ID:           reminderID,
		TenantID:     tenantID,
		ContractID:   contractID,
		RemindAt:     time.Now().Add(-30 * time.Minute),
		ReminderType: ReminderTypeRenewal,
		Subject:      "Renewal fällig",
		Message:      "Bitte Vertrag verlängern.",
		Status:       ReminderStatusPending,
		CreatedAt:    time.Now(),
	}
	repo.reminders[due.ID] = due

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := NewReminderWorker(repo, emitter, logger)

	err := w.processReminders(context.Background())
	require.NoError(t, err)

	// Exactly one event emitted
	assert.Equal(t, int32(1), emitter.calls.Load(), "expected exactly 1 emit call")

	// Payload fields
	p := emitter.last
	assert.Equal(t, "vertraege.reminder.due", p.Type)
	assert.Equal(t, tenantID, p.TenantID)
	assert.Equal(t, contractID.String(), p.ResourceID)
	assert.Equal(t, "Renewal fällig", p.Title)
	assert.Equal(t, "Bitte Vertrag verlängern.", p.Body)
	assert.Contains(t, p.DeepLink, contractID.String())
	assert.Equal(t, "vertraege", p.ModuleID)

	// Reminder is marked sent (idempotency: second run will not re-emit)
	assert.Equal(t, ReminderStatusSent, repo.reminders[due.ID].Status)

	// Second run — emitter must NOT be called again
	err = w.processReminders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(1), emitter.calls.Load(), "second run must not re-emit already-sent reminder")
}
