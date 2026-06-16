package calendar

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestBookingPages_GlobalActiveSlugUnique verifies migration 000149: the partial
// unique index idx_booking_pages_active_slug makes a duplicate ACTIVE slug
// impossible across tenants, so the unauth public lookup (GetBookingPageBySlug,
// which has no tenant filter) can never resolve ambiguously. An INACTIVE
// duplicate slug remains allowed (the index is WHERE active = true).
func TestBookingPages_GlobalActiveSlugUnique(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// A user + calendar per tenant (calendars.owner_id and booking_pages.calendar_id are FKs).
	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("slug-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)
	calA := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": testutil.TenantA, "name": "Cal A", "owner_id": userA,
	})
	defer testutil.CleanupRow(t, pool, "calendars", calA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantB,
		"email":         fmt.Sprintf("slug-test-%s@tenantb.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)
	calB := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": testutil.TenantB, "name": "Cal B", "owner_id": userB,
	})
	defer testutil.CleanupRow(t, pool, "calendars", calB)

	const slug = "demo-slug"
	sysCtx := testutil.WithSystemCtx(context.Background())

	// TenantA registers the active slug — fine.
	pageA := testutil.SeedRow(t, pool, "booking_pages", map[string]any{
		"tenant_id": testutil.TenantA, "calendar_id": calA, "slug": slug, "company_name": "A GmbH",
	})
	defer testutil.CleanupRow(t, pool, "booking_pages", pageA)

	// TenantB tries the SAME active slug — must be rejected by the global index.
	_, err := pool.Exec(sysCtx,
		`INSERT INTO booking_pages (tenant_id, calendar_id, slug, company_name, active)
		 VALUES ($1, $2, $3, $4, true)`,
		testutil.TenantB, calB, slug, "B GmbH",
	)
	require.Error(t, err, "a second ACTIVE page with the same slug must violate idx_booking_pages_active_slug")
	assert.Contains(t, err.Error(), "idx_booking_pages_active_slug")

	// TenantB CAN register the same slug INACTIVE (partial index is WHERE active = true).
	pageBInactive := testutil.SeedRow(t, pool, "booking_pages", map[string]any{
		"tenant_id": testutil.TenantB, "calendar_id": calB, "slug": slug,
		"company_name": "B GmbH", "active": false,
	})
	defer testutil.CleanupRow(t, pool, "booking_pages", pageBInactive)
}
