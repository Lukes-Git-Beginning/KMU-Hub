package creditnote

// postgres_repository_queries_real_sql_test.go runs the read-side queries of the
// repository (List, GetByInvoiceID, ListForDATEVExport, SumGrossByInvoiceInTx)
// against the local Postgres as kmuhub_app. The same queries are also exercised in
// repository_coverage_integration_test.go, but that file sits behind
// //go:build integration (testcontainers) and therefore counts for neither the
// PR coverage gate nor a plain `go test` — these untagged tests do.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedTenantWithInvoice creates a tenant and one invoice for it, both removed again
// on cleanup.
func seedInvoiceFor(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"invoice_date": time.Now().UTC(), "due_date": time.Now().UTC().AddDate(0, 0, 14),
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })
	return invoiceID
}

// seedTenantWithInvoice creates a tenant and one invoice for it, both removed again
// on cleanup.
func seedTenantWithInvoice(t *testing.T, pool *pgxpool.Pool, name string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, name)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	return tenantID, seedInvoiceFor(t, pool, tenantID)
}

func createCreditNote(t *testing.T, pool *pgxpool.Pool, repo *PostgresRepository, cn *models.CreditNote) {
	t.Helper()
	ctx := testutil.WithTenantCtx(context.Background(), cn.TenantID)
	require.NoError(t, repo.Create(ctx, cn))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_credit_notes", cn.ID) })
}

