package quote

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// These tests run against DATABASE_URL as kmuhub_app (RLS enforced) and are
// NOT behind the `integration` build tag, so they count towards the CI gate
// and coverage. The tagged *_db_test.go files need testcontainers.

func envQuoteRepo(t *testing.T) (*PostgresRepository, *pgxpool.Pool) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	return NewPostgresRepository(pool), pool
}

// envTenant creates a tenant row and removes everything the tests wrote for it.
func envTenant(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, context.Context) {
	t.Helper()
	id := uuid.New()
	testutil.EnsureTenant(t, pool, id, "quote-env-"+id.String()[:8])
	sys := testutil.WithSystemCtx(context.Background())
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM finance_quotes WHERE tenant_id = $1`,
			`DELETE FROM finance_number_sequences WHERE tenant_id = $1`,
			`DELETE FROM company_settings WHERE tenant_id = $1`,
			`DELETE FROM tenants WHERE id = $1`,
		} {
			if _, err := pool.Exec(sys, q, id); err != nil {
				t.Logf("cleanup %q: %v", q, err)
			}
		}
	})
	return id, testutil.WithTenantCtx(context.Background(), id)
}

// envDeal inserts a deal (with the user and stage it needs) because
// finance_quotes.deal_id is a foreign key.
func envDeal(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	sys := testutil.WithSystemCtx(context.Background())
	userID, stageID, dealID := uuid.New(), uuid.New(), uuid.New()
	for _, st := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users (id, email, password_hash, tenant_id) VALUES ($1, $2, 'x', $3)`, []any{userID, userID.String() + "@quote-env.test", tenantID}},
		{`INSERT INTO pipeline_stages (id, name, tenant_id) VALUES ($1, 'Env', $2)`, []any{stageID, tenantID}},
		{`INSERT INTO deals (id, name, stage_id, created_by, tenant_id) VALUES ($1, 'Env Deal', $2, $3, $4)`, []any{dealID, stageID, userID, tenantID}},
	} {
		if _, err := pool.Exec(sys, st.sql, st.args...); err != nil {
			t.Fatalf("seed %q: %v", st.sql, err)
		}
	}
	t.Cleanup(func() {
		for _, st := range []struct {
			sql string
			id  uuid.UUID
		}{
			{`DELETE FROM deals WHERE id = $1`, dealID},
			{`DELETE FROM pipeline_stages WHERE id = $1`, stageID},
			{`DELETE FROM users WHERE id = $1`, userID},
		} {
			if _, err := pool.Exec(sys, st.sql, st.id); err != nil {
				t.Logf("cleanup %q: %v", st.sql, err)
			}
		}
	})
	return dealID
}

func envLines() []models.LineItem {
	return []models.LineItem{
		{Description: "Analyse", Quantity: decimal.RequireFromString("1"), UnitPrice: decimal.RequireFromString("150.00"), TaxRate: decimal.RequireFromString("19"), LineTotal: decimal.RequireFromString("150.00")},
		{Description: "Reisezeit", Quantity: decimal.RequireFromString("2.5"), UnitPrice: decimal.RequireFromString("20.00"), TaxRate: decimal.RequireFromString("7"), LineTotal: decimal.RequireFromString("50.00")},
	}
}

