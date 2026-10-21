package gdpr

// DB tests for the export sweeper. They are deliberately NOT parallel: they
// take the sweep advisory lock (and the loop test holds it between ticks), so
// a parallel sibling would find it held and skip.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// recordingExportHandler notes which user it was asked to export and under
// which tenant, so a test can tell the sweeper re-ran a request in the right
// tenant context.
type recordingExportHandler struct {
	mu      sync.Mutex
	users   []uuid.UUID
	tenants []string
}

func (h *recordingExportHandler) ModuleName() string { return "recording" }

func (h *recordingExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	tenant, _ := middleware.GetTenantID(ctx)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.users = append(h.users, userID)
	h.tenants = append(h.tenants, tenant.String())
	return []byte(`{}`), nil
}

func (h *recordingExportHandler) calls(userID uuid.UUID) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, u := range h.users {
		if u == userID {
			n++
		}
	}
	return n
}

// seedStalledExport files an export request in the given status whose last
// generation started `age` ago.
func seedStalledExport(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, status string, age time.Duration) uuid.UUID {
	t.Helper()
	started := time.Now().UTC().Add(-age)
	return testutil.SeedRow(t, pool, "gdpr_export_requests", map[string]any{
		"tenant_id":             tenantID,
		"user_id":               userID,
		"status":                status,
		"requested_at":          started.Add(-time.Minute),
		"reviewed_at":           started.Add(-time.Minute),
		"processing_started_at": started,
	})
}

func exportStatus(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var status string
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT status FROM gdpr_export_requests WHERE id = $1`, id).Scan(&status))
	return status
}

func newSweepFixture(t *testing.T) (*pgxpool.Pool, *Service, *recordingExportHandler, uuid.UUID, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Export Sweeper Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	userID := seedGDPRUser(t, pool, tenantID)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	handler := &recordingExportHandler{}
	svc := NewService(NewPostgresRepository(pool))
	svc.RegisterExportHandler(handler)
	return pool, svc, handler, tenantID, userID
}

// A request that sat in "processing" past the lease is re-run to completion
// in its own tenant; one inside the lease is left alone; a replica holding the
// sweep lock makes the call skip. The lock is held on a second connection.
func TestRunExportSweep_RerunsStalledProcessingPastLease(t *testing.T) {
	pool, svc, handler, tenantID, userID := newSweepFixture(t)
	ctx := context.Background()

	stale := seedStalledExport(t, pool, tenantID, userID, models.ExportStatusProcessing, 2*ExportLease)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "gdpr_export_requests", stale) })
	fresh := seedStalledExport(t, pool, tenantID, userID, models.ExportStatusProcessing, time.Minute)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "gdpr_export_requests", fresh) })

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()
	var locked bool
	require.NoError(t, conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, exportSweepLockKey).Scan(&locked))
	require.True(t, locked, "test setup must win the lock first")

	ran, err := RunExportSweep(ctx, pool, svc, ExportLease)
	require.NoError(t, err)
	assert.False(t, ran, "a replica holding the lock must make the sweep skip")
	assert.Equal(t, 0, handler.calls(userID), "skipped sweep must not run anything")

	_, err = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, exportSweepLockKey)
	require.NoError(t, err)

	ran, err = RunExportSweep(ctx, pool, svc, ExportLease)
	require.NoError(t, err)
	assert.True(t, ran)

	assert.Equal(t, models.ExportStatusReady, exportStatus(t, pool, stale), "stalled request must be generated")
	assert.Equal(t, models.ExportStatusProcessing, exportStatus(t, pool, fresh), "request inside the lease must be left alone")
	assert.Equal(t, 1, handler.calls(userID), "only the stalled request is re-run")
	assert.Equal(t, []string{tenantID.String()}, handler.tenants, "re-run must carry the request's tenant")
}

// A request a failed run reverted to "approved" is retried once its lease is
// up, not before.
func TestRunExportSweep_RetriesApprovedPastLease(t *testing.T) {
	pool, svc, handler, tenantID, userID := newSweepFixture(t)

	young := seedStalledExport(t, pool, tenantID, userID, models.ExportStatusApproved, time.Minute)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "gdpr_export_requests", young) })

	ran, err := RunExportSweep(context.Background(), pool, svc, ExportLease)
	require.NoError(t, err)
	assert.True(t, ran)
	assert.Equal(t, models.ExportStatusApproved, exportStatus(t, pool, young))
	assert.Equal(t, 0, handler.calls(userID))

	old := seedStalledExport(t, pool, tenantID, userID, models.ExportStatusApproved, 2*ExportLease)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "gdpr_export_requests", old) })

	_, err = RunExportSweep(context.Background(), pool, svc, ExportLease)
	require.NoError(t, err)
	assert.Equal(t, models.ExportStatusReady, exportStatus(t, pool, old))
	assert.Equal(t, models.ExportStatusApproved, exportStatus(t, pool, young))
}

// The loop sweeps more than once: a request that goes stale AFTER the first
// pass is picked up by a later tick. A one-shot sweeper would fail this.
func TestExportSweeper_SweepsOnEveryTick(t *testing.T) {
	pool, svc, _, tenantID, userID := newSweepFixture(t)

	first := seedStalledExport(t, pool, tenantID, userID, models.ExportStatusProcessing, 2*ExportLease)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "gdpr_export_requests", first) })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ExportSweeper(ctx, pool, svc, ExportLease, 0, 25*time.Millisecond)
	}()
	defer func() {
		cancel()
		<-done
	}()

	require.Eventually(t, func() bool {
		return exportStatus(t, pool, first) == models.ExportStatusReady
	}, 5*time.Second, 20*time.Millisecond, "first pass must finish the stalled request")

	second := seedStalledExport(t, pool, tenantID, userID, models.ExportStatusProcessing, 2*ExportLease)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "gdpr_export_requests", second) })

	require.Eventually(t, func() bool {
		return exportStatus(t, pool, second) == models.ExportStatusReady
	}, 5*time.Second, 20*time.Millisecond, "a later tick must pick up a request that went stale after the first pass")
}

// ExecuteExportAsync stamps the lease clock when generation starts; the
// stamp is what the sweeper measures against.
func TestExecuteExportAsync_StampsProcessingStartedAt(t *testing.T) {
	pool, svc, _, tenantID, userID := newSweepFixture(t)

	id := seedStalledExport(t, pool, tenantID, userID, models.ExportStatusApproved, 3*time.Hour)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "gdpr_export_requests", id) })

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	require.NoError(t, svc.ExecuteExportAsync(ctx, id))

	var started time.Time
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT processing_started_at FROM gdpr_export_requests WHERE id = $1`, id).Scan(&started))
	assert.WithinDuration(t, time.Now(), started, time.Minute, "processing_started_at must be refreshed when generation starts")
}
