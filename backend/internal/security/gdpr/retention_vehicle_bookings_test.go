package gdpr

// Coverage for VehicleBookingRetentionHandler. Cases that matter: only
// bookings whose ends_at is past the cutoff are Due, the plan is tenant
// scoped, Apply deletes and the second Plan run finds nothing left
// (idempotency).

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

func TestVehicleBookingRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewVehicleBookingRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.False(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.Equal(t, "vehicle_bookings", handler.ResourceType())
	assert.Equal(t, "vehicle_bookings", handler.Table())
}

func TestVehicleBookingRetentionHandler_PlanOnlyMatchesPastEndsAtAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vehicle Booking Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "vb-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	vehicleID := seedRetentionVehicle(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "vehicles", vehicleID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Vehicle Booking Retention Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherUserID := seedExportUser(t, pool, otherTenantID, "vb-retention-plan-other-user")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)
	otherVehicleID := seedRetentionVehicle(t, pool, otherTenantID)
	defer testutil.CleanupRow(t, pool, "vehicles", otherVehicleID)

	now := time.Now().UTC()
	finishedOld := seedVehicleBooking(t, pool, tenantID, vehicleID, userID, now.AddDate(0, 0, -400), now.AddDate(0, 0, -399))
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", finishedOld)

	finishedFresh := seedVehicleBooking(t, pool, tenantID, vehicleID, userID, now.AddDate(0, 0, -2), now.AddDate(0, 0, -1))
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", finishedFresh)

	// Still running booking, old start but no end past the cutoff — must
	// never be Due, ends_at is the gate, not starts_at.
	ongoing := seedVehicleBooking(t, pool, tenantID, vehicleID, userID, now.AddDate(0, 0, -400), now.AddDate(1, 0, 0))
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", ongoing)

	// Another tenant's finished-and-old booking must never leak into this
	// tenant's plan.
	otherFinishedOld := seedVehicleBooking(t, pool, otherTenantID, otherVehicleID, otherUserID, now.AddDate(0, 0, -400), now.AddDate(0, 0, -399))
	defer testutil.CleanupRow(t, pool, "vehicle_bookings", otherFinishedOld)

	handler := NewVehicleBookingRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := now.AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{finishedOld}, plan.Due)
	assert.Empty(t, plan.Skipped)
}

func TestVehicleBookingRetentionHandler_ApplyDeletesAndIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vehicle Booking Retention Apply Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "vb-retention-apply-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	vehicleID := seedRetentionVehicle(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "vehicles", vehicleID)

	now := time.Now().UTC()
	bookingID := seedVehicleBooking(t, pool, tenantID, vehicleID, userID, now.AddDate(0, 0, -400), now.AddDate(0, 0, -399))

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewVehicleBookingRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{bookingID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM vehicle_bookings WHERE id = $1`, bookingID).Scan(&count))
	assert.Zero(t, count, "deleted booking must be gone")

	cutoff := now.AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, bookingID, "a second run must find nothing left to delete")
}

func TestVehicleBookingRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewVehicleBookingRetentionHandler(nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "")
	require.Error(t, err)
}

// seedRetentionVehicle seeds a minimal vehicle for a booking to reference.
func seedRetentionVehicle(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "vehicles", map[string]any{
		"tenant_id":     tenantID,
		"license_plate": "RT-" + uuid.New().String()[:6],
		"make":          "VW",
		"model":         "Caddy",
		"year":          2023,
	})
}

// seedVehicleBooking seeds a minimal vehicle_bookings row.
func seedVehicleBooking(t *testing.T, pool *pgxpool.Pool, tenantID, vehicleID, userID uuid.UUID, startsAt, endsAt time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "vehicle_bookings", map[string]any{
		"tenant_id":  tenantID,
		"vehicle_id": vehicleID,
		"user_id":    userID,
		"starts_at":  startsAt,
		"ends_at":    endsAt,
	})
}
