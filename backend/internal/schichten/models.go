package schichten

import (
	"time"

	"github.com/google/uuid"
)

// ShiftStatus represents the publication lifecycle of a shift.
type ShiftStatus string

const (
	ShiftStatusDraft     ShiftStatus = "draft"
	ShiftStatusPublished ShiftStatus = "published"
)

// Shift represents a single work shift in the schedule.
type Shift struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    uuid.UUID   `json:"tenant_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time"`
	Status      ShiftStatus `json:"status"`
	Location    *string     `json:"location,omitempty"`
	// Capacity is the maximum number of employees that can be assigned to this shift.
	// nil means unlimited.
	Capacity  *int       `json:"capacity,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ShiftAssignment links an employee to a shift.
type ShiftAssignment struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	ShiftID    uuid.UUID  `json:"shift_id"`
	EmployeeID uuid.UUID  `json:"employee_id"`
	AssignedAt time.Time  `json:"assigned_at"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"`
}

// ShiftTemplate defines a repeating weekly pattern for shift generation.
type ShiftTemplate struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	DayOfWeek       int       `json:"day_of_week"`       // 0=Sunday … 6=Saturday
	StartHour       int       `json:"start_hour"`        // 0–23
	StartMinute     int       `json:"start_minute"`      // 0–59
	DurationMinutes int       `json:"duration_minutes"`  // >0
	Location        *string   `json:"location,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ShiftStats is an aggregated summary of shift data for a tenant.
type ShiftStats struct {
	TotalShifts       int
	PublishedShifts   int
	DraftShifts       int
	TotalAssignments  int
	UniqueEmployees   int
}
