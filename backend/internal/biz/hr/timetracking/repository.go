package timetracking

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// balanceStatuses are the entry states that count towards a daily or weekly
// balance. 'correction_pending' does not (it is a proposal) and neither does
// 'superseded' (an approved correction already replaced it) — leaving the original
// in here is what made a corrected day count twice.
//
// The set lives here once and travels into every aggregate as a query parameter,
// so no single query can drift away from the definition.
var balanceStatuses = []string{
	string(models.WorkTimeStatusActive),
	string(models.WorkTimeStatusCompleted),
	string(models.WorkTimeStatusCorrectionApproved),
}

// billableStatuses are the states an invoice or a project breakdown may bill.
// Same set minus 'active': a running shift has no final duration yet.
var billableStatuses = []string{
	string(models.WorkTimeStatusCompleted),
	string(models.WorkTimeStatusCorrectionApproved),
}

// WorkTimeRepository defines the interface for work time entry persistence.
//
// Every method scopes to a tenant. Where no tenantID parameter appears the entry
// itself carries one (Create, Update, ApproveCorrection) or the filter does
// (List). The RLS policy on hr_work_time_entries (migration 000123) is the second
// line, not the first: a write with `WHERE id = $1` alone would reach another
// tenant's row in any context where app.tenant_id is unset.
type WorkTimeRepository interface {
	Create(ctx context.Context, entry *models.HRWorkTimeEntry) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.HRWorkTimeEntry, error)
	GetActiveShift(ctx context.Context, tenantID, employeeID uuid.UUID) (*models.HRWorkTimeEntry, error)
	Update(ctx context.Context, entry *models.HRWorkTimeEntry) error
	// ApproveCorrection writes the approved correction and marks the entry it
	// replaces as superseded in one transaction. Split into two writes, a failure
	// between them leaves the day either double-counted or missing entirely.
	// originalID may be nil for a correction without a resolvable original.
	ApproveCorrection(ctx context.Context, correction *models.HRWorkTimeEntry, originalID *uuid.UUID) error
	List(ctx context.Context, filter WorkTimeFilter) ([]*models.HRWorkTimeEntry, int, error)
	GetPreviousShiftEnd(ctx context.Context, tenantID, employeeID uuid.UUID, before time.Time) (*time.Time, error)
	GetDailySummary(ctx context.Context, tenantID, employeeID uuid.UUID, date time.Time) (*DailySummary, error)
	// GetDailySummaryRange returns one DailySummary per day for [start, start+numDays)
	// in a single aggregated query (replaces numDays per-day GetDailySummary calls).
	GetDailySummaryRange(ctx context.Context, tenantID, employeeID uuid.UUID, start time.Time, numDays int) ([]DailySummary, error)
	GetWeeklySummary(ctx context.Context, tenantID, employeeID uuid.UUID, weekStart time.Time) (*WeeklySummary, error)
	// GetWeeklySummaryBatch returns the weekly summary for each employee in a single
	// aggregated query, replacing N per-employee GetWeeklySummary calls.
	GetWeeklySummaryBatch(ctx context.Context, tenantID uuid.UUID, employeeIDs []uuid.UUID, weekStart time.Time) (map[uuid.UUID]*WeeklySummary, error)
	// GetActiveShiftEmployeeIDs returns the set of employee IDs that currently have an
	// active shift, in a single query, replacing N per-employee GetActiveShift calls.
	GetActiveShiftEmployeeIDs(ctx context.Context, tenantID uuid.UUID, employeeIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// ReserveWorkTimeForInvoice locks and marks billed, in one transaction, the
	// billable entries (see billableStatuses) with a non-NULL net_work_minutes
	// for an employee in the given inclusive date range that are not already
	// billed. Returns the total completed net_work_minutes and the individual
	// entry IDs. A second call for the same employee/period sees zero — the
	// double-billing guard for CreateInvoiceFromTimeEntries.
	ReserveWorkTimeForInvoice(ctx context.Context, tenantID, employeeID uuid.UUID, from, to time.Time) (totalMinutes int, entryIDs []string, err error)
	// ConfirmInvoiceReservation records which invoice claimed a reservation
	// (best-effort traceability; the double-billing guard is already enforced).
	ConfirmInvoiceReservation(ctx context.Context, tenantID, invoiceID uuid.UUID, entryIDs []string) error
	// ReleaseInvoiceReservation undoes a reservation whose invoice was never
	// created, so the entries become billable again.
	ReleaseInvoiceReservation(ctx context.Context, tenantID uuid.UUID, entryIDs []string) error
	// GetProjectBreakdown returns per-project aggregated net_work_minutes for the given employee and date range.
	GetProjectBreakdown(ctx context.Context, tenantID, employeeID uuid.UUID, dateFrom, dateTo time.Time) ([]ProjectBreakdown, error)
}

// BreakRepository defines the interface for break entry persistence.
// Every method is tenant-scoped: Create and Update read the tenant off the
// entry, the read paths take it as a parameter. work_time_entry_id alone would
// only be scoped as long as the caller resolved the shift correctly first.
type BreakRepository interface {
	Create(ctx context.Context, entry *models.HRBreakEntry) error
	GetActiveBreak(ctx context.Context, tenantID, workTimeEntryID uuid.UUID) (*models.HRBreakEntry, error)
	Update(ctx context.Context, entry *models.HRBreakEntry) error
	ListByWorkTimeEntry(ctx context.Context, tenantID, workTimeEntryID uuid.UUID) ([]*models.HRBreakEntry, error)
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
	// lean: passed through from the HTTP handler but never read here —
	// deduplication happens one layer up in middleware.Idempotency
	// (route_hr.go's authWithIdempotency chain), not in this service. Upgrade
	// trigger: a caller of CreateManualEntry that bypasses the gateway's
	// idempotency middleware (e.g. a future internal/worker path).
	IdempotencyKey string
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
