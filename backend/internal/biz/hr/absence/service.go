// Package absence provides the absence calendar query service.
// It queries approved leave requests and returns them formatted for
// a Gantt-style calendar view, with configurable reason visibility
// based on HR company settings.
package absence

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
)

// Service handles absence calendar queries.
type Service struct {
	repo         AbsenceRepository
	settingsRepo leave.HRSettingsRepository
}

// NewService creates a new absence service.
func NewService(repo AbsenceRepository, settingsRepo leave.HRSettingsRepository) *Service {
	return &Service{
		repo:         repo,
		settingsRepo: settingsRepo,
	}
}

// GetAbsenceCalendar returns absence entries for the given filter.
// If ShowAbsenceReason is false in company settings, leave type names
// are replaced with "Abwesend" and colors with a neutral gray.
func (s *Service) GetAbsenceCalendar(ctx context.Context, tenantID uuid.UUID, filter AbsenceFilter) ([]*AbsenceEntry, error) {
	if filter.EndDate.Before(filter.StartDate) {
		return nil, ErrInvalidDateRange
	}

	filter.TenantID = tenantID

	entries, err := s.repo.GetAbsenceCalendar(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Check if we should mask absence reasons
	settings, settingsErr := s.settingsRepo.GetByTenant(ctx, tenantID)
	if settingsErr != nil {
		slog.Warn("failed to load HR settings for absence calendar, showing reasons by default",
			"tenant_id", tenantID,
			"error", settingsErr,
		)
		return entries, nil
	}

	if !settings.ShowAbsenceReason {
		for _, entry := range entries {
			entry.LeaveTypeName = "Abwesend"
			entry.LeaveTypeKey = "absent"
			entry.Color = "#9ca3af"
		}
	}

	return entries, nil
}
