// Package recurring provides the business logic for recurring invoice schedules
// (Abo-Rechnungen). A schedule is a template that emits draft invoices at a fixed
// interval; the emitted invoice goes through the regular invoice service, so
// numbering, tax calculation and GoBD rules stay in exactly one place.
//
// Generation is idempotent per period: the period a run consumes is claimed via a
// unique row in finance_recurring_runs before any invoice is written. A retried
// request, a double click or a second scheduler pass finds the claim and returns
// the invoice that already exists. A duplicated invoice is a real business event,
// not a display glitch, so this guard lives in the database, not in a Go check.
package recurring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
)

// InvoiceCreator creates the draft invoice a schedule emits. Satisfied by
// *invoice.Service — injected so the recurring service never reimplements tax
// calculation, currency defaults or persistence.
type InvoiceCreator interface {
	Create(ctx context.Context, input invoice.CreateInput) (*models.Invoice, error)
}

// InvoiceReader loads an already emitted invoice. Needed for the idempotent
// replay: a repeated generate must answer with the invoice of the claimed period
// instead of creating a second one. Satisfied by invoice.Repository.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// Service handles recurring invoice business logic.
type Service struct {
	repo           Repository
	invoiceCreator InvoiceCreator
	invoiceReader  InvoiceReader
}

// NewService creates a new recurring invoice service.
func NewService(repo Repository, invoiceCreator InvoiceCreator, invoiceReader InvoiceReader) *Service {
	return &Service{repo: repo, invoiceCreator: invoiceCreator, invoiceReader: invoiceReader}
}

// CreateInput contains the data needed to create a schedule.
type CreateInput struct {
	TenantID         uuid.UUID
	Title            string
	Customer         models.CustomerSnapshot
	LineItems        []models.LineItem
	TaxMode          string
	Currency         string
	Interval         string
	StartDate        time.Time
	EndDate          *time.Time
	PaymentTermsDays int
	Notes            string
	UserID           uuid.UUID
}

// UpdateInput carries a partial update. A nil field stays untouched — the
// frontend sends Partial<CreateRecurringInvoiceRequest>.
type UpdateInput struct {
	Title            *string
	Customer         *models.CustomerSnapshot
	LineItems        []models.LineItem
	TaxMode          *string
	Currency         *string
	Interval         *string
	StartDate        *time.Time
	EndDate          *time.Time
	ClearEndDate     bool
	PaymentTermsDays *int
	Notes            *string
}

