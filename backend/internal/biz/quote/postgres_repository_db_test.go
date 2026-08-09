//go:build integration

package quote

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testsupport/pgtc"
)

// ============================================================================
// PostgresRepository.Delete / UpdateStatus — previously 0.0% covered.
// ============================================================================

// TestRepository_Delete_RemovesQuoteAndLines verifies Delete removes the quote
// row and (via ON DELETE CASCADE on finance_quote_lines) its line items, and
// is tenant-scoped: a foreign tenant's delete call must not affect the row.
func TestRepository_Delete_RemovesQuoteAndLines(t *testing.T) {
	appPool, superPoolURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superPoolURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	otherTenant := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	otherCtx := pgtc.TenantCtx(context.Background(), otherTenant)
	seedTenantQ(t, appPool, ctx, tenantID)
	seedTenantQ(t, appPool, otherCtx, otherTenant)

	repo := NewPostgresRepository(appPool)
	q := makeQuote(tenantID, twoQuoteLines())
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := countQuoteLineRows(t, superPool, q.ID); n != 2 {
		t.Fatalf("after Create: got %d line rows, want 2", n)
	}

	// A foreign tenant's Delete must not remove the row (RLS-scoped statement
	// affects 0 rows, no error).
	if err := repo.Delete(otherCtx, otherTenant, q.ID); err != nil {
		t.Fatalf("cross-tenant Delete should not error: %v", err)
	}
	if _, err := repo.GetByID(ctx, tenantID, q.ID); err != nil {
		t.Fatalf("quote must still exist after cross-tenant delete attempt: %v", err)
	}

	if err := repo.Delete(ctx, tenantID, q.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, tenantID, q.ID); err == nil {
		t.Error("expected ErrQuoteNotFound after Delete, got nil error")
	}
	if n := countQuoteLineRows(t, superPool, q.ID); n != 0 {
		t.Errorf("after Delete: got %d orphan line rows, want 0 (cascade)", n)
	}
}

