package vermietung

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListObjectsFilter holds optional filtering for object list queries.
type ListObjectsFilter struct {
	Search     string
	Category   *string
	ActiveOnly bool
}

// ListRentalsFilter holds optional filtering for rental list queries.
type ListRentalsFilter struct {
	ObjectID *uuid.UUID
	Status   *RentalStatus
	From     *time.Time
	To       *time.Time
}

// Repository defines the persistence interface for the vermietung module.
type Repository interface {
	// Objects
	CreateObject(ctx context.Context, obj *RentalObject) error
	UpdateObject(ctx context.Context, obj *RentalObject) error
	SoftDeleteObject(ctx context.Context, tenantID, objectID uuid.UUID) error
	GetObject(ctx context.Context, tenantID, objectID uuid.UUID) (*RentalObject, error)
	ListObjects(ctx context.Context, tenantID uuid.UUID, filter ListObjectsFilter, offset, limit int) ([]*RentalObject, int, error)

	// Rentals
	CreateRental(ctx context.Context, rental *Rental) error
	UpdateRental(ctx context.Context, rental *Rental) error
	DeleteRental(ctx context.Context, tenantID, rentalID uuid.UUID) error
	GetRental(ctx context.Context, tenantID, rentalID uuid.UUID) (*Rental, error)
	ListRentals(ctx context.Context, tenantID uuid.UUID, filter ListRentalsFilter, offset, limit int) ([]*Rental, int, error)

	// HasOverlap returns true when any non-cancelled rental for objectID overlaps
	// the given date range, optionally excluding a specific rental (for updates).
	HasOverlap(ctx context.Context, tenantID, objectID uuid.UUID, start, end time.Time, excludeRentalID *uuid.UUID) (bool, error)

	// Inspections
	CreateInspection(ctx context.Context, ins *RentalInspection) error
	UpdateInspection(ctx context.Context, ins *RentalInspection) error
	GetInspection(ctx context.Context, tenantID, inspectionID uuid.UUID) (*RentalInspection, error)
	GetInspectionByKind(ctx context.Context, tenantID, rentalID uuid.UUID, kind InspectionKind) (*RentalInspection, error)
	ListInspections(ctx context.Context, tenantID, rentalID uuid.UUID, offset, limit int) ([]*RentalInspection, int, error)
}
