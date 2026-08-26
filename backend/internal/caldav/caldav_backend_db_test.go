package caldav

// DB-backed coverage for checkCalendarWritePermission and listEventExceptions
// against the real calendars/calendar_members/event_exceptions schema and
// its RLS tenant_isolation policies (migration 000124).
//
// CONFIRMED PRODUCTION BUG (verified against the local Postgres, see
// fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx at the end
// of BACKLOG.yml): neither function wraps its query in sysctx.With(ctx) or
// receives a tenant via context the way resolveTenantID does. The CalDAV
// Basic-Auth request path (basicAuthMiddleware -> CtxWithUser, see
// route_caldav.go) never carries a tenant -- there is no JWT to derive one
// from. Under RLS (kmuhub_app, FORCE ROW LEVEL SECURITY), a query with no
// app.tenant_id GUC and app.role != 'system' sees zero rows for every tenant
// (internal/database/postgres.go:39-43, "the safe default"). The tests
// below prove this with the exact context CalDAV requests actually carry:
//   - TestCheckCalendarWritePermission_RealCalDAVContext_OwnerBlocked proves
//     PutCalendarObject/DeleteCalendarObject 404 for the calendar's own
//     owner -- every CalDAV write is broken, matching the "my calendar
//     won't sync" symptom the unit's scope description called out.
//   - TestListEventExceptions_RealCalDAVContext_SilentlyEmpty proves
//     recurring-event overrides/cancellations silently vanish from CalDAV
//     reads instead of erroring, which is worse: no client ever sees a
//     failure to retry.
//
// The *_WithTenantCtx tests use testutil.WithTenantCtx to prove the
// permission-branching logic itself (owner/edit/admin/view/no-membership/
// not-found) is otherwise correct -- i.e. once the missing sysctx wrap is
// fixed, the intended behaviour is this. They double as regression coverage
// for that fix.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

type calendarFixture struct {
	tenantID   uuid.UUID
	ownerID    uuid.UUID
	calendarID uuid.UUID
}

// seedCalendarFixture creates a tenant, an owning user, and a calendar owned
// by that user, all under system context (bypasses RLS for setup).
func seedCalendarFixture(t *testing.T, pool *pgxpool.Pool) calendarFixture {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CalDAV Permission Test Tenant")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("caldav-perm-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", ownerID) })

	calendarID := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": tenantID,
		"name":      "Fixture Calendar",
		"owner_id":  ownerID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "calendars", calendarID) })

	return calendarFixture{tenantID: tenantID, ownerID: ownerID, calendarID: calendarID}
}

// ============================================================================
// checkCalendarWritePermission -- WithTenantCtx (intended behaviour)
// ============================================================================

func TestCheckCalendarWritePermission_OwnerAllowed_WithTenantCtx(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenantID)

	err := backend.checkCalendarWritePermission(ctx, fx.ownerID, fx.calendarID)

	assert.NoError(t, err)
}

func TestCheckCalendarWritePermission_MemberWithEditAllowed_WithTenantCtx(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	memberID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     fx.tenantID,
		"email":         fmt.Sprintf("caldav-editmember-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", memberID) })
	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx,
		`INSERT INTO calendar_members (calendar_id, user_id, permission, tenant_id) VALUES ($1, $2, 'edit', $3)`,
		fx.calendarID, memberID, fx.tenantID)
	require.NoError(t, err)

	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenantID)

	err = backend.checkCalendarWritePermission(ctx, memberID, fx.calendarID)

	assert.NoError(t, err)
}

func TestCheckCalendarWritePermission_MemberWithAdminAllowed_WithTenantCtx(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	memberID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     fx.tenantID,
		"email":         fmt.Sprintf("caldav-adminmember-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", memberID) })
	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx,
		`INSERT INTO calendar_members (calendar_id, user_id, permission, tenant_id) VALUES ($1, $2, 'admin', $3)`,
		fx.calendarID, memberID, fx.tenantID)
	require.NoError(t, err)

	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenantID)

	err = backend.checkCalendarWritePermission(ctx, memberID, fx.calendarID)

	assert.NoError(t, err)
}

func TestCheckCalendarWritePermission_MemberWithViewForbidden_WithTenantCtx(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	memberID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     fx.tenantID,
		"email":         fmt.Sprintf("caldav-viewmember-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", memberID) })
	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx,
		`INSERT INTO calendar_members (calendar_id, user_id, permission, tenant_id) VALUES ($1, $2, 'view', $3)`,
		fx.calendarID, memberID, fx.tenantID)
	require.NoError(t, err)

	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenantID)

	err = backend.checkCalendarWritePermission(ctx, memberID, fx.calendarID)

	assert.Equal(t, 403, webdavStatusCode(t, err))
}

func TestCheckCalendarWritePermission_NoMembershipForbidden_WithTenantCtx(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	strangerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     fx.tenantID,
		"email":         fmt.Sprintf("caldav-stranger-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", strangerID) })

	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenantID)

	err := backend.checkCalendarWritePermission(ctx, strangerID, fx.calendarID)

	assert.Equal(t, 403, webdavStatusCode(t, err))
}

