// Package quote provides the business logic for managing quotes (Angebote).
// Quotes support the full lifecycle: draft -> sent -> accepted/rejected/expired,
// with tax calculation via the tax package and optional deal value auto-sync
// back to CRM when a quote is linked to a deal.
package quote

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles quote business logic.
type Service struct {
	repo            Repository
	numberSeqRepo   NumberSequenceRepo
	companySettings CompanySettingsRepo
	dealUpdater     DealValueUpdater
}

// NewService creates a new quote service.
// dealUpdater may be nil; if nil, deal value sync is skipped (standalone mode).
func NewService(
	repo Repository,
	numberSeqRepo NumberSequenceRepo,
	companySettings CompanySettingsRepo,
	dealUpdater DealValueUpdater,
) *Service {
	return &Service{
		repo:            repo,
		numberSeqRepo:   numberSeqRepo,
		companySettings: companySettings,
		dealUpdater:     dealUpdater,
	}
}

// CreateInput contains the data needed to create a quote.
type CreateInput struct {
	TenantID        uuid.UUID
	CustomerName    string
	CustomerAddress string
	CustomerEmail   string
	CustomerUStIDNr string
	TaxMode         string
	LineItems       []models.LineItem
	ValidUntil      *time.Time
	Notes           string
	DealID          *uuid.UUID
	UserID          uuid.UUID
}

// Create creates a new quote with tax calculation and optional deal sync.
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.Quote, error) {
	if len(input.LineItems) == 0 {
		return nil, ErrNoLineItems
	}

	// Calculate tax
	taxMode := parseTaxMode(input.TaxMode)
	taxItems := toTaxLineItems(input.LineItems)
	breakdown := tax.Calculate(taxItems, taxMode)

	// Apply calculated line totals back to the model items
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

	// Determine valid_until: explicit or from company settings default
	validUntil := input.ValidUntil
	if validUntil == nil {
		settings, settingsErr := s.companySettings.GetByTenantID(ctx, input.TenantID)
		if settingsErr != nil {
			slog.Warn("failed to load company settings for quote validity default",
				"tenant_id", input.TenantID,
				"error", settingsErr,
			)
		}
		if settings != nil && settings.DefaultQuoteValidityDays > 0 {
			expiry := time.Now().AddDate(0, 0, settings.DefaultQuoteValidityDays)
			validUntil = &expiry
		}
	}

	now := time.Now()
	quote := &models.Quote{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		QuoteNumber:     "", // Assigned when sent
		Status:          models.QuoteStatusDraft,
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
		ValidUntil:      validUntil,
		Notes:           input.Notes,
		DealID:          input.DealID,
		CreatedBy:       input.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if createErr := s.repo.Create(ctx, quote); createErr != nil {
		return nil, createErr
	}

	slog.Info("quote created",
		"quote_id", quote.ID,
		"tenant_id", quote.TenantID,
		"gross_total", quote.GrossTotal,
		"deal_id", quote.DealID,
	)

	// Deal value auto-sync (locked decision: quote gross_total syncs to CRM deal)
	if input.DealID != nil && s.dealUpdater != nil {
		if syncErr := s.dealUpdater.UpdateDealValue(ctx, *input.DealID, breakdown.GrossTotal); syncErr != nil {
			slog.Warn("failed to sync deal value after quote creation",
				"quote_id", quote.ID,
				"deal_id", *input.DealID,
				"gross_total", breakdown.GrossTotal,
				"error", syncErr,
			)
			// Graceful degradation: quote creation succeeds even if deal sync fails
		}
	}

	return quote, nil
}

// GetByID retrieves a quote by ID.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List retrieves quotes with optional filtering.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Quote, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

// UpdateInput contains the data that can be updated on a quote.
type UpdateInput struct {
	CustomerName    *string
	CustomerAddress *string
	CustomerEmail   *string
	CustomerUStIDNr *string
	TaxMode         *string
	LineItems       []models.LineItem // nil = no change, empty = error
	ValidUntil      *time.Time
	Notes           *string
	DealID          *uuid.UUID
}

// Update updates an existing quote. Only draft quotes can be modified.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, input UpdateInput) (*models.Quote, error) {
	quote, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if quote.Status != models.QuoteStatusDraft {
		return nil, ErrQuoteNotDraft
	}

	if input.CustomerName != nil {
		quote.CustomerName = *input.CustomerName
	}
	if input.CustomerAddress != nil {
		quote.CustomerAddress = *input.CustomerAddress
	}
	if input.CustomerEmail != nil {
		quote.CustomerEmail = *input.CustomerEmail
	}
	if input.CustomerUStIDNr != nil {
		quote.CustomerUStIDNr = *input.CustomerUStIDNr
	}
	if input.Notes != nil {
		quote.Notes = *input.Notes
	}
	if input.ValidUntil != nil {
		quote.ValidUntil = input.ValidUntil
	}
	if input.DealID != nil {
		quote.DealID = input.DealID
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
		quote.TaxMode = *input.TaxMode
		needsRecalc = true
	}

	if needsRecalc {
		lineItems := input.LineItems
		if lineItems == nil {
			// Load existing line items for tax recalculation
			existingItems, unmarshalErr := unmarshalLineItems(quote.LineItems)
			if unmarshalErr != nil {
				return nil, unmarshalErr
			}
			lineItems = existingItems
		}

		// Calculate tax
		taxMode := parseTaxMode(quote.TaxMode)
		taxItems := toTaxLineItems(lineItems)
		breakdown := tax.Calculate(taxItems, taxMode)

		// Update line totals
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

		quote.LineItems = lineItemsJSON
		quote.TaxBreakdownRaw = taxBreakdownJSON
		quote.Subtotal = breakdown.Subtotal
		quote.TotalTax = breakdown.TotalTax
		quote.GrossTotal = breakdown.GrossTotal
	}

	quote.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, quote); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("quote updated",
		"quote_id", quote.ID,
		"tenant_id", quote.TenantID,
		"gross_total", quote.GrossTotal,
	)

	// Deal value auto-sync on modification (locked decision)
	if quote.DealID != nil && s.dealUpdater != nil {
		if syncErr := s.dealUpdater.UpdateDealValue(ctx, *quote.DealID, quote.GrossTotal); syncErr != nil {
			slog.Warn("failed to sync deal value after quote update",
				"quote_id", quote.ID,
				"deal_id", *quote.DealID,
				"gross_total", quote.GrossTotal,
				"error", syncErr,
			)
			// Graceful degradation: quote update succeeds even if deal sync fails
		}
	}

	return quote, nil
}

