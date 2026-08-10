package recurring

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Fakes
// ============================================================================

type fakeRepo struct {
	schedules  map[uuid.UUID]*models.RecurringInvoice
	runs       map[string]*Run
	claims     int
	releases   int
	releaseErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		schedules: map[uuid.UUID]*models.RecurringInvoice{},
		runs:      map[string]*Run{},
	}
}

func runKey(recurringID uuid.UUID, period time.Time) string {
	return recurringID.String() + "|" + period.Format(dateLayout)
}

func (f *fakeRepo) Create(_ context.Context, rec *models.RecurringInvoice) error {
	copied := *rec
	f.schedules[rec.ID] = &copied
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.RecurringInvoice, error) {
	rec, ok := f.schedules[id]
	if !ok || rec.TenantID != tenantID {
		return nil, ErrNotFound
	}
	copied := *rec
	return &copied, nil
}

func (f *fakeRepo) List(_ context.Context, tenantID uuid.UUID, _ ListFilter) ([]*models.RecurringInvoice, int, error) {
	items := make([]*models.RecurringInvoice, 0, len(f.schedules))
	for _, rec := range f.schedules {
		if rec.TenantID == tenantID {
			items = append(items, rec)
		}
	}
	return items, len(items), nil
}

func (f *fakeRepo) Update(_ context.Context, rec *models.RecurringInvoice) error {
	if _, ok := f.schedules[rec.ID]; !ok {
		return ErrNotFound
	}
	copied := *rec
	f.schedules[rec.ID] = &copied
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, _, id uuid.UUID) error {
	if _, ok := f.schedules[id]; !ok {
		return ErrNotFound
	}
	delete(f.schedules, id)
	return nil
}

func (f *fakeRepo) ClaimPeriod(_ context.Context, tenantID, recurringID uuid.UUID, period time.Time) (*Run, bool, error) {
	f.claims++
	key := runKey(recurringID, period)
	if existing, ok := f.runs[key]; ok {
		copied := *existing
		return &copied, false, nil
	}
	run := &Run{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RecurringID: recurringID,
		PeriodDate:  period,
		CreatedAt:   time.Now(),
	}
	f.runs[key] = run
	copied := *run
	return &copied, true, nil
}

func (f *fakeRepo) AttachInvoice(_ context.Context, _, runID, invoiceID uuid.UUID) error {
	for _, run := range f.runs {
		if run.ID == runID {
			id := invoiceID
			run.InvoiceID = &id
			return nil
		}
	}
	return errors.New("run not found")
}

func (f *fakeRepo) ReleasePeriod(_ context.Context, _, runID uuid.UUID) error {
	f.releases++
	if f.releaseErr != nil {
		return f.releaseErr
	}
	for key, run := range f.runs {
		if run.ID == runID && run.InvoiceID == nil {
			delete(f.runs, key)
			return nil
		}
	}
	return nil
}

type fakeInvoices struct {
	created []invoice.CreateInput
	stored  map[uuid.UUID]*models.Invoice
	failWith error
}

func newFakeInvoices() *fakeInvoices {
	return &fakeInvoices{stored: map[uuid.UUID]*models.Invoice{}}
}

func (f *fakeInvoices) Create(_ context.Context, in invoice.CreateInput) (*models.Invoice, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.created = append(f.created, in)
	inv := &models.Invoice{
		ID:           uuid.New(),
		TenantID:     in.TenantID,
		Status:       models.InvoiceStatusDraft,
		CustomerName: in.CustomerName,
		InvoiceDate:  in.InvoiceDate,
		Currency:     in.Currency,
		RecurringID:  in.RecurringID,
	}
	f.stored[inv.ID] = inv
	return inv, nil
}

func (f *fakeInvoices) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	inv, ok := f.stored[id]
	if !ok || inv.TenantID != tenantID {
		return nil, errors.New("invoice not found")
	}
	return inv, nil
}

// ============================================================================
// Helpers
// ============================================================================

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}

func lineItems() []models.LineItem {
	return []models.LineItem{{
		Description: "CRM Lizenz Enterprise",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   decimal.RequireFromString("790.00"),
		TaxRate:     decimal.NewFromInt(19),
	}}
}

