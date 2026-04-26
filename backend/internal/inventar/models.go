package inventar

import (
	"time"

	"github.com/google/uuid"
)

// MovementType represents the direction of an inventory movement.
type MovementType string

const (
	MovementTypeIn         MovementType = "in"
	MovementTypeOut        MovementType = "out"
	MovementTypeAdjustment MovementType = "adjustment"
	MovementTypeTransfer   MovementType = "transfer"
)

// WarningStatus represents the lifecycle state of a stock warning.
type WarningStatus string

const (
	WarningStatusActive       WarningStatus = "active"
	WarningStatusAcknowledged WarningStatus = "acknowledged"
	WarningStatusResolved     WarningStatus = "resolved"
)

// Item represents a stock-keeping unit in the inventory.
type Item struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	SKU         string     `json:"sku"`
	Barcode     *string    `json:"barcode,omitempty"`
	Quantity    int64      `json:"quantity"`
	MinQuantity int64      `json:"min_quantity"`
	Unit        string     `json:"unit"`
	Location    *string    `json:"location,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// Movement records a single stock change event.
type Movement struct {
	ID           uuid.UUID    `json:"id"`
	TenantID     uuid.UUID    `json:"tenant_id"`
	ItemID       uuid.UUID    `json:"item_id"`
	MovementType MovementType `json:"movement_type"`
	Quantity     int64        `json:"quantity"`
	PerformedBy  *uuid.UUID   `json:"performed_by,omitempty"`
	Reason       string       `json:"reason"`
	CreatedAt    time.Time    `json:"created_at"`
}

// Warning represents an automatic or manual low-stock alert.
type Warning struct {
	ID              uuid.UUID     `json:"id"`
	TenantID        uuid.UUID     `json:"tenant_id"`
	ItemID          uuid.UUID     `json:"item_id"`
	Threshold       int64         `json:"threshold"`
	CurrentQuantity int64         `json:"current_quantity"`
	Status          WarningStatus `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	AcknowledgedAt  *time.Time    `json:"acknowledged_at,omitempty"`
	AcknowledgedBy  *uuid.UUID    `json:"acknowledged_by,omitempty"`
}