func envQuote(t *testing.T, tenantID uuid.UUID, lines []models.LineItem) *models.Quote {
	t.Helper()
	raw, err := json.Marshal(lines)
	if err != nil {
		t.Fatalf("marshal lines: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return &models.Quote{
		ID:           uuid.New(),
		TenantID:     tenantID,
		QuoteNumber:  "AN-ENV-" + uuid.NewString()[:8],
		Status:       models.QuoteStatusDraft,
		CustomerName: "Env AG",
		TaxMode:      models.TaxModeStandard,
		LineItems:    raw,
		Subtotal:     decimal.RequireFromString("200.00"),
		TotalTax:     decimal.RequireFromString("38.00"),
		GrossTotal:   decimal.RequireFromString("238.00"),
		Currency:     "EUR",
		CreatedBy:    uuid.New(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func envParsedLines(t *testing.T, q *models.Quote) []models.LineItem {
	t.Helper()
	items, err := ParseLineItems(q.LineItems)
	if err != nil {
		t.Fatalf("ParseLineItems: %v", err)
	}
	return items
}

func TestEnvRepo_CreateGetByID_RoundtripsLinesInPositionOrder(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantID, ctx := envTenant(t, pool)

	q := envQuote(t, tenantID, envLines())
	dealID := envDeal(t, pool, tenantID)
	q.DealID = &dealID
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, tenantID, q.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.QuoteNumber != q.QuoteNumber || got.CustomerName != "Env AG" || got.Currency != "EUR" {
		t.Errorf("header fields not roundtripped: %+v", got)
	}
	if got.DealID == nil || *got.DealID != dealID {
		t.Errorf("DealID: got %v, want %s", got.DealID, dealID)
	}
	if !got.GrossTotal.Equal(decimal.RequireFromString("238.00")) {
		t.Errorf("GrossTotal: got %s, want 238.00", got.GrossTotal)
	}
	if got.ConvertedInvoiceNumber != nil {
		t.Errorf("ConvertedInvoiceNumber: got %v, want nil for an unconverted quote", *got.ConvertedInvoiceNumber)
	}
	lines := envParsedLines(t, got)
	if len(lines) != 2 {
		t.Fatalf("line count: got %d, want 2", len(lines))
	}
	if lines[0].Position != 1 || lines[1].Position != 2 {
		t.Errorf("positions: got %d,%d want 1,2 (missing positions are numbered by index)", lines[0].Position, lines[1].Position)
	}
	if lines[0].Description != "Analyse" || !lines[1].Quantity.Equal(decimal.RequireFromString("2.5")) {
		t.Errorf("line content not roundtripped: %+v", lines)
	}
}

func TestEnvRepo_Create_BadLineJSONFailsAndLeavesNoQuote(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantID, ctx := envTenant(t, pool)

	q := envQuote(t, tenantID, envLines())
	q.LineItems = json.RawMessage(`{"not":"an array"}`)
	if err := repo.Create(ctx, q); err == nil {
		t.Fatal("Create with malformed line items: want error, got nil")
	}
	if _, err := repo.GetByID(ctx, tenantID, q.ID); err != ErrQuoteNotFound {
		t.Errorf("quote header must be rolled back with the failed lines: got err %v", err)
	}
}

func TestEnvRepo_GetByID_OtherTenantIsNotFound(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantA, ctxA := envTenant(t, pool)
	tenantB, ctxB := envTenant(t, pool)

	q := envQuote(t, tenantA, envLines())
	if err := repo.Create(ctxA, q); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := repo.GetByID(ctxB, tenantB, q.ID); err != ErrQuoteNotFound {
		t.Errorf("GetByID from foreign tenant: got %v, want ErrQuoteNotFound", err)
	}
	// Even naming the right tenant in the argument, RLS on the pool must hide the row.
	if _, err := repo.GetByID(ctxB, tenantA, q.ID); err != ErrQuoteNotFound {
		t.Errorf("GetByID with foreign tenant arg under own ctx: got %v, want ErrQuoteNotFound", err)
	}
}

func TestEnvRepo_Update_ReplacesLinesAndFields(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantID, ctx := envTenant(t, pool)

	q := envQuote(t, tenantID, envLines())
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	replacement := []models.LineItem{
		{Description: "Nur eine Zeile", Quantity: decimal.RequireFromString("3"), UnitPrice: decimal.RequireFromString("10"), TaxRate: decimal.RequireFromString("19"), LineTotal: decimal.RequireFromString("30")},
	}
	raw, _ := json.Marshal(replacement)
	q.LineItems = raw
	q.CustomerName = "Geändert GmbH"
	q.Status = models.QuoteStatusSent
	q.Notes = "n"
	q.GrossTotal = decimal.RequireFromString("35.70")
	q.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := repo.Update(ctx, q); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, tenantID, q.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.CustomerName != "Geändert GmbH" || got.Status != models.QuoteStatusSent {
		t.Errorf("fields not updated: name=%q status=%q", got.CustomerName, got.Status)
	}
	if !got.GrossTotal.Equal(decimal.RequireFromString("35.70")) {
		t.Errorf("GrossTotal: got %s, want 35.70", got.GrossTotal)
	}
	lines := envParsedLines(t, got)
	if len(lines) != 1 || lines[0].Description != "Nur eine Zeile" {
		t.Errorf("old lines must be replaced, got %+v", lines)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM finance_quote_lines WHERE quote_id = $1`, q.ID).Scan(&n); err != nil {
		t.Fatalf("count lines: %v", err)
	}
	if n != 1 {
		t.Errorf("orphan line rows: got %d, want 1", n)
	}
}

func TestEnvRepo_Update_BadLineJSONRollsBackWholeUpdate(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantID, ctx := envTenant(t, pool)

	q := envQuote(t, tenantID, envLines())
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}
	q.CustomerName = "Darf nicht ankommen"
	q.LineItems = json.RawMessage(`[`)
	if err := repo.Update(ctx, q); err == nil {
		t.Fatal("Update with malformed line items: want error, got nil")
	}
	got, err := repo.GetByID(ctx, tenantID, q.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.CustomerName != "Env AG" {
		t.Errorf("header update must roll back with the failed lines: name=%q", got.CustomerName)
	}
	if lines := envParsedLines(t, got); len(lines) != 2 {
		t.Errorf("original lines must survive a failed update: got %d", len(lines))
	}
}

func TestEnvRepo_UpdateStatusAndDelete_AreTenantScoped(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantA, ctxA := envTenant(t, pool)
	tenantB, ctxB := envTenant(t, pool)

	q := envQuote(t, tenantA, envLines())
	if err := repo.Create(ctxA, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateStatus(ctxB, tenantB, q.ID, models.QuoteStatusAccepted); err != nil {
		t.Fatalf("foreign UpdateStatus should be a silent no-op: %v", err)
	}
	if err := repo.Delete(ctxB, tenantB, q.ID); err != nil {
		t.Fatalf("foreign Delete should be a silent no-op: %v", err)
	}
	got, err := repo.GetByID(ctxA, tenantA, q.ID)
	if err != nil {
		t.Fatalf("quote must survive foreign Delete: %v", err)
	}
	if got.Status != models.QuoteStatusDraft {
		t.Errorf("foreign UpdateStatus changed the status to %q", got.Status)
	}

	if err := repo.UpdateStatus(ctxA, tenantA, q.ID, models.QuoteStatusAccepted); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ = repo.GetByID(ctxA, tenantA, q.ID)
	if got.Status != models.QuoteStatusAccepted {
		t.Errorf("status: got %q, want %q", got.Status, models.QuoteStatusAccepted)
	}

	if err := repo.Delete(ctxA, tenantA, q.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctxA, tenantA, q.ID); err != ErrQuoteNotFound {
		t.Errorf("after Delete: got %v, want ErrQuoteNotFound", err)
	}
	var n int
	if err := pool.QueryRow(ctxA, `SELECT COUNT(*) FROM finance_quote_lines WHERE quote_id = $1`, q.ID).Scan(&n); err != nil {
		t.Fatalf("count lines: %v", err)
	}
	if n != 0 {
		t.Errorf("lines must cascade with the quote, got %d", n)
	}
}

func TestEnvRepo_List_FiltersPaginatesAndStaysInTenant(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantA, ctxA := envTenant(t, pool)
	tenantB, ctxB := envTenant(t, pool)

	base := time.Now().UTC().Truncate(time.Second).Add(-10 * time.Hour)
	deal := envDeal(t, pool, tenantA)
	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		q := envQuote(t, tenantA, envLines())
		q.CreatedAt = base.Add(time.Duration(i) * time.Hour)
		q.UpdatedAt = q.CreatedAt
		if i == 0 {
			q.Status = models.QuoteStatusSent
			q.DealID = &deal
		}
		if err := repo.Create(ctxA, q); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		ids = append(ids, q.ID)
	}
	foreign := envQuote(t, tenantB, envLines())
	if err := repo.Create(ctxB, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}

	all, total, err := repo.List(ctxA, tenantA, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("List: total=%d len=%d, want 3/3 (foreign tenant row must not appear)", total, len(all))
	}
	if all[0].ID != ids[2] || all[2].ID != ids[0] {
		t.Errorf("order must be created_at DESC: got %s..%s", all[0].ID, all[2].ID)
	}
	if got := envParsedLines(t, all[0]); len(got) != 2 {
		t.Errorf("List must bulk-load lines, got %d for first quote", len(got))
	}

	sent, total, err := repo.List(ctxA, tenantA, ListFilter{Status: models.QuoteStatusSent})
	if err != nil || total != 1 || len(sent) != 1 || sent[0].ID != ids[0] {
		t.Errorf("status filter: err=%v total=%d len=%d", err, total, len(sent))
	}
	byDeal, total, err := repo.List(ctxA, tenantA, ListFilter{DealID: &deal})
	if err != nil || total != 1 || len(byDeal) != 1 || byDeal[0].ID != ids[0] {
		t.Errorf("deal filter: err=%v total=%d len=%d", err, total, len(byDeal))
	}

	from, to := base.Add(30*time.Minute), base.Add(90*time.Minute)
	window, total, err := repo.List(ctxA, tenantA, ListFilter{DateFrom: &from, DateTo: &to})
	if err != nil || total != 1 || len(window) != 1 || window[0].ID != ids[1] {
		t.Errorf("date window: err=%v total=%d len=%d", err, total, len(window))
	}

	page, total, err := repo.List(ctxA, tenantA, ListFilter{Limit: 2, Offset: 2})
	if err != nil || total != 3 || len(page) != 1 || page[0].ID != ids[0] {
		t.Errorf("pagination: err=%v total=%d len=%d (total counts all matches, not the page)", err, total, len(page))
	}
}

func TestEnvRepo_GetByDealID_ScopedToTenantAndDeal(t *testing.T) {
	repo, pool := envQuoteRepo(t)
	tenantA, ctxA := envTenant(t, pool)
	tenantB, ctxB := envTenant(t, pool)

	deal := envDeal(t, pool, tenantA)
	mine := envQuote(t, tenantA, envLines())
	mine.DealID = &deal
	other := envQuote(t, tenantA, envLines())
	otherDeal := envDeal(t, pool, tenantA)
	other.DealID = &otherDeal
	foreign := envQuote(t, tenantB, envLines())
	foreignDeal := envDeal(t, pool, tenantB)
	foreign.DealID = &foreignDeal
	for _, q := range []*models.Quote{mine, other} {
		if err := repo.Create(ctxA, q); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	if err := repo.Create(ctxB, foreign); err != nil {
		t.Fatalf("Create foreign: %v", err)
	}

	got, err := repo.GetByDealID(ctxA, tenantA, deal)
	if err != nil {
		t.Fatalf("GetByDealID: %v", err)
	}
	if len(got) != 1 || got[0].ID != mine.ID {
		t.Fatalf("got %d quotes, want only the tenant's quote for the deal", len(got))
	}
	if lines := envParsedLines(t, got[0]); len(lines) != 2 {
		t.Errorf("GetByDealID must load lines, got %d", len(lines))
	}

	none, err := repo.GetByDealID(ctxA, tenantA, uuid.New())
	if err != nil || len(none) != 0 {
		t.Errorf("unknown deal: err=%v len=%d, want empty", err, len(none))
	}
}

func TestEnvNumberSequence_GapFreeAndInfo(t *testing.T) {
	_, pool := envQuoteRepo(t)
	tenantA, ctxA := envTenant(t, pool)
	tenantB, ctxB := envTenant(t, pool)
	seq := NewPostgresNumberSequenceRepo(pool)

	info, err := seq.GetSequenceInfo(ctxA, tenantA, "quote", 2026)
	if err != nil || info != nil {
		t.Fatalf("unseen sequence: info=%v err=%v, want nil/nil", info, err)
	}

	for i, want := range []string{"AN-2026-0001", "AN-2026-0002", "AN-2026-0003"} {
		got, err := seq.NextNumber(ctxA, tenantA, "quote", 2026, "AN")
		if err != nil || got != want {
			t.Fatalf("NextNumber #%d: got %q err=%v, want %q", i+1, got, err, want)
		}
	}
	// Other fiscal year and other tenant count independently.
	if got, _ := seq.NextNumber(ctxA, tenantA, "quote", 2027, "AN"); got != "AN-2027-0001" {
		t.Errorf("new fiscal year: got %q, want AN-2027-0001", got)
	}
	if got, _ := seq.NextNumber(ctxB, tenantB, "quote", 2026, "AN"); got != "AN-2026-0001" {
		t.Errorf("other tenant: got %q, want AN-2026-0001", got)
	}

	info, err = seq.GetSequenceInfo(ctxA, tenantA, "quote", 2026)
	if err != nil || info == nil || info.CurrentNumber != 3 || info.FiscalYear != 2026 {
		t.Errorf("GetSequenceInfo: info=%+v err=%v, want current 3 / FY 2026", info, err)
	}
}

func TestEnvNumberSequence_InTxRollbackBurnsNoNumber(t *testing.T) {
	_, pool := envQuoteRepo(t)
	tenantID, ctx := envTenant(t, pool)
	seq := NewPostgresNumberSequenceRepo(pool)

	if got, _ := seq.NextNumber(ctx, tenantID, "quote", 2026, "AN"); got != "AN-2026-0001" {
		t.Fatalf("first number: got %q", got)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after the explicit Rollback; keeps a failed assertion from holding the sequence lock
	burned, err := seq.NextNumberInTx(ctx, tx, tenantID, "quote", 2026, "AN")
	if err != nil || burned != "AN-2026-0002" {
		t.Fatalf("NextNumberInTx: got %q err=%v", burned, err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	// GoBD: the rolled-back number must be handed out again, no gap.
	if got, _ := seq.NextNumber(ctx, tenantID, "quote", 2026, "AN"); got != "AN-2026-0002" {
		t.Errorf("after rollback: got %q, want AN-2026-0002 (no gap)", got)
	}
}

func TestEnvCompanySettings_UpsertRoundtripAndTenantIsolation(t *testing.T) {
	_, pool := envQuoteRepo(t)
	tenantA, ctxA := envTenant(t, pool)
	tenantB, ctxB := envTenant(t, pool)
	repo := NewPostgresCompanySettingsRepo(pool)

	if got, err := repo.GetByTenantID(ctxA, tenantA); err != nil || got != nil {
		t.Fatalf("before Upsert: got %+v err=%v, want nil/nil", got, err)
	}

	s := &models.CompanySettings{
		TenantID:                 tenantA,
		Name:                     "Zentria GmbH",
		IBAN:                     "DE02120300000000202051",
		IsKleinunternehmer:       true,
		DefaultPaymentTermsDays:  14,
		DefaultQuoteValidityDays: 21,
		Basiszinssatz:            decimal.RequireFromString("-0.88"),
		DefaultCurrency:          "CHF",
		DatevBeraterNr:           "12345",
		UpdatedAt:                time.Now().UTC(),
	}
	if err := repo.Upsert(ctxA, s); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if s.ID == uuid.Nil {
		t.Error("Upsert must assign an ID when none is set")
	}

	got, err := repo.GetByTenantID(ctxA, tenantA)
	if err != nil || got == nil {
		t.Fatalf("GetByTenantID: got %+v err=%v", got, err)
	}
	if got.Name != "Zentria GmbH" || got.IBAN != "DE02120300000000202051" || !got.IsKleinunternehmer ||
		got.DefaultPaymentTermsDays != 14 || got.DefaultQuoteValidityDays != 21 ||
		got.DefaultCurrency != "CHF" || got.DatevBeraterNr != "12345" {
		t.Errorf("settings not roundtripped: %+v", got)
	}
	if !got.Basiszinssatz.Equal(decimal.RequireFromString("-0.88")) {
		t.Errorf("Basiszinssatz: got %s, want -0.88", got.Basiszinssatz)
	}

	// Second upsert updates the single row; an empty currency must not blank the column.
	s.Name = "Zentria UG"
	s.DefaultCurrency = ""
	if err := repo.Upsert(ctxA, s); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if got, err := repo.GetByTenantID(ctxA, tenantA); err != nil || got.Name != "Zentria UG" || got.DefaultCurrency != models.DefaultCurrency {
		t.Errorf("after second Upsert: %+v err=%v, want name updated and currency %q", got, err, models.DefaultCurrency)
	}
	var rows int
	if err := pool.QueryRow(ctxA, `SELECT COUNT(*) FROM company_settings WHERE tenant_id = $1`, tenantA).Scan(&rows); err != nil || rows != 1 {
		t.Errorf("rows after second Upsert: got %d err=%v, want 1", rows, err)
	}

	// Tenant B sees nothing of tenant A, even when asking for A's id.
	if got, err := repo.GetByTenantID(ctxB, tenantA); err != nil || got != nil {
		t.Errorf("foreign tenant read: got %+v err=%v, want nil/nil", got, err)
	}
	if got, err := repo.GetByTenantID(ctxB, tenantB); err != nil || got != nil {
		t.Errorf("own tenant without settings: got %+v err=%v, want nil/nil", got, err)
	}
}

func TestEnvParseHelpers(t *testing.T) {
	if items, err := ParseLineItems(nil); err != nil || items != nil {
		t.Errorf("ParseLineItems(nil): %v %v, want nil/nil", items, err)
	}
	if _, err := ParseLineItems(json.RawMessage(`{`)); err == nil {
		t.Error("ParseLineItems(malformed): want error")
	}
	if tb, err := ParseTaxBreakdown(nil); err != nil || tb != nil {
		t.Errorf("ParseTaxBreakdown(nil): %v %v, want nil/nil", tb, err)
	}
	if _, err := ParseTaxBreakdown(json.RawMessage(`[`)); err == nil {
		t.Error("ParseTaxBreakdown(malformed): want error")
	}
	raw, err := marshalTaxBreakdown(&models.TaxBreakdown{})
	if err != nil || len(raw) == 0 {
		t.Fatalf("marshalTaxBreakdown: %q %v", raw, err)
	}
	if tb, err := ParseTaxBreakdown(raw); err != nil || tb == nil {
		t.Errorf("ParseTaxBreakdown(roundtrip): %v %v", tb, err)
	}
	if raw, err := marshalTaxBreakdown(nil); err != nil || raw != nil {
		t.Errorf("marshalTaxBreakdown(nil): %q %v, want nil/nil", raw, err)
	}
	std := parseTaxMode(models.TaxModeStandard)
	if got := parseTaxMode("unknown"); got != std {
		t.Errorf("unknown mode must fall back to standard, got %v", got)
	}
	rc, klein := parseTaxMode(models.TaxModeReverseCharge), parseTaxMode(models.TaxModeKleinunternehmer)
	if rc == klein || rc == std || klein == std {
		t.Errorf("tax modes must be distinct: rc=%v klein=%v std=%v", rc, klein, std)
	}
}
