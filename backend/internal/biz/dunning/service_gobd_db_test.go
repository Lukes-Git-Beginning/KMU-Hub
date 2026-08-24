package dunning

// service_gobd_db_test.go exercises UpdateDunningStatus and SendDunningNotice
// (service_gobd.go) through the real PostgresRepository instead of
// MockRepository. The mock-based tests in service_test.go cover the branching
// logic; these prove the same behavior survives a round trip through actual
// SQL — the COALESCE(sent_at) fix from postgres_repository_db_test.go and the
// tenant-scoped UPDATE/SELECT predicates only prove something when driven
// through the Service methods a caller (gRPC handler) actually calls.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestService_UpdateDunningStatus_RealSQL_DraftToSent_SetsSentAt proves the
// service-level draft->sent transition persists sent_at through the real
// UPDATE, not just on the in-memory returned record.
func TestService_UpdateDunningStatus_RealSQL_DraftToSent_SetsSentAt(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning Service UpdateStatus Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedDunningInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc := NewService(repo, &MockConfigRepository{}, NewMockInvoiceReader())

	rec := newTestDunningRecord(tenantID, invoiceID, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctx, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	result, err := svc.UpdateDunningStatus(ctx, tenantID, rec.ID, models.DunningStatusSent)
	require.NoError(t, err)
	require.NotNil(t, result.SentAt, "returned record must carry sent_at")

	got, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DunningStatusSent, got.Status, "status must be persisted, not just on the in-memory record")
	require.NotNil(t, got.SentAt, "sent_at must be persisted by the real UPDATE")
	assert.WithinDuration(t, *result.SentAt, *got.SentAt, time.Second)
}

// TestService_UpdateDunningStatus_RealSQL_AdminOverride_SentToDraft_PreservesSentAt
// pins down the admin-override transition the package comment calls out as
// "intentionally permissive": a sent record moved back to draft (e.g. a
// correction) is an allowed, deliberate transition, not a bug — and it must
// not wipe the sent_at that already recorded when the notice went out
// (the COALESCE(sent_at) fix this package's Block-B unit made in
// postgres_repository.go). Losing sent_at here would erase the audit trail
// GoBD requires for "when was this dunning actually sent".
func TestService_UpdateDunningStatus_RealSQL_AdminOverride_SentToDraft_PreservesSentAt(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning Service AdminOverride Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedDunningInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc := NewService(repo, &MockConfigRepository{}, NewMockInvoiceReader())

	rec := newTestDunningRecord(tenantID, invoiceID, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctx, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	_, err := svc.UpdateDunningStatus(ctx, tenantID, rec.ID, models.DunningStatusSent)
	require.NoError(t, err)

	// Read the persisted value back (TIMESTAMPTZ truncates to microseconds;
	// comparing against the in-memory time.Now() from the service would fail
	// on nanosecond precision alone, not on an actual bug).
	afterSend, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	require.NotNil(t, afterSend.SentAt)
	originalSentAt := *afterSend.SentAt

	// Admin override: correction back to draft. Allowed by design
	// (service_gobd.go doc comment), must not touch sent_at.
	back, err := svc.UpdateDunningStatus(ctx, tenantID, rec.ID, models.DunningStatusDraft)
	require.NoError(t, err, "sent -> draft must be an allowed admin override, not rejected")
	assert.Equal(t, models.DunningStatusDraft, back.Status)

	got, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DunningStatusDraft, got.Status)
	require.NotNil(t, got.SentAt, "sent_at must survive the admin override back to draft")
	assert.True(t, originalSentAt.Equal(*got.SentAt), "sent_at must remain the original timestamp, not be cleared or bumped")
}

// TestService_UpdateDunningStatus_RealSQL_CrossTenant_NotFound proves the
// service surfaces ErrDunningNotFound (not a silent no-op or another tenant's
// record) when called with a foreign tenant ID against the real repository.
func TestService_UpdateDunningStatus_RealSQL_CrossTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Dunning Service CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Dunning Service CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedDunningInvoice(t, pool, tenantA)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	svcB := NewService(repo, &MockConfigRepository{}, NewMockInvoiceReader())

	rec := newTestDunningRecord(tenantA, invoiceA, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctxA, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	_, err := svcB.UpdateDunningStatus(ctxB, tenantB, rec.ID, models.DunningStatusSent)
	assert.ErrorIs(t, err, ErrDunningNotFound, "a foreign tenant must not be able to update another tenant's dunning record")
}

// TestService_SendDunningNotice_RealSQL_DoubleCall_SecondCallRejected is the
// idempotency question the unit notes call out explicitly: a second
// SendDunningNotice on an already-sent record must not send/record a second
// notice or move sent_at forward — driven through the real repository so the
// draft-check-then-update is proven against actual row state, not a mock's
// in-memory map.
func TestService_SendDunningNotice_RealSQL_DoubleCall_SecondCallRejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dunning Service SendTwice Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedDunningInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	svc := NewService(repo, &MockConfigRepository{}, NewMockInvoiceReader())

	rec := newTestDunningRecord(tenantID, invoiceID, 1, models.DunningStatusDraft)
	require.NoError(t, repo.Create(ctx, rec))
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", rec.ID)

	userID := uuid.New()
	_, err := svc.SendDunningNotice(ctx, tenantID, rec.ID, userID)
	require.NoError(t, err)

	// Read the persisted value back (TIMESTAMPTZ truncates to microseconds;
	// comparing against the in-memory time.Now() from the service would fail
	// on nanosecond precision alone, not on an actual bug).
	afterFirst, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	require.NotNil(t, afterFirst.SentAt)
	firstSentAt := *afterFirst.SentAt

	_, err = svc.SendDunningNotice(ctx, tenantID, rec.ID, userID)
	assert.ErrorIs(t, err, ErrDunningNotDraft, "a second send on an already-sent record must be rejected, not create a second notice")

	got, err := repo.GetByID(ctx, tenantID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DunningStatusSent, got.Status)
	require.NotNil(t, got.SentAt)
	assert.True(t, firstSentAt.Equal(*got.SentAt), "the rejected second call must not move sent_at forward")
}
