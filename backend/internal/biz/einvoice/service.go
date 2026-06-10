package einvoice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles business logic for inbound electronic invoices.
type Service struct {
	repo Repository
}

// NewService creates a new einvoice Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Import
// ============================================================================

// ImportInput contains everything needed to import an inbound e-invoice.
type ImportInput struct {
	TenantID  uuid.UUID
	CreatedBy uuid.UUID
	Filename  string
	Content   []byte // PDF or XML bytes
	MimeType  string // "application/pdf", "application/xml", "text/xml"
}

// Import processes an inbound e-invoice file (PDF or XML), parses it, and
// persists it to finance_incoming_invoices.
//
// For PDFs: extracts the embedded XML first (returns ErrNoEmbeddedXML if absent).
// For XML:  used directly.
//
// Returns ErrDuplicateInvoice if the same supplier+invoice_number already exists
// for this tenant.
func (s *Service) Import(ctx context.Context, input ImportInput) (*models.IncomingInvoice, error) {
	var xmlBytes []byte

	switch {
	case strings.Contains(input.MimeType, "pdf"):
		var err error
		xmlBytes, err = ExtractXMLFromPDF(input.Content)
		if err != nil {
			return nil, err
		}
	case strings.Contains(input.MimeType, "xml"):
		xmlBytes = input.Content
	default:
		// Try PDF extraction first, then treat as XML
		extracted, err := ExtractXMLFromPDF(input.Content)
		if err == nil {
			xmlBytes = extracted
		} else {
			xmlBytes = input.Content
		}
	}

	format := DetectFormat(xmlBytes)

	var parsed *ParsedInvoice
	var parseErr error
	switch format {
	case "zugferd_cii":
		parsed, parseErr = ParseCII(xmlBytes)
	case "xrechnung_ubl":
		parsed, parseErr = ParseUBL(xmlBytes)
	default:
		// Attempt CII parse on xml_only; fall back to UBL; else error
		parsed, parseErr = ParseCII(xmlBytes)
		if parseErr != nil {
			parsed, parseErr = ParseUBL(xmlBytes)
		}
		if parseErr != nil {
			return nil, ErrUnsupportedFormat
		}
		format = parsed.SourceFormat
		if format == "" {
			format = "xml_only"
		}
	}
	if parseErr != nil {
		return nil, parseErr
	}

	// Marshal line items and tax breakdown as JSONB
	lineItemsJSON, err := marshalLineItems(parsed.LineItems)
	if err != nil {
		return nil, fmt.Errorf("marshal line items: %w", err)
	}
	taxBreakdownJSON, err := marshalTaxBreakdown(parsed.TaxBreakdown)
	if err != nil {
		return nil, fmt.Errorf("marshal tax breakdown: %w", err)
	}

	now := time.Now().UTC()
	inv := &models.IncomingInvoice{
		ID:               uuid.New(),
		TenantID:         input.TenantID,
		SupplierName:     parsed.SupplierName,
		SupplierAddress:  parsed.SupplierAddress,
		SupplierVATID:    parsed.SupplierVATID,
		InvoiceNumber:    parsed.InvoiceNumber,
		InvoiceDate:      parsed.InvoiceDate,
		DueDate:          parsed.DueDate,
		Currency:         parsed.Currency,
		LineItems:        lineItemsJSON,
		TaxBreakdownRaw:  taxBreakdownJSON,
		Subtotal:         parsed.Subtotal,
		TotalTax:         parsed.TotalTax,
		GrossTotal:       parsed.GrossTotal,
		SourceFormat:     format,
		OriginalXML:      string(xmlBytes),
		OriginalFilename: input.Filename,
		Status:           models.IncomingInvoiceStatusReceived,
		CreatedBy:        input.CreatedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, err
	}

	slog.Info("incoming invoice imported",
		"id", inv.ID,
		"tenant_id", input.TenantID,
		"supplier", inv.SupplierName,
		"invoice_number", inv.InvoiceNumber,
		"format", format,
	)

	return inv, nil
}

// ============================================================================
// Read
// ============================================================================

// Get retrieves an incoming invoice by ID for the given tenant.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.IncomingInvoice, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// ListInput contains pagination and filter parameters for List.
type ListInput struct {
	Status   string
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	PerPage  int
}

// List retrieves incoming invoices with optional filters and pagination.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, input ListInput) ([]*models.IncomingInvoice, int, error) {
	limit := input.PerPage
	if limit <= 0 {
		limit = 50
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	return s.repo.List(ctx, tenantID, ListFilter{
		Status:   input.Status,
		DateFrom: input.DateFrom,
		DateTo:   input.DateTo,
		Limit:    limit,
		Offset:   offset,
	})
}

// ============================================================================
// Status update
// ============================================================================

// validTransitions defines which status changes are allowed.
// "booked" is a terminal write-only state reachable only from "reviewed".
var validTransitions = map[string][]string{
	models.IncomingInvoiceStatusReceived: {
		models.IncomingInvoiceStatusReviewed,
		models.IncomingInvoiceStatusRejected,
	},
	models.IncomingInvoiceStatusReviewed: {
		models.IncomingInvoiceStatusBooked,
		models.IncomingInvoiceStatusRejected,
	},
	// booked and rejected are terminal
}

// UpdateStatus transitions an incoming invoice to a new status.
//
// Allowed transitions:
//   - received → reviewed | rejected
//   - reviewed → booked | rejected
//   - booked, rejected: terminal (no further transitions)
func (s *Service) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, newStatus string) (*models.IncomingInvoice, error) {
	inv, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	allowed := validTransitions[inv.Status]
	if !contains(allowed, newStatus) {
		return nil, fmt.Errorf("%w: %s → %s", ErrInvalidStatusTransition, inv.Status, newStatus)
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateStatus(ctx, tenantID, id, newStatus, now); err != nil {
		return nil, err
	}

	inv.Status = newStatus
	inv.UpdatedAt = now
	return inv, nil
}

// ============================================================================
// Helpers
// ============================================================================

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// marshalLineItems converts ParsedLineItems to JSONB-compatible json.RawMessage.
func marshalLineItems(items []ParsedLineItem) (json.RawMessage, error) {
	type jsonItem struct {
		Position    int             `json:"position"`
		Description string          `json:"description"`
		Quantity    decimal.Decimal `json:"quantity"`
		UnitPrice   decimal.Decimal `json:"unit_price"`
		TaxRate     decimal.Decimal `json:"tax_rate"`
		LineTotal   decimal.Decimal `json:"line_total"`
	}
	out := make([]jsonItem, 0, len(items))
	for _, li := range items {
		out = append(out, jsonItem{
			Position:    li.Position,
			Description: li.Description,
			Quantity:    li.Quantity,
			UnitPrice:   li.UnitPrice,
			TaxRate:     li.TaxRate,
			LineTotal:   li.LineTotal,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// marshalTaxBreakdown converts ParsedTaxEntries to JSONB-compatible json.RawMessage.
func marshalTaxBreakdown(entries []ParsedTaxEntry) (json.RawMessage, error) {
	type jsonEntry struct {
		TaxRate    decimal.Decimal `json:"tax_rate"`
		TaxableNet decimal.Decimal `json:"taxable_net"`
		TaxAmount  decimal.Decimal `json:"tax_amount"`
	}
	out := make([]jsonEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, jsonEntry{
			TaxRate:    e.TaxRate,
			TaxableNet: e.TaxableNet,
			TaxAmount:  e.TaxAmount,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
