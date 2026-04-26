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

	// Production Plans
	CreatePlan(ctx context.Context, plan *ProductionPlan) error
	UpdatePlan(ctx context.Context, plan *ProductionPlan) error
	GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*ProductionPlan, error)

	// Capacity
	GetCapacityOverview(ctx context.Context, tenantID uuid.UUID, machineID string, planID uuid.UUID) (*CapacityOverview, error)
}
