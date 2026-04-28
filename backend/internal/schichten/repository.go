package schichten

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListShiftsFilter holds optional filters for shift list queries.
type ListShiftsFilter struct {
	Status *ShiftStatus
	From   *time.Time
	To     *time.Time
}

// Repository defines the persistence interface for the schichten module.
type Repository interface {
	// Shifts
	CreateShift(ctx context.Context, shift *Shift) error
	UpdateShift(ctx context.Context, shift *Shift) error
	DeleteShift(ctx context.Context, tenantID, shiftID uuid.UUID) error
	GetShift(ctx context.Context, tenantID, shiftID uuid.UUID) (*Shift, error)
	ListShifts(ctx context.Context, tenantID uuid.UUID, filter ListShiftsFilter, offset, limit int) ([]*Shift, int, error)
	PublishShifts(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (int64, error)

	// Assignments
	CreateAssignment(ctx context.Context, a *ShiftAssignment) error
	DeleteAssignment(ctx context.Context, tenantID, shiftID, employeeID uuid.UUID) error
	GetAssignment(ctx context.Context, tenantID, shiftID, employeeID uuid.UUID) (*ShiftAssignment, error)
	ListAssignments(ctx context.Context, tenantID, shiftID uuid.UUID) ([]*ShiftAssignment, error)
	// CountAssignments returns the number of assignments for a shift (capacity guard).
	CountAssignments(ctx context.Context, tenantID, shiftID uuid.UUID) (int, error)

	// ArbZG: latest shift that ends strictly before newStart for the given employee (bidirectional check)
	LatestShiftEndBeforeForEmployee(ctx context.Context, tenantID, employeeID uuid.UUID, before time.Time) (*time.Time, error)
	// ArbZG: earliest shift that starts strictly after newEnd for the given employee (bidirectional check)
	EarliestShiftStartAfterForEmployee(ctx context.Context, tenantID, employeeID uuid.UUID, after time.Time) (*time.Time, error)
	// ShiftExistsForTemplate checks for an identical shift (idempotency for ApplyTemplate)
	ShiftExistsForTemplate(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time, title string) (bool, error)

	// Templates
	CreateTemplate(ctx context.Context, t *ShiftTemplate) error
	UpdateTemplate(ctx context.Context, t *ShiftTemplate) error
	DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error
	GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*ShiftTemplate, error)
	ListTemplates(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*ShiftTemplate, int, error)

	// Stats
	GetStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*ShiftStats, error)
}
