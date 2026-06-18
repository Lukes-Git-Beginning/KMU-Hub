package produktion

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListOrdersFilter contains filtering options for listing production orders.
type ListOrdersFilter struct {
	Status    *OrderStatus
	Priority  *int
	DateFrom  *time.Time
	DateTo    *time.Time
}

// ListBookingsFilter contains filtering options for listing machine bookings.
type ListBookingsFilter struct {
	MachineID         *string
	DateFrom          *time.Time
	DateTo            *time.Time
	ProductionOrderID *uuid.UUID
}

// Repository defines the interface for produktion persistence.
type Repository interface {
	// Production Orders
	CreateOrder(ctx context.Context, order *ProductionOrder) error
	UpdateOrder(ctx context.Context, order *ProductionOrder) error
	GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*ProductionOrder, error)
	ListOrders(ctx context.Context, tenantID uuid.UUID, filter ListOrdersFilter, offset, limit int) ([]*ProductionOrder, int, error)
	DeleteOrder(ctx context.Context, tenantID, orderID uuid.UUID) error
	OrderNumberExists(ctx context.Context, tenantID uuid.UUID, orderNumber string, excludeID *uuid.UUID) (bool, error)

	// Machine Bookings
	CreateBooking(ctx context.Context, booking *MachineBooking) error
	UpdateBooking(ctx context.Context, booking *MachineBooking) error
	GetBooking(ctx context.Context, tenantID, bookingID uuid.UUID) (*MachineBooking, error)
	ListBookings(ctx context.Context, tenantID uuid.UUID, filter ListBookingsFilter, offset, limit int) ([]*MachineBooking, int, error)
	DeleteBooking(ctx context.Context, tenantID, bookingID uuid.UUID) error
	// FindConflictingBooking checks for an overlapping active booking for the same machine.
	// excludeID can be set to exclude a specific booking (for updates). Returns nil if no conflict.
	FindConflictingBooking(ctx context.Context, tenantID uuid.UUID, machineID string, startsAt, endsAt time.Time, excludeID *uuid.UUID) (*uuid.UUID, error)

	// CreateBookingWithLock atomically checks for conflicts and inserts a new booking under a
	// pg_advisory_xact_lock keyed by tenant_id||':'||machine_id. This serialises concurrent
	// create-booking requests for the same (tenant, machine) pair and prevents double-booking.
	//
	// Return values:
	//   conflictID non-nil, err nil  → a conflicting booking exists; caller should return ErrBookingConflict
	//   conflictID nil, err nil      → booking created successfully
	//   conflictID nil, err non-nil  → unexpected DB error
	CreateBookingWithLock(ctx context.Context, booking *MachineBooking) (conflictID *uuid.UUID, err error)

	// Production Plans
	CreatePlan(ctx context.Context, plan *ProductionPlan) error
	UpdatePlan(ctx context.Context, plan *ProductionPlan) error
	GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*ProductionPlan, error)

	// Capacity
	GetCapacityOverview(ctx context.Context, tenantID uuid.UUID, machineID string, planID uuid.UUID) (*CapacityOverview, error)

	// BOMs
	CreateBOM(ctx context.Context, bom *BOM) error
	UpdateBOM(ctx context.Context, bom *BOM) error
	GetBOM(ctx context.Context, tenantID, bomID uuid.UUID) (*BOM, error)
	ListBOMs(ctx context.Context, tenantID uuid.UUID, activeOnly *bool, offset, limit int) ([]*BOM, int, error)
	DeleteBOM(ctx context.Context, tenantID, bomID uuid.UUID) error

	// Work Steps
	CreateWorkStep(ctx context.Context, step *WorkStep) error
	UpdateWorkStep(ctx context.Context, step *WorkStep) error
	GetWorkStep(ctx context.Context, tenantID, stepID uuid.UUID) (*WorkStep, error)
	ListWorkSteps(ctx context.Context, tenantID, orderID uuid.UUID) ([]*WorkStep, error)
	DeleteWorkStep(ctx context.Context, tenantID, stepID uuid.UUID) error

	// Machines
	CreateMachine(ctx context.Context, machine *Machine) error
	UpdateMachine(ctx context.Context, machine *Machine) error
	GetMachine(ctx context.Context, tenantID, machineID uuid.UUID) (*Machine, error)
	ListMachines(ctx context.Context, tenantID uuid.UUID, status *MachineStatus, offset, limit int) ([]*Machine, int, error)
	DeleteMachine(ctx context.Context, tenantID, machineID uuid.UUID) error

	// Quality Checks
	CreateQualityCheck(ctx context.Context, check *QualityCheck) error
	GetQualityCheck(ctx context.Context, tenantID, checkID uuid.UUID) (*QualityCheck, error)
	ListQualityChecks(ctx context.Context, tenantID uuid.UUID, orderID *uuid.UUID, offset, limit int) ([]*QualityCheck, int, error)
}
