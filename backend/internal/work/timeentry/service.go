package timeentry

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles time entry business logic
type Service struct {
	repo Repository
}

// NewService creates a new time entry service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// StartTimer starts a new timer for a user on a task.
// If the user already has a running timer on another task, it stops that timer first.
func (s *Service) StartTimer(ctx context.Context, taskID, userID, tenantID uuid.UUID) (*models.TimeEntry, *models.TimeEntry, error) {
	// Check for and stop any existing active timer
	var stoppedEntry *models.TimeEntry
	existing, err := s.repo.GetActiveTimer(ctx, userID, tenantID)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		stopped, stopErr := s.repo.StopActiveTimer(ctx, userID, tenantID)
		if stopErr != nil {
			return nil, nil, stopErr
		}
		stoppedEntry = stopped
		slog.Info("auto-stopped previous timer",
			"previous_task_id", existing.TaskID,
			"user_id", userID,
		)
	}

	now := time.Now()
	entry := &models.TimeEntry{
		ID:        uuid.New(),
		TenantID:  tenantID,
		TaskID:    taskID,
		UserID:    userID,
		StartedAt: now,
		IsManual:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if createErr := s.repo.Create(ctx, entry); createErr != nil {
		return nil, nil, createErr
	}

	slog.Info("timer started",
		"time_entry_id", entry.ID,
		"task_id", taskID,
		"user_id", userID,
	)

	return entry, stoppedEntry, nil
}

// StopTimer stops the active timer for a user
func (s *Service) StopTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.TimeEntry, error) {
	stopped, err := s.repo.StopActiveTimer(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	if stopped == nil {
		return nil, ErrNoActiveTimer
	}

	slog.Info("timer stopped",
		"time_entry_id", stopped.ID,
		"task_id", stopped.TaskID,
		"user_id", userID,
		"duration_seconds", stopped.DurationSeconds,
	)

	return stopped, nil
}

// GetActiveTimer returns the currently running timer for a user, or nil
func (s *Service) GetActiveTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.ActiveTimer, error) {
	return s.repo.GetActiveTimer(ctx, userID, tenantID)
}

// ManualEntryInput contains the data for creating a manual time entry
type ManualEntryInput struct {
	TenantID        uuid.UUID
	TaskID          uuid.UUID
	UserID          uuid.UUID
	StartedAt       time.Time
	DurationSeconds int
	Description     *string
}

// AddManualEntry creates a manual time entry
func (s *Service) AddManualEntry(ctx context.Context, input ManualEntryInput) (*models.TimeEntry, error) {
	if input.DurationSeconds <= 0 {
		return nil, ErrInvalidDuration
	}

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if len(desc) > 2000 {
			return nil, ErrDescriptionTooLong
		}
		input.Description = &desc
	}

	now := time.Now()
	endedAt := input.StartedAt.Add(time.Duration(input.DurationSeconds) * time.Second)
	durationSec := input.DurationSeconds

	entry := &models.TimeEntry{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		TaskID:          input.TaskID,
		UserID:          input.UserID,
		StartedAt:       input.StartedAt,
		EndedAt:         &endedAt,
		DurationSeconds: &durationSec,
		Description:     input.Description,
		IsManual:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return nil, err
	}

	slog.Info("manual time entry added",
		"time_entry_id", entry.ID,
		"task_id", input.TaskID,
		"user_id", input.UserID,
		"duration_seconds", input.DurationSeconds,
	)

	return entry, nil
}

// UpdateEntryInput contains the data for updating a time entry
type UpdateEntryInput struct {
	StartedAt       *time.Time
	DurationSeconds *int
	Description     *string
}

// UpdateEntry updates an existing time entry
func (s *Service) UpdateEntry(ctx context.Context, entryID, actorID, tenantID uuid.UUID, input UpdateEntryInput) (*models.TimeEntryWithUser, error) {
	existing, err := s.repo.GetByID(ctx, entryID, tenantID)
	if err != nil {
		return nil, err
	}

	// Only the user who created the entry can edit it
	if existing.UserID != actorID {
		return nil, ErrCannotEditOthers
	}

	entry := &existing.TimeEntry

	if input.StartedAt != nil {
		entry.StartedAt = *input.StartedAt
	}

	if input.DurationSeconds != nil {
		if *input.DurationSeconds <= 0 {
			return nil, ErrInvalidDuration
		}
		dur := *input.DurationSeconds
		entry.DurationSeconds = &dur
		endedAt := entry.StartedAt.Add(time.Duration(dur) * time.Second)
		entry.EndedAt = &endedAt
	}

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if len(desc) > 2000 {
			return nil, ErrDescriptionTooLong
		}
		entry.Description = &desc
	}

	entry.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, entry); updateErr != nil {
		return nil, updateErr
	}

	return s.repo.GetByID(ctx, entryID, tenantID)
}

// DeleteEntry deletes a time entry
func (s *Service) DeleteEntry(ctx context.Context, entryID, actorID, tenantID uuid.UUID, isAdmin bool) error {
	existing, err := s.repo.GetByID(ctx, entryID, tenantID)
	if err != nil {
		return err
	}

	// Only the owner or admin can delete
	if existing.UserID != actorID && !isAdmin {
		return ErrCannotDeleteOthers
	}

	return s.repo.Delete(ctx, entryID, tenantID)
}

// ListByTask returns time entries for a task
func (s *Service) ListByTask(ctx context.Context, taskID, tenantID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error) {
	return s.repo.ListByTask(ctx, taskID, tenantID, page, pageSize)
}

// GetTaskTimeSummary returns the aggregate time summary for a task
func (s *Service) GetTaskTimeSummary(ctx context.Context, taskID, tenantID uuid.UUID) (*models.TimeEntrySummary, error) {
	return s.repo.GetTaskTimeSummary(ctx, taskID, tenantID)
}

// ListBillable returns completed time entries across the tenant for invoicing
// (Stunden -> Rechnung).
func (s *Service) ListBillable(ctx context.Context, tenantID uuid.UUID) ([]models.BillableTimeEntry, error) {
	return s.repo.ListBillable(ctx, tenantID)
}
