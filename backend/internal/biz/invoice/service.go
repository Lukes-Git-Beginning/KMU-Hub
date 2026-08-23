// Package invoice provides the business logic for managing invoices (Rechnungen).
// Invoices enforce GoBD compliance: once sent, they become immutable with a complete
// JSONB snapshot (line items, tax breakdown, customer data, company data). Sequential
// numbering is gap-free using SELECT FOR UPDATE on finance_number_sequences.
package invoice

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
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// txBeginner begins a transaction. Satisfied by *pgxpool.Pool in production and
// injected as a fake in unit tests so Send's orchestration can be exercised
// without a database (real atomicity is covered by the integration test).
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// StornoCreator reverses an issued invoice by creating and finalizing a mirroring
// credit note and cancelling the invoice atomically. Implemented by
// creditnote.Service (structural — no import either way). Injected so Cancel can
// enforce GoBD §146: a sent invoice is never silently flipped to cancelled, it is
// reversed by a Stornorechnung. Returns the credit note id and number.
type StornoCreator interface {
	StornoInvoice(ctx context.Context, tenantID, invoiceID, userID uuid.UUID) (uuid.UUID, string, error)
}

// Service handles invoice business logic with GoBD compliance.
type Service struct {
	repo            Repository
	numberSeqRepo   NumberSequenceRepo
	companySettings CompanySettingsRepo
	quoteReader     QuoteReader
	emitter         EventEmitter
	// pool backs the atomic Send transaction (number assignment + status/number
	// update in one tx). May be nil in unit tests that do not exercise Send.
	pool txBeginner
	// stornoCreator reverses an issued invoice via a credit note. Set via
	// SetStornoCreator; required to cancel a sent/overdue invoice.
	stornoCreator StornoCreator
}

// SetStornoCreator wires the credit-note-backed Storno used to cancel issued
// invoices. Kept out of NewService so existing constructor callers and tests are
// unchanged.
func (s *Service) SetStornoCreator(c StornoCreator) {
	s.stornoCreator = c
}

// NewService creates a new invoice service.
// quoteReader may be nil if quote-to-invoice conversion is not needed.
// pool is required for Send (atomic numbering); it may be nil in unit tests
// that do not call Send.
func NewService(
	repo Repository,
	numberSeqRepo NumberSequenceRepo,
	companySettings CompanySettingsRepo,
	quoteReader QuoteReader,
	pool txBeginner,
) *Service {
	return &Service{
		repo:            repo,
		numberSeqRepo:   numberSeqRepo,
		companySettings: companySettings,
		quoteReader:     quoteReader,
		pool:            pool,
	}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.emitter = emitter
}

// emitEvent emits a notification event if the emitter is configured.
// Nil-safe: does nothing if no emitter is set.
func (s *Service) emitEvent(ctx context.Context, eventType, actorID, resourceID string, targetUserIDs []string, title, body, deepLink string) {
	if s.emitter == nil {
		return
	}
	payload := models.EventPayload{
		Type:          eventType,
		Priority:      "normal",
		ActorID:       actorID,
		ModuleID:      event.ModuleBiz,
		ResourceID:    resourceID,
		TargetUserIDs: targetUserIDs,
		Title:         title,
		Body:          body,
		DeepLink:      deepLink,
		Timestamp:     time.Now(),
	}
	if err := s.emitter.EmitBizEvent(ctx, payload); err != nil {
		slog.Error("failed to emit biz event",
			"type", eventType,
			"resource_id", resourceID,
			"error", err,
		)
	}
}

// CreateInput contains the data needed to create an invoice.
type CreateInput struct {
	TenantID         uuid.UUID
	CustomerName     string
	CustomerAddress  string
	CustomerEmail    string
	CustomerUStIDNr  string
	TaxMode          string
	LineItems        []models.LineItem
	InvoiceDate      time.Time
	DeliveryDate     *time.Time
	PaymentTermsDays int
	Notes            string
	SourceQuoteID    *uuid.UUID
	// ContactID links this invoice to a CRM contact for Contact-360 view.
	ContactID *uuid.UUID
	// RecurringID marks this invoice as emitted by a recurring schedule
	// (Migration 000246). Set by recurring.Service.Generate, nil otherwise.
	RecurringID *uuid.UUID
	// Currency overrides the tenant default (ISO 4217). Empty means "use the
	// company setting" — a recurring schedule bills in its own currency, which
	// may differ from the tenant default.
	Currency string
	UserID   uuid.UUID
}

