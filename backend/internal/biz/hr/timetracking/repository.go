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
	// AggregateWorkTimeForInvoice returns the total completed net_work_minutes and the
	// individual entry IDs for an employee in the given inclusive date range.
	// Only entries with status='completed' and a non-NULL net_work_minutes are considered.
	AggregateWorkTimeForInvoice(ctx context.Context, tenantID, employeeID uuid.UUID, from, to time.Time) (totalMinutes int, entryIDs []string, err error)
	// GetProjectBreakdown returns per-project aggregated net_work_minutes for the given employee and date range.
	GetProjectBreakdown(ctx context.Context, tenantID, employeeID uuid.UUID, dateFrom, dateTo time.Time) ([]ProjectBreakdown, error)
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

// TimeCategoryRepository defines CRUD for hr_time_categories.
type TimeCategoryRepository interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeCategory, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.HRTimeCategory, error)
	Create(ctx context.Context, cat *models.HRTimeCategory) error
	Update(ctx context.Context, cat *models.HRTimeCategory) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TimeTemplateRepository defines CRUD for hr_time_templates.
type TimeTemplateRepository interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeTemplate, error)
	Create(ctx context.Context, t *models.HRTimeTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TimeProjectRepository defines CRUD for hr_time_projects.
type TimeProjectRepository interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*models.HRTimeProject, error)
	Create(ctx context.Context, p *models.HRTimeProject) error
}

// WeekApprovalRepository defines lifecycle for hr_week_approvals.
type WeekApprovalRepository interface {
	GetByEmployeeWeek(ctx context.Context, tenantID, employeeID uuid.UUID, weekStart time.Time) (*models.HRWeekApproval, error)
	Upsert(ctx context.Context, w *models.HRWeekApproval) error
	ListByWeek(ctx context.Context, tenantID uuid.UUID, weekStart time.Time) ([]*models.HRWeekApproval, error)
}

// ManualEntryInput contains data for creating a manual work time entry.
type ManualEntryInput struct {
	TenantID        uuid.UUID
	EmployeeID      uuid.UUID
	ClockIn         time.Time
	ClockOut        time.Time
	BreakMinutes    int
	ProjectID       *uuid.UUID
	CategoryID      *uuid.UUID
	Activity        string
	Note            string
	Billable        bool
	LocationLat     *float64
	LocationLng     *float64
	LocationAddress string
	IdempotencyKey  string
}

// TeamTimeEntry holds aggregated time info for one employee for a week.
type TeamTimeEntry struct {
	EmployeeID    uuid.UUID
	Name          string
	Department    string
	WeekMinutes   int
	TargetMinutes int
	OvertimeMinutes int
	ClockedIn     bool
	WeekStatus    string
}

// DayTrendEntry is one day of analytics data.
type DayTrendEntry struct {
	Date          time.Time
	NetMinutes    int
	TargetMinutes int
}

// ProjectBreakdown is time grouped by project.
type ProjectBreakdown struct {
	ProjectID   uuid.UUID
	ProjectName string
	Minutes     int
}

// TimeAnalytics contains aggregated analytics for a period.
type TimeAnalytics struct {
	TotalMinutes    int
	TargetMinutes   int
	OvertimeMinutes int
	AvgDailyMinutes int
	DayTrend        []DayTrendEntry
	ByProject       []ProjectBreakdown
}