// Create persists a new active schedule. The first run is the start date.
func (s *Service) Create(ctx context.Context, in CreateInput) (*models.RecurringInvoice, error) {
	if len(in.LineItems) == 0 {
		return nil, ErrNoLineItems
	}
	if !validInterval(in.Interval) {
		return nil, ErrInvalidInterval
	}
	start := dateOnly(in.StartDate)
	if start.IsZero() {
		start = dateOnly(time.Now())
	}
	end := normalizeEnd(in.EndDate)
	if end != nil && end.Before(start) {
		return nil, ErrInvalidDateRange
	}

	lineItemsJSON, breakdownJSON, err := priceLineItems(in.LineItems, in.TaxMode)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	rec := &models.RecurringInvoice{
		ID:               uuid.New(),
		TenantID:         in.TenantID,
		Title:            in.Title,
		CustomerName:     in.Customer.Name,
		CustomerAddress:  in.Customer.Address,
		CustomerEmail:    in.Customer.Email,
		CustomerUStIDNr:  in.Customer.UStIDNr,
		TaxMode:          normalizeTaxMode(in.TaxMode),
		LineItems:        lineItemsJSON,
		TaxBreakdownRaw:  breakdownJSON,
		Currency:         defaultString(in.Currency, models.DefaultCurrency),
		Interval:         in.Interval,
		Status:           models.RecurringStatusActive,
		StartDate:        start,
		EndDate:          end,
		NextRun:          start,
		PaymentTermsDays: in.PaymentTermsDays,
		Notes:            in.Notes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if in.UserID != uuid.Nil {
		userID := in.UserID
		rec.CreatedBy = &userID
	}
	if rec.Title == "" {
		rec.Title = rec.CustomerName
	}

	if createErr := s.repo.Create(ctx, rec); createErr != nil {
		return nil, createErr
	}
	slog.Info("recurring invoice created",
		"recurring_id", rec.ID,
		"tenant_id", rec.TenantID,
		"interval", rec.Interval,
		"next_run", rec.NextRun.Format(dateLayout),
	)
	return rec, nil
}

// Get returns a single schedule of the tenant.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.RecurringInvoice, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List returns the tenant's schedules, newest first.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.RecurringInvoice, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

// Update applies a partial change to a schedule.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, in UpdateInput) (*models.RecurringInvoice, error) {
	rec, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		rec.Title = *in.Title
	}
	if in.Customer != nil {
		rec.CustomerName = in.Customer.Name
		rec.CustomerAddress = in.Customer.Address
		rec.CustomerEmail = in.Customer.Email
		rec.CustomerUStIDNr = in.Customer.UStIDNr
	}
	if in.TaxMode != nil {
		rec.TaxMode = normalizeTaxMode(*in.TaxMode)
	}
	if in.Currency != nil && *in.Currency != "" {
		rec.Currency = *in.Currency
	}
	if in.Interval != nil {
		if !validInterval(*in.Interval) {
			return nil, ErrInvalidInterval
		}
		rec.Interval = *in.Interval
	}
	if in.PaymentTermsDays != nil {
		rec.PaymentTermsDays = *in.PaymentTermsDays
	}
	if in.Notes != nil {
		rec.Notes = *in.Notes
	}
	if len(in.LineItems) > 0 {
		lineItemsJSON, breakdownJSON, priceErr := priceLineItems(in.LineItems, rec.TaxMode)
		if priceErr != nil {
			return nil, priceErr
		}
		rec.LineItems = lineItemsJSON
		rec.TaxBreakdownRaw = breakdownJSON
	}
	if in.StartDate != nil {
		rec.StartDate = dateOnly(*in.StartDate)
	}
	switch {
	case in.ClearEndDate:
		rec.EndDate = nil
	case in.EndDate != nil:
		rec.EndDate = normalizeEnd(in.EndDate)
	}
	if rec.EndDate != nil && rec.EndDate.Before(rec.StartDate) {
		return nil, ErrInvalidDateRange
	}

	// Re-anchor the schedule when its cadence changed: the next run is always the
	// start date advanced by the number of periods already emitted.
	if in.StartDate != nil || in.Interval != nil {
		rec.NextRun = nextRunFor(rec.StartDate, rec.Interval, rec.GeneratedCount)
	}
	if rec.Status == models.RecurringStatusActive && exhausted(rec) {
		rec.Status = models.RecurringStatusEnded
	}
	rec.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// SetStatus pauses, resumes or ends a schedule. Resuming a schedule whose end
// date has passed is a no-op on the status: it stays ended.
func (s *Service) SetStatus(ctx context.Context, tenantID, id uuid.UUID, status string) (*models.RecurringInvoice, error) {
	if status != models.RecurringStatusActive &&
		status != models.RecurringStatusPaused &&
		status != models.RecurringStatusEnded {
		return nil, ErrInvalidStatus
	}
	rec, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if status == models.RecurringStatusActive && rec.Status == models.RecurringStatusEnded {
		return rec, nil
	}
	if status == models.RecurringStatusActive && exhausted(rec) {
		status = models.RecurringStatusEnded
	}
	rec.Status = status
	rec.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, rec); err != nil {
		return nil, err
	}
	slog.Info("recurring invoice status changed",
		"recurring_id", rec.ID, "tenant_id", rec.TenantID, "status", rec.Status)
	return rec, nil
}

