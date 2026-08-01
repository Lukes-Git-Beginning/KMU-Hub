package models

import (
	"encoding/json"
	"errors"
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
	TaxModeStandard         = "standard"
	TaxModeReverseCharge    = "reverse_charge"
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

// Recurring invoice intervals (Migration 000246).
const (
	RecurringIntervalWeekly    = "weekly"
	RecurringIntervalMonthly   = "monthly"
	RecurringIntervalQuarterly = "quarterly"
	RecurringIntervalYearly    = "yearly"
)

// Recurring invoice schedule statuses (Migration 000246).
const (
	RecurringStatusActive = "active"
	RecurringStatusPaused = "paused"
	RecurringStatusEnded  = "ended"
)

// ============================================================================
// Domain Models
// ============================================================================

// CompanySettings holds per-tenant company information used on invoices and quotes.
type CompanySettings struct {
	ID                       uuid.UUID       `json:"id"`
	TenantID                 uuid.UUID       `json:"tenant_id"`
	Name                     string          `json:"name"`
	Street                   string          `json:"street"`
	PLZ                      string          `json:"plz"`
	City                     string          `json:"city"`
	Country                  string          `json:"country"`
	Steuernummer             string          `json:"steuernummer"`
	UStIDNr                  string          `json:"ust_id_nr"`
	Handelsregister          string          `json:"handelsregister"`
	BankName                 string          `json:"bank_name"`
	IBAN                     string          `json:"iban"`
	BIC                      string          `json:"bic"`
	LogoURL                  string          `json:"logo_url"`
	AccentColor              string          `json:"accent_color"`
	IsKleinunternehmer       bool            `json:"is_kleinunternehmer"`
	DefaultPaymentTermsDays  int             `json:"default_payment_terms_days"`
	DefaultQuoteValidityDays int             `json:"default_quote_validity_days"`
	Basiszinssatz            decimal.Decimal `json:"basiszinssatz"`
	// DefaultCurrency is the ISO 4217 code applied to new finance documents (B6).
	DefaultCurrency string `json:"default_currency"`
	// DatevBeraterNr / DatevMandantNr fill the DATEV EXTF header so an imported
	// Buchungsstapel is assigned to the right client. Empty until configured.
	DatevBeraterNr string    `json:"datev_berater_nr"`
	DatevMandantNr string    `json:"datev_mandant_nr"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
	Name    string `json:"name"`
	Address string `json:"address"`
	Email   string `json:"email"`
	UStIDNr string `json:"ust_id_nr"`
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
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	InvoiceNumber      string          `json:"invoice_number"`
	Status             string          `json:"status"`
	CustomerName       string          `json:"customer_name"`
	CustomerAddress    string          `json:"customer_address"`
	CustomerEmail      string          `json:"customer_email"`
	CustomerUStIDNr    string          `json:"customer_ust_id_nr"`
	CompanySnapshotRaw json.RawMessage `json:"company_snapshot"`
	TaxMode            string          `json:"tax_mode"`
	LineItems          json.RawMessage `json:"line_items"`
	TaxBreakdownRaw    json.RawMessage `json:"tax_breakdown"`
	Subtotal           decimal.Decimal `json:"subtotal"`
	TotalTax           decimal.Decimal `json:"total_tax"`
	GrossTotal         decimal.Decimal `json:"gross_total"`
	Currency           string          `json:"currency"`
	InvoiceDate        time.Time       `json:"invoice_date"`
	DeliveryDate       *time.Time      `json:"delivery_date,omitempty"`
	DueDate            time.Time       `json:"due_date"`
	PaymentTerms       string          `json:"payment_terms"`
	SnapshotData       json.RawMessage `json:"snapshot_data"`
	SourceQuoteID      *uuid.UUID      `json:"source_quote_id,omitempty"`
	Notes              string          `json:"notes"`
	ZUGFeRDProfile     *string         `json:"zugferd_profile,omitempty"`
	TimeTrackingSource json.RawMessage `json:"time_tracking_source,omitempty"`
	// LockedAt and LockedBy replace the snapshot_data lock hack (ADR-0007 / Migration 000132).
	// Set by LockInvoice; non-nil means the invoice is administratively locked (GoBD §146).
	LockedAt *time.Time `json:"locked_at,omitempty"`
	LockedBy *uuid.UUID `json:"locked_by,omitempty"`
	// ContactID links this invoice to a CRM contact for Contact-360 view (Migration 000141).
	// Populated via backfill (source_quote_id→deal→contact) and on manual creation.
	ContactID *uuid.UUID `json:"contact_id,omitempty"`
	// Source is the invoice provenance (Migration 000243): InvoiceSourceCosmi for
	// Cosmi-issued invoices (own GoBD RE-YYYY-NNNN number space) or InvoiceSourceBexio
	// for read-only mirrors pulled from Bexio. ExternalID/ExternalNumber carry the
	// source-system identity and are NULL for Cosmi invoices.
	Source         string  `json:"source"`
	ExternalID     *string `json:"external_id,omitempty"`
	ExternalNumber *string `json:"external_number,omitempty"`
	// RecurringID links this invoice to the schedule that emitted it
	// (Migration 000246). NULL for manually created invoices.
	RecurringID *uuid.UUID `json:"recurring_id,omitempty"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
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

// Chain node statuses (presentational, independent of the underlying
// document's own status field — e.g. a quote that led to an invoice is
// "completed" as a chain step regardless of its own status column).
const (
	ChainNodeCompleted = "completed"
	ChainNodeActive    = "active"
	ChainNodePending   = "pending"
	ChainNodeCancelled = "cancelled"
	ChainNodeOverdue   = "overdue"
)

// Chain node document types.
const (
	ChainNodeQuote      = "quote"
	ChainNodeInvoice    = "invoice"
	ChainNodePayment    = "payment"
	ChainNodeCreditNote = "credit-note"
	ChainNodeDunning    = "dunning"
)

// ChainNode is one step in a document's lifecycle (Belegkette): a quote,
// invoice, payment, dunning notice, or credit note. Number and Date are empty
// for a step that has not happened yet — the pending-payment placeholder on
// an invoice that is not fully settled.
type ChainNode struct {
	Type   string          `json:"type"`
	Number string          `json:"number"`
	Date   *time.Time      `json:"date,omitempty"`
	Amount decimal.Decimal `json:"amount"`
	Status string          `json:"status"`
}

// DocumentChain traces one customer document from quote through invoice to
// payment, dunning, or credit note — GET /finance/document-chains.
type DocumentChain struct {
	ID         uuid.UUID       `json:"id"`
	Customer   string          `json:"customer"`
	Currency   string          `json:"currency"`
	TotalValue decimal.Decimal `json:"total_value"`
	IsComplete bool            `json:"is_complete"`
	Nodes      []ChainNode     `json:"nodes"`
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
	QuotesPending   int             `json:"quotes_pending"`
	ConversionRate  decimal.Decimal `json:"conversion_rate"`
	AverageDealSize decimal.Decimal `json:"average_deal_size"`
}

// RecurringInvoice is a schedule that emits draft invoices at a fixed interval
// until the optional end date is reached (Migration 000246). It is a template,
// never itself a bookkeeping document: it carries no number and never enters the
// GoBD journal — only the invoices it emits do.
type RecurringInvoice struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	Title            string          `json:"title"`
	CustomerName     string          `json:"customer_name"`
	CustomerAddress  string          `json:"customer_address"`
	CustomerEmail    string          `json:"customer_email"`
	CustomerUStIDNr  string          `json:"customer_ust_id_nr"`
	TaxMode          string          `json:"tax_mode"`
	LineItems        json.RawMessage `json:"line_items"`
	TaxBreakdownRaw  json.RawMessage `json:"tax_breakdown"`
	Currency         string          `json:"currency"`
	Interval         string          `json:"interval"`
	Status           string          `json:"status"`
	StartDate        time.Time       `json:"start_date"`
	EndDate          *time.Time      `json:"end_date,omitempty"`
	NextRun          time.Time       `json:"next_run"`
	PaymentTermsDays int             `json:"payment_terms_days"`
	GeneratedCount   int             `json:"generated_count"`
	LastGeneratedAt  *time.Time      `json:"last_generated_at,omitempty"`
	Notes            string          `json:"notes"`
	CreatedBy        *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// InvoiceStatusBreakdown holds invoice counts by status.
type InvoiceStatusBreakdown struct {
	Draft     int `json:"draft"`
	Sent      int `json:"sent"`
	Overdue   int `json:"overdue"`
	Paid      int `json:"paid"`
	Cancelled int `json:"cancelled"`
}

// ============================================================================
// Open Items (Offene Posten) — receivables aging
// ============================================================================

// Aging bucket keys. A receivable falls into exactly one bucket, by how many
// days it is past its due date.
const (
	AgingBucketCurrent = "current" // not yet due
	AgingBucketD30     = "d30"     // 1-30 days overdue
	AgingBucketD60     = "d60"     // 31-60 days overdue
	AgingBucketD60Plus = "d60plus" // more than 60 days overdue
)

// agingBucketKeys is indexed by bucket index; agingBucketUpperDays holds the
// inclusive upper day bound of every bucket except the last (which is open
// ended). Both the Go classification below and the SQL aggregation in the
// invoice repository derive from these two slices — the repository passes the
// bounds as query parameters instead of restating them, so there is one source
// of truth for where a bucket starts.
var (
	agingBucketKeys      = []string{AgingBucketCurrent, AgingBucketD30, AgingBucketD60, AgingBucketD60Plus}
	agingBucketUpperDays = []int{0, 30, 60}
)

// AgingBucketUpperDays returns the inclusive upper day bounds of all buckets
// except the open-ended last one.
func AgingBucketUpperDays() []int {
	out := make([]int, len(agingBucketUpperDays))
	copy(out, agingBucketUpperDays)
	return out
}

// AgingBucketFor classifies a day count past due into a bucket key. A negative
// or zero count means the item is not due yet.
func AgingBucketFor(daysOverdue int) string {
	return AgingBucketKeyAt(AgingBucketIndexFor(daysOverdue))
}

// AgingBucketIndexFor returns the bucket index for a day count past due.
func AgingBucketIndexFor(daysOverdue int) int {
	for i, upper := range agingBucketUpperDays {
		if daysOverdue <= upper {
			return i
		}
	}
	return len(agingBucketUpperDays)
}

// AgingBucketIndexOf returns the index of a bucket key, or -1 if the key names
// no bucket.
func AgingBucketIndexOf(key string) int {
	for i, candidate := range agingBucketKeys {
		if candidate == key {
			return i
		}
	}
	return -1
}

// AgingBucketKeyAt maps a bucket index to its key, falling back to the
// open-ended bucket for an index out of range (a repository that returns an
// unexpected index must not silently produce an empty bucket label).
func AgingBucketKeyAt(index int) string {
	if index < 0 || index >= len(agingBucketKeys) {
		return agingBucketKeys[len(agingBucketKeys)-1]
	}
	return agingBucketKeys[index]
}

// ErrUnknownAgingBucket is returned when a caller asks for a bucket key that is
// not one of the four above. The service validates the key at the boundary; the
// repository repeats the check so a future caller cannot turn a typo into an
// empty receivables list.
var ErrUnknownAgingBucket = errors.New("unknown aging bucket")

// OpenItemFilter selects and pages the open-items list. It lives here rather
// than in the dunning package so the invoice repository can satisfy the reader
// interface without importing dunning.
type OpenItemFilter struct {
	// AsOf is the reference date for days-overdue and bucket classification.
	// Passed in rather than taken from NOW() in SQL so the aging of a given data
	// set is reproducible in a test.
	AsOf time.Time
	// Bucket restricts the page to one aging bucket key (empty = all buckets).
	Bucket string
	// OverdueOnly drops items that are not past their due date yet.
	OverdueOnly bool
	Limit       int
	Offset      int
}

// OpenItem is one unpaid receivable: a sent or overdue invoice with its
// remaining amount. OpenAmount is the invoice gross total minus everything
// already recorded against it in finance_payments — an invoice with a partial
// payment is still an open item, for the residual only.
type OpenItem struct {
	InvoiceID     uuid.UUID       `json:"invoice_id"`
	InvoiceNumber string          `json:"invoice_number"`
	Status        string          `json:"status"`
	CustomerName  string          `json:"customer_name"`
	CustomerEmail string          `json:"customer_email"`
	Currency      string          `json:"currency"`
	GrossTotal    decimal.Decimal `json:"gross_total"`
	PaidAmount    decimal.Decimal `json:"paid_amount"`
	OpenAmount    decimal.Decimal `json:"open_amount"`
	InvoiceDate   time.Time       `json:"invoice_date"`
	DueDate       time.Time       `json:"due_date"`
	DaysOverdue   int             `json:"days_overdue"`
	AgingBucket   string          `json:"aging_bucket"`
	DunningLevel  int             `json:"dunning_level"`
	DunningStatus string          `json:"dunning_status,omitempty"`
	LastDunnedAt  *time.Time      `json:"last_dunned_at,omitempty"`
	ContactID     *uuid.UUID      `json:"contact_id,omitempty"`
}

// OpenItemBucketTotal is the aggregate of one aging bucket in one currency,
// over all open items of the tenant — not just the requested page.
type OpenItemBucketTotal struct {
	Currency string          `json:"currency"`
	Bucket   string          `json:"bucket"`
	Count    int             `json:"count"`
	Amount   decimal.Decimal `json:"amount"`
	// DaysOverdueSum is the summed day count of the bucket's items, used to
	// derive the average. Repository-internal, never on the wire.
	DaysOverdueSum int `json:"-"`
	// BucketIndex is what the repository actually groups by; the key above is
	// resolved from it in the service.
	BucketIndex int `json:"-"`
}

// OpenItemCurrencyTotal sums a tenant's receivables in one currency. There is
// one entry per currency in use: without a stored exchange rate the backend has
// no honest way to fold CHF mirrors and EUR invoices into a single figure.
type OpenItemCurrencyTotal struct {
	Currency       string          `json:"currency"`
	OpenAmount     decimal.Decimal `json:"open_amount"`
	OpenCount      int             `json:"open_count"`
	OverdueAmount  decimal.Decimal `json:"overdue_amount"`
	OverdueCount   int             `json:"overdue_count"`
	AvgDaysOverdue int             `json:"avg_days_overdue"`
}

// OpenItemsSummary carries the tenant-wide aggregates alongside a page of items.
type OpenItemsSummary struct {
	Totals  []*OpenItemCurrencyTotal `json:"totals"`
	Buckets []*OpenItemBucketTotal   `json:"buckets"`
}

// ============================================================================
// Bank statement import (Zahlungsabgleich) — Migration 000247
// ============================================================================

// Bank statement file formats accepted by the import.
const (
	BankStatementFormatCAMT053 = "camt053"
	BankStatementFormatMT940   = "mt940"
)

// Reconciliation state of a single bank transaction.
const (
	// BankMatchUnmatched: nothing plausible found.
	BankMatchUnmatched = "unmatched"
	// BankMatchSuggested: exactly one candidate, but not certain enough to book
	// on its own. A human confirms it.
	BankMatchSuggested = "suggested"
	// BankMatchMatched: booked against an invoice, with a payment recorded.
	BankMatchMatched = "matched"
	// BankMatchIgnored: deliberately taken out of the queue (own transfer, fee).
	BankMatchIgnored = "ignored"
)

// BankStatement is one imported account statement file.
type BankStatement struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Format   string    `json:"format"`
	Filename string    `json:"filename"`
	// ContentHash is the SHA-256 of the uploaded bytes and the idempotency key
	// of the import: the same file twice resolves to this same statement.
	ContentHash      string           `json:"content_hash"`
	AccountIBAN      string           `json:"account_iban"`
	StatementRef     string           `json:"statement_ref"`
	Currency         string           `json:"currency"`
	OpeningBalance   *decimal.Decimal `json:"opening_balance,omitempty"`
	ClosingBalance   *decimal.Decimal `json:"closing_balance,omitempty"`
	StatementDate    *time.Time       `json:"statement_date,omitempty"`
	TransactionCount int              `json:"transaction_count"`
	ImportedBy       *uuid.UUID       `json:"imported_by,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// BankTransaction is one entry of a statement plus its reconciliation state.
// Everything down to RemittanceInfo is what the bank reported and is never
// rewritten; the match fields below carry what this system made of it.
type BankTransaction struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	StatementID uuid.UUID  `json:"statement_id"`
	EntryRef    string     `json:"entry_ref"`
	EndToEndID  string     `json:"end_to_end_id"`
	ValueDate   time.Time  `json:"value_date"`
	BookingDate *time.Time `json:"booking_date,omitempty"`
	// Amount is signed: a credit is positive, a debit negative.
	Amount           decimal.Decimal `json:"amount"`
	Currency         string          `json:"currency"`
	CounterpartyName string          `json:"counterparty_name"`
	CounterpartyIBAN string          `json:"counterparty_iban"`
	RemittanceInfo   string          `json:"remittance_info"`
	MatchStatus      string          `json:"match_status"`
	MatchReason      string          `json:"match_reason"`
	MatchedInvoiceID *uuid.UUID      `json:"matched_invoice_id,omitempty"`
	// MatchedInvoiceNumber is joined on read and never written: the number
	// belongs to the invoice, and a copy here would age the moment that invoice
	// is renumbered.
	MatchedInvoiceNumber string     `json:"matched_invoice_number,omitempty"`
	PaymentID            *uuid.UUID `json:"payment_id,omitempty"`
	ReconciledAt         *time.Time `json:"reconciled_at,omitempty"`
	ReconciledBy         *uuid.UUID `json:"reconciled_by,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

// IsCredit reports whether the entry moved money into the account. Only credits
// can settle a receivable.
func (t *BankTransaction) IsCredit() bool {
	return t.Amount.IsPositive()
}

// BankTransactionFilter selects and pages the transaction list.
type BankTransactionFilter struct {
	// StatementID restricts to one import (nil = across all statements).
	StatementID *uuid.UUID
	// MatchStatus restricts to one reconciliation state (empty = all).
	MatchStatus string
	// ExcludeMatchStatus drops states from an otherwise unfiltered list. The
	// reconciliation queue uses it to hide entries someone deliberately set
	// aside without hiding them from a caller that asks for them by name.
	ExcludeMatchStatus []string
	Limit              int
	Offset             int
}