// Create creates a new draft invoice with tax calculation.
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.Invoice, error) {
	if len(input.LineItems) == 0 {
		return nil, ErrNoLineItems
	}

	// Calculate tax
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

	// Load company settings once (best-effort) for the payment-terms and currency
	// defaults below.
	settings, settingsErr := s.companySettings.GetByTenantID(ctx, input.TenantID)
	if settingsErr != nil {
		slog.Warn("failed to load company settings for invoice defaults",
			"tenant_id", input.TenantID,
			"error", settingsErr,
		)
	}

	// Determine payment terms and due date
	paymentTermsDays := input.PaymentTermsDays
	if paymentTermsDays <= 0 {
		if settings != nil && settings.DefaultPaymentTermsDays > 0 {
			paymentTermsDays = settings.DefaultPaymentTermsDays
		} else {
			paymentTermsDays = 30 // Fallback default
		}
	}

	// Currency: explicit input wins, then the tenant default from company
	// settings, then the system default (EUR).
	currency := models.DefaultCurrency
	if settings != nil && settings.DefaultCurrency != "" {
		currency = settings.DefaultCurrency
	}
	if input.Currency != "" {
		currency = input.Currency
	}

	invoiceDate := input.InvoiceDate
	if invoiceDate.IsZero() {
		invoiceDate = time.Now()
	}
	dueDate := invoiceDate.AddDate(0, 0, paymentTermsDays)

	paymentTerms := fmt.Sprintf("%d Tage netto", paymentTermsDays)

	now := time.Now()
	inv := &models.Invoice{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		InvoiceNumber:   "", // Assigned when sent
		Status:          models.InvoiceStatusDraft,
		CustomerName:    input.CustomerName,
		CustomerAddress: input.CustomerAddress,
		CustomerEmail:   input.CustomerEmail,
		CustomerUStIDNr: input.CustomerUStIDNr,
		TaxMode:         input.TaxMode,
		LineItems:       lineItemsJSON,
		TaxBreakdownRaw: taxBreakdownJSON,
		Subtotal:        breakdown.Subtotal,
		TotalTax:        breakdown.TotalTax,
		GrossTotal:      breakdown.GrossTotal,
		Currency:        currency,
		InvoiceDate:     invoiceDate,
		DeliveryDate:    input.DeliveryDate,
		DueDate:         dueDate,
		PaymentTerms:    paymentTerms,
		SourceQuoteID:   input.SourceQuoteID,
		Notes:           input.Notes,
		ContactID:       input.ContactID,
		RecurringID:     input.RecurringID,
		CreatedBy:       input.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if createErr := s.repo.Create(ctx, inv); createErr != nil {
		return nil, createErr
	}

	slog.Info("invoice created",
		"invoice_id", inv.ID,
		"tenant_id", inv.TenantID,
		"gross_total", inv.GrossTotal,
		"source_quote_id", inv.SourceQuoteID,
	)

	s.emitEvent(ctx, event.EventInvoiceCreated, input.UserID.String(), inv.ID.String(), nil,
		"Rechnung erstellt", inv.CustomerName, "/finanzen/rechnungen/"+inv.ID.String())

	return inv, nil
}

// UpsertImported creates or updates a read-only invoice mirror pulled from an external
// accounting system (source='bexio'). It deliberately bypasses Create/Send and the
// RE-YYYY-NNNN sequence: the invoice keeps its external number, so the GoBD gap-free
// journal (source='cosmi') is never touched. Idempotent per (tenant_id, source,
// external_id). Imported invoices are immutable in Cosmi afterwards (ErrExternalReadOnly).
func (s *Service) UpsertImported(ctx context.Context, tenantID uuid.UUID, in models.ImportedInvoiceInput) (*models.Invoice, error) {
	if in.ExternalID == "" {
		return nil, fmt.Errorf("import invoice: external id required")
	}
	// The external document number becomes invoice_number, which is unique per tenant.
	// Requiring it also keeps number-less external drafts out of the mirror — only
	// issued external invoices are mirrored.
	if in.ExternalNumber == "" {
		return nil, fmt.Errorf("import invoice %s: external number required", in.ExternalID)
	}

	status := in.Status
	if status == "" {
		status = models.InvoiceStatusSent
	}

	invoiceDate := in.InvoiceDate
	if invoiceDate.IsZero() {
		invoiceDate = time.Now()
	}
	dueDate := invoiceDate
	if in.DueDate != nil {
		dueDate = *in.DueDate
	}

	currency := in.Currency
	if currency == "" {
		currency = models.DefaultCurrency
	}

	lineItemsJSON, err := marshalLineItems(toLineItems(in.Lines))
	if err != nil {
		return nil, err
	}

	var contactID *uuid.UUID
	if in.ContactID != uuid.Nil {
		cid := in.ContactID
		contactID = &cid
	}

	externalID := in.ExternalID
	externalNumber := in.ExternalNumber
	now := time.Now()
	inv := &models.Invoice{
		ID:              uuid.New(),
		TenantID:        tenantID,
		InvoiceNumber:   in.ExternalNumber,
		Status:          status,
		Source:          models.InvoiceSourceBexio,
		ExternalID:      &externalID,
		ExternalNumber:  &externalNumber,
		CustomerName:    in.CustomerName,
		CustomerAddress: in.CustomerAddress,
		CustomerEmail:   in.CustomerEmail,
		TaxMode:         models.TaxModeStandard,
		LineItems:       lineItemsJSON,
		Subtotal:        in.Subtotal,
		TotalTax:        in.TotalTax,
		GrossTotal:      in.GrossTotal,
		Currency:        currency,
		InvoiceDate:     invoiceDate,
		DueDate:         dueDate,
		ContactID:       contactID,
		CreatedBy:       uuid.Nil, // system/sync import (created_by is NOT NULL, no FK)
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if upsertErr := s.repo.UpsertImported(ctx, inv); upsertErr != nil {
		return nil, upsertErr
	}

	slog.Info("bexio invoice imported (read-only mirror)",
		"invoice_id", inv.ID,
		"tenant_id", tenantID,
		"external_id", in.ExternalID,
		"external_number", in.ExternalNumber,
		"status", inv.Status,
	)

	return inv, nil
}

// GetByID retrieves an invoice by ID.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// LinkTimeTracking attaches a time_tracking_source JSONB payload to an invoice.
// This is best-effort: callers should treat errors as non-fatal.
func (s *Service) LinkTimeTracking(ctx context.Context, tenantID, invoiceID uuid.UUID, src json.RawMessage) error {
	return s.repo.LinkTimeTracking(ctx, tenantID, invoiceID, src)
}

// List retrieves invoices with optional filtering.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Invoice, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

// ListDocumentChains returns every document chain of the tenant. See
// Repository.ListDocumentChains.
func (s *Service) ListDocumentChains(ctx context.Context, tenantID uuid.UUID) ([]*models.DocumentChain, error) {
	return s.repo.ListDocumentChains(ctx, tenantID)
}

// ListTransactions returns the tenant's consolidated payment ledger. See
// Repository.ListTransactions.
func (s *Service) ListTransactions(ctx context.Context, tenantID uuid.UUID) ([]*models.FinanceTransaction, error) {
	return s.repo.ListTransactions(ctx, tenantID)
}

// ListForDATEVExport returns one keyset-paged page of sent/paid/overdue invoices in
// [fromDate, toDate] for the DATEV export. See Repository.ListForDATEVExport.
func (s *Service) ListForDATEVExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time, afterDate *time.Time, afterID *uuid.UUID, limit int) ([]*models.Invoice, error) {
	return s.repo.ListForDATEVExport(ctx, tenantID, fromDate, toDate, afterDate, afterID, limit)
}

// UpdateInput contains the data that can be updated on a draft invoice.
type UpdateInput struct {
	CustomerName     *string
	CustomerAddress  *string
	CustomerEmail    *string
	CustomerUStIDNr  *string
	TaxMode          *string
	LineItems        []models.LineItem // nil = no change, empty = error
	InvoiceDate      *time.Time
	DeliveryDate     *time.Time
	PaymentTermsDays *int
	Notes            *string
}

// Update updates an existing invoice. ONLY draft invoices can be modified (GoBD enforcement).
// If the invoice has been sent, it is IMMUTABLE and returns ErrInvoiceImmutable.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, input UpdateInput) (*models.Invoice, error) {
	inv, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Imported (source='bexio') invoices are read-only mirrors of the external book.
	if inv.Source == models.InvoiceSourceBexio {
		return nil, ErrExternalReadOnly
	}

	// GoBD enforcement: only draft invoices can be modified
	if inv.Status != models.InvoiceStatusDraft {
		return nil, ErrInvoiceImmutable
	}

	// GoBD §146: administratively locked invoices are technically immutable
	if isInvoiceLocked(inv) {
		return nil, ErrInvoiceLocked
	}

	if input.CustomerName != nil {
		inv.CustomerName = *input.CustomerName
	}
	if input.CustomerAddress != nil {
		inv.CustomerAddress = *input.CustomerAddress
	}
	if input.CustomerEmail != nil {
		inv.CustomerEmail = *input.CustomerEmail
	}
	if input.CustomerUStIDNr != nil {
		inv.CustomerUStIDNr = *input.CustomerUStIDNr
	}
	if input.Notes != nil {
		inv.Notes = *input.Notes
	}
	if input.InvoiceDate != nil {
		inv.InvoiceDate = *input.InvoiceDate
	}
	if input.DeliveryDate != nil {
		inv.DeliveryDate = input.DeliveryDate
	}

	// Recalculate due date if payment terms or invoice date changed
	if input.PaymentTermsDays != nil {
		days := *input.PaymentTermsDays
		inv.DueDate = inv.InvoiceDate.AddDate(0, 0, days)
		inv.PaymentTerms = fmt.Sprintf("%d Tage netto", days)
	}

	// Recalculate tax if line items or tax mode changed
	needsRecalc := false
	if input.LineItems != nil {
		if len(input.LineItems) == 0 {
			return nil, ErrNoLineItems
		}
		needsRecalc = true
	}
	if input.TaxMode != nil {
		inv.TaxMode = *input.TaxMode
		needsRecalc = true
	}

	if needsRecalc {
		lineItems := input.LineItems
		if lineItems == nil {
			existingItems, unmarshalErr := unmarshalLineItems(inv.LineItems)
			if unmarshalErr != nil {
				return nil, unmarshalErr
			}
			lineItems = existingItems
		}

		taxMode := parseTaxMode(inv.TaxMode)
		taxItems := toTaxLineItems(lineItems)
		breakdown := tax.Calculate(taxItems, taxMode)

		for i := range lineItems {
			lineItems[i].LineTotal = lineItems[i].Quantity.Mul(lineItems[i].UnitPrice)
		}

		lineItemsJSON, marshalErr := marshalLineItems(lineItems)
		if marshalErr != nil {
			return nil, marshalErr
		}

		taxBreakdown := &models.TaxBreakdown{
			Subtotal:   breakdown.Subtotal,
			TaxByRate:  breakdown.TaxByRate,
			TotalTax:   breakdown.TotalTax,
			GrossTotal: breakdown.GrossTotal,
		}
		taxBreakdownJSON, tbErr := marshalTaxBreakdown(taxBreakdown)
		if tbErr != nil {
			return nil, tbErr
		}

		inv.LineItems = lineItemsJSON
		inv.TaxBreakdownRaw = taxBreakdownJSON
		inv.Subtotal = breakdown.Subtotal
		inv.TotalTax = breakdown.TotalTax
		inv.GrossTotal = breakdown.GrossTotal
	}

	inv.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, inv); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("invoice updated",
		"invoice_id", inv.ID,
		"tenant_id", inv.TenantID,
		"gross_total", inv.GrossTotal,
	)

	return inv, nil
}

