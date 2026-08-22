package dunning

// postgres_repository_db_test.go exercises PostgresRepository against the
// real schema instead of a mock. This package has never run a single query
// against Postgres before (no DB test at all, and outside the "Finance & HR
// Integration Tests" CI job): a dunning record decides who receives a
// collections notice for a real overdue invoice, so a wrong tenant filter or
// a broken total counter changes what a customer sees or receives. Untagged
// (no //go:build integration) so it runs in the normal local gate and the
// coverage job — see BACKLOG.yml Befund 2.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedDunningInvoice inserts a minimal finance_invoices row under the given
// tenant so dunning rows (fk_finance_dunning_invoice) have a parent to point
// at. Mirrors the pattern in payment/postgres_repository_db_test.go.
func seedDunningInvoice(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	invoiceID := uuid.New()
	id := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id":             invoiceID,
		"tenant_id":      tenantID,
		"created_by":     uuid.New(),
		"invoice_date":   time.Now(),
		"due_date":       time.Now().AddDate(0, 0, -14),
		"invoice_number": "DUN-" + invoiceID.String()[:8],
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
	return id
}

func newTestDunningRecord(tenantID, invoiceID uuid.UUID, level int, status string) *models.DunningRecord {
	return &models.DunningRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Level:     level,
		Status:    status,
		Fee:       decimal.RequireFromString("5.00"),
		Interest:  decimal.RequireFromString("0.00"),
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
	}
}

// TestPostgresRepository_Create_GetByID_RoundTripsExactDecimal proves the
// scan-as-string-then-decimal.NewFromString path in GetByID never routes fee
// and interest through float64 (ADR-0007).
func TestPostgresRepository_Create_GetByID_RoundTripsExactDecimal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning Core Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedDunningInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	rec := newTestDunningRecord(tenantID, invoiceID, 1, models.DunningStatusDraft)
	rec.Fee = decimal.RequireFromString("12.33")
	rec.Interest = decimal.RequireFromString("1.07")
	require.NoError(t, repo.Create(ctx, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	got, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("12.33").Equal(got.Fee), "fee must survive as exact decimal, got %s", got.Fee)
	assert.True(t, decimal.RequireFromString("1.07").Equal(got.Interest), "interest must survive as exact decimal, got %s", got.Interest)
	assert.Equal(t, 1, got.Level)
	assert.Equal(t, models.DunningStatusDraft, got.Status)
	assert.Nil(t, got.SentAt)
}

// TestPostgresRepository_GetByID_CrossTenant_ReturnsNotFound proves the
// explicit tenant_id parameter (backed by RLS) actually refuses a foreign
// tenant, not just that the SQL string contains a WHERE clause.
func TestPostgresRepository_GetByID_CrossTenant_ReturnsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dunning CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Dunning CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedDunningInvoice(t, pool, tenantA)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	rec := newTestDunningRecord(tenantA, invoiceA, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctxA, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	_, err := repo.GetByID(ctxB, tenantB, rec.ID)
	assert.ErrorIs(t, err, ErrDunningNotFound, "a foreign tenant must never see another tenant's dunning record")

	got, err := repo.GetByID(ctxA, tenantA, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, got.ID)
}

