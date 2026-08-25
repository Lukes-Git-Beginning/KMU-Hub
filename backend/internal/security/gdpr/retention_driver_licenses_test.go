package gdpr

// Coverage for DriverLicenseRetentionHandler. The case that matters most:
// the newest checked_at row per driver_id is protected from deletion no
// matter how old it is, even when it is the driver's only row — otherwise
// the proof of a license control could vanish entirely. Also covers plan
// tenant scoping, Apply deleting only the non-latest rows, and idempotency.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestDriverLicenseRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewDriverLicenseRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.False(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.Equal(t, "driver_licenses", handler.ResourceType())
	assert.Equal(t, "driver_licenses", handler.Table())
}

func TestDriverLicenseRetentionHandler_PlanKeepsLatestPerDriverEvenWhenOnlyRowIsAncient(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Driver License Retention Sole Row Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	driverID := seedExportUser(t, pool, tenantID, "dl-retention-sole-driver")
	defer testutil.CleanupRow(t, pool, "users", driverID)

	ancient := time.Now().UTC().AddDate(0, 0, -3000)
	soleCheck := seedDriverLicenseCheck(t, pool, tenantID, driverID, ancient)
	defer testutil.CleanupRow(t, pool, "driver_licenses", soleCheck)

	handler := NewDriverLicenseRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Empty(t, plan.Due, "a driver's only check must never be Due, no matter its age")
	require.Len(t, plan.Skipped, 1)
	assert.Equal(t, soleCheck, plan.Skipped[0].RecordID)
	assert.Contains(t, plan.Skipped[0].Reason, "aktuelle Kontrollzeile")
}

func TestDriverLicenseRetentionHandler_PlanDeletesOlderRowsButKeepsNewestAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Driver License Retention History Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	driverID := seedExportUser(t, pool, tenantID, "dl-retention-history-driver")
	defer testutil.CleanupRow(t, pool, "users", driverID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Driver License Retention Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherDriverID := seedExportUser(t, pool, otherTenantID, "dl-retention-other-driver")
	defer testutil.CleanupRow(t, pool, "users", otherDriverID)

	now := time.Now().UTC()
	oldCheck := seedDriverLicenseCheck(t, pool, tenantID, driverID, now.AddDate(0, 0, -400))
	defer testutil.CleanupRow(t, pool, "driver_licenses", oldCheck)
	freshCheck := seedDriverLicenseCheck(t, pool, tenantID, driverID, now.AddDate(0, 0, -1))
	defer testutil.CleanupRow(t, pool, "driver_licenses", freshCheck)

	// Another tenant's old-and-only check must never leak into this tenant's
	// plan, either as Due or as Skipped.
	otherOldCheck := seedDriverLicenseCheck(t, pool, otherTenantID, otherDriverID, now.AddDate(0, 0, -400))
	defer testutil.CleanupRow(t, pool, "driver_licenses", otherOldCheck)

	handler := NewDriverLicenseRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := now.AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{oldCheck}, plan.Due, "only the superseded row is Due")
	assert.Empty(t, plan.Skipped, "the newest row is not past the cutoff, so it never even reaches Skipped here")
}

func TestDriverLicenseRetentionHandler_ApplyDeletesAndIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Driver License Retention Apply Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	driverID := seedExportUser(t, pool, tenantID, "dl-retention-apply-driver")
	defer testutil.CleanupRow(t, pool, "users", driverID)

	now := time.Now().UTC()
	oldCheck := seedDriverLicenseCheck(t, pool, tenantID, driverID, now.AddDate(0, 0, -400))
	freshCheck := seedDriverLicenseCheck(t, pool, tenantID, driverID, now.AddDate(0, 0, -1))
	defer testutil.CleanupRow(t, pool, "driver_licenses", freshCheck)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewDriverLicenseRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{oldCheck}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM driver_licenses WHERE id = $1`, oldCheck).Scan(&count))
	assert.Zero(t, count, "deleted check must be gone")

	cutoff := now.AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, oldCheck, "a second run must find nothing left to delete")
}

func TestDriverLicenseRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewDriverLicenseRetentionHandler(nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "")
	require.Error(t, err)
}

// seedDriverLicenseCheck seeds a minimal driver_licenses row (one control
// visit).
func seedDriverLicenseCheck(t *testing.T, pool *pgxpool.Pool, tenantID, driverID uuid.UUID, checkedAt time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "driver_licenses", map[string]any{
		"tenant_id":           tenantID,
		"driver_id":           driverID,
		"license_classes":     []string{"B"},
		"checked_at":          checkedAt,
		"next_check_due_date": checkedAt.AddDate(1, 0, 0),
	})
}