// Send transitions a draft invoice to sent. This is the IMMUTABILITY POINT:
// - Assigns gap-free sequential invoice number (RE-{year}-{padded}, e.g., RE-2026-0001)
// - Builds complete JSONB snapshot (line items, tax breakdown, customer data, company data)
// - Sets invoice_date if not already set
// After this call, the invoice can NEVER be modified (GoBD compliance).
func (s *Service) Send(ctx context.Context, tenantID, id, userID uuid.UUID) error {
	if s.pool == nil {
		return fmt.Errorf("invoice service: pool not configured (required for Send)")
	}

	inv, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Imported (source='bexio') invoices are read-only mirrors of the external book.
	if inv.Source == models.InvoiceSourceBexio {
		return ErrExternalReadOnly
	}

	if inv.Status != models.InvoiceStatusDraft {
		return ErrInvoiceNotDraft
	}

	// Load company settings for snapshot (read-only; the number is assigned in the
	// transaction below).
	settings, settingsErr := s.companySettings.GetByTenantID(ctx, tenantID)
	if settingsErr != nil {
		slog.Warn("failed to load company settings for invoice snapshot",
			"tenant_id", tenantID,
			"error", settingsErr,
		)
	}

	// Build company snapshot
	companySnapshotJSON, csErr := marshalCompanySnapshot(settings)
	if csErr != nil {
		return fmt.Errorf("build company snapshot: %w", csErr)
	}

	fiscalYear := time.Now().Year()

	// Assign the sequential number and persist the sent state in ONE transaction.
	// Previously NextNumber committed its own tx before repo.Update ran in a second
	// tx; a failed update left the number consumed (GoBD gap). Coupling both in one
	// tx makes the rollback atomic: a failed update returns the number.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin send tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	number, numErr := s.numberSeqRepo.NextNumberInTx(ctx, tx, tenantID, models.DocumentTypeInvoice, fiscalYear, "RE")
	if numErr != nil {
		return fmt.Errorf("assign invoice number: %w", numErr)
	}

	// Build complete invoice snapshot (the entire document at send time)
	snapshot := map[string]any{
		"invoice_number": number,
		"customer": models.CustomerSnapshot{
			Name:    inv.CustomerName,
			Address: inv.CustomerAddress,
			Email:   inv.CustomerEmail,
			UStIDNr: inv.CustomerUStIDNr,
		},
		"line_items":    inv.LineItems,
		"tax_breakdown": inv.TaxBreakdownRaw,
		"subtotal":      inv.Subtotal.String(),
		"total_tax":     inv.TotalTax.String(),
		"gross_total":   inv.GrossTotal.String(),
		"invoice_date":  inv.InvoiceDate.Format("2006-01-02"),
		"due_date":      inv.DueDate.Format("2006-01-02"),
		"payment_terms": inv.PaymentTerms,
		"tax_mode":      inv.TaxMode,
		"notes":         inv.Notes,
		"sent_at":       time.Now().Format(time.RFC3339),
		"sent_by":       userID.String(),
	}
	snapshotJSON, snErr := json.Marshal(snapshot)
	if snErr != nil {
		return fmt.Errorf("build invoice snapshot: %w", snErr)
	}

	// Set invoice date if not yet set
	if inv.InvoiceDate.IsZero() {
		inv.InvoiceDate = time.Now()
	}

	inv.InvoiceNumber = number
	inv.Status = models.InvoiceStatusSent
	inv.CompanySnapshotRaw = companySnapshotJSON
	inv.SnapshotData = snapshotJSON
	inv.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateInTx(ctx, tx, inv); updateErr != nil {
		return updateErr
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit send tx: %w", commitErr)
	}

	slog.Info("invoice sent (now immutable)",
		"invoice_id", inv.ID,
		"tenant_id", tenantID,
		"invoice_number", number,
		"gross_total", inv.GrossTotal,
	)

	s.emitEvent(ctx, event.EventInvoiceSent, userID.String(), inv.ID.String(), nil,
		"Rechnung versendet", inv.InvoiceNumber, "/finanzen/rechnungen/"+inv.ID.String())

	return nil
}