func newTestService() (*Service, *fakeRepo, *fakeInvoices) {
	repo := newFakeRepo()
	invoices := newFakeInvoices()
	return NewService(repo, invoices, invoices), repo, invoices
}

func createSchedule(t *testing.T, svc *Service, tenantID uuid.UUID, in CreateInput) *models.RecurringInvoice {
	t.Helper()
	if in.LineItems == nil {
		in.LineItems = lineItems()
	}
	if in.Interval == "" {
		in.Interval = models.RecurringIntervalMonthly
	}
	in.TenantID = tenantID
	rec, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	return rec
}

// ============================================================================
// Create
// ============================================================================

func TestCreate_PricesLineItemsAndSetsFirstRun(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()

	rec := createSchedule(t, svc, tenantID, CreateInput{
		Title:            "CRM-Lizenz",
		StartDate:        mustDate(t, "2026-03-15"),
		PaymentTermsDays: 14,
	})

	if rec.Status != models.RecurringStatusActive {
		t.Errorf("status = %q, want active", rec.Status)
	}
	if got := rec.NextRun.Format(dateLayout); got != "2026-03-15" {
		t.Errorf("next_run = %s, want the start date", got)
	}
	if rec.Currency != models.DefaultCurrency {
		t.Errorf("currency = %q, want the EUR default", rec.Currency)
	}

	var breakdown models.TaxBreakdown
	if err := json.Unmarshal(rec.TaxBreakdownRaw, &breakdown); err != nil {
		t.Fatalf("unmarshal tax breakdown: %v", err)
	}
	if got := breakdown.GrossTotal.StringFixed(2); got != "940.10" {
		t.Errorf("gross_total = %s, want 940.10 (790.00 + 19%%)", got)
	}

	var items []models.LineItem
	if err := json.Unmarshal(rec.LineItems, &items); err != nil {
		t.Fatalf("unmarshal line items: %v", err)
	}
	if got := items[0].LineTotal.StringFixed(2); got != "790.00" {
		t.Errorf("line_total = %s, want 790.00 — an unpriced line renders as 0,00 EUR", got)
	}
}

func TestCreate_RejectsInvalidInput(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	start := mustDate(t, "2026-03-15")
	earlier := mustDate(t, "2026-03-01")

	cases := []struct {
		name string
		in   CreateInput
		want error
	}{
		{"no line items", CreateInput{Interval: models.RecurringIntervalMonthly, StartDate: start}, ErrNoLineItems},
		{"bad interval", CreateInput{LineItems: lineItems(), Interval: "daily", StartDate: start}, ErrInvalidInterval},
		{"end before start", CreateInput{LineItems: lineItems(), Interval: models.RecurringIntervalMonthly, StartDate: start, EndDate: &earlier}, ErrInvalidDateRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.TenantID = tenantID
			if _, err := svc.Create(context.Background(), tc.in); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// ============================================================================
// Generate — the idempotency guarantee
// ============================================================================

func TestGenerate_EmitsInvoiceAndAdvancesSchedule(t *testing.T) {
	svc, _, invoices := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{
		Title:            "CRM-Lizenz",
		StartDate:        mustDate(t, "2026-03-15"),
		PaymentTermsDays: 14,
		Currency:         "CHF",
	})

	inv, updated, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if inv.RecurringID == nil || *inv.RecurringID != rec.ID {
		t.Error("emitted invoice is not linked back to its schedule")
	}
	if inv.Currency != "CHF" {
		t.Errorf("invoice currency = %q, want the schedule currency CHF", inv.Currency)
	}
	if got := invoices.created[0].InvoiceDate.Format(dateLayout); got != "2026-03-15" {
		t.Errorf("invoice_date = %s, want the billed period 2026-03-15", got)
	}
	if updated.GeneratedCount != 1 {
		t.Errorf("generated_count = %d, want 1", updated.GeneratedCount)
	}
	if got := updated.NextRun.Format(dateLayout); got != "2026-04-15" {
		t.Errorf("next_run = %s, want 2026-04-15", got)
	}
	if updated.LastGeneratedAt == nil || updated.LastGeneratedAt.Format(dateLayout) != "2026-03-15" {
		t.Errorf("last_generated_at = %v, want 2026-03-15", updated.LastGeneratedAt)
	}
}

// A retried request, a double click or a second scheduler pass must not bill the
// customer twice. The second call has to hand back the very same invoice.
func TestGenerate_IsIdempotentPerPeriod(t *testing.T) {
	svc, repo, invoices := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	first, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New())
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}

	// Rewind the schedule to the same period, as a stale client or a replayed
	// request would: the run ledger, not the schedule state, is the guard.
	stored := repo.schedules[rec.ID]
	stored.NextRun = mustDate(t, "2026-03-15")
	stored.Status = models.RecurringStatusActive

	second, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New())
	if err != nil {
		t.Fatalf("second generate: %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("replay returned invoice %s, want the already emitted %s", second.ID, first.ID)
	}
	if len(invoices.created) != 1 {
		t.Errorf("%d invoices created for one period, want exactly 1", len(invoices.created))
	}
}

