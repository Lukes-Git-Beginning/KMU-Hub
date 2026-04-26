package produktion

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents the lifecycle state of a production order.
type OrderStatus string

const (
	OrderStatusPlanned    OrderStatus = "planned"
	OrderStatusInProgress OrderStatus = "in_progress"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// BookingStatus represents the lifecycle state of a machine booking.
type BookingStatus string

const (
	BookingStatusBooked    BookingStatus = "booked"
	BookingStatusInUse     BookingStatus = "in_use"
	BookingStatusCompleted BookingStatus = "completed"
	BookingStatusCancelled BookingStatus = "cancelled"
)

// PlanStatus represents the lifecycle state of a production plan.
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusActive    PlanStatus = "active"
	PlanStatusCompleted PlanStatus = "completed"
)

// ProductionOrder represents a manufacturing/production job.
type ProductionOrder struct {
	ID           uuid.UUID   `json:"id"`
	TenantID     uuid.UUID   `json:"tenant_id"`
	OrderNumber  string      `json:"order_number"`
	ProductName  string      `json:"product_name"`
	Quantity     int         `json:"quantity"`
	Status       OrderStatus `json:"status"`
	PlannedStart time.Time   `json:"planned_start"`
	PlannedEnd   time.Time   `json:"planned_end"`
	ActualStart  *time.Time  `json:"actual_start,omitempty"`
	ActualEnd    *time.Time  `json:"actual_end,omitempty"`
	Priority     int         `json:"priority"` // 1-5, default 3
	Notes        string      `json:"notes"`
	CreatedBy    *uuid.UUID  `json:"created_by,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// MachineBooking represents a time-slot reservation on a machine for a production order.
type MachineBooking struct {
	ID                uuid.UUID     `json:"id"`
	TenantID          uuid.UUID     `json:"tenant_id"`
	MachineID         string        `json:"machine_id"`
	ProductionOrderID uuid.UUID     `json:"production_order_id"`
	StartsAt          time.Time     `json:"starts_at"`
	EndsAt            time.Time     `json:"ends_at"`
	Status            BookingStatus `json:"status"`
	Notes             string        `json:"notes"`
	CreatedBy         *uuid.UUID    `json:"created_by,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// ProductionPlan represents a weekly capacity plan for production.
type ProductionPlan struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	Name                 string     `json:"name"`
	WeekNumber           int        `json:"week_number"`
	Year                 int        `json:"year"`
	TotalCapacityHours   float64    `json:"total_capacity_hours"`
	PlannedCapacityHours float64    `json:"planned_capacity_hours"`
	Status               PlanStatus `json:"status"`
	Notes                string     `json:"notes"`
	CreatedBy            *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// CapacityOverview holds aggregated booking hours for a machine within a plan period.
type CapacityOverview struct {
	MachineID          string  `json:"machine_id"`
	TotalCapacityHours float64 `json:"total_capacity_hours"`
	BookedHours        float64 `json:"booked_hours"`
	AvailableHours     float64 `json:"available_hours"`
}