// MarkPaid transitions a sent or overdue invoice to paid.
func (s *Service) MarkPaid(ctx context.Context, tenantID, id uuid.UUID) error {
	inv, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Imported (source='bexio') invoices are read-only mirrors of the external book.
	if inv.Source == models.InvoiceSourceBexio {
		return ErrExternalReadOnly
	}

	switch inv.Status {
	case models.InvoiceStatusSent, models.InvoiceStatusOverdue:
		// Valid transitions
	case models.InvoiceStatusPaid:
		return ErrInvoiceAlreadyPaid
	case models.InvoiceStatusCancelled:
		return ErrInvoiceAlreadyCancelled
	default:
		return ErrInvoiceNotDraft
	}

	// GoBD §146: administratively locked invoices are technically immutable
	if isInvoiceLocked(inv) {
		return ErrInvoiceLocked
	}

	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, models.InvoiceStatusPaid); updateErr != nil {
		return updateErr
	}

	slog.Info("invoice marked paid",
		"invoice_id", id,
		"tenant_id", tenantID,
	)
	return nil
}

// Cancel cancels an invoice. A draft (never issued) is cancelled directly. A
// sent or overdue invoice has already been issued, so GoBD §146 forbids a silent
// status flip: it is reversed by a Stornorechnung (credit note) created and
// committed atomically with the cancellation via the injected StornoCreator.
// Paid invoices cannot be cancelled (issue a credit note manually instead).
func (s *Service) Cancel(ctx context.Context, tenantID, id, userID uuid.UUID) error {
	inv, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Imported (source='bexio') invoices are read-only mirrors of the external book.
	if inv.Source == models.InvoiceSourceBexio {
		return ErrExternalReadOnly
	}

	switch inv.Status {
	case models.InvoiceStatusDraft:
		// Never issued — safe to discard directly.
	case models.InvoiceStatusSent, models.InvoiceStatusOverdue:
		// Issued — must be reversed by a credit note (handled below).
	case models.InvoiceStatusPaid:
		return ErrInvoiceAlreadyPaid
	case models.InvoiceStatusCancelled:
		return ErrInvoiceAlreadyCancelled
	default:
		return fmt.Errorf("cannot cancel invoice with status %s", inv.Status)
	}

	// GoBD §146: administratively locked invoices are technically immutable.
	if isInvoiceLocked(inv) {
		return ErrInvoiceLocked
	}

	if inv.Status == models.InvoiceStatusSent || inv.Status == models.InvoiceStatusOverdue {
		if s.stornoCreator == nil {
			// Never silently flip an issued invoice when no Storno path is wired.
			return ErrStornoUnavailable
		}
		cnID, cnNumber, stornoErr := s.stornoCreator.StornoInvoice(ctx, tenantID, id, userID)
		if stornoErr != nil {
			return fmt.Errorf("storno invoice %s: %w", id, stornoErr)
		}
		slog.Info("issued invoice cancelled via storno credit note",
			"invoice_id", id,
			"tenant_id", tenantID,
			"credit_note_id", cnID,
			"credit_note_number", cnNumber,
			"cancelled_by", userID,
		)
		// The Storno tx already flipped the invoice to cancelled.
		return nil
	}

	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, models.InvoiceStatusCancelled); updateErr != nil {
		return updateErr
	}

	slog.Info("draft invoice cancelled",
		"invoice_id", id,
		"tenant_id", tenantID,
		"cancelled_by", userID,
	)
	return nil
}