// TestPostgresRepository_List_FilterByStatusAndLevel_TotalIndependentOfLimit
// is the classic pagination bug class named in the unit scope: the total
// counter must reflect the filter, not the LIMIT, and a Limit smaller than
// the matching set must not shrink the reported total.
func TestPostgresRepository_List_FilterByStatusAndLevel_TotalIndependentOfLimit(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning List Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceA := seedDunningInvoice(t, pool, tenantID)
	invoiceB := seedDunningInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	records := []*models.DunningRecord{
		newTestDunningRecord(tenantID, invoiceA, 1, models.DunningStatusDraft),
		newTestDunningRecord(tenantID, invoiceA, 1, models.DunningStatusSent),
		newTestDunningRecord(tenantID, invoiceA, 2, models.DunningStatusSent),
		newTestDunningRecord(tenantID, invoiceA, 3, models.DunningStatusPaid),
		newTestDunningRecord(tenantID, invoiceB, 1, models.DunningStatusDraft),
	}
	for _, r := range records {
		require.NoError(t, repo.Create(ctx, r))
		defer testutil.CleanupRow(t, pool, "finance_dunning_records", r.ID)
	}

	// Filter by status alone: two "sent" records across two levels.
	list, total, err := repo.List(ctx, tenantID, ListFilter{Status: models.DunningStatusSent})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, list, 2)

	// Same filter with a Limit smaller than the match count: total must stay 2.
	limited, totalLimited, err := repo.List(ctx, tenantID, ListFilter{Status: models.DunningStatusSent, Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, 2, totalLimited, "total must reflect the filter, not the LIMIT")
	assert.Len(t, limited, 1)

	// Filter by level alone: three level-1 records (two on invoiceA, one on invoiceB).
	byLevel, totalByLevel, err := repo.List(ctx, tenantID, ListFilter{Level: 1})
	require.NoError(t, err)
	assert.Equal(t, 3, totalByLevel)
	assert.Len(t, byLevel, 3)

	// Filter by invoice ID: all four records on invoiceA.
	byInvoice, totalByInvoice, err := repo.List(ctx, tenantID, ListFilter{InvoiceID: &invoiceA})
	require.NoError(t, err)
	assert.Equal(t, 4, totalByInvoice)
	assert.Len(t, byInvoice, 4)
}

// TestPostgresRepository_List_TenantScoped proves a foreign tenant's dunning
// records never leak into another tenant's list, even unfiltered.
func TestPostgresRepository_List_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dunning TenantScope A")
	testutil.EnsureTenant(t, pool, tenantB, "Dunning TenantScope B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedDunningInvoice(t, pool, tenantA)
	invoiceB := seedDunningInvoice(t, pool, tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	recA := newTestDunningRecord(tenantA, invoiceA, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctxA, recA))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", recA.ID)

	recB := newTestDunningRecord(tenantB, invoiceB, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctxB, recB))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", recB.ID)

	listA, totalA, err := repo.List(ctxA, tenantA, ListFilter{})
	require.NoError(t, err)
	assert.Equal(t, 1, totalA)
	require.Len(t, listA, 1)
	assert.Equal(t, recA.ID, listA[0].ID)
}

// TestPostgresRepository_UpdateStatus_SentAtNil_LeavesColumnUnchanged is the
// mutations-probe target: passing sentAt = nil (an admin override that is not
// "the notice was just sent") must not clear an already-recorded sent_at —
// exactly the failure mode that would silently empty a dunning history.
func TestPostgresRepository_UpdateStatus_SentAtNil_LeavesColumnUnchanged(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning UpdateStatus SentAtNil")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedDunningInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	rec := newTestDunningRecord(tenantID, invoiceID, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctx, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	sentAt := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	require.NoError(t, repo.UpdateStatus(ctx, tenantID, rec.ID, models.DunningStatusSent, &sentAt))

	got, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	require.NotNil(t, got.SentAt)
	assert.True(t, sentAt.Equal(*got.SentAt))

	// Admin override back to draft (e.g. a correction) — passes sentAt = nil,
	// mirroring service_gobd.go's UpdateDunningStatus.
	require.NoError(t, repo.UpdateStatus(ctx, tenantID, rec.ID, models.DunningStatusDraft, nil))

	got, err = repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DunningStatusDraft, got.Status)
	require.NotNil(t, got.SentAt, "sent_at must survive a status update with sentAt = nil")
	assert.True(t, sentAt.Equal(*got.SentAt), "sent_at must remain the original timestamp, not be cleared")
}

