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
