package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================================
// Status Constants
// ============================================================================

// DefaultCurrency is the fallback ISO 4217 currency code applied to finance
// documents when a tenant has no configured default (B6 / Migration 000216).
// It mirrors the DB column default on finance_invoices/credit_notes/quotes.
const DefaultCurrency = "EUR"

// Quote statuses
const (
	QuoteStatusDraft    = "draft"
	QuoteStatusSent     = "sent"
	QuoteStatusAccepted = "accepted"
	QuoteStatusRejected = "rejected"
	QuoteStatusExpired  = "expired"
)

// Invoice statuses
const (
	InvoiceStatusDraft     = "draft"
	InvoiceStatusSent      = "sent"
	InvoiceStatusPaid      = "paid"
	InvoiceStatusOverdue   = "overdue"
	InvoiceStatusCancelled = "cancelled"
)

// Credit note statuses
const (
	CreditNoteStatusDraft = "draft"
	CreditNoteStatusSent  = "sent"
)

// Dunning statuses
const (
	DunningStatusDraft = "draft"
	DunningStatusSent  = "sent"
	DunningStatusPaid  = "paid"
)

// Tax modes
const (
	TaxModeStandard        = "standard"
	TaxModeReverseCharge   = "reverse_charge"
	TaxModeKleinunternehmer = "kleinunternehmer"
)

// Payment methods
const (
	PaymentMethodBankTransfer = "bank_transfer"
	PaymentMethodCash         = "cash"
	PaymentMethodCreditCard   = "credit_card"
	PaymentMethodOther        = "other"
)

// Document types for number sequences
const (
	DocumentTypeQuote      = "quote"
	DocumentTypeInvoice    = "invoice"
	DocumentTypeCreditNote = "credit_note"
)

// ============================================================================
// Domain Models
// ============================================================================

// CompanySettings holds per-tenant company information used on invoices and quotes.
type CompanySettings struct {
	ID                      uuid.UUID       `json:"id"`
	TenantID                uuid.UUID       `json:"tenant_id"`
	Name                    string          `json:"name"`
	Street                  string          `json:"street"`
	PLZ                     string          `json:"plz"`
	City                    string          `json:"city"`
	Country                 string          `json:"country"`
	Steuernummer            string          `json:"steuernummer"`
	UStIDNr                 string          `json:"ust_id_nr"`
	Handelsregister         string          `json:"handelsregister"`
	BankName                string          `json:"bank_name"`
	IBAN                    string          `json:"iban"`
	BIC                     string          `json:"bic"`
	LogoURL                 string          `json:"logo_url"`
	AccentColor             string          `json:"accent_color"`
	IsKleinunternehmer      bool            `json:"is_kleinunternehmer"`
	DefaultPaymentTermsDays int             `json:"default_payment_terms_days"`
	DefaultQuoteValidityDays int            `json:"default_quote_validity_days"`
	Basiszinssatz           decimal.Decimal `json:"basiszinssatz"`
	// DefaultCurrency is the ISO 4217 code applied to new finance documents (B6).
	DefaultCurrency         string          `json:"default_currency"`
	// DatevBeraterNr / DatevMandantNr fill the DATEV EXTF header so an imported
	// Buchungsstapel is assigned to the right client. Empty until configured.
	DatevBeraterNr          string          `json:"datev_berater_nr"`
	DatevMandantNr          string          `json:"datev_mandant_nr"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

// NumberSequence tracks auto-incrementing document numbers per tenant and fiscal year.
type NumberSequence struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	DocumentType  string    `json:"document_type"`
	Prefix        string    `json:"prefix"`
	CurrentNumber int       `json:"current_number"`
	FiscalYear    int       `json:"fiscal_year"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LineItem represents a single line on a quote, invoice, or credit note.
// Stored in the relational line tables (finance_invoice_lines, finance_quote_lines,
// finance_credit_note_lines). The LineItems json.RawMessage field on Quote/Invoice/
// CreditNote is an in-memory transport populated by the repository from those tables.
type LineItem struct {
	ID          string          `json:"id"`
	Position    int             `json:"position"`
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	TaxRate     decimal.Decimal `json:"tax_rate"`
	LineTotal   decimal.Decimal `json:"line_total"`
}

// TaxBreakdown holds the computed tax summary for a financial document.
// Stored as JSONB in the database.
type TaxBreakdown struct {
	Subtotal   decimal.Decimal            `json:"subtotal"`
	TaxByRate  map[string]decimal.Decimal `json:"tax_by_rate"`
	TotalTax   decimal.Decimal            `json:"total_tax"`
	GrossTotal decimal.Decimal            `json:"gross_total"`
}

// CustomerSnapshot captures customer details at document creation time.
// Embedded in quotes, invoices, and credit notes as denormalized data.
type CustomerSnapshot struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Email    string `json:"email"`
	UStIDNr  string `json:"ust_id_nr"`
}

