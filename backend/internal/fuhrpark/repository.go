package fuhrpark

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListVehiclesFilter holds optional filtering for vehicle list queries.
type ListVehiclesFilter struct {
	Search   string
	Status   *VehicleStatus
	FuelType *string
}

// ListServicesFilter holds optional filtering for service list queries.
type ListServicesFilter struct {
	VehicleID *uuid.UUID
	Status    *ServiceStatus
}

// ListDamagesFilter holds optional filtering for damage list queries.
type ListDamagesFilter struct {
	VehicleID *uuid.UUID
	Status    *DamageStatus
}

// Repository defines the persistence interface for the fuhrpark module.
type Repository interface {
	// Vehicles
	CreateVehicle(ctx context.Context, v *Vehicle) error
	UpdateVehicle(ctx context.Context, v *Vehicle) error
	SoftDeleteVehicle(ctx context.Context, tenantID, vehicleID uuid.UUID) error
	GetVehicle(ctx context.Context, tenantID, vehicleID uuid.UUID) (*Vehicle, error)
	ListVehicles(ctx context.Context, tenantID uuid.UUID, filter ListVehiclesFilter, offset, limit int) ([]*Vehicle, int, error)
	PlateExists(ctx context.Context, tenantID uuid.UUID, plate string, excludeID *uuid.UUID) (bool, error)

	// Services
	CreateService(ctx context.Context, s *VehicleService) error
	UpdateService(ctx context.Context, s *VehicleService) error
	DeleteService(ctx context.Context, tenantID, serviceID uuid.UUID) error
	GetService(ctx context.Context, tenantID, serviceID uuid.UUID) (*VehicleService, error)
	ListServices(ctx context.Context, tenantID uuid.UUID, filter ListServicesFilter, offset, limit int) ([]*VehicleService, int, error)

	// Damages
	CreateDamage(ctx context.Context, d *VehicleDamage) error
	UpdateDamage(ctx context.Context, d *VehicleDamage) error
	GetDamage(ctx context.Context, tenantID, damageID uuid.UUID) (*VehicleDamage, error)
	ListDamages(ctx context.Context, tenantID uuid.UUID, filter ListDamagesFilter, offset, limit int) ([]*VehicleDamage, int, error)

	// History
	GetVehicleHistory(ctx context.Context, tenantID, vehicleID uuid.UUID, offset, limit int) ([]*HistoryEntry, int, error)

	// TUEV Cron
	// FindVehiclesDueTuev returns vehicles where tuev_due_date falls in [from, to]
	// and tuev_reminder_sent_at is NULL or < windowStart (idempotency guard).
	// The query is intentionally cross-tenant: the cron must check all tenants.
	FindVehiclesDueTuev(ctx context.Context, from, to time.Time) ([]*Vehicle, error)
	// MarkTuevReminderSent stamps tuev_reminder_sent_at = now on a vehicle.
	// tenant_id is required to prevent cross-tenant stamp manipulation.
	MarkTuevReminderSent(ctx context.Context, vehicleID, tenantID uuid.UUID) error
}