func TestGenerate_ReleasesClaimWhenInvoiceCreationFails(t *testing.T) {
	svc, repo, invoices := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	invoices.failWith = errors.New("database down")
	if _, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New()); err == nil {
		t.Fatal("generate: want error when the invoice cannot be written")
	}
	if repo.releases != 1 {
		t.Fatalf("released %d claims, want 1 — a stuck claim blocks the period forever", repo.releases)
	}

	// The period must be claimable again after the failure.
	invoices.failWith = nil
	if _, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New()); err != nil {
		t.Fatalf("retry after failure: %v", err)
	}
	if len(invoices.created) != 1 {
		t.Errorf("%d invoices created, want 1", len(invoices.created))
	}
}

func TestGenerate_RefusesPausedSchedule(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	if _, err := svc.SetStatus(context.Background(), tenantID, rec.ID, models.RecurringStatusPaused); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if _, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New()); !errors.Is(err, ErrNotActive) {
		t.Errorf("err = %v, want ErrNotActive", err)
	}
}

func TestGenerate_EndsScheduleAfterLastPeriod(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	end := mustDate(t, "2026-04-01")
	rec := createSchedule(t, svc, tenantID, CreateInput{
		StartDate: mustDate(t, "2026-03-15"),
		EndDate:   &end,
	})

	_, updated, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if updated.Status != models.RecurringStatusEnded {
		t.Errorf("status = %q, want ended — next run 2026-04-15 is past the end date", updated.Status)
	}
	if _, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New()); !errors.Is(err, ErrNotActive) {
		t.Errorf("err = %v, want ErrNotActive after the schedule ended", err)
	}
}

func TestGenerate_UnknownScheduleOfOtherTenant(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	if _, _, err := svc.Generate(context.Background(), uuid.New(), rec.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound for a foreign tenant", err)
	}
}

// ============================================================================
// Status + schedule arithmetic
// ============================================================================

func TestSetStatus_ResumeKeepsEndedSchedulesEnded(t *testing.T) {
	svc, repo, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	stored := repo.schedules[rec.ID]
	stored.Status = models.RecurringStatusEnded

	resumed, err := svc.SetStatus(context.Background(), tenantID, rec.ID, models.RecurringStatusActive)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Status != models.RecurringStatusEnded {
		t.Errorf("status = %q, want ended", resumed.Status)
	}
}

func TestNextRunFor_AnchorsAtStartAndClampsMonthEnd(t *testing.T) {
	cases := []struct {
		name     string
		start    string
		interval string
		periods  int
		want     string
	}{
		{"monthly from the 31st clamps to February", "2026-01-31", models.RecurringIntervalMonthly, 1, "2026-02-28"},
		{"monthly stays anchored on the 31st", "2026-01-31", models.RecurringIntervalMonthly, 2, "2026-03-31"},
		{"weekly", "2026-03-15", models.RecurringIntervalWeekly, 3, "2026-04-05"},
		{"quarterly", "2026-01-15", models.RecurringIntervalQuarterly, 2, "2026-07-15"},
		{"yearly over a leap day", "2028-02-29", models.RecurringIntervalYearly, 1, "2029-02-28"},
		{"zero periods is the start", "2026-03-15", models.RecurringIntervalMonthly, 0, "2026-03-15"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, err := time.Parse(dateLayout, tc.start)
			if err != nil {
				t.Fatalf("parse start: %v", err)
			}
			if got := nextRunFor(start, tc.interval, tc.periods).Format(dateLayout); got != tc.want {
				t.Errorf("nextRunFor(%s, %s, %d) = %s, want %s", tc.start, tc.interval, tc.periods, got, tc.want)
			}
		})
	}
}