// CompanySnapshot captures company details at invoice send time.
// Frozen into the invoice so it remains accurate even if company settings change.
type CompanySnapshot struct {
	Name            string `json:"name"`
	Street          string `json:"street"`
	PLZ             string `json:"plz"`
	City            string `json:"city"`
	Country         string `json:"country"`
	Steuernummer    string `json:"steuernummer"`
	UStIDNr         string `json:"ust_id_nr"`
	Handelsregister string `json:"handelsregister"`
	BankName        string `json:"bank_name"`
	IBAN            string `json:"iban"`
	BIC             string `json:"bic"`
	LogoURL         string `json:"logo_url"`
	AccentColor     string `json:"accent_color"`
}

// Quote represents a price offer (Angebot) to a customer.
type Quote struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	QuoteNumber     string          `json:"quote_number"`
	Status          string          `json:"status"`
	CustomerName    string          `json:"customer_name"`
	CustomerAddress string          `json:"customer_address"`
	CustomerEmail   string          `json:"customer_email"`
	CustomerUStIDNr string          `json:"customer_ust_id_nr"`
	TaxMode         string          `json:"tax_mode"`
	LineItems       json.RawMessage `json:"line_items"`
	TaxBreakdownRaw json.RawMessage `json:"tax_breakdown"`
	Subtotal        decimal.Decimal `json:"subtotal"`
	TotalTax        decimal.Decimal `json:"total_tax"`
	GrossTotal      decimal.Decimal `json:"gross_total"`
	Currency        string          `json:"currency"`
	ValidUntil      *time.Time      `json:"valid_until,omitempty"`
	Notes           string          `json:"notes"`
	DealID          *uuid.UUID      `json:"deal_id,omitempty"`
	SourceQuoteID   *uuid.UUID      `json:"source_quote_id,omitempty"`
	CreatedBy       uuid.UUID       `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Invoice represents a bill (Rechnung) sent to a customer.
type Invoice struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	InvoiceNumber     string          `json:"invoice_number"`
	Status            string          `json:"status"`
	CustomerName      string          `json:"customer_name"`
	CustomerAddress   string          `json:"customer_address"`
	CustomerEmail     string          `json:"customer_email"`
	CustomerUStIDNr   string          `json:"customer_ust_id_nr"`
	CompanySnapshotRaw json.RawMessage `json:"company_snapshot"`
	TaxMode           string          `json:"tax_mode"`
	LineItems         json.RawMessage `json:"line_items"`
	TaxBreakdownRaw   json.RawMessage `json:"tax_breakdown"`
	Subtotal          decimal.Decimal `json:"subtotal"`
	TotalTax          decimal.Decimal `json:"total_tax"`
	GrossTotal        decimal.Decimal `json:"gross_total"`
	Currency          string          `json:"currency"`
	InvoiceDate       time.Time       `json:"invoice_date"`
	DeliveryDate      *time.Time      `json:"delivery_date,omitempty"`
	DueDate           time.Time       `json:"due_date"`
	PaymentTerms      string          `json:"payment_terms"`
	SnapshotData      json.RawMessage `json:"snapshot_data"`
	SourceQuoteID     *uuid.UUID      `json:"source_quote_id,omitempty"`
	Notes             string          `json:"notes"`
	ZUGFeRDProfile    *string         `json:"zugferd_profile,omitempty"`
	TimeTrackingSource json.RawMessage `json:"time_tracking_source,omitempty"`
	// LockedAt and LockedBy replace the snapshot_data lock hack (ADR-0007 / Migration 000132).
	// Set by LockInvoice; non-nil means the invoice is administratively locked (GoBD §146).
	LockedAt          *time.Time      `json:"locked_at,omitempty"`
	LockedBy          *uuid.UUID      `json:"locked_by,omitempty"`
	// ContactID links this invoice to a CRM contact for Contact-360 view (Migration 000141).
	// Populated via backfill (source_quote_id→deal→contact) and on manual creation.
	ContactID         *uuid.UUID      `json:"contact_id,omitempty"`
	// Source is the invoice provenance (Migration 000243): InvoiceSourceCosmi for
	// Cosmi-issued invoices (own GoBD RE-YYYY-NNNN number space) or InvoiceSourceBexio
	// for read-only mirrors pulled from Bexio. ExternalID/ExternalNumber carry the
	// source-system identity and are NULL for Cosmi invoices.
	Source            string          `json:"source"`
	ExternalID        *string         `json:"external_id,omitempty"`
	ExternalNumber    *string         `json:"external_number,omitempty"`
	CreatedBy         uuid.UUID       `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// CreditNote represents a credit (Gutschrift) against an invoice.
