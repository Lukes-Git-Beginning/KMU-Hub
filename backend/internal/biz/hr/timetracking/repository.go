package timetracking

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// WorkTimeRepository defines the interface for work time entry persistence.
type WorkTimeRepository interface {
	Create(ctx context.Context, entry *models.HRWorkTimeEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.HRWorkTimeEntry, error)
	GetActiveShift(ctx context.Context, employeeID uuid.UUID) (*models.HRWorkTimeEntry, error)
	Update(ctx context.Context, entry *models.HRWorkTimeEntry) error
	List(ctx context.Context, filter WorkTimeFilter) ([]*models.HRWorkTimeEntry, int, error)
	GetPreviousShiftEnd(ctx context.Context, employeeID uuid.UUID, before time.Time) (*time.Time, error)
	GetDailySummary(ctx context.Context, employeeID uuid.UUID, date time.Time) (*DailySummary, error)
	GetWeeklySummary(ctx context.Context, employeeID uuid.UUID, weekStart time.Time) (*WeeklySummary, error)
}

// BreakRepository defines the interface for break entry persistence.
type BreakRepository interface {
	Create(ctx context.Context, entry *models.HRBreakEntry) error
	GetActiveBreak(ctx context.Context, workTimeEntryID uuid.UUID) (*models.HRBreakEntry, error)
	Update(ctx context.Context, entry *models.HRBreakEntry) error
	ListByWorkTimeEntry(ctx context.Context, workTimeEntryID uuid.UUID) ([]*models.HRBreakEntry, error)
}

// WorkTimeFilter contains filtering options for listing work time entries.
type WorkTimeFilter struct {
	TenantID   uuid.UUID
	EmployeeID *uuid.UUID
	DateFrom   *time.Time
	DateTo     *time.Time
	Status     *string
	Page       int
	PerPage    int
}

// DailySummary contains aggregated work time data for a single date.
type DailySummary struct {
	Date               time.Time
	TotalWorkedMinutes int
	TotalBreakMinutes  int
	NetWorkMinutes     int
	OvertimeMinutes    int // net - contracted daily hours * 60
	EntryCount         int
}

// WeeklySummary contains aggregated work time data for a week (Monday-Sunday).
type WeeklySummary struct {
	WeekStart           time.Time
	Days                []DailySummary
	TotalWorkedMinutes  int
	TotalBreakMinutes   int
	NetWorkMinutes      int
	TotalOvertimeMinutes int
}

// CorrectionInput contains the data needed to submit a time correction.
type CorrectionInput struct {
	OriginalEntryID      uuid.UUID
	CorrectedClockIn     time.Time
	CorrectedClockOut    time.Time
	CorrectedBreakMinutes int
	Reason               string
}

// WorkTimeStatus is a convenience struct for the header quick-toggle button.
type WorkTimeStatus struct {
	IsClockedIn       bool
	IsOnBreak         bool
	CurrentShiftStart *time.Time
	CurrentBreakStart *time.Time
	TodayTotalMinutes int
	ArbZGSeverity     string
}
