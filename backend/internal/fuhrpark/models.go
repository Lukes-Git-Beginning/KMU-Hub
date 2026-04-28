package fuhrpark

import (
	"time"

	"github.com/google/uuid"
)

// ServiceStatus represents the lifecycle state of a vehicle service entry.
type ServiceStatus string

const (
	ServiceStatusScheduled  ServiceStatus = "scheduled"
	ServiceStatusInProgress ServiceStatus = "in_progress"
	ServiceStatusCompleted  ServiceStatus = "completed"
	ServiceStatusCancelled  ServiceStatus = "cancelled"
)

// DamageStatus represents the lifecycle state of a damage report.
type DamageStatus string

const (
	DamageStatusReported  DamageStatus = "reported"
	DamageStatusInRepair  DamageStatus = "in_repair"
	DamageStatusResolved  DamageStatus = "resolved"
)

// VehicleStatus represents the operational state of a vehicle.
type VehicleStatus string

const (
	VehicleStatusActive          VehicleStatus = "active"
	VehicleStatusInService       VehicleStatus = "in_service"
	VehicleStatusInactive        VehicleStatus = "inactive"
	VehicleStatusDecommissioned  VehicleStatus = "decommissioned"
)

// Vehicle represents a fleet vehicle.
type Vehicle struct {
	ID                   uuid.UUID     `json:"id"`
	TenantID             uuid.UUID     `json:"tenant_id"`
	LicensePlate         string        `json:"license_plate"`
	Make                 string        `json:"make"`
	Model                string        `json:"model"`
	Year                 int           `json:"year"`
	VIN                  *string       `json:"vin,omitempty"`
	Color                *string       `json:"color,omitempty"`
	FuelType             string        `json:"fuel_type"`
	Status               VehicleStatus `json:"status"`
	MileageKm            int64         `json:"mileage_km"`
	TuevDueDate          *time.Time    `json:"tuev_due_date,omitempty"`
	TuevReminderSentAt   *time.Time    `json:"tuev_reminder_sent_at,omitempty"`
	AssignedDriverID     *uuid.UUID    `json:"assigned_driver_id,omitempty"`
	Notes                *string       `json:"notes,omitempty"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
	DeletedAt            *time.Time    `json:"deleted_at,omitempty"`
}

// VehicleService records a maintenance or inspection event for a vehicle.
type VehicleService struct {
	ID          uuid.UUID     `json:"id"`
	TenantID    uuid.UUID     `json:"tenant_id"`
	VehicleID   uuid.UUID     `json:"vehicle_id"`
	ServiceType string        `json:"service_type"`
	Description *string       `json:"description,omitempty"`
	ScheduledAt time.Time     `json:"scheduled_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	CostCents   *int64        `json:"cost_cents,omitempty"`
	Workshop    *string       `json:"workshop,omitempty"`
	MileageKm   *int64        `json:"mileage_km,omitempty"`
	Status      ServiceStatus `json:"status"`
	Notes       *string       `json:"notes,omitempty"`
	CreatedBy   *uuid.UUID    `json:"created_by,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// VehicleDamage records a reported damage incident for a vehicle.
type VehicleDamage struct {
	ID          uuid.UUID    `json:"id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	VehicleID   uuid.UUID    `json:"vehicle_id"`
	Description string       `json:"description"`
	Severity    string       `json:"severity"`
	Status      DamageStatus `json:"status"`
	ReportedBy  *uuid.UUID   `json:"reported_by,omitempty"`
	ResolvedBy  *uuid.UUID   `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time   `json:"resolved_at,omitempty"`
	PhotoKeys   []string     `json:"photo_keys"`
	CostCents   *int64       `json:"cost_cents,omitempty"`
	Notes       *string      `json:"notes,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// HistoryEntry is a unified timeline entry for a vehicle (service or damage).
type HistoryEntry struct {
	Kind       string    `json:"kind"` // "service" | "damage"
	ID         uuid.UUID `json:"id"`
	Summary    string    `json:"summary"`
	OccurredAt time.Time `json:"occurred_at"`
}
