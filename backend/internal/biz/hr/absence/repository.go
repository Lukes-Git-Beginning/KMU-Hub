package absence

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// AbsenceRepository defines the interface for querying absence data.
type AbsenceRepository interface {
	GetAbsenceCalendar(ctx context.Context, filter AbsenceFilter) ([]*AbsenceEntry, error)
}

// AbsenceFilter contains filtering options for the absence calendar query.
type AbsenceFilter struct {
	TenantID   uuid.UUID
	StartDate  time.Time
	EndDate    time.Time
	Department string // Optional department filter
}

// AbsenceEntry represents a single absence entry for the calendar view.
type AbsenceEntry struct {
	EmployeeID         uuid.UUID                `json:"employee_id"`
	EmployeeName       string                   `json:"employee_name"`
	Department         string                   `json:"department"`
	LeaveTypeName      string                   `json:"leave_type_name"`
	LeaveTypeKey       string                   `json:"leave_type_key"`
	Color              string                   `json:"color"`
	StartDate          time.Time                `json:"start_date"`
	EndDate            time.Time                `json:"end_date"`
	IsHalfDayStart     bool                     `json:"is_half_day_start"`
	HalfDayPeriodStart *models.HalfDayPeriod    `json:"half_day_period_start,omitempty"`
	IsHalfDayEnd       bool                     `json:"is_half_day_end"`
	HalfDayPeriodEnd   *models.HalfDayPeriod    `json:"half_day_period_end,omitempty"`
	Status             models.LeaveRequestStatus `json:"status"`
}
