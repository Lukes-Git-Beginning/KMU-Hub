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
	BomID        *uuid.UUID  `json:"bom_id,omitempty"`
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

// WorkStepStatus represents the lifecycle state of a work step.
type WorkStepStatus string

const (
	WorkStepStatusPending    WorkStepStatus = "pending"
	WorkStepStatusInProgress WorkStepStatus = "in_progress"
	WorkStepStatusCompleted  WorkStepStatus = "completed"
	WorkStepStatusSkipped    WorkStepStatus = "skipped"
)

// MachineStatus represents the operational state of a machine.
type MachineStatus string

const (
	MachineStatusAvailable   MachineStatus = "available"
	MachineStatusInUse       MachineStatus = "in_use"
	MachineStatusMaintenance MachineStatus = "maintenance"
)

// BomItem is a single line item within a BOM.
type BomItem struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	BomID        uuid.UUID `json:"bom_id"`
	MaterialName string    `json:"material_name"`
	Quantity     float64   `json:"quantity"`
	Unit         string    `json:"unit"`
	SortOrder    int       `json:"sort_order"`
}

// BOM (Bill of Materials / Stückliste) defines the components of a product.
type BOM struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ProductName string     `json:"product_name"`
	SKU         string     `json:"sku"`
	Version     string     `json:"version"`
	Active      bool       `json:"active"`
	Notes       string     `json:"notes"`
	Items       []*BomItem `json:"items"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// WorkStep represents a single operation/step within a production order.
type WorkStep struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	OrderID         uuid.UUID      `json:"order_id"`
	StepNr          int            `json:"step_nr"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	DurationMinutes int            `json:"duration_minutes"`
	Status          WorkStepStatus `json:"status"`
	Assignee        string         `json:"assignee"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Machine represents a production machine managed as an entity.
type Machine struct {
	ID        uuid.UUID     `json:"id"`
	TenantID  uuid.UUID     `json:"tenant_id"`
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Status    MachineStatus `json:"status"`
	Notes     string        `json:"notes"`
	CreatedBy *uuid.UUID    `json:"created_by,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// QualityCheck represents a quality inspection for a production order.
type QualityCheck struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	OrderID      uuid.UUID  `json:"order_id"`
	Inspector    string     `json:"inspector"`
	CheckedAt    time.Time  `json:"checked_at"`
	Passed       bool       `json:"passed"`
	DefectsFound int        `json:"defects_found"`
	Notes        string     `json:"notes"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