// Delete removes a schedule. Already emitted invoices survive — their
// recurring_id is nulled by the FK, they remain regular invoices.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// Generate emits the invoice for the schedule's current period and advances the
// schedule. It is idempotent: calling it twice for the same period returns the
// same invoice and never emits a second one.
func (s *Service) Generate(ctx context.Context, tenantID, id, userID uuid.UUID) (*models.Invoice, *models.RecurringInvoice, error) {
	rec, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	if rec.Status != models.RecurringStatusActive {
		return nil, nil, ErrNotActive
	}
	period := dateOnly(rec.NextRun)
	if rec.EndDate != nil && period.After(*rec.EndDate) {
		return nil, nil, ErrScheduleExhausted
	}

	run, claimed, err := s.repo.ClaimPeriod(ctx, tenantID, rec.ID, period)
	if err != nil {
		return nil, nil, err
	}
	if !claimed {
		// Replay: this period was already emitted. Answer with the existing
		// invoice instead of creating a second one for the same period.
		if run.InvoiceID == nil {
			return nil, nil, fmt.Errorf("recurring period %s is being generated concurrently", period.Format(dateLayout))
		}
		inv, getErr := s.invoiceReader.GetByID(ctx, tenantID, *run.InvoiceID)
		if getErr != nil {
			return nil, nil, getErr
		}
		slog.Info("recurring invoice generation replayed",
			"recurring_id", rec.ID, "tenant_id", tenantID,
			"period", period.Format(dateLayout), "invoice_id", inv.ID)
		return inv, rec, nil
	}

	items, err := invoice.ParseLineItems(rec.LineItems)
	if err != nil {
		s.releaseClaim(ctx, tenantID, run.ID)
		return nil, nil, fmt.Errorf("parse recurring line items: %w", err)
	}
	if len(items) == 0 {
		s.releaseClaim(ctx, tenantID, run.ID)
		return nil, nil, ErrNoLineItems
	}

	recurringID := rec.ID
	inv, err := s.invoiceCreator.Create(ctx, invoice.CreateInput{
		TenantID:         tenantID,
		CustomerName:     rec.CustomerName,
		CustomerAddress:  rec.CustomerAddress,
		CustomerEmail:    rec.CustomerEmail,
		CustomerUStIDNr:  rec.CustomerUStIDNr,
		TaxMode:          rec.TaxMode,
		LineItems:        items,
		InvoiceDate:      period,
		PaymentTermsDays: rec.PaymentTermsDays,
		Notes:            generatedNote(rec),
		RecurringID:      &recurringID,
		Currency:         rec.Currency,
		UserID:           userID,
	})
	if err != nil {
		// Free the claim, otherwise the period stays blocked forever and the
		// schedule can never emit its invoice.
		s.releaseClaim(ctx, tenantID, run.ID)
		return nil, nil, err
	}

	if attachErr := s.repo.AttachInvoice(ctx, tenantID, run.ID, inv.ID); attachErr != nil {
		// The invoice exists; losing the back-reference would let a retry emit a
		// second one, so this is an error, not a warning.
		return nil, nil, attachErr
	}

	rec.GeneratedCount++
	lastGenerated := period
	rec.LastGeneratedAt = &lastGenerated
	rec.NextRun = nextRunFor(rec.StartDate, rec.Interval, rec.GeneratedCount)
	if exhausted(rec) {
		rec.Status = models.RecurringStatusEnded
	}
	rec.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, rec); err != nil {
		return nil, nil, err
	}

	slog.Info("recurring invoice generated",
		"recurring_id", rec.ID, "tenant_id", tenantID,
		"invoice_id", inv.ID, "period", period.Format(dateLayout),
		"next_run", rec.NextRun.Format(dateLayout), "status", rec.Status)

	return inv, rec, nil
}

func (s *Service) releaseClaim(ctx context.Context, tenantID, runID uuid.UUID) {
	if err := s.repo.ReleasePeriod(ctx, tenantID, runID); err != nil {
		slog.Error("failed to release recurring period claim",
			"tenant_id", tenantID, "run_id", runID, "error", err)
	}
}

// ============================================================================
// Helpers
// ============================================================================

// dateLayout is the wire format for all schedule dates (YYYY-MM-DD).
const dateLayout = "2006-01-02"

func validInterval(interval string) bool {
	switch interval {
	case models.RecurringIntervalWeekly, models.RecurringIntervalMonthly,
		models.RecurringIntervalQuarterly, models.RecurringIntervalYearly:
		return true
	default:
		return false
	}
}