// TestPostgresRepository_GetByInvoiceID_OrderedByLevel_CrossTenantEmpty proves
// GetByInvoiceID returns every record for the invoice ordered by level
// ascending, and that a foreign tenant with the correct invoice ID gets
// nothing back.
func TestPostgresRepository_GetByInvoiceID_OrderedByLevel_CrossTenantEmpty(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dunning GetByInvoiceID A")
	testutil.EnsureTenant(t, pool, tenantB, "Dunning GetByInvoiceID B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedDunningInvoice(t, pool, tenantA)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	level2 := newTestDunningRecord(tenantA, invoiceA, 2, models.DunningStatusSent)
	level1 := newTestDunningRecord(tenantA, invoiceA, 1, models.DunningStatusPaid)
	require.NoError(t, repo.Create(ctxA, level2))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", level2.ID)
	require.NoError(t, repo.Create(ctxA, level1))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", level1.ID)

	got, err := repo.GetByInvoiceID(ctxA, tenantA, invoiceA)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Level, "must be ordered by level ascending")
	assert.Equal(t, 2, got[1].Level)

	// Same invoice ID, but a foreign tenant must see nothing (RLS + explicit filter).
	gotB, err := repo.GetByInvoiceID(ctxB, tenantB, invoiceA)
	require.NoError(t, err)
	assert.Empty(t, gotB)
}

// TestPostgresRepository_GetByInvoiceIDs_EmptyUnknownAndNoRecords covers the
// three edge cases the batch path must not get wrong: an empty ID slice (must
// not become an unbounded query), an unknown invoice ID mixed with a known
// one (must not fail, must just be absent from the map), and an invoice with
// no dunning records (must not appear as a nil-valued map entry that would
// panic a caller ranging over it).
func TestPostgresRepository_GetByInvoiceIDs_EmptyUnknownAndNoRecords(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning GetByInvoiceIDs Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceWithRecords := seedDunningInvoice(t, pool, tenantID)
	invoiceWithoutRecords := seedDunningInvoice(t, pool, tenantID)
	unknownInvoiceID := uuid.New()

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	rec1 := newTestDunningRecord(tenantID, invoiceWithRecords, 1, models.DunningStatusDraft)
	rec2 := newTestDunningRecord(tenantID, invoiceWithRecords, 2, models.DunningStatusSent)
	require.NoError(t, repo.Create(ctx, rec1))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec1.ID)
	require.NoError(t, repo.Create(ctx, rec2))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec2.ID)

	// Empty slice must return an empty, non-nil map without querying.
	empty, err := repo.GetByInvoiceIDs(ctx, tenantID, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	byInvoice, err := repo.GetByInvoiceIDs(ctx, tenantID, []uuid.UUID{
		invoiceWithRecords, invoiceWithoutRecords, unknownInvoiceID,
	})
	require.NoError(t, err)
	require.Len(t, byInvoice[invoiceWithRecords], 2, "unbekannte und leere Rechnungen duerfen den bekannten Treffer nicht stoeren")
	assert.Equal(t, 1, byInvoice[invoiceWithRecords][0].Level, "grouped records stay ordered by level ascending")
	assert.Equal(t, 2, byInvoice[invoiceWithRecords][1].Level)

	_, hasEmpty := byInvoice[invoiceWithoutRecords]
	assert.False(t, hasEmpty, "an invoice with no dunning records must be absent from the map, not a nil-valued entry")
	_, hasUnknown := byInvoice[unknownInvoiceID]
	assert.False(t, hasUnknown, "an unknown invoice ID must not appear in the result")
}

// TestPostgresRepository_GetHighestLevelByInvoiceID_TieBreaksOnMostRecent
// covers the case named in the unit notes: two records at the same (highest)
// level are schema-legal (no unique constraint on invoice_id+level), and
// `ORDER BY level DESC` alone would make the pick nondeterministic. The fix
// adds `created_at DESC` as a tiebreaker; this test pins that behavior down.
func TestPostgresRepository_GetHighestLevelByInvoiceID_TieBreaksOnMostRecent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning GetHighestLevel Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedDunningInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	older := newTestDunningRecord(tenantID, invoiceID, 1, models.DunningStatusSent)
	require.NoError(t, repo.Create(ctx, older))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", older.ID)

	tieOlder := newTestDunningRecord(tenantID, invoiceID, 2, models.DunningStatusSent)
	tieOlder.CreatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, repo.Create(ctx, tieOlder))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", tieOlder.ID)

	tieNewer := newTestDunningRecord(tenantID, invoiceID, 2, models.DunningStatusDraft)
	tieNewer.CreatedAt = time.Now()
	require.NoError(t, repo.Create(ctx, tieNewer))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", tieNewer.ID)

	got, err := repo.GetHighestLevelByInvoiceID(ctx, tenantID, invoiceID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 2, got.Level)
	assert.Equal(t, tieNewer.ID, got.ID, "a tie on level must break on the most recently created record")
}