func TestPostgresRepository_List_FiltersPaginationAndTenantScoping(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, invA := seedTenantWithInvoice(t, pool, "CreditNote List Tenant A")
	invA2 := seedInvoiceFor(t, pool, tenantA)
	tenantB, invB := seedTenantWithInvoice(t, pool, "CreditNote List Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	base := time.Now().UTC().Truncate(time.Second)
	mk := func(tenant, inv uuid.UUID, age time.Duration) *models.CreditNote {
		cn := baseCreditNote(tenant, inv, oneCreditNoteLine())
		cn.CreatedAt = base.Add(-age)
		cn.UpdatedAt = cn.CreatedAt
		return cn
	}
	cnDraft := mk(tenantA, invA, 3*time.Minute)
	cnSent := mk(tenantA, invA, 2*time.Minute)
	cnSent.Status = models.CreditNoteStatusSent
	cnSent.CreditNoteNumber = "GS-LIST-" + uuid.New().String()[:8]
	cnOther := mk(tenantA, invA2, time.Minute)
	cnB := mk(tenantB, invB, 0)
	for _, cn := range []*models.CreditNote{cnDraft, cnSent, cnOther, cnB} {
		createCreditNote(t, pool, repo, cn)
	}

	all, total, err := repo.List(ctxA, tenantA, ListFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, all, 3)
	assert.Equal(t, []uuid.UUID{cnOther.ID, cnSent.ID, cnDraft.ID},
		[]uuid.UUID{all[0].ID, all[1].ID, all[2].ID}, "newest first (created_at DESC)")
	for _, cn := range all {
		assert.Equal(t, tenantA, cn.TenantID)
		assert.NotEmpty(t, cn.LineItems, "line items are bulk-loaded for the page")
	}

	sentOnly, sentTotal, err := repo.List(ctxA, tenantA, ListFilter{Status: models.CreditNoteStatusSent, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, sentTotal)
	require.Len(t, sentOnly, 1)
	assert.Equal(t, cnSent.ID, sentOnly[0].ID)

	byInvoice, invTotal, err := repo.List(ctxA, tenantA, ListFilter{OriginalInvID: &invA2, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, invTotal)
	require.Len(t, byInvoice, 1)
	assert.Equal(t, cnOther.ID, byInvoice[0].ID)

	page1, totalP1, err := repo.List(ctxA, tenantA, ListFilter{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, totalP1)
	require.Len(t, page1, 2)
	page2, totalP2, err := repo.List(ctxA, tenantA, ListFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, totalP2, "total counts the filtered set, not the page")
	require.Len(t, page2, 1)
	assert.Equal(t, cnDraft.ID, page2[0].ID)

	// Limits outside 1..100 fall back to the default page size instead of erroring.
	capped, _, err := repo.List(ctxA, tenantA, ListFilter{Limit: 1000})
	require.NoError(t, err)
	assert.Len(t, capped, 3)

	// Tenant B sees only its own row.
	bList, bTotal, err := repo.List(ctxB, tenantB, ListFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, bTotal)
	require.Len(t, bList, 1)
	assert.Equal(t, cnB.ID, bList[0].ID)

	// An explicit foreign tenant argument under tenant A's context matches nothing.
	none, noneTotal, err := repo.List(ctxA, tenantB, ListFilter{Limit: 10})
	require.NoError(t, err)
	assert.Zero(t, noneTotal)
	assert.Empty(t, none)
}

func TestPostgresRepository_GetByInvoiceID_TenantScopedAndNewestFirst(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, invA := seedTenantWithInvoice(t, pool, "CreditNote ByInvoice Tenant A")
	tenantB, invB := seedTenantWithInvoice(t, pool, "CreditNote ByInvoice Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	older := baseCreditNote(tenantA, invA, oneCreditNoteLine())
	older.CreatedAt = older.CreatedAt.Add(-time.Minute)
	newer := baseCreditNote(tenantA, invA, oneCreditNoteLine())
	cnB := baseCreditNote(tenantB, invB, oneCreditNoteLine())
	for _, cn := range []*models.CreditNote{older, newer, cnB} {
		createCreditNote(t, pool, repo, cn)
	}

	got, err := repo.GetByInvoiceID(ctxA, tenantA, invA)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, newer.ID, got[0].ID, "newest credit note first")
	assert.Equal(t, older.ID, got[1].ID)
	assert.NotEmpty(t, got[0].LineItems, "lines are bulk-loaded")

	none, err := repo.GetByInvoiceID(ctxA, tenantA, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, none)

	// The double-storno guard depends on this: another tenant never sees A's notes.
	wrongTenant, err := repo.GetByInvoiceID(ctxB, tenantB, invA)
	require.NoError(t, err)
	assert.Empty(t, wrongTenant)

	own, err := repo.GetByInvoiceID(ctxB, tenantB, invB)
	require.NoError(t, err)
	require.Len(t, own, 1)
	assert.Equal(t, cnB.ID, own[0].ID)
}

func TestPostgresRepository_ListForDATEVExport_RangeStatusAndKeysetPaging(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID, invID := seedTenantWithInvoice(t, pool, "CreditNote DATEV Tenant")
	otherTenant, otherInv := seedTenantWithInvoice(t, pool, "CreditNote DATEV Other Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	mkSent := func(tenant, inv uuid.UUID, offsetDays int) *models.CreditNote {
		cn := baseCreditNote(tenant, inv, oneCreditNoteLine())
		cn.Status = models.CreditNoteStatusSent
		cn.CreditNoteNumber = "GS-DATEV-" + uuid.New().String()[:8]
		cn.CreatedAt = base.AddDate(0, 0, offsetDays)
		cn.UpdatedAt = cn.CreatedAt
		return cn
	}
	in1, in2, in3 := mkSent(tenantID, invID, 0), mkSent(tenantID, invID, 1), mkSent(tenantID, invID, 2)
	before, after := mkSent(tenantID, invID, -1), mkSent(tenantID, invID, 5)
	foreign := mkSent(otherTenant, otherInv, 1)
	draft := baseCreditNote(tenantID, invID, oneCreditNoteLine())
	draft.CreatedAt, draft.UpdatedAt = base, base
	for _, cn := range []*models.CreditNote{in1, in2, in3, before, after, foreign, draft} {
		createCreditNote(t, pool, repo, cn)
	}

	from := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	page1, err := repo.ListForDATEVExport(ctx, tenantID, from, to, nil, nil, 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.Equal(t, in1.ID, page1[0].ID)
	assert.Equal(t, in2.ID, page1[1].ID)

	lastAt, lastID := page1[1].CreatedAt, page1[1].ID
	page2, err := repo.ListForDATEVExport(ctx, tenantID, from, to, &lastAt, &lastID, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, in3.ID, page2[0].ID, "the 12th at 12:00 is inside the inclusive end day")
	assert.NotEmpty(t, page2[0].LineItems, "lines are bulk-loaded")

	all, err := repo.ListForDATEVExport(ctx, tenantID, from, to, nil, nil, 100)
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for _, cn := range all {
		ids[cn.ID] = true
		assert.Equal(t, tenantID, cn.TenantID)
		assert.Equal(t, models.CreditNoteStatusSent, cn.Status)
	}
	assert.Equal(t, map[uuid.UUID]bool{in1.ID: true, in2.ID: true, in3.ID: true}, ids,
		"drafts, other days and other tenants never reach the export")

	// The service method is a plain pass-through to the same query.
	svc := &Service{repo: repo}
	viaSvc, err := svc.ListForDATEVExport(ctx, tenantID, from, to, nil, nil, 100)
	require.NoError(t, err)
	assert.Len(t, viaSvc, 3)
}

func TestPostgresRepository_SumGrossByInvoiceInTx_CountsDraftsAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, invA := seedTenantWithInvoice(t, pool, "CreditNote Sum Tenant A")
	tenantB, _ := seedTenantWithInvoice(t, pool, "CreditNote Sum Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)

	draft := baseCreditNote(tenantA, invA, oneCreditNoteLine())
	draft.GrossTotal = decimal.RequireFromString("59.50")
	sent := baseCreditNote(tenantA, invA, oneCreditNoteLine())
	sent.Status = models.CreditNoteStatusSent
	sent.CreditNoteNumber = "GS-SUM-" + uuid.New().String()[:8]
	sent.GrossTotal = decimal.RequireFromString("11.90")
	for _, cn := range []*models.CreditNote{draft, sent} {
		createCreditNote(t, pool, repo, cn)
	}

	sum := func(ctx context.Context, tenant, inv uuid.UUID) decimal.Decimal {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx) //nolint:errcheck
		got, err := repo.SumGrossByInvoiceInTx(ctx, tx, tenant, inv)
		require.NoError(t, err)
		return got
	}

	assert.Equal(t, "71.4", sum(ctxA, tenantA, invA).String(), "drafts count against the limit, like sent notes")
	assert.True(t, sum(ctxA, tenantA, uuid.New()).IsZero(), "an invoice without credit notes sums to zero")
	assert.True(t, sum(ctxA, tenantB, invA).IsZero(), "another tenant's id never sums A's notes")
}

func TestParseTaxMode(t *testing.T) {
	cases := map[string]tax.TaxMode{
		models.TaxModeStandard:         tax.ModeStandard,
		models.TaxModeReverseCharge:    tax.ModeReverseCharge,
		models.TaxModeKleinunternehmer: tax.ModeKleinunternehmer,
		"":                             tax.ModeStandard,
		"unknown":                      tax.ModeStandard,
	}
	for in, want := range cases {
		assert.Equal(t, want, parseTaxMode(in), "mode %q", in)
	}
}