func normalizeTaxMode(mode string) string {
	switch mode {
	case models.TaxModeReverseCharge, models.TaxModeKleinunternehmer:
		return mode
	default:
		return models.TaxModeStandard
	}
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// dateOnly strips the time component so date arithmetic and the DATE columns
// agree — a schedule runs on a day, not at an instant.
func dateOnly(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeEnd(end *time.Time) *time.Time {
	if end == nil || end.IsZero() {
		return nil
	}
	d := dateOnly(*end)
	return &d
}

func exhausted(rec *models.RecurringInvoice) bool {
	return rec.EndDate != nil && rec.NextRun.After(*rec.EndDate)
}

// nextRunFor returns the run date after `periods` completed periods, always
// anchored at the start date. Anchoring instead of repeatedly advancing the
// previous run keeps a monthly schedule that starts on the 31st on the 31st:
// advancing step by step would clamp it to the 28th in February and leave it
// there forever.
func nextRunFor(start time.Time, interval string, periods int) time.Time {
	start = dateOnly(start)
	if periods < 0 {
		periods = 0
	}
	switch interval {
	case models.RecurringIntervalWeekly:
		return start.AddDate(0, 0, 7*periods)
	case models.RecurringIntervalQuarterly:
		return addMonthsClamped(start, 3*periods)
	case models.RecurringIntervalYearly:
		return addMonthsClamped(start, 12*periods)
	default: // monthly
		return addMonthsClamped(start, periods)
	}
}

// addMonthsClamped adds months and clamps the day to the last day of the target
// month. time.AddDate normalizes overflow instead (31 Jan + 1 month = 3 March),
// which would silently bill a month late.
func addMonthsClamped(t time.Time, months int) time.Time {
	year := t.Year()
	month := int(t.Month()) - 1 + months
	year += month / 12
	month %= 12
	if month < 0 {
		month += 12
		year--
	}
	target := time.Month(month + 1)
	day := t.Day()
	if last := daysInMonth(year, target); day > last {
		day = last
	}
	return time.Date(year, target, day, 0, 0, 0, 0, time.UTC)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func generatedNote(rec *models.RecurringInvoice) string {
	if rec.Notes != "" {
		return rec.Notes
	}
	return fmt.Sprintf("Automatisch erzeugt aus %q", rec.Title)
}

// priceLineItems computes line totals and the tax breakdown for a schedule and
// returns both as JSONB payloads, using the same tax calculator as invoices so a
// generated invoice matches the schedule preview to the cent.
func priceLineItems(items []models.LineItem, taxMode string) (lineItemsJSON, breakdownJSON json.RawMessage, err error) {
	priced := make([]models.LineItem, len(items))
	taxItems := make([]tax.LineItem, len(items))
	for i, item := range items {
		item.LineTotal = tax.LineTotal(item.Quantity, item.UnitPrice)
		if item.Position == 0 {
			item.Position = i + 1
		}
		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		priced[i] = item
		taxItems[i] = tax.LineItem{
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			TaxRate:   item.TaxRate,
		}
	}

	breakdown := tax.Calculate(taxItems, toTaxMode(taxMode))
	lineItemsJSON, err = json.Marshal(priced)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal line items: %w", err)
	}
	breakdownJSON, err = json.Marshal(models.TaxBreakdown{
		Subtotal:   breakdown.Subtotal,
		TaxByRate:  breakdown.TaxByRate,
		TotalTax:   breakdown.TotalTax,
		GrossTotal: breakdown.GrossTotal,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("marshal tax breakdown: %w", err)
	}
	return lineItemsJSON, breakdownJSON, nil
}

func toTaxMode(mode string) tax.TaxMode {
	switch mode {
	case models.TaxModeReverseCharge:
		return tax.ModeReverseCharge
	case models.TaxModeKleinunternehmer:
		return tax.ModeKleinunternehmer
	default:
		return tax.ModeStandard
	}
}

// IsNotFound reports whether err is the "no such schedule" case, so callers can
// map it to gRPC NotFound without importing the sentinel everywhere.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
