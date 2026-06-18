package einkauf

import (
	"time"

	"github.com/google/uuid"
)

// CatalogItem represents a product in a supplier's catalog.
type CatalogItem struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	SupplierID  uuid.UUID `json:"supplier_id"`
	Name        string    `json:"name"`
	SKU         string    `json:"sku"`
	Category    string    `json:"category"`
	Price       string    `json:"price"`         // numeric string
	Currency    string    `json:"currency"`
	Unit        string    `json:"unit"`
	Available   bool      `json:"available"`
	MinOrderQty string    `json:"min_order_qty"` // numeric string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RatingCategory classifies a supplier rating dimension.
type RatingCategory string

const (
	RatingCategoryQuality  RatingCategory = "quality"
	RatingCategoryDelivery RatingCategory = "delivery"
	RatingCategoryPrice    RatingCategory = "price"
)

// SupplierRating represents a rating for a supplier in a specific category.
type SupplierRating struct {
	ID         uuid.UUID      `json:"id"`
	TenantID   uuid.UUID      `json:"tenant_id"`
	SupplierID uuid.UUID      `json:"supplier_id"`
	Category   RatingCategory `json:"category"`
	Rating     int32          `json:"rating"` // 1–5
	Comment    string         `json:"comment"`
	RatedBy    *uuid.UUID     `json:"rated_by,omitempty"`
	RatedAt    time.Time      `json:"rated_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// ContractStatus is the lifecycle state of a framework contract.
type ContractStatus string

const (
	ContractStatusDraft   ContractStatus = "draft"
	ContractStatusActive  ContractStatus = "active"
	ContractStatusExpired ContractStatus = "expired"
)

// FrameworkContract represents a framework / blanket purchase agreement.
type FrameworkContract struct {
	ID         uuid.UUID      `json:"id"`
	TenantID   uuid.UUID      `json:"tenant_id"`
	SupplierID uuid.UUID      `json:"supplier_id"`
	Title      string         `json:"title"`
	ContractNr string         `json:"contract_nr"`
	StartDate  *time.Time     `json:"start_date,omitempty"`
	EndDate    *time.Time     `json:"end_date,omitempty"`
	TotalValue string         `json:"total_value"` // numeric string
	UsedValue  string         `json:"used_value"`  // numeric string
	Currency   string         `json:"currency"`
	Status     ContractStatus `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`

	// Items are populated by GetFrameworkContractWithItems.
	Items []*FrameworkContractItem `json:"items,omitempty"`
}

// FrameworkContractItem is a line item inside a framework contract.
type FrameworkContractItem struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	ContractID uuid.UUID `json:"contract_id"`
	Name       string    `json:"name"`
	UnitPrice  string    `json:"unit_price"` // numeric string
	Unit       string    `json:"unit"`
	AgreedQty  string    `json:"agreed_qty"` // numeric string
	CalledQty  string    `json:"called_qty"` // numeric string
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// FrameworkContractCall records a call-off (abruf) against a framework contract.
type FrameworkContractCall struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	ContractID uuid.UUID  `json:"contract_id"`
	POID       *uuid.UUID `json:"po_id,omitempty"`
	Amount     string     `json:"amount"` // numeric string
	Currency   string     `json:"currency"`
	CalledAt   time.Time  `json:"called_at"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
