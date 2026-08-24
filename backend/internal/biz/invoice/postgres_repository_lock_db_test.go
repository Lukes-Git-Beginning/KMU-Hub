package invoice

// postgres_repository_lock_db_test.go exercises the GoBD administrative lock
// (SetLock, postgres_repository.go:386) and its interaction with GetOverdue
// (:399) against real Postgres. Untagged (testutil.SkipIfNoDB) so it runs in
// the local gate and counts toward this package's coverage number, unlike the
// equivalent tagged TestInvoiceLockColumns / TestGetOverdueReturnsOnlySentPastDue
// in integration_test.go (build tag "integration", separate CI job, no
// coverage collection).
//
// FINDING documented here, not fixed at the repository level: GetOverdue does
// NOT filter by locked_at -- a locked, sent, past-due invoice is still
// returned. That is intentional (dunning.Service.invReader.GetOverdue reuses
// this exact query to build dunning candidates, and a locked invoice is still
// legitimately due). The GoBD §146 write barrier instead sits one layer up, in
// Service.DetectOverdue, which now skips locked invoices before calling
// UpdateStatus -- see TestService_DetectOverdue_SkipsLockedInvoice in
// service_test.go for that half of the proof.
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

// TestPostgresRepository_SetLock_PersistsAndRoundTrips proves locked_at and
// locked_by survive a write/read round trip against real Postgres -- the
// dedicated lock columns introduced by Migration 000132 (ADR-0007), replacing
// the earlier snapshot_data lock hack.
func TestPostgresRepository_SetLock_PersistsAndRoundTrips(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Lock Roundtrip Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	inv := newListTestInvoice(tenantID, time.Now().UTC(), time.Now().UTC().AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, inv)

	lockedAt := time.Now().UTC().Truncate(time.Second)
	lockedBy := uuid.New()
	require.NoError(t, repo.SetLock(ctx, tenantID, inv.ID, lockedAt, lockedBy))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)

	require.NotNil(t, got.LockedAt)
	assert.True(t, got.LockedAt.Equal(lockedAt), "LockedAt: got %v, want %v", got.LockedAt, lockedAt)
	require.NotNil(t, got.LockedBy)
	assert.Equal(t, lockedBy, *got.LockedBy)
}

// TestPostgresRepository_SetLock_UnknownInvoiceIsNoop proves SetLock against a
// non-existent (or cross-tenant) id affects zero rows and returns no error --
// the UPDATE's WHERE clause is id AND tenant_id, no existence check beforehand.
// A caller relying on an error to detect "invoice not found" would be wrong;
// documented so LockInvoice's own repo.GetByID-first call (service_gobd.go:81)
// is understood as load-bearing, not redundant.
func TestPostgresRepository_SetLock_UnknownInvoiceIsNoop(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Lock Unknown Invoice Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	err := repo.SetLock(ctx, tenantID, uuid.New(), time.Now().UTC(), uuid.New())
	assert.NoError(t, err)
}

// TestPostgresRepository_GetOverdue_IncludesLockedInvoice is the read-side half
// of the FINDING above: a locked, sent, past-due invoice is still returned by
// GetOverdue. The write barrier is enforced by Service.DetectOverdue, not by
// this query.
func TestPostgresRepository_GetOverdue_IncludesLockedInvoice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Lock Overdue Read Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now.AddDate(0, 0, -30), now.Add(-48*time.Hour))
	inv.Status = models.InvoiceStatusSent
	createListTestInvoice(t, repo, pool, ctx, inv)

	require.NoError(t, repo.SetLock(ctx, tenantID, inv.ID, now, uuid.New()))

	results, err := repo.GetOverdue(ctx, tenantID)
	require.NoError(t, err)

	var found bool
	for _, r := range results {
		if r.ID == inv.ID {
			found = true
			require.NotNil(t, r.LockedAt, "GetOverdue must still surface the lock state to its caller")
		}
	}
	assert.True(t, found, "GetOverdue must include a locked, sent, past-due invoice -- dunning candidate selection relies on this")
}
