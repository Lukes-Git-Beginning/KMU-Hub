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

// LocationType represents the physical type of an inventory location.
type LocationType string

const (
	LocationTypeWarehouse LocationType = "warehouse"
	LocationTypeStore     LocationType = "store"
	LocationTypeVehicle   LocationType = "vehicle"
)

// InventurStatus represents the lifecycle of an inventory count session.
type InventurStatus string

const (
	InventurStatusOpen      InventurStatus = "open"
	InventurStatusCounting  InventurStatus = "counting"
	InventurStatusReview    InventurStatus = "review"
	InventurStatusCompleted InventurStatus = "completed"
)

// Item represents a stock-keeping unit in the inventory.
type Item struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	Name                string     `json:"name"`
	SKU                 string     `json:"sku"`
	Barcode             *string    `json:"barcode,omitempty"`
	Quantity            int64      `json:"quantity"`
	MinQuantity         int64      `json:"min_quantity"`
	Unit                string     `json:"unit"`
	Location            *string    `json:"location,omitempty"`
	LocationID          *uuid.UUID `json:"location_id,omitempty"`
	Category            *string    `json:"category,omitempty"`
	Price               *float64   `json:"price,omitempty"`
	Currency            *string    `json:"currency,omitempty"`
	BatchNumber         *string    `json:"batch_number,omitempty"`
	SerialNumbers       []string   `json:"serial_numbers,omitempty"`
	LinkedPurchaseOrder *string    `json:"linked_purchase_order,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
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
	Reference    *string      `json:"reference,omitempty"`
	LocationFrom *string      `json:"location_from,omitempty"`
	LocationTo   *string      `json:"location_to,omitempty"`
	Notes        *string      `json:"notes,omitempty"`
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

// InventoryLocation represents a physical storage location.
type InventoryLocation struct {
	ID        uuid.UUID    `json:"id"`
	TenantID  uuid.UUID    `json:"tenant_id"`
	Name      string       `json:"name"`
	Address   string       `json:"address"`
	Type      LocationType `json:"type"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt *time.Time   `json:"deleted_at,omitempty"`
}

// InventurSession represents a physical inventory count session.
type InventurSession struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	Name       string          `json:"name"`
	Date       time.Time       `json:"date"`
	Status     InventurStatus  `json:"status"`
	LocationID *uuid.UUID      `json:"location_id,omitempty"`
	CreatedBy  *uuid.UUID      `json:"created_by,omitempty"`
	Counts     []InventurCount `json:"counts,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// InventurCount represents one item's count within an inventory session.
type InventurCount struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	SessionID uuid.UUID  `json:"session_id"`
	ItemID    uuid.UUID  `json:"item_id"`
	Expected  int64      `json:"expected"`
	Counted   *int64     `json:"counted,omitempty"`
	CountedAt *time.Time `json:"counted_at,omitempty"`
}