func TestCheckCalendarWritePermission_CalendarNotFound_WithTenantCtx(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CalDAV Permission Test Tenant NotFound")
	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	err := backend.checkCalendarWritePermission(ctx, uuid.New(), uuid.New())

	assert.Equal(t, 404, webdavStatusCode(t, err))
}

// ============================================================================
// checkCalendarWritePermission -- the real CalDAV Basic-Auth context (bug)
// ============================================================================

func TestCheckCalendarWritePermission_RealCalDAVContext_OwnerBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	backend := &CalDAVBackend{pool: pool}

	// This is exactly what PutCalendarObject/DeleteCalendarObject pass in
	// production: a context stamped only via CtxWithUser (basicAuthMiddleware),
	// never a tenant -- CalDAV authenticates over Basic Auth, not JWT.
	realCalDAVCtx := CtxWithUser(context.Background(), fx.ownerID)

	err := backend.checkCalendarWritePermission(realCalDAVCtx, fx.ownerID, fx.calendarID)

	require.Error(t, err, "documents the confirmed bug: RLS silently hides the "+
		"calendar from its own owner because no tenant GUC is set for this ctx")
	assert.Equal(t, 404, webdavStatusCode(t, err))
}

// ============================================================================
// listEventExceptions -- WithTenantCtx (intended behaviour)
// ============================================================================

func TestListEventExceptions_WithTenantCtx_ReturnsSeededExceptionsInDateOrder(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   fx.tenantID,
		"calendar_id": fx.calendarID,
		"title":       "Recurring Standup",
		"start_time":  "2026-03-01T09:00:00Z",
		"end_time":    "2026-03-01T09:30:00Z",
		"created_by":  fx.ownerID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "calendar_events", eventID) })

	later := testutil.SeedRow(t, pool, "event_exceptions", map[string]any{
		"tenant_id":     fx.tenantID,
		"event_id":      eventID,
		"original_date": "2026-03-15",
		"is_cancelled":  true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "event_exceptions", later) })
	earlier := testutil.SeedRow(t, pool, "event_exceptions", map[string]any{
		"tenant_id":     fx.tenantID,
		"event_id":      eventID,
		"original_date": "2026-03-08",
		"is_cancelled":  false,
		"title":         "Standup (moved to afternoon)",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "event_exceptions", earlier) })

	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenantID)

	exceptions, err := backend.listEventExceptions(ctx, eventID)

	require.NoError(t, err)
	require.Len(t, exceptions, 2)
	assert.Equal(t, earlier, exceptions[0].ID, "expected ORDER BY original_date ascending")
	assert.Equal(t, later, exceptions[1].ID)
	assert.False(t, exceptions[0].IsCancelled)
	require.NotNil(t, exceptions[0].Title)
	assert.Equal(t, "Standup (moved to afternoon)", *exceptions[0].Title)
	assert.True(t, exceptions[1].IsCancelled)
}

func TestListEventExceptions_NoExceptions_ReturnsEmptySliceNotNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   fx.tenantID,
		"calendar_id": fx.calendarID,
		"title":       "No Exceptions Event",
		"start_time":  "2026-03-01T09:00:00Z",
		"end_time":    "2026-03-01T09:30:00Z",
		"created_by":  fx.ownerID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "calendar_events", eventID) })

	backend := &CalDAVBackend{pool: pool}
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenantID)

	exceptions, err := backend.listEventExceptions(ctx, eventID)

	require.NoError(t, err)
	assert.NotNil(t, exceptions)
	assert.Empty(t, exceptions)
}

// ============================================================================
// listEventExceptions -- the real CalDAV Basic-Auth context (bug)
// ============================================================================

func TestListEventExceptions_RealCalDAVContext_SilentlyEmpty(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   fx.tenantID,
		"calendar_id": fx.calendarID,
		"title":       "Recurring Standup",
		"start_time":  "2026-03-01T09:00:00Z",
		"end_time":    "2026-03-01T09:30:00Z",
		"created_by":  fx.ownerID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "calendar_events", eventID) })
	excID := testutil.SeedRow(t, pool, "event_exceptions", map[string]any{
		"tenant_id":     fx.tenantID,
		"event_id":      eventID,
		"original_date": "2026-03-15",
		"is_cancelled":  true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "event_exceptions", excID) })

	backend := &CalDAVBackend{pool: pool}
	// GetCalendarObject/expandedEventsToObjects call listEventExceptions with
	// exactly this ctx in production -- no tenant, Basic-Auth only.
	realCalDAVCtx := CtxWithUser(context.Background(), fx.ownerID)

	exceptions, err := backend.listEventExceptions(realCalDAVCtx, eventID)

	require.NoError(t, err, "the query itself never errors -- RLS filters "+
		"rows away silently, which is the dangerous half of this bug: no "+
		"CalDAV client ever sees a failure to retry, the cancelled/overridden "+
		"occurrence just never shows up")
	assert.Empty(t, exceptions, "documents the confirmed bug: a seeded "+
		"exception is invisible under the real CalDAV request context")
}
