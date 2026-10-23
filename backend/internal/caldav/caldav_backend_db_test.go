package caldav

// DB-backed coverage for checkCalendarWritePermission and listEventExceptions
// against the real calendars/calendar_members/event_exceptions schema and
// its RLS tenant_isolation policies (migration 000124).
//
// The CalDAV Basic-Auth request path never carries a JWT, so the tenant has
// to come from NewCtxInjector (basicAuthMiddleware, route_caldav.go): it
// resolves the user's tenant and stamps it under middleware.TenantIDKey, which
// database.NewPostgresPool's PrepareConn hook turns into app.tenant_id. Without
// it RLS (kmuhub_app, FORCE ROW LEVEL SECURITY) hides every row. The
// *_RealCalDAVContext_* tests build their context through the production
// injector -- nothing hand-stamped -- and prove:
//   - the calendar's owner may write (PutCalendarObject/DeleteCalendarObject),
//   - recurring-event exceptions are visible to CalDAV reads,
//   - a user from another tenant still gets 404 / no rows, i.e. the fix is
//     the tenant context, not a blanket RLS bypass.
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
// checkCalendarWritePermission -- the real CalDAV Basic-Auth context
// ============================================================================

func TestCheckCalendarWritePermission_RealCalDAVContext_OwnerAllowed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	backend := &CalDAVBackend{pool: pool}

	// What PutCalendarObject/DeleteCalendarObject receive in production: the
	// context basicAuthMiddleware builds after validating the app password.
	realCalDAVCtx := realCalDAVCtx(t, pool, fx.ownerID)

	err := backend.checkCalendarWritePermission(realCalDAVCtx, fx.ownerID, fx.calendarID)

	assert.NoError(t, err)
}

func TestCheckCalendarWritePermission_RealCalDAVContext_OtherTenantSeesNoCalendar(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := seedCalendarFixture(t, pool)
	foreignUserID := seedForeignTenantUser(t, pool)
	backend := &CalDAVBackend{pool: pool}

	err := backend.checkCalendarWritePermission(
		realCalDAVCtx(t, pool, foreignUserID), foreignUserID, fx.calendarID)

	assert.Equal(t, 404, webdavStatusCode(t, err),
		"another tenant's calendar must stay invisible under the real CalDAV context")
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
// listEventExceptions -- the real CalDAV Basic-Auth context
// ============================================================================

func TestListEventExceptions_RealCalDAVContext_ReturnsExceptions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx, eventID, excID := seedEventWithException(t, pool)
	backend := &CalDAVBackend{pool: pool}

	// GetCalendarObject/expandedEventsToObjects call listEventExceptions with
	// exactly this ctx in production.
	exceptions, err := backend.listEventExceptions(realCalDAVCtx(t, pool, fx.ownerID), eventID)

	require.NoError(t, err)
	require.Len(t, exceptions, 1)
	assert.Equal(t, excID, exceptions[0].ID)
	assert.True(t, exceptions[0].IsCancelled)
}

func TestListEventExceptions_RealCalDAVContext_OtherTenantSeesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	_, eventID, _ := seedEventWithException(t, pool)
	foreignUserID := seedForeignTenantUser(t, pool)
	backend := &CalDAVBackend{pool: pool}

	exceptions, err := backend.listEventExceptions(realCalDAVCtx(t, pool, foreignUserID), eventID)

	require.NoError(t, err)
	assert.Empty(t, exceptions, "another tenant's exceptions must not leak")
}

// realCalDAVCtx builds the context a real CalDAV request carries: exactly
// what basicAuthMiddleware produces from the production injector.
func realCalDAVCtx(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) context.Context {
	t.Helper()
	ctx, err := NewCtxInjector(pool)(context.Background(), userID)
	require.NoError(t, err)
	return ctx
}

// seedForeignTenantUser creates a user in a second tenant.
func seedForeignTenantUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CalDAV Permission Test Tenant Foreign")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("caldav-foreign-%s@tenantb.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	return userID
}

// seedEventWithException creates a fixture calendar with one event carrying a
// cancelled occurrence.
func seedEventWithException(t *testing.T, pool *pgxpool.Pool) (calendarFixture, uuid.UUID, uuid.UUID) {
	t.Helper()
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
	return fx, eventID, excID
}
