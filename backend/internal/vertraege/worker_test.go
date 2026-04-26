package vertraege

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	w := NewReminderWorker(repo, logger)

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
	w := NewReminderWorker(repo, logger)

	err := w.processReminders(context.Background())
	require.NoError(t, err)

	assert.Equal(t, ReminderStatusPending, repo.reminders[future.ID].Status)
}

func TestReminderWorker_ProcessExpiry_LogsCount(t *testing.T) {
	repo := newMockRepository()
	repo.expiredCount = 3 // simulate 3 contracts expired

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := NewReminderWorker(repo, logger)

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
	w := NewReminderWorker(repo, logger)

	err := w.processReminders(context.Background())
	require.NoError(t, err)

	// Status should remain Sent (not re-sent)
	assert.Equal(t, ReminderStatusSent, repo.reminders[sent.ID].Status)
}
