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
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
)

// txBeginner begins a transaction. Satisfied by *pgxpool.Pool in production and
// injected as a fake in unit tests so Send's orchestration can be exercised
// without a database (real atomicity is covered by the integration test).
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service handles credit note business logic.
type Service struct {
	repo          Repository
	invoiceReader InvoiceReader
	numberSeqRepo NumberSequenceRepo
	// pool backs the atomic Send transaction (number assignment + status/number
	// update in one tx). May be nil in unit tests that do not exercise Send.
	pool txBeginner
	// invoiceCanceller cancels the original invoice inside the Storno tx. Set via
	// SetInvoiceStatusUpdater; required only for StornoInvoice.
	invoiceCanceller InvoiceStatusUpdater
}

// SetInvoiceStatusUpdater wires the in-tx invoice canceller used by StornoInvoice.
// Kept out of NewService so existing constructor callers and tests are unchanged.
func (s *Service) SetInvoiceStatusUpdater(u InvoiceStatusUpdater) {
	s.invoiceCanceller = u
}

// NewService creates a new credit note service.
// pool is required for Send (atomic numbering); it may be nil in unit tests
// that do not call Send.
func NewService(
	repo Repository,
	invoiceReader InvoiceReader,
	numberSeqRepo NumberSequenceRepo,
	pool txBeginner,
) *Service {
	return &Service{
		repo:          repo,
		invoiceReader: invoiceReader,
		numberSeqRepo: numberSeqRepo,
		pool:          pool,
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

	// A credit note inherits the currency of the invoice it credits.
	currency := inv.Currency
	if currency == "" {
		currency = models.DefaultCurrency
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
		Currency:          currency,
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

// ListForDATEVExport returns one keyset-paged page of sent credit notes in
// [fromDate, toDate] for the DATEV export. See Repository.ListForDATEVExport.
func (s *Service) ListForDATEVExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time, afterDate *time.Time, afterID *uuid.UUID, limit int) ([]*models.CreditNote, error) {
	return s.repo.ListForDATEVExport(ctx, tenantID, fromDate, toDate, afterDate, afterID, limit)
}

// Send transitions a draft credit note to sent and assigns a gap-free GS number.
// Format: GS-{year}-{padded} e.g., GS-2026-0001
func (s *Service) Send(ctx context.Context, tenantID, id, userID uuid.UUID) error {
	if s.pool == nil {
		return fmt.Errorf("credit note service: pool not configured (required for Send)")
	}

	cn, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if cn.Status != models.CreditNoteStatusDraft {
		return ErrCreditNoteNotDraft
	}

	fiscalYear := time.Now().Year()

	// Assign the sequential number and persist the sent state in ONE transaction so
	// a failed update rolls the number assignment back (GoBD: no burned numbers).
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin send tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	number, numErr := s.numberSeqRepo.NextNumberInTx(ctx, tx, tenantID, models.DocumentTypeCreditNote, fiscalYear, "GS")
	if numErr != nil {
		return fmt.Errorf("assign credit note number: %w", numErr)
	}

	cn.CreditNoteNumber = number
	cn.Status = models.CreditNoteStatusSent
	cn.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateInTx(ctx, tx, cn); updateErr != nil {
		return updateErr
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit send tx: %w", commitErr)
	}

	slog.Info("credit note sent",
		"credit_note_id", cn.ID,
		"tenant_id", tenantID,
		"credit_note_number", number,
		"gross_total", cn.GrossTotal,
	)
	return nil
}

// StornoInvoice reverses an issued invoice GoBD-compliantly: it creates a full
// mirroring credit note (Stornorechnung) and, in a single transaction, finalizes
// it with a gap-free GS number AND flips the invoice to cancelled. GoBD §146
// forbids silently deleting an issued invoice — it must be reversed by a credit
// note. The credit note mirrors the invoice's line items with their original
// (positive) amounts; the credit-note document type is what makes it a reversal.
//
// Returns the finalized credit note's id and number. It refuses to act if the
// invoice already carries a sent credit note, so a partially-credited invoice is
// never silently over-credited (manual handling required there).
func (s *Service) StornoInvoice(ctx context.Context, tenantID, invoiceID, userID uuid.UUID) (uuid.UUID, string, error) {
	if s.pool == nil {
		return uuid.Nil, "", fmt.Errorf("credit note service: pool not configured (required for Storno)")
	}
	if s.invoiceCanceller == nil {
		return uuid.Nil, "", fmt.Errorf("credit note service: invoice status updater not configured (required for Storno)")
	}

	inv, err := s.invoiceReader.GetByID(ctx, tenantID, invoiceID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("load invoice for storno: %w", err)
	}

	// Guard: never auto-storno on top of an existing sent credit note.
	existing, exErr := s.repo.GetByInvoiceID(ctx, tenantID, invoiceID)
	if exErr != nil {
		return uuid.Nil, "", fmt.Errorf("check existing credit notes: %w", exErr)
	}
	for _, c := range existing {
		if c.Status == models.CreditNoteStatusSent {
			return uuid.Nil, "", ErrInvoiceAlreadyCredited
		}
	}

	var items []models.LineItem
	if umErr := json.Unmarshal(inv.LineItems, &items); umErr != nil {
		return uuid.Nil, "", fmt.Errorf("unmarshal invoice line items: %w", umErr)
	}

	// Create the reversing credit note as a draft (mirrors the invoice 1:1).
	cn, err := s.Create(ctx, CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: invoiceID,
		LineItems:         items,
		TaxMode:           inv.TaxMode,
		Reason:            fmt.Sprintf("Storno der Rechnung %s", inv.InvoiceNumber),
		UserID:            userID,
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("create storno credit note: %w", err)
	}

	// Finalize the credit note (gap-free GS number) AND cancel the invoice in ONE
	// transaction so the books are never left half-reversed.
	fiscalYear := time.Now().Year()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("begin storno tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	number, numErr := s.numberSeqRepo.NextNumberInTx(ctx, tx, tenantID, models.DocumentTypeCreditNote, fiscalYear, "GS")
	if numErr != nil {
		return uuid.Nil, "", fmt.Errorf("assign storno number: %w", numErr)
	}

	cn.CreditNoteNumber = number
	cn.Status = models.CreditNoteStatusSent
	cn.UpdatedAt = time.Now()
	if updErr := s.repo.UpdateInTx(ctx, tx, cn); updErr != nil {
		return uuid.Nil, "", fmt.Errorf("finalize storno credit note: %w", updErr)
	}

	if cancelErr := s.invoiceCanceller.UpdateStatusInTx(ctx, tx, tenantID, invoiceID, models.InvoiceStatusCancelled); cancelErr != nil {
		return uuid.Nil, "", fmt.Errorf("cancel invoice in storno tx: %w", cancelErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return uuid.Nil, "", fmt.Errorf("commit storno tx: %w", commitErr)
	}

	slog.Info("invoice reversed by storno credit note",
		"invoice_id", invoiceID,
		"credit_note_id", cn.ID,
		"credit_note_number", number,
		"tenant_id", tenantID,
	)
	return cn.ID, number, nil
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
