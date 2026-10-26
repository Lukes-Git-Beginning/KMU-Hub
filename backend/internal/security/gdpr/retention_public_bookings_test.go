package gdpr

// Coverage for PublicBookingRetentionHandler (feat-retention-public-bookings-24m,
// Lauf 17): an appointment date before the cutoff day is due, a younger one is
// not (whatever its status), Plan (the dry-run path) deletes nothing, Apply is
// tenant-scoped and idempotent.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPublicBookingRetentionHandler_Metadata(t *testing.T) {
	t.Parallel()

	h := NewPublicBookingRetentionHandler(nil)
	assert.Equal(t, "public_bookings", h.ResourceType())
	assert.Equal(t, "public_bookings", h.Table())
	assert.NotEmpty(t, h.DateColumn())
	assert.True(t, h.SupportsAction(models.RetentionActionDelete))
	assert.False(t, h.SupportsAction(models.RetentionActionAnonymize))

	_, err := h.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "x")
	require.Error(t, err, "must refuse an action it cannot perform instead of deleting")
}

func TestPublicBookingRetentionHandler_PlanAndApply(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID, otherTenant := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Booking Retention Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	testutil.EnsureTenant(t, pool, otherTenant, "Booking Retention Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })

	sys := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	// page seeds a calendar plus a booking page for one tenant.
	page := func(tenant uuid.UUID) uuid.UUID {
		owner := seedExportUser(t, pool, tenant, "booking-retention-owner")
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", owner) })
		cal := testutil.SeedRow(t, pool, "calendars", map[string]any{
			"tenant_id": tenant, "name": "Booking Retention", "owner_id": owner,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "calendars", cal) })
		p := testutil.SeedRow(t, pool, "booking_pages", map[string]any{
			"tenant_id": tenant, "calendar_id": cal,
			"slug":         "ret-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			"company_name": "Retention GmbH",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "booking_pages", p) })
		return p
	}
	pageID, otherPage := page(tenantID), page(otherTenant)

	booking := func(tenant, pageID uuid.UUID, date time.Time, status string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "public_bookings", map[string]any{
			"tenant_id": tenant, "booking_page_id": pageID, "service_id": "svc",
			"customer_name": "Erika Muster", "customer_email": "ret-" + uuid.NewString() + "@example.com",
			"date": date, "time_slot": "09:00", "status": status,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "public_bookings", id) })
		return id
	}
	ago := func(months int) time.Time { return now.AddDate(0, -months, 0) }

	old := booking(tenantID, pageID, ago(30), "completed")
	oldCancelled := booking(tenantID, pageID, ago(26), "cancelled")
	justOver := booking(tenantID, pageID, now.AddDate(-2, 0, -2), "confirmed")
	// Appointment day still inside the window: not due, whatever the status.
	fresh := booking(tenantID, pageID, ago(6), "completed")
	future := booking(tenantID, pageID, now.AddDate(0, 1, 0), "confirmed")
	otherOld := booking(otherTenant, otherPage, ago(30), "completed")

	h := NewPublicBookingRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := now.AddDate(-2, 0, 0)

	plan, err := h.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{old, oldCancelled, justOver}, plan.Due)
	assert.Empty(t, plan.Skipped)

	// RLS hides the other tenant above; under the system context only the
	// handler's own tenant filter keeps its old row out of the plan.
	sysPlan, err := h.Plan(sys, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, sysPlan.Due, otherOld)

	count := func(id uuid.UUID) int {
		var n int
		require.NoError(t, pool.QueryRow(sys, `SELECT COUNT(*) FROM public_bookings WHERE id = $1`, id).Scan(&n))
		return n
	}
	for _, id := range []uuid.UUID{old, oldCancelled, justOver, fresh, future, otherOld} {
		assert.Equal(t, 1, count(id), "Plan is the dry-run path and must not delete")
	}

	// An id from another tenant handed to this tenant's Apply is never deleted,
	// even where RLS would let the statement see it (system context).
	foreign, err := h.Apply(sys, tenantID, []uuid.UUID{otherOld}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, foreign)

	affected, err := h.Apply(ctx, tenantID, plan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 3, affected)
	for _, id := range []uuid.UUID{old, oldCancelled, justOver} {
		assert.Zero(t, count(id))
	}
	for _, id := range []uuid.UUID{fresh, future, otherOld} {
		assert.Equal(t, 1, count(id))
	}

	again, err := h.Apply(ctx, tenantID, plan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, again)
}