// DetectOverdue bulk-updates all sent invoices past their due_date to overdue.
// Returns the number of invoices marked as overdue.
func (s *Service) DetectOverdue(ctx context.Context, tenantID uuid.UUID) (int, error) {
	overdueInvoices, err := s.repo.GetOverdue(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, inv := range overdueInvoices {
		// GoBD §146: an administratively locked invoice is technically immutable,
		// same barrier Update/MarkPaid/Cancel already enforce via isInvoiceLocked.
		// GetOverdue does not filter by locked_at (dunning.Service reuses it and
		// must still see a locked invoice as due), so the skip belongs here.
		if isInvoiceLocked(inv) {
			slog.Warn("skipped overdue transition for locked invoice",
				"invoice_id", inv.ID,
				"tenant_id", tenantID,
			)
			continue
		}
		if updateErr := s.repo.UpdateStatus(ctx, tenantID, inv.ID, models.InvoiceStatusOverdue); updateErr != nil {
			slog.Warn("failed to mark invoice as overdue",
				"invoice_id", inv.ID,
				"error", updateErr,
			)
			continue
		}
		count++
	}

	if count > 0 {
		slog.Info("invoices marked overdue",
			"tenant_id", tenantID,
			"count", count,
		)
	}

	return count, nil
}

// CreateFromQuote creates a new draft invoice from an accepted quote.
// Implements the copy-and-link pattern: copies line items and customer data,
// sets source_quote_id for traceability. Quote must have "accepted" status.
func (s *Service) CreateFromQuote(ctx context.Context, tenantID, quoteID, userID uuid.UUID) (*models.Invoice, error) {
	if s.quoteReader == nil {
		return nil, fmt.Errorf("quote reader not configured")
	}

	quote, err := s.quoteReader.GetByID(ctx, tenantID, quoteID)
	if err != nil {
		return nil, err
	}

	if quote.Status != models.QuoteStatusAccepted {
		return nil, ErrQuoteNotAccepted
	}

	// Parse the quote's line items for reuse
	lineItems, parseErr := unmarshalLineItems(quote.LineItems)
	if parseErr != nil {
		return nil, fmt.Errorf("parse quote line items: %w", parseErr)
	}

	// Determine payment terms from company settings
	paymentTermsDays := 30 // Default
	settings, settingsErr := s.companySettings.GetByTenantID(ctx, tenantID)
	if settingsErr != nil {
		slog.Warn("failed to load company settings for quote-to-invoice conversion",
			"tenant_id", tenantID,
			"error", settingsErr,
		)
	}
	if settings != nil && settings.DefaultPaymentTermsDays > 0 {
		paymentTermsDays = settings.DefaultPaymentTermsDays
	}

	input := CreateInput{
		TenantID:         tenantID,
		CustomerName:     quote.CustomerName,
		CustomerAddress:  quote.CustomerAddress,
		CustomerEmail:    quote.CustomerEmail,
		CustomerUStIDNr:  quote.CustomerUStIDNr,
		TaxMode:          quote.TaxMode,
		LineItems:        lineItems,
		InvoiceDate:      time.Now(),
		PaymentTermsDays: paymentTermsDays,
		Notes:            quote.Notes,
		SourceQuoteID:    &quoteID,
		UserID:           userID,
	}

	invoice, createErr := s.Create(ctx, input)
	if createErr != nil {
		return nil, createErr
	}

	slog.Info("invoice created from quote",
		"invoice_id", invoice.ID,
		"quote_id", quoteID,
		"tenant_id", tenantID,
	)

	return invoice, nil
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

// toLineItems converts imported invoice lines to model line items for storage.
// The line total is taken verbatim from the source system (no recalculation): imported
// invoices are read-only mirrors, not Cosmi-computed documents.
func toLineItems(lines []models.ImportedInvoiceLine) []models.LineItem {
	items := make([]models.LineItem, len(lines))
	for i, l := range lines {
		items[i] = models.LineItem{
			Position:    l.Position,
			Description: l.Description,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
			TaxRate:     l.TaxRate,
			LineTotal:   l.LineTotal,
		}
	}
	return items
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

// ParseLineItems is a convenience function for callers to parse JSONB line items.
func ParseLineItems(raw json.RawMessage) ([]models.LineItem, error) {
	return unmarshalLineItems(raw)
}

// ParseTaxBreakdown is a convenience function for callers to parse JSONB tax breakdown.
func ParseTaxBreakdown(raw json.RawMessage) (*models.TaxBreakdown, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var tb models.TaxBreakdown
	if err := json.Unmarshal(raw, &tb); err != nil {
		return nil, err
	}
	return &tb, nil
}
