package vermietung

import (
	"time"

	"github.com/google/uuid"
)

// RentalStatus represents the lifecycle state of a rental.
type RentalStatus string

const (
	RentalStatusReserved  RentalStatus = "reserved"
	RentalStatusActive    RentalStatus = "active"
	RentalStatusCompleted RentalStatus = "completed"
	RentalStatusCancelled RentalStatus = "cancelled"
)

// InspectionKind distinguishes handover from return inspections.
type InspectionKind string

const (
	InspectionKindHandover InspectionKind = "handover"
	InspectionKindReturn   InspectionKind = "return"
)

// RentalObject represents an asset that can be rented out.
type RentalObject struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Category    string     `json:"category"`
	DailyRate   float64    `json:"daily_rate"`
	Deposit     float64    `json:"deposit"`
	Location    *string    `json:"location,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// Rental represents a booking of a RentalObject for a date range.
type Rental struct {
	ID            uuid.UUID    `json:"id"`
	TenantID      uuid.UUID    `json:"tenant_id"`
	ObjectID      uuid.UUID    `json:"object_id"`
	ContactID     *uuid.UUID   `json:"contact_id,omitempty"`
	RenterName    string       `json:"renter_name"`
	StartDate     time.Time    `json:"start_date"`
	EndDate       time.Time    `json:"end_date"`
	Status        RentalStatus `json:"status"`
	TotalPrice    float64      `json:"total_price"`
	DepositPaid   bool         `json:"deposit_paid"`
	Notes         *string      `json:"notes,omitempty"`
	SignatureData *string      `json:"signature_data,omitempty"`
	SignedAt      *time.Time   `json:"signed_at,omitempty"`
	SignedBy      *string      `json:"signed_by,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// RentalInspection records the condition of a rental object at handover or return.
type RentalInspection struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	RentalID      uuid.UUID       `json:"rental_id"`
	Kind          InspectionKind  `json:"kind"`
	Notes         string          `json:"notes"`
	PhotoURLs     []string        `json:"photo_urls"`
	PerformedBy   *uuid.UUID      `json:"performed_by,omitempty"`
	SignatureData *string         `json:"signature_data,omitempty"`
	Checklist     []ChecklistItem `json:"checklist"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// ChecklistItem is one position of an inspection's structured condition checklist,
// e.g. {label: "Windschutzscheibe", condition: "intakt", remark: "leichter Steinschlag"}.
type ChecklistItem struct {
	Label     string `json:"label"`
	Condition string `json:"condition"`
	Remark    string `json:"remark,omitempty"`
}