func TestUpdate_ReanchorsNextRunOnIntervalChange(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-01-15")})

	if _, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New()); err != nil {
		t.Fatalf("generate: %v", err)
	}

	quarterly := models.RecurringIntervalQuarterly
	updated, err := svc.Update(context.Background(), tenantID, rec.ID, UpdateInput{Interval: &quarterly})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := updated.NextRun.Format(dateLayout); got != "2026-04-15" {
		t.Errorf("next_run = %s, want 2026-04-15 (one quarter after the start)", got)
	}
}

func TestUpdate_RejectsInvalidInterval(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	bad := "fortnightly"
	if _, err := svc.Update(context.Background(), tenantID, rec.ID, UpdateInput{Interval: &bad}); !errors.Is(err, ErrInvalidInterval) {
		t.Errorf("err = %v, want ErrInvalidInterval", err)
	}
}

// ============================================================================
// Get / List / Delete / IsNotFound
// ============================================================================

func TestGet_ReturnsScheduleOrNotFound(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	got, err := svc.Get(context.Background(), tenantID, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != rec.ID {
		t.Errorf("id = %s, want %s", got.ID, rec.ID)
	}

	if _, err := svc.Get(context.Background(), tenantID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound for an unknown id", err)
	}
}

func TestList_ReturnsOnlyTheTenantsSchedules(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	otherTenant := uuid.New()
	createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})
	createSchedule(t, svc, otherTenant, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	items, total, err := svc.List(context.Background(), tenantID, ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total = %d, len(items) = %d, want 1 and 1 — a foreign tenant's schedule leaked into the list", total, len(items))
	}
}

func TestDelete_RemovesScheduleOrReturnsNotFound(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	if err := svc.Delete(context.Background(), tenantID, rec.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), tenantID, rec.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound after delete", err)
	}
	if err := svc.Delete(context.Background(), tenantID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound deleting an unknown schedule", err)
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(ErrNotFound) {
		t.Error("IsNotFound(ErrNotFound) = false, want true")
	}
	if IsNotFound(errors.New("some other error")) {
		t.Error("IsNotFound(other) = true, want false")
	}
}

// ============================================================================
// SetStatus — remaining branches
// ============================================================================

func TestSetStatus_RejectsUnknownStatus(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	if _, err := svc.SetStatus(context.Background(), tenantID, rec.ID, "cancelled"); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("err = %v, want ErrInvalidStatus", err)
	}
}

func TestSetStatus_ResumingAnExhaustedScheduleEndsItInstead(t *testing.T) {
	svc, repo, _ := newTestService()
	tenantID := uuid.New()
	end := mustDate(t, "2026-04-01")
	rec := createSchedule(t, svc, tenantID, CreateInput{
		StartDate: mustDate(t, "2026-03-15"),
		EndDate:   &end,
	})

	// A paused schedule whose next run has, by whatever means, already moved
	// past its end date — resuming it must not silently reactivate a schedule
	// that can never generate again.
	stored := repo.schedules[rec.ID]
	stored.Status = models.RecurringStatusPaused
	stored.NextRun = mustDate(t, "2026-05-15")

	resumed, err := svc.SetStatus(context.Background(), tenantID, rec.ID, models.RecurringStatusActive)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Status != models.RecurringStatusEnded {
		t.Errorf("status = %q, want ended — next run is past the end date", resumed.Status)
	}
}

// ============================================================================
// Generate — the defensive exhausted-period guard
// ============================================================================

func TestGenerate_RefusesPeriodPastEndDate(t *testing.T) {
	svc, repo, _ := newTestService()
	tenantID := uuid.New()
	end := mustDate(t, "2026-04-01")
	rec := createSchedule(t, svc, tenantID, CreateInput{
		StartDate: mustDate(t, "2026-03-15"),
		EndDate:   &end,
	})

	// The schedule claims to still be active with a next run already past its
	// own end date — a state the service must refuse to emit for, whatever
	// external means produced it.
	stored := repo.schedules[rec.ID]
	stored.NextRun = mustDate(t, "2026-05-15")

	if _, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New()); !errors.Is(err, ErrScheduleExhausted) {
		t.Errorf("err = %v, want ErrScheduleExhausted", err)
	}
}