// TestRepository_UpdateStatus_IsTenantScoped verifies UpdateStatus changes the
// status column and that a foreign tenant cannot flip another tenant's status.
func TestRepository_UpdateStatus_IsTenantScoped(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	otherTenant := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	otherCtx := pgtc.TenantCtx(context.Background(), otherTenant)
	seedTenantQ(t, appPool, ctx, tenantID)
	seedTenantQ(t, appPool, otherCtx, otherTenant)

	repo := NewPostgresRepository(appPool)
	q := makeQuote(tenantID, twoQuoteLines())
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateStatus(otherCtx, otherTenant, q.ID, models.QuoteStatusSent); err != nil {
		t.Fatalf("cross-tenant UpdateStatus should not error: %v", err)
	}
	got, err := repo.GetByID(ctx, tenantID, q.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != models.QuoteStatusDraft {
		t.Errorf("status after cross-tenant UpdateStatus: got %q, want draft (unchanged)", got.Status)
	}

	if err := repo.UpdateStatus(ctx, tenantID, q.ID, models.QuoteStatusSent); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, err = repo.GetByID(ctx, tenantID, q.ID)
	if err != nil {
		t.Fatalf("GetByID after UpdateStatus: %v", err)
	}
	if got.Status != models.QuoteStatusSent {
		t.Errorf("status after UpdateStatus: got %q, want sent", got.Status)
	}
}

// ============================================================================
// PostgresNumberSequenceRepo — previously 0.0% covered anywhere in the repo,
// despite backing GoBD gap-free numbering for quotes, invoices and credit
// notes. NextNumber/NextNumberInTx/GetSequenceInfo have never run against
// real Postgres.
// ============================================================================

func TestNumberSequenceRepo_SequentialWithinTenantAndYear(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresNumberSequenceRepo(appPool)
	year := time.Now().Year()

	n1, err := repo.NextNumber(ctx, tenantID, models.DocumentTypeQuote, year, "AN")
	if err != nil {
		t.Fatalf("NextNumber #1: %v", err)
	}
	n2, err := repo.NextNumber(ctx, tenantID, models.DocumentTypeQuote, year, "AN")
	if err != nil {
		t.Fatalf("NextNumber #2: %v", err)
	}
	n3, err := repo.NextNumber(ctx, tenantID, models.DocumentTypeQuote, year, "AN")
	if err != nil {
		t.Fatalf("NextNumber #3: %v", err)
	}

	wantPrefix := "AN-"
	if n1 == n2 || n2 == n3 || n1 == n3 {
		t.Fatalf("sequence must be strictly increasing, got %q, %q, %q", n1, n2, n3)
	}
	for _, n := range []string{n1, n2, n3} {
		if len(n) < len(wantPrefix) || n[:len(wantPrefix)] != wantPrefix {
			t.Errorf("number %q does not start with prefix %q", n, wantPrefix)
		}
	}

	info, err := repo.GetSequenceInfo(ctx, tenantID, models.DocumentTypeQuote, year)
	if err != nil {
		t.Fatalf("GetSequenceInfo: %v", err)
	}
	if info == nil || info.CurrentNumber != 3 {
		t.Fatalf("GetSequenceInfo: got %+v, want CurrentNumber=3", info)
	}
}

// TestNumberSequenceRepo_IndependentPerFiscalYearAndTenant verifies that a new
// fiscal year and a different tenant each start their own gap-free sequence
// at 1, rather than sharing a single counter.
func TestNumberSequenceRepo_IndependentPerFiscalYearAndTenant(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxA := pgtc.TenantCtx(context.Background(), tenantA)
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)
	seedTenantQ(t, appPool, ctxA, tenantA)
	seedTenantQ(t, appPool, ctxB, tenantB)

	repo := NewPostgresNumberSequenceRepo(appPool)
	year := time.Now().Year()

	if _, err := repo.NextNumber(ctxA, tenantA, models.DocumentTypeQuote, year, "AN"); err != nil {
		t.Fatalf("NextNumber tenantA year: %v", err)
	}
	if _, err := repo.NextNumber(ctxA, tenantA, models.DocumentTypeQuote, year, "AN"); err != nil {
		t.Fatalf("NextNumber tenantA year #2: %v", err)
	}

	// A different fiscal year for the same tenant starts fresh at 1.
	nextYear, err := repo.NextNumber(ctxA, tenantA, models.DocumentTypeQuote, year+1, "AN")
	if err != nil {
		t.Fatalf("NextNumber tenantA nextYear: %v", err)
	}
	infoNextYear, err := repo.GetSequenceInfo(ctxA, tenantA, models.DocumentTypeQuote, year+1)
	if err != nil {
		t.Fatalf("GetSequenceInfo nextYear: %v", err)
	}
	if infoNextYear == nil || infoNextYear.CurrentNumber != 1 {
		t.Fatalf("new fiscal year sequence: got %+v, want CurrentNumber=1", infoNextYear)
	}
	wantNextYear := "AN-" + strconv.Itoa(year+1) + "-0001"
	if nextYear != wantNextYear {
		t.Errorf("nextYear number: got %q, want %q", nextYear, wantNextYear)
	}

	// A different tenant, same document type and year, also starts fresh at 1 —
	// a forgotten tenant_id condition here would leak tenant A's counter.
	firstB, err := repo.NextNumber(ctxB, tenantB, models.DocumentTypeQuote, year, "AN")
	if err != nil {
		t.Fatalf("NextNumber tenantB: %v", err)
	}
	wantFirstB := "AN-" + strconv.Itoa(year) + "-0001"
	if firstB != wantFirstB {
		t.Errorf("tenantB first number: got %q, want %q (must not continue tenantA's counter)", firstB, wantFirstB)
	}
}

// TestNumberSequenceRepo_GetSequenceInfo_NilForUnseenSequence covers the
// pgx.ErrNoRows branch: a sequence that was never assigned must return
// (nil, nil), not an error.
func TestNumberSequenceRepo_GetSequenceInfo_NilForUnseenSequence(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresNumberSequenceRepo(appPool)
	info, err := repo.GetSequenceInfo(ctx, tenantID, models.DocumentTypeQuote, time.Now().Year())
	if err != nil {
		t.Fatalf("GetSequenceInfo for unseen sequence should not error: %v", err)
	}
	if info != nil {
		t.Errorf("GetSequenceInfo for unseen sequence: got %+v, want nil", info)
	}
}