type CreditNote struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	CreditNoteNumber  string          `json:"credit_note_number"`
	Status            string          `json:"status"`
	OriginalInvoiceID uuid.UUID       `json:"original_invoice_id"`
	CustomerName      string          `json:"customer_name"`
	CustomerAddress   string          `json:"customer_address"`
	CustomerEmail     string          `json:"customer_email"`
	CustomerUStIDNr   string          `json:"customer_ust_id_nr"`
	TaxMode           string          `json:"tax_mode"`
	LineItems         json.RawMessage `json:"line_items"`
	TaxBreakdownRaw   json.RawMessage `json:"tax_breakdown"`
	Subtotal          decimal.Decimal `json:"subtotal"`
	TotalTax          decimal.Decimal `json:"total_tax"`
	GrossTotal        decimal.Decimal `json:"gross_total"`
	Currency          string          `json:"currency"`
	Reason            string          `json:"reason"`
	CreatedBy         uuid.UUID       `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// Payment represents a payment received against an invoice.
type Payment struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	InvoiceID   uuid.UUID       `json:"invoice_id"`
	Amount      decimal.Decimal `json:"amount"`
	PaymentDate time.Time       `json:"payment_date"`
	Method      string          `json:"method"`
	Reference   string          `json:"reference"`
	Notes       string          `json:"notes"`
	CreatedBy   uuid.UUID       `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	// IdempotencyKey is the client-supplied Idempotency-Key used for DB-level
	// payment deduplication (F5). Empty for payments recorded without one.
	// Internal only — not exposed in API responses.
	IdempotencyKey string `json:"-"`
}

// DunningRecord represents a dunning notice (Mahnung) for an overdue invoice.
type DunningRecord struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	InvoiceID uuid.UUID       `json:"invoice_id"`
	Level     int             `json:"level"`
	Status    string          `json:"status"`
	Fee       decimal.Decimal `json:"fee"`
	Interest  decimal.Decimal `json:"interest"`
	SentAt    *time.Time      `json:"sent_at,omitempty"`
	CreatedBy uuid.UUID       `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
}

// DunningConfig holds per-tenant dunning escalation settings.
type DunningConfig struct {
	ID                    uuid.UUID       `json:"id"`
	TenantID              uuid.UUID       `json:"tenant_id"`
	Level1DaysAfterDue    int             `json:"level1_days_after_due"`
	Level2DaysAfterLevel1 int             `json:"level2_days_after_level1"`
	Level3DaysAfterLevel2 int             `json:"level3_days_after_level2"`
	Level1Fee             decimal.Decimal `json:"level1_fee"`
	Level2Fee             decimal.Decimal `json:"level2_fee"`
	Level3Fee             decimal.Decimal `json:"level3_fee"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// ============================================================================
// Dashboard Models
// ============================================================================

// FinanceDashboard holds the aggregated finance dashboard metrics.
type FinanceDashboard struct {
	Revenue         RevenueMetrics         `json:"revenue"`
	Pipeline        PipelineMetrics        `json:"pipeline"`
	StatusBreakdown InvoiceStatusBreakdown `json:"status_breakdown"`
	RecentInvoices  []*Invoice             `json:"recent_invoices"`
	ExpiringQuotes  []*Quote               `json:"expiring_quotes"`
	PendingDunnings []*DunningRecord       `json:"pending_dunnings"`
	RevenueForecast decimal.Decimal        `json:"revenue_forecast"`
}

// RevenueMetrics holds the revenue aggregation data.
type RevenueMetrics struct {
	TotalInvoiced    decimal.Decimal `json:"total_invoiced"`
	TotalPaid        decimal.Decimal `json:"total_paid"`
	TotalOutstanding decimal.Decimal `json:"total_outstanding"`
	OverdueAmount    decimal.Decimal `json:"overdue_amount"`
}

// PipelineMetrics holds the sales pipeline aggregation data.
type PipelineMetrics struct {
	QuotesPending  int             `json:"quotes_pending"`
	ConversionRate decimal.Decimal `json:"conversion_rate"`
	AverageDealSize decimal.Decimal `json:"average_deal_size"`
}

// InvoiceStatusBreakdown holds invoice counts by status.
type InvoiceStatusBreakdown struct {
	Draft     int `json:"draft"`
	Sent      int `json:"sent"`
	Overdue   int `json:"overdue"`
	Paid      int `json:"paid"`
	Cancelled int `json:"cancelled"`
}
