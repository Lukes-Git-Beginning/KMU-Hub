package einkauf

import (
	"time"

	"github.com/google/uuid"
)

// POStatus represents the lifecycle state of a purchase order.
type POStatus string

const (
	POStatusDraft             POStatus = "draft"
	POStatusSubmitted         POStatus = "submitted"
	POStatusSent              POStatus = "sent"
	POStatusPartiallyReceived POStatus = "partially_received"
	POStatusReceived          POStatus = "received"
	POStatusClosed            POStatus = "closed"
	POStatusCancelled         POStatus = "cancelled"
)

// Supplier represents a vendor/supplier in the purchasing module.
type Supplier struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Name         string     `json:"name"`
	ContactID    *uuid.UUID `json:"contact_id,omitempty"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	Address      string     `json:"address"`
	PaymentTerms string     `json:"payment_terms"`
	Notes        string     `json:"notes"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// PurchaseOrder represents a purchase order sent to a supplier.
//
// Wareneingangs-Tracking: Die Spalten received_quantity in den zugehoerigen
// po_lines werden lokal in dieser Datenbank gepflegt. Der Abgleich mit dem
// Inventar-Modul (inventory_movements) ist ein Sprint-3-Item — bis dahin
// repraesentieren received_quantity-Werte nur den lokalen Stand ohne
// Auswirkung auf den globalen Lagerbestand.
type PurchaseOrder struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	SupplierID           uuid.UUID  `json:"supplier_id"`
	PONumber             string     `json:"po_number"`
	Status               POStatus   `json:"status"`
	OrderDate            time.Time  `json:"order_date"`
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date,omitempty"`
	TotalAmount          string     `json:"total_amount"` // numeric as string to avoid float precision issues
	Currency             string     `json:"currency"`
	Notes                string     `json:"notes"`
	CreatedBy            *uuid.UUID `json:"created_by,omitempty"`
	// FrameworkContractID links this PO to a framework contract (optional).
	// When set, every goods receipt records a call-off for the newly received value.
	FrameworkContractID  *uuid.UUID `json:"framework_contract_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// Lines are populated on GetPO — not stored directly on the PO row.
	Lines []*POLine `json:"lines,omitempty"`

	// ReceiptCallOff is set only on the PO returned by ReceiveGoods and
	// PartialReceive, and only when that receipt touched a framework contract.
	ReceiptCallOff *ReceiptCallOff `json:"-"`
}

// GoodsReceipt is one goods receipt: the cumulative received quantity per line
// and the PO status that follows from it.
type GoodsReceipt struct {
	TenantID uuid.UUID
	POID     uuid.UUID
	Lines    []PartialReceiveItem // one entry per line, cumulative quantity
	Status   POStatus
}

// ReceiptCallOff reports what a goods receipt did to the PO's framework contract.
// CallID is nil when nothing was booked; Warning is set when the call-off
// exceeded the budget or had to be skipped.
type ReceiptCallOff struct {
	ContractID     uuid.UUID
	CallID         *uuid.UUID
	Amount         string // net value of the newly received quantities
	BudgetExceeded bool
	Remaining      string // contract's remaining value before this call-off
	Warning        string
}

// POLine represents a single line item within a purchase order.
type POLine struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	POID             uuid.UUID `json:"po_id"`
	ProductName      string    `json:"product_name"`
	SKU              string    `json:"sku,omitempty"`
	Quantity         string    `json:"quantity"`         // numeric
	UnitPrice        string    `json:"unit_price"`       // numeric
	TaxRate          string    `json:"tax_rate"`         // numeric, default 0
	ReceivedQuantity string    `json:"received_quantity"` // numeric, default 0
	LinePosition     int       `json:"line_position"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
