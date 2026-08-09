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

// SwapRequestFilter holds optional filters for swap request list queries.
type SwapRequestFilter struct {
	ShiftID *uuid.UUID
	Status  *SwapRequestStatus
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
	// JArbSchG: whether the employee's HR profile is flagged as a minor. False
	// for employees without a profile — the flag is opt-in, an absent profile
	// is not a claim about age.
	IsMinorEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (bool, error)

	// Templates
	CreateTemplate(ctx context.Context, t *ShiftTemplate) error
	UpdateTemplate(ctx context.Context, t *ShiftTemplate) error
	DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error
	GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*ShiftTemplate, error)
	ListTemplates(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*ShiftTemplate, int, error)

	// Stats
	GetStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*ShiftStats, error)

	// SwapRequests
	CreateSwapRequest(ctx context.Context, req *SwapRequest) error
	GetSwapRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*SwapRequest, error)
	ListSwapRequests(ctx context.Context, tenantID uuid.UUID, filter SwapRequestFilter, offset, limit int) ([]*SwapRequest, int, error)
	// UpdateSwapRequestStatus sets the status of a swap request and (on approval) atomically
	// swaps the two assignments within the same transaction.
	UpdateSwapRequestStatus(ctx context.Context, tenantID, requestID uuid.UUID, status SwapRequestStatus) error
	// SwapAssignmentsForRequest atomically swaps the employee of two assignments identified
	// by the swap request's assignment data. Called inside ApproveSwapRequest.
	SwapAssignmentsForRequest(ctx context.Context, req *SwapRequest) error
}