// ============================================================================
// Service.Send against a real pool — the atomic number-assignment + status
// transition that service.go explicitly calls out as GoBD-critical (a
// previous bug committed the number in a separate tx from the status update,
// leaving a consumed-but-unassigned number on partial failure). Exercised
// here through the real Service, real repos and a real pool for the first
// time — the existing service_test.go only exercises this with a
// noopTxBeginner + mock repos, which cannot catch a broken tx coupling.
// ============================================================================

// TestSend_RealDB_AssignsNumberAndStatusAtomically proves the happy path
// commits both the sequence increment and the quote row in one transaction.
func TestSend_RealDB_AssignsNumberAndStatusAtomically(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	userID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	numRepo := NewPostgresNumberSequenceRepo(appPool)
	csRepo := NewPostgresCompanySettingsRepo(appPool)
	svc := NewService(repo, numRepo, csRepo, nil, appPool)

	q := makeQuote(tenantID, twoQuoteLines())
	q.Status = models.QuoteStatusDraft
	q.QuoteNumber = ""
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Send(ctx, tenantID, q.ID, userID); err != nil {
		t.Fatalf("Send: %v", err)
	}

	got, err := repo.GetByID(ctx, tenantID, q.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != models.QuoteStatusSent {
		t.Errorf("status after Send: got %q, want sent", got.Status)
	}
	year := time.Now().Year()
	wantNumber := "AN-" + strconv.Itoa(year) + "-0001"
	if got.QuoteNumber != wantNumber {
		t.Errorf("quote_number after Send: got %q, want %q", got.QuoteNumber, wantNumber)
	}

	info, err := numRepo.GetSequenceInfo(ctx, tenantID, models.DocumentTypeQuote, year)
	if err != nil {
		t.Fatalf("GetSequenceInfo: %v", err)
	}
	if info == nil || info.CurrentNumber != 1 {
		t.Fatalf("sequence after Send: got %+v, want CurrentNumber=1", info)
	}
}

// TestSend_RealDB_FailedUpdateRollsBackNumberAssignment is the regression test
// for the GoBD gap bug described in service.go's Send comment: if the status
// update fails AFTER the number was assigned within the same transaction, the
// whole transaction (including the sequence increment) must roll back, so the
// number is not burned. This replicates the exact statement pair Send uses
// (NextNumberInTx + UpdateInTx in one tx) but forces UpdateInTx to fail via
// the chk_finance_quotes_status CHECK constraint, since Send() itself always
// writes a valid status and cannot be driven into this failure from outside.
func TestSend_RealDB_FailedUpdateRollsBackNumberAssignment(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	numRepo := NewPostgresNumberSequenceRepo(appPool)
	year := time.Now().Year()

	q := makeQuote(tenantID, twoQuoteLines())
	q.Status = models.QuoteStatusDraft
	q.QuoteNumber = ""
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tx, err := appPool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	number, err := numRepo.NextNumberInTx(ctx, tx, tenantID, models.DocumentTypeQuote, year, "AN")
	if err != nil {
		t.Fatalf("NextNumberInTx: %v", err)
	}
	if number != "AN-"+strconv.Itoa(year)+"-0001" {
		t.Fatalf("unexpected first number: %q", number)
	}

	q.QuoteNumber = number
	q.Status = "bogus-status" // violates chk_finance_quotes_status
	updateErr := repo.UpdateInTx(ctx, tx, q)
	if updateErr == nil {
		t.Fatal("expected UpdateInTx to fail on invalid status (CHECK constraint), got nil")
	}
	if rbErr := tx.Rollback(ctx); rbErr != nil {
		t.Fatalf("Rollback: %v", rbErr)
	}

	// The quote itself must be unaffected — still draft, no number.
	got, err := repo.GetByID(ctx, tenantID, q.ID)
	if err != nil {
		t.Fatalf("GetByID after rollback: %v", err)
	}
	if got.Status != models.QuoteStatusDraft || got.QuoteNumber != "" {
		t.Errorf("quote after rolled-back Send: got status=%q number=%q, want draft/empty", got.Status, got.QuoteNumber)
	}

	// The critical assertion: the sequence increment rolled back too, so the
	// next real Send reuses number 0001 instead of skipping to 0002 (a
	// GoBD gap — a number consumed by a document that was never issued).
	again, err := numRepo.NextNumber(ctx, tenantID, models.DocumentTypeQuote, year, "AN")
	if err != nil {
		t.Fatalf("NextNumber after rollback: %v", err)
	}
	if again != "AN-"+strconv.Itoa(year)+"-0001" {
		t.Errorf("number after rolled-back Send: got %q, want AN-%d-0001 (rollback must return the burned number)", again, year)
	}
}

// ============================================================================
// PostgresCompanySettingsRepo — previously 0.0% covered. Basiszinssatz is
// scanned as a string and parsed to decimal (ADR-0007); a naive float
// round-trip would lose precision, and Basiszinssatz has been negative in
// real German base-rate history, so the sign must survive too.
// ============================================================================

func TestCompanySettingsRepo_UpsertAndGetByTenantID_RoundtripsDecimalPrecision(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresCompanySettingsRepo(appPool)

	// Not configured yet.
	none, err := repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenantID before Upsert: %v", err)
	}
	if none != nil {
		t.Fatalf("GetByTenantID before Upsert: got %+v, want nil", none)
	}

	settings := &models.CompanySettings{
		TenantID:                 tenantID,
		Name:                     "Zentria GmbH",
		DefaultQuoteValidityDays: 21,
		Basiszinssatz:            decimal.RequireFromString("-0.88"),
		DefaultCurrency:          "CHF",
		UpdatedAt:                time.Now().UTC().Truncate(time.Second),
	}
	if err := repo.Upsert(ctx, settings); err != nil {
		t.Fatalf("Upsert (insert): %v", err)
	}

	got, err := repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenantID after insert: %v", err)
	}
	if got == nil {
		t.Fatal("GetByTenantID after insert: got nil")
	}
	if !got.Basiszinssatz.Equal(decimal.RequireFromString("-0.88")) {
		t.Errorf("Basiszinssatz: got %s, want -0.88 (sign/precision must survive the string round-trip)", got.Basiszinssatz)
	}
	if got.DefaultCurrency != "CHF" {
		t.Errorf("DefaultCurrency: got %q, want CHF", got.DefaultCurrency)
	}
	if got.Name != "Zentria GmbH" {
		t.Errorf("Name: got %q, want Zentria GmbH", got.Name)
	}

	// Second Upsert on the same tenant must UPDATE (ON CONFLICT), not insert a
	// second row — uq_company_settings_tenant would reject a real duplicate,
	// but an app-level bug forming two rows differently would silently pass
	// GetByTenantID (LIMIT 1 semantics of a bare SELECT). Verify the value
	// actually changed instead.
	settings.Basiszinssatz = decimal.RequireFromString("1.95")
	settings.DefaultCurrency = "" // empty must NOT persist as "" (B6 guard)
	if err := repo.Upsert(ctx, settings); err != nil {
		t.Fatalf("Upsert (update): %v", err)
	}
	got2, err := repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenantID after update: %v", err)
	}
	if !got2.Basiszinssatz.Equal(decimal.RequireFromString("1.95")) {
		t.Errorf("Basiszinssatz after update: got %s, want 1.95", got2.Basiszinssatz)
	}
	// B6 guard: an empty DefaultCurrency falls back to the system default (EUR),
	// not to the previously-persisted tenant value (CHF) — it does not "keep old
	// value on empty", it substitutes the column default so the field is never
	// blank. Confirmed by reading Upsert's defaultCurrency fallback.
	if got2.DefaultCurrency != models.DefaultCurrency {
		t.Errorf("DefaultCurrency after empty-string Upsert: got %q, want %q (system default, never blank)", got2.DefaultCurrency, models.DefaultCurrency)
	}
}