// TestPostgresRepository_GetHighestLevelByInvoiceID_NoRecords_ReturnsNilNil
// proves the "no dunning yet" path returns (nil, nil), not an error — callers
// use this to decide the invoice has no history at all.
func TestPostgresRepository_GetHighestLevelByInvoiceID_NoRecords_ReturnsNilNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dunning GetHighestLevel NoRecords A")
	testutil.EnsureTenant(t, pool, tenantB, "Dunning GetHighestLevel NoRecords B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedDunningInvoice(t, pool, tenantA)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	got, err := repo.GetHighestLevelByInvoiceID(ctxA, tenantA, invoiceA)
	require.NoError(t, err)
	assert.Nil(t, got, "an invoice without dunning records must yield (nil, nil), not an error")

	rec := newTestDunningRecord(tenantA, invoiceA, 3, models.DunningStatusSent)
	require.NoError(t, repo.Create(ctxA, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	// Same invoice ID, foreign tenant: must not see tenant A's record.
	gotB, err := repo.GetHighestLevelByInvoiceID(ctxB, tenantB, invoiceA)
	require.NoError(t, err)
	assert.Nil(t, gotB, "a foreign tenant must never see another tenant's dunning record")
}

// TestPostgresRepository_UpdateStatus_CrossTenant_ZeroRowsAffected proves an
// UPDATE issued with a foreign tenant ID touches zero rows, using the exact
// query the repository runs so the assertion is against the real predicate,
// not a proxy for it.
func TestPostgresRepository_UpdateStatus_CrossTenant_ZeroRowsAffected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dunning UpdateStatus TenantA")
	testutil.EnsureTenant(t, pool, tenantB, "Dunning UpdateStatus TenantB")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedDunningInvoice(t, pool, tenantA)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	rec := newTestDunningRecord(tenantA, invoiceA, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctxA, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		`UPDATE finance_dunning_records SET status = $1, sent_at = COALESCE($2, sent_at) WHERE tenant_id = $3 AND id = $4`,
		0, models.DunningStatusPaid, nil, tenantB, rec.ID,
	)

	got, err := repo.GetByID(ctxA, tenantA, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DunningStatusDraft, got.Status, "cross-tenant update must not have changed the row")
}

func newTestDunningConfig(tenantID uuid.UUID) *models.DunningConfig {
	return &models.DunningConfig{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		Level1DaysAfterDue:    14,
		Level2DaysAfterLevel1: 14,
		Level3DaysAfterLevel2: 14,
		Level1Fee:             decimal.RequireFromString("0.00"),
		Level2Fee:             decimal.RequireFromString("5.00"),
		Level3Fee:             decimal.RequireFromString("10.00"),
		UpdatedAt:             time.Now(),
	}
}

// TestPostgresConfigRepository_Get_NoConfig_ReturnsNilNil proves Get signals
// "nothing configured yet" as (nil, nil), not an error — Service.GetConfig
// (service.go:83) relies on exactly this to decide whether to create the
// default config.
func TestPostgresConfigRepository_Get_NoConfig_ReturnsNilNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning Config No Config Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresConfigRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	got, err := repo.Get(ctx, tenantID)
	require.NoError(t, err)
	assert.Nil(t, got, "a tenant without a config row must yield (nil, nil), not an error")
}