func TestGenerate_UsesScheduleNotesVerbatimWhenSet(t *testing.T) {
	svc, _, invoices := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{
		StartDate: mustDate(t, "2026-03-15"),
		Notes:     "Wartungsvertrag Q1",
	})

	if _, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New()); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := invoices.created[0].Notes; got != "Wartungsvertrag Q1" {
		t.Errorf("notes = %q, want the schedule's own notes verbatim", got)
	}
}

// releaseClaim only logs its own error and never surfaces it to the caller —
// the original invoice-creation error must still win, and a broken release
// must not panic or mask it.
func TestReleaseClaim_SwallowsItsOwnErrorAndKeepsTheOriginal(t *testing.T) {
	svc, repo, invoices := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	invoices.failWith = errors.New("database down")
	repo.releaseErr = errors.New("release also failed")

	_, _, err := svc.Generate(context.Background(), tenantID, rec.ID, uuid.New())
	if err == nil || err.Error() != "database down" {
		t.Fatalf("err = %v, want the original invoice-creation error", err)
	}
	if repo.releases != 1 {
		t.Errorf("releases = %d, want 1 — the release must still be attempted", repo.releases)
	}
}

// ============================================================================
// Create — remaining branches
// ============================================================================

func TestCreate_TitleFallsBackToCustomerName(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()

	rec, err := svc.Create(context.Background(), CreateInput{
		Customer:  models.CustomerSnapshot{Name: "Musterfirma GmbH"},
		LineItems: lineItems(),
		Interval:  models.RecurringIntervalMonthly,
		StartDate: mustDate(t, "2026-03-15"),
		TenantID:  tenantID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rec.Title != "Musterfirma GmbH" {
		t.Errorf("title = %q, want the customer name as fallback", rec.Title)
	}
}

func TestCreate_SetsCreatedByWhenUserIDGiven(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()

	rec, err := svc.Create(context.Background(), CreateInput{
		Title:     "CRM-Lizenz",
		LineItems: lineItems(),
		Interval:  models.RecurringIntervalMonthly,
		StartDate: mustDate(t, "2026-03-15"),
		TenantID:  tenantID,
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rec.CreatedBy == nil || *rec.CreatedBy != userID {
		t.Errorf("created_by = %v, want %s", rec.CreatedBy, userID)
	}
}

func TestCreate_DefaultsStartDateToToday(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()

	rec, err := svc.Create(context.Background(), CreateInput{
		Title:     "CRM-Lizenz",
		LineItems: lineItems(),
		Interval:  models.RecurringIntervalMonthly,
		TenantID:  tenantID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if want := dateOnly(time.Now()); !rec.StartDate.Equal(want) {
		t.Errorf("start_date = %s, want today (%s) when none is given", rec.StartDate.Format(dateLayout), want.Format(dateLayout))
	}
}

// ============================================================================
// Update — remaining field and error branches
// ============================================================================

func TestUpdate_AppliesEveryOptionalField(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	title := "Neuer Titel"
	currency := "CHF"
	terms := 21
	notes := "Aktualisiert"
	customer := models.CustomerSnapshot{Name: "Neuer Kunde", Email: "neu@example.com"}
	newItems := []models.LineItem{{
		Description: "Zusatzmodul",
		Quantity:    decimal.NewFromInt(2),
		UnitPrice:   decimal.RequireFromString("50.00"),
		TaxRate:     decimal.NewFromInt(19),
	}}

	updated, err := svc.Update(context.Background(), tenantID, rec.ID, UpdateInput{
		Title:            &title,
		Customer:         &customer,
		LineItems:        newItems,
		Currency:         &currency,
		PaymentTermsDays: &terms,
		Notes:            &notes,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != title {
		t.Errorf("title = %q, want %q", updated.Title, title)
	}
	if updated.CustomerName != customer.Name || updated.CustomerEmail != customer.Email {
		t.Errorf("customer = %q/%q, want %q/%q", updated.CustomerName, updated.CustomerEmail, customer.Name, customer.Email)
	}
	if updated.Currency != currency {
		t.Errorf("currency = %q, want %q", updated.Currency, currency)
	}
	if updated.PaymentTermsDays != terms {
		t.Errorf("payment_terms_days = %d, want %d", updated.PaymentTermsDays, terms)
	}
	if updated.Notes != notes {
		t.Errorf("notes = %q, want %q", updated.Notes, notes)
	}
	var items []models.LineItem
	if err := json.Unmarshal(updated.LineItems, &items); err != nil {
		t.Fatalf("unmarshal line items: %v", err)
	}
	if len(items) != 1 || items[0].Description != "Zusatzmodul" {
		t.Errorf("line_items = %+v, want the new items to have replaced the old ones", items)
	}
}

func TestUpdate_EmptyCurrencyIsIgnored(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15"), Currency: "CHF"})

	empty := ""
	updated, err := svc.Update(context.Background(), tenantID, rec.ID, UpdateInput{Currency: &empty})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Currency != "CHF" {
		t.Errorf("currency = %q, want the original CHF to survive an empty-string update", updated.Currency)
	}
}

func TestUpdate_ClearEndDateWins(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	end := mustDate(t, "2026-06-01")
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15"), EndDate: &end})

	updated, err := svc.Update(context.Background(), tenantID, rec.ID, UpdateInput{ClearEndDate: true})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.EndDate != nil {
		t.Errorf("end_date = %v, want nil after ClearEndDate", updated.EndDate)
	}
}

func TestUpdate_SetsNewEndDate(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	end := mustDate(t, "2026-09-01")
	updated, err := svc.Update(context.Background(), tenantID, rec.ID, UpdateInput{EndDate: &end})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.EndDate == nil || !updated.EndDate.Equal(end) {
		t.Errorf("end_date = %v, want %s", updated.EndDate, end.Format(dateLayout))
	}
}

func TestUpdate_RejectsEndDateBeforeStartDate(t *testing.T) {
	svc, _, _ := newTestService()
	tenantID := uuid.New()
	rec := createSchedule(t, svc, tenantID, CreateInput{StartDate: mustDate(t, "2026-03-15")})

	tooEarly := mustDate(t, "2026-01-01")
	if _, err := svc.Update(context.Background(), tenantID, rec.ID, UpdateInput{EndDate: &tooEarly}); !errors.Is(err, ErrInvalidDateRange) {
		t.Errorf("err = %v, want ErrInvalidDateRange", err)
	}
}

// ============================================================================
// Tax mode branches — reverse charge and Kleinunternehmer both zero the tax,
// but only via their own normalizeTaxMode/toTaxMode path, not the default.
// ============================================================================

func TestCreate_ReverseChargeAndKleinunternehmerZeroTheTax(t *testing.T) {
	cases := []struct {
		name string
		mode string
	}{
		{"reverse charge", models.TaxModeReverseCharge},
		{"kleinunternehmer", models.TaxModeKleinunternehmer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService()
			tenantID := uuid.New()

			rec, err := svc.Create(context.Background(), CreateInput{
				Title:     "CRM-Lizenz",
				LineItems: lineItems(),
				TaxMode:   tc.mode,
				Interval:  models.RecurringIntervalMonthly,
				StartDate: mustDate(t, "2026-03-15"),
				TenantID:  tenantID,
			})
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if rec.TaxMode != tc.mode {
				t.Errorf("tax_mode = %q, want %q", rec.TaxMode, tc.mode)
			}
			var breakdown models.TaxBreakdown
			if err := json.Unmarshal(rec.TaxBreakdownRaw, &breakdown); err != nil {
				t.Fatalf("unmarshal tax breakdown: %v", err)
			}
			if !breakdown.TotalTax.IsZero() {
				t.Errorf("total_tax = %s, want zero for %s", breakdown.TotalTax, tc.mode)
			}
		})
	}
}

// ============================================================================
// addMonthsClamped — the negative-months wrap-around branch. nextRunFor never
// calls it with a negative delta today, but the arithmetic must still be
// correct if a future caller ever advances a schedule backwards.
// ============================================================================

func TestAddMonthsClamped_NegativeMonthsWrapsTheYear(t *testing.T) {
	start := mustDate(t, "2026-03-15")
	got := addMonthsClamped(start, -5)
	if want := "2025-10-15"; got.Format(dateLayout) != want {
		t.Errorf("addMonthsClamped(2026-03-15, -5) = %s, want %s", got.Format(dateLayout), want)
	}
}
