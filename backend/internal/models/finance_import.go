package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Invoice provenance (finance_invoices.source, Migration 000243). Distinguishes
// Cosmi-issued invoices (own GoBD number space, RE-YYYY-NNNN) from read-only mirrors
// pulled from an external accounting system such as Bexio.
const (
	InvoiceSourceCosmi = "cosmi"
	InvoiceSourceBexio = "bexio"
)

// ImportedInvoiceInput is the read-only import shape for an invoice pulled from an
// external accounting system. It is integration-agnostic: any adapter maps its source
// records to this DTO and the finance service imports them via Service.UpsertImported.
// Imported invoices keep their external number and never enter the RE-YYYY-NNNN
// sequence, so the GoBD gap-free journal (source='cosmi') stays untouched.
type ImportedInvoiceInput struct {
	ExternalID      string    // external system id (e.g. Bexio kb_invoice.id)
	ExternalNumber  string    // external document number (shown as invoice_number)
	Status          string    // mapped Cosmi status (draft/sent/paid/cancelled)
	ContactID       uuid.UUID // resolved Cosmi contact; uuid.Nil when unresolved
	CustomerName    string
	CustomerEmail   string
	CustomerAddress string
	InvoiceDate     time.Time
	DueDate         *time.Time
	Currency        string
	Subtotal        decimal.Decimal
	TotalTax        decimal.Decimal
	GrossTotal      decimal.Decimal
	Lines           []ImportedInvoiceLine
	// ExternalUpdatedAt is the source-side updated_at, used for last-write-wins in the
	// sync mapping (bexio_entity_mappings).
	ExternalUpdatedAt *time.Time
}

// ImportedInvoiceLine is a single external invoice position in the Cosmi line shape.
type ImportedInvoiceLine struct {
	Position    int
	Description string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	TaxRate     decimal.Decimal
	LineTotal   decimal.Decimal
}
