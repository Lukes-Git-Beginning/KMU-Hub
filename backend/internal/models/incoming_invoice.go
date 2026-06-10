package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================================
// Incoming Invoice — E-Rechnung Eingang
// ============================================================================

// IncomingInvoiceStatus constants for finance_incoming_invoices.status.
const (
	IncomingInvoiceStatusReceived = "received"
	IncomingInvoiceStatusReviewed = "reviewed"
	IncomingInvoiceStatusBooked   = "booked"
	IncomingInvoiceStatusRejected = "rejected"
)

// IncomingInvoiceSourceFormat constants for finance_incoming_invoices.source_format.
const (
	IncomingInvoiceFormatZUGFeRDCII   = "zugferd_cii"
	IncomingInvoiceFormatXRechnungUBL = "xrechnung_ubl"
	IncomingInvoiceFormatXMLOnly      = "xml_only"
)

// IncomingInvoiceLineItem represents a parsed line item from an inbound e-invoice.
// Stored as JSONB in finance_incoming_invoices.line_items.
type IncomingInvoiceLineItem struct {
	Position    int             `json:"position"`
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	TaxRate     decimal.Decimal `json:"tax_rate"`
	LineTotal   decimal.Decimal `json:"line_total"`
}

// IncomingInvoiceTaxEntry holds a single tax-rate bucket for the tax breakdown.
// Stored as JSONB array in finance_incoming_invoices.tax_breakdown.
type IncomingInvoiceTaxEntry struct {
	TaxRate    decimal.Decimal `json:"tax_rate"`
	TaxableNet decimal.Decimal `json:"taxable_net"`
	TaxAmount  decimal.Decimal `json:"tax_amount"`
}

// IncomingInvoice represents an inbound electronic invoice (Eingangsrechnung)
// parsed from a ZUGFeRD/Factur-X CII or XRechnung UBL document.
type IncomingInvoice struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	SupplierName     string          `json:"supplier_name"`
	SupplierAddress  string          `json:"supplier_address"`
	SupplierVATID    string          `json:"supplier_vat_id"`
	InvoiceNumber    string          `json:"invoice_number"`
	InvoiceDate      time.Time       `json:"invoice_date"`
	DueDate          *time.Time      `json:"due_date,omitempty"`
	Currency         string          `json:"currency"`
	LineItems        json.RawMessage `json:"line_items"`
	TaxBreakdownRaw  json.RawMessage `json:"tax_breakdown"`
	Subtotal         decimal.Decimal `json:"subtotal"`
	TotalTax         decimal.Decimal `json:"total_tax"`
	GrossTotal       decimal.Decimal `json:"gross_total"`
	SourceFormat     string          `json:"source_format"`
	OriginalXML      string          `json:"original_xml"`
	OriginalFilename string          `json:"original_filename"`
	Status           string          `json:"status"`
	CreatedBy        uuid.UUID       `json:"created_by"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}