// Delete removes a quote. Only draft quotes can be deleted.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	quote, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if quote.Status != models.QuoteStatusDraft {
		return ErrQuoteNotDraft
	}

	if deleteErr := s.repo.Delete(ctx, tenantID, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("quote deleted",
		"quote_id", id,
		"tenant_id", tenantID,
	)
	return nil
}

// Send transitions a quote from draft to sent and assigns a quote number.
func (s *Service) Send(ctx context.Context, tenantID, id, userID uuid.UUID) error {
	quote, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if quote.Status != models.QuoteStatusDraft {
		return ErrQuoteNotDraft
	}

	// Assign quote number via gap-free sequence
	fiscalYear := time.Now().Year()
	number, numErr := s.numberSeqRepo.NextNumber(ctx, tenantID, models.DocumentTypeQuote, fiscalYear, "AN")
	if numErr != nil {
		return numErr
	}

	quote.QuoteNumber = number
	quote.Status = models.QuoteStatusSent
	quote.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, quote); updateErr != nil {
		return updateErr
	}

	slog.Info("quote sent",
		"quote_id", quote.ID,
		"tenant_id", tenantID,
		"quote_number", number,
	)
	return nil
}

// Accept transitions a quote from sent to accepted.
func (s *Service) Accept(ctx context.Context, tenantID, id uuid.UUID) error {
	quote, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if quote.Status != models.QuoteStatusSent {
		return ErrQuoteNotSent
	}

	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, models.QuoteStatusAccepted); updateErr != nil {
		return updateErr
	}

	slog.Info("quote accepted",
		"quote_id", id,
		"tenant_id", tenantID,
	)
	return nil
}

// Reject transitions a quote from sent to rejected.
func (s *Service) Reject(ctx context.Context, tenantID, id uuid.UUID) error {
	quote, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if quote.Status != models.QuoteStatusSent {
		return ErrQuoteNotSent
	}

	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, models.QuoteStatusRejected); updateErr != nil {
		return updateErr
	}

	slog.Info("quote rejected",
		"quote_id", id,
		"tenant_id", tenantID,
	)
	return nil
}

// ExpireQuotes bulk-updates all sent quotes past their valid_until date to expired.
// Returns the number of quotes expired.
func (s *Service) ExpireQuotes(ctx context.Context, tenantID uuid.UUID) (int, error) {
	// List all sent quotes for this tenant
	filter := ListFilter{
		Status: models.QuoteStatusSent,
		Limit:  1000, // Process in large batches
	}

	quotes, _, err := s.repo.List(ctx, tenantID, filter)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	expired := 0

	for _, q := range quotes {
		if q.ValidUntil != nil && q.ValidUntil.Before(now) {
			if updateErr := s.repo.UpdateStatus(ctx, tenantID, q.ID, models.QuoteStatusExpired); updateErr != nil {
				slog.Warn("failed to expire quote",
					"quote_id", q.ID,
					"error", updateErr,
				)
				continue
			}
			expired++
		}
	}

	if expired > 0 {
		slog.Info("quotes expired",
			"tenant_id", tenantID,
			"count", expired,
		)
	}

	return expired, nil
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
