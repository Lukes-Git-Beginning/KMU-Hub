// Package creditnote provides the business logic for managing credit notes (Gutschriften).
// Credit notes reference an original invoice and use their own GS-prefix number sequence.
// They are created as drafts and sent with a gap-free sequential number (GS-2026-0001).
package creditnote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles credit note business logic.
type Service struct {
	repo          Repository
	invoiceReader InvoiceReader
	numberSeqRepo NumberSequenceRepo
}

// NewService creates a new credit note service.
func NewService(
	repo Repository,
	invoiceReader InvoiceReader,
	numberSeqRepo NumberSequenceRepo,
) *Service {
	return &Service{
		repo:          repo,
		invoiceReader: invoiceReader,
		numberSeqRepo: numberSeqRepo,
	}
}

// CreateInput contains the data needed to create a credit note.
type CreateInput struct {
	TenantID          uuid.UUID
	OriginalInvoiceID uuid.UUID
	LineItems         []models.LineItem
	TaxMode           string
	Reason            string
	UserID            uuid.UUID
}

// Create creates a new draft credit note against an existing invoice.
// The original invoice must be in sent, paid, or overdue status.
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.CreditNote, error) {
	if len(input.LineItems) == 0 {
		return nil, ErrNoLineItems
	}

	// Validate original invoice exists and is in a valid state
	inv, err := s.invoiceReader.GetByID(ctx, input.TenantID, input.OriginalInvoiceID)
	if err != nil {
		return nil, fmt.Errorf("get original invoice: %w", err)
	}

	switch inv.Status {
	case models.InvoiceStatusSent, models.InvoiceStatusPaid, models.InvoiceStatusOverdue:
		// Valid states for credit note creation
	default:
		return nil, ErrInvalidInvoiceForCredit
	}

	// Calculate tax using the specified mode
	taxMode := parseTaxMode(input.TaxMode)
	taxItems := toTaxLineItems(input.LineItems)
	breakdown := tax.Calculate(taxItems, taxMode)

	// Apply calculated line totals
	for i := range input.LineItems {
		input.LineItems[i].LineTotal = input.LineItems[i].Quantity.Mul(input.LineItems[i].UnitPrice)
	}

	// Marshal for JSONB storage
	lineItemsJSON, err := marshalLineItems(input.LineItems)
	if err != nil {
		return nil, err
	}

	taxBreakdown := &models.TaxBreakdown{
		Subtotal:   breakdown.Subtotal,
		TaxByRate:  breakdown.TaxByRate,
		TotalTax:   breakdown.TotalTax,
		GrossTotal: breakdown.GrossTotal,
	}
	taxBreakdownJSON, err := marshalTaxBreakdown(taxBreakdown)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cn := &models.CreditNote{
		ID:                uuid.New(),
		TenantID:          input.TenantID,
		CreditNoteNumber:  "", // Assigned when sent
		Status:            models.CreditNoteStatusDraft,
		OriginalInvoiceID: input.OriginalInvoiceID,
		CustomerName:      inv.CustomerName,
		CustomerAddress:   inv.CustomerAddress,
		CustomerEmail:     inv.CustomerEmail,
		CustomerUStIDNr:   inv.CustomerUStIDNr,
		TaxMode:           input.TaxMode,
		LineItems:         lineItemsJSON,
		TaxBreakdownRaw:   taxBreakdownJSON,
		Subtotal:          breakdown.Subtotal,
		TotalTax:          breakdown.TotalTax,
		GrossTotal:        breakdown.GrossTotal,
		Reason:            input.Reason,
		CreatedBy:         input.UserID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if createErr := s.repo.Create(ctx, cn); createErr != nil {
		return nil, createErr
	}

	slog.Info("credit note created",
		"credit_note_id", cn.ID,
		"tenant_id", cn.TenantID,
		"original_invoice_id", cn.OriginalInvoiceID,
		"gross_total", cn.GrossTotal,
	)

	return cn, nil
}

// GetByID retrieves a credit note by ID.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.CreditNote, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List retrieves credit notes with optional filtering.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.CreditNote, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

// Send transitions a draft credit note to sent and assigns a gap-free GS number.
// Format: GS-{year}-{padded} e.g., GS-2026-0001
func (s *Service) Send(ctx context.Context, tenantID, id, userID uuid.UUID) error {
	cn, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if cn.Status != models.CreditNoteStatusDraft {
		return ErrCreditNoteNotDraft
	}

	// Assign sequential credit note number via gap-free sequence
	fiscalYear := time.Now().Year()
	number, numErr := s.numberSeqRepo.NextNumber(ctx, tenantID, models.DocumentTypeCreditNote, fiscalYear, "GS")
	if numErr != nil {
		return fmt.Errorf("assign credit note number: %w", numErr)
	}

	cn.CreditNoteNumber = number
	cn.Status = models.CreditNoteStatusSent
	cn.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, cn); updateErr != nil {
		return updateErr
	}

	slog.Info("credit note sent",
		"credit_note_id", cn.ID,
		"tenant_id", tenantID,
		"credit_note_number", number,
		"gross_total", cn.GrossTotal,
	)
	return nil
}

// parseTaxMode converts a string tax mode to the tax package TaxMode type.
func parseTaxMode(mode string) tax.TaxMode {
	switch mode {
	case models.TaxModeReverseCharge:
		return tax.ModeReverseCharge
	case models.TaxModeKleinunternehmer:
		return tax.ModeKleinunternehmer
	default:
		return tax.ModeStandard
	}
}

// toTaxLineItems converts model line items to tax calculation line items.
func toTaxLineItems(items []models.LineItem) []tax.LineItem {
	taxItems := make([]tax.LineItem, len(items))
	for i, item := range items {
		taxItems[i] = tax.LineItem{
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			TaxRate:   item.TaxRate,
		}
	}
	return taxItems
}

// marshalLineItems converts model line items to JSON for storage.
func marshalLineItems(items []models.LineItem) (json.RawMessage, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal line items: %w", err)
	}
	return data, nil
}

// marshalTaxBreakdown converts a TaxBreakdown to JSON for storage.
func marshalTaxBreakdown(tb *models.TaxBreakdown) (json.RawMessage, error) {
	if tb == nil {
		return nil, nil
	}
	data, err := json.Marshal(tb)
	if err != nil {
		return nil, fmt.Errorf("marshal tax breakdown: %w", err)
	}
	return data, nil
}
