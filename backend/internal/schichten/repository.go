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

	// ArbZG: latest shift that ends before newStart for the given employee
	LatestShiftEndBeforeForEmployee(ctx context.Context, tenantID, employeeID uuid.UUID, before time.Time) (*time.Time, error)

	// Templates
	CreateTemplate(ctx context.Context, t *ShiftTemplate) error
	UpdateTemplate(ctx context.Context, t *ShiftTemplate) error
	DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error
	GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*ShiftTemplate, error)
	ListTemplates(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*ShiftTemplate, int, error)

	// Stats
	GetStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*ShiftStats, error)
}
