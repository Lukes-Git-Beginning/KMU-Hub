package invoice

// postgres_repository_status_db_test.go exercises UpdateStatus and
// UpdateStatusInTx (postgres_repository.go:363/375) against real Postgres.
// Both are narrow write paths that bypass the full Update: any allowed-vs-
// forbidden status transition check (isInvoiceLocked, the switch statements
// in Service.MarkPaid/Cancel) lives entirely one layer up. This file proves
// what the repository itself does and does NOT enforce, so that layer-up
// dependency is documented rather than assumed.
//
// FINDING: UpdateStatus carries no lock (GoBD §146) awareness at all -- a
// locked invoice's status flips exactly like an unlocked one. The only
// callers that go through Service.MarkPaid/Cancel are protected by their own
// isInvoiceLocked check before reaching the repository. internal/biz/payment
// calls UpdateStatusInTx directly (cmd/biz/main.go wires the raw repository
// as its InvoiceStatusUpdater, bypassing invoice.Service entirely) and, until
// this same iteration, had no equivalent check -- see
// TestRecord_NoAutoTransitionWhenInvoiceLocked and
// TestDelete_NoRevertWhenInvoiceLocked in internal/biz/payment/service_test.go
// for the fix and its proof.
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

func TestPostgresRepository_UpdateStatus_PersistsAndRoundTrips(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "UpdateStatus Roundtrip Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	inv.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ctx, inv)

	require.NoError(t, repo.UpdateStatus(ctx, tenantID, inv.ID, models.InvoiceStatusPaid))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, got.Status)
	assert.True(t, got.UpdatedAt.After(inv.UpdatedAt), "UpdateStatus must bump updated_at")
}

// TestPostgresRepository_UpdateStatus_CrossTenantIsNoop proves an UpdateStatus
// call with a foreign tenant id affects zero rows and returns no error --
// same silent-noop shape as SetLock (postgres_repository_lock_db_test.go), a
// caller relying on an error to detect "wrong tenant" would be wrong.
func TestPostgresRepository_UpdateStatus_CrossTenantIsNoop(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	ownerTenant := uuid.New()
	testutil.EnsureTenant(t, pool, ownerTenant, "UpdateStatus Owner Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", ownerTenant)

	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "UpdateStatus Foreign Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", foreignTenant)

	repo := NewPostgresRepository(pool)
	ownerCtx := testutil.WithTenantCtx(context.Background(), ownerTenant)

	now := time.Now().UTC()
	inv := newListTestInvoice(ownerTenant, now, now.AddDate(0, 0, 14))
	inv.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ownerCtx, inv)

	// Attempt the update scoped to the wrong tenant.
	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignTenant)
	require.NoError(t, repo.UpdateStatus(foreignCtx, foreignTenant, inv.ID, models.InvoiceStatusPaid))

	got, err := repo.GetByID(ownerCtx, ownerTenant, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusSent, got.Status, "a cross-tenant UpdateStatus must affect zero rows")
}

// TestPostgresRepository_UpdateStatus_IgnoresLock is the FINDING documented
// in the file header made concrete: UpdateStatus flips a locked invoice's
// status exactly like an unlocked one. The repository has no lock column in
// its WHERE clause -- the GoBD §146 barrier is enforced entirely by callers
// (Service.MarkPaid/Cancel, payment.Service.transitionToPaidInTx/
// revertPaidStatusInTx), not by this method.
func TestPostgresRepository_UpdateStatus_IgnoresLock(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "UpdateStatus Lock Bypass Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	inv.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ctx, inv)

	require.NoError(t, repo.SetLock(ctx, tenantID, inv.ID, now, uuid.New()))

	require.NoError(t, repo.UpdateStatus(ctx, tenantID, inv.ID, models.InvoiceStatusPaid))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, got.Status, "the repository itself does not check locked_at -- this is why every caller must")
	require.NotNil(t, got.LockedAt, "the lock itself must remain untouched by the status write")
}

// TestPostgresRepository_UpdateStatusInTx_RollbackLeavesNothing proves an
// UpdateStatusInTx call inside a transaction that is rolled back afterward
// (mirroring payment.Service.transitionToPaidInTx/revertPaidStatusInTx, which
// couple this write with a payment insert/delete in one caller-owned tx)
// leaves the original status untouched.
func TestPostgresRepository_UpdateStatusInTx_RollbackLeavesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "UpdateStatusInTx Rollback Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	inv.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ctx, inv)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck // safety net if an assertion below fails before the explicit rollback
	require.NoError(t, repo.UpdateStatusInTx(ctx, tx, tenantID, inv.ID, models.InvoiceStatusPaid), "the write itself is valid inside the tx")
	require.NoError(t, tx.Rollback(ctx))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusSent, got.Status, "a rolled-back UpdateStatusInTx must leave the original status in place")
}