// TestPostgresConfigRepository_Upsert_Get_RoundTripsExactDecimal proves the
// insert branch of the upsert and that fees survive the string+decimal scan
// path (ADR-0007) exactly, not through float64.
func TestPostgresConfigRepository_Upsert_Get_RoundTripsExactDecimal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning Config Roundtrip Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresConfigRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	config := newTestDunningConfig(tenantID)
	config.Level1Fee = decimal.RequireFromString("12.34")
	config.Level2Fee = decimal.RequireFromString("0.07")
	config.Level3Fee = decimal.RequireFromString("199.99")
	require.NoError(t, repo.Upsert(ctx, config))
	defer testutil.CleanupRow(t, pool, "finance_dunning_config", config.ID)

	got, err := repo.Get(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, decimal.RequireFromString("12.34").Equal(got.Level1Fee), "got %s", got.Level1Fee)
	assert.True(t, decimal.RequireFromString("0.07").Equal(got.Level2Fee), "got %s", got.Level2Fee)
	assert.True(t, decimal.RequireFromString("199.99").Equal(got.Level3Fee), "got %s", got.Level3Fee)
	assert.Equal(t, config.Level1DaysAfterDue, got.Level1DaysAfterDue)
	assert.Equal(t, config.Level2DaysAfterLevel1, got.Level2DaysAfterLevel1)
	assert.Equal(t, config.Level3DaysAfterLevel2, got.Level3DaysAfterLevel2)
}

// TestPostgresConfigRepository_Upsert_UpdateBranch_OverwritesInPlace proves
// the ON CONFLICT (tenant_id) DO UPDATE branch: a second Upsert for the same
// tenant changes the existing row's values rather than erroring or creating
// a second row (the table carries a UNIQUE constraint on tenant_id, so a
// silent duplicate is impossible — but a broken ON CONFLICT target would
// surface as a constraint violation here, not silently).
func TestPostgresConfigRepository_Upsert_UpdateBranch_OverwritesInPlace(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning Config Update Branch Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresConfigRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	config := newTestDunningConfig(tenantID)
	require.NoError(t, repo.Upsert(ctx, config))
	defer testutil.CleanupRow(t, pool, "finance_dunning_config", config.ID)

	config.Level1DaysAfterDue = 21
	config.Level1Fee = decimal.RequireFromString("2.50")
	config.UpdatedAt = time.Now()
	require.NoError(t, repo.Upsert(ctx, config))

	got, err := repo.Get(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 21, got.Level1DaysAfterDue)
	assert.True(t, decimal.RequireFromString("2.50").Equal(got.Level1Fee), "got %s", got.Level1Fee)

	var rowCount int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT count(*) FROM finance_dunning_config WHERE tenant_id = $1`, tenantID,
	).Scan(&rowCount))
	assert.Equal(t, 1, rowCount, "a second Upsert must update the existing row, not insert a duplicate")
}

// TestPostgresConfigRepository_Upsert_TwoTenants_Independent proves the
// classic ON CONFLICT tenant-leak shape: two tenants upserting their own
// config must never see or overwrite each other's values.
func TestPostgresConfigRepository_Upsert_TwoTenants_Independent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dunning Config Two Tenants A")
	testutil.EnsureTenant(t, pool, tenantB, "Dunning Config Two Tenants B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresConfigRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	configA := newTestDunningConfig(tenantA)
	configA.Level1Fee = decimal.RequireFromString("1.00")
	configB := newTestDunningConfig(tenantB)
	configB.Level1Fee = decimal.RequireFromString("9.00")

	require.NoError(t, repo.Upsert(ctxA, configA))
	defer testutil.CleanupRow(t, pool, "finance_dunning_config", configA.ID)
	require.NoError(t, repo.Upsert(ctxB, configB))
	defer testutil.CleanupRow(t, pool, "finance_dunning_config", configB.ID)

	gotA, err := repo.Get(ctxA, tenantA)
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.True(t, decimal.RequireFromString("1.00").Equal(gotA.Level1Fee), "tenant A must keep its own fee, not tenant B's — got %s", gotA.Level1Fee)

	gotB, err := repo.Get(ctxB, tenantB)
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.True(t, decimal.RequireFromString("9.00").Equal(gotB.Level1Fee), "tenant B must keep its own fee, not tenant A's — got %s", gotB.Level1Fee)

	// Cross-tenant read: RLS must hide tenant A's config from tenant B's
	// session even though the WHERE clause also filters by tenant_id.
	crossRead, err := repo.Get(ctxB, tenantA)
	require.NoError(t, err)
	assert.Nil(t, crossRead, "a foreign tenant's config row must never be visible")
}
