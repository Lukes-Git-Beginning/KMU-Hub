package caldav

// This file covers the parts of CalDAVBackend that don't need a live
// Postgres or a real CalendarService RPC round trip: the pure identity/path
// helpers, the always-403 CreateCalendar stub, the queryTimeRange filter
// extraction (previously inlined in QueryCalendarObjects, extracted here so
// its branches are directly testable), and the calendarClient()
// error-propagation path shared by every RPC-backed method. There is no
// bufconn stub for CalendarServiceClient in this repo (same boundary noted
// throughout internal/gateway's *_test.go files), so success responses from
// ListCalendars/GetCalendar/GetCalendarObject/etc. cannot be exercised here;
// what IS testable and matters for production is that a missing/unreachable
// calendar service surfaces as the correct WebDAV status, not a panic or a
// silently swallowed error.

import (
	"context"
	"testing"
	"time"

	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/gateway"
)

// emptyRegistry returns a ServiceRegistry with no registered services --
// calendarClient() fails immediately with a plain (non-gRPC-status) error.
func emptyRegistry() *gateway.ServiceRegistry {
	return gateway.NewServiceRegistry(nil)
}

// registryWithWorkService returns a registry with "work" registered at a
// dummy address. grpc.NewClient is non-blocking, so calendarClient()
// succeeds; the RPC itself fails at call time with codes.Unavailable.
func registryWithWorkService() *gateway.ServiceRegistry {
	reg := gateway.NewServiceRegistry(nil)
	reg.Register("work", "localhost:0")
	return reg
}

// ============================================================================
// CurrentUserPrincipal / CalendarHomeSetPath
// ============================================================================

func TestCurrentUserPrincipal_NoUserInCtx_Returns401(t *testing.T) {
	b := &CalDAVBackend{registry: emptyRegistry()}

	_, err := b.CurrentUserPrincipal(context.Background())

	assert.Equal(t, 401, webdavStatusCode(t, err))
}

func TestCurrentUserPrincipal_AuthenticatedUser_ReturnsPrincipalPath(t *testing.T) {
	b := &CalDAVBackend{registry: emptyRegistry()}
	userID := uuid.New()
	ctx := CtxWithUser(context.Background(), userID)

	path, err := b.CurrentUserPrincipal(ctx)

	require.NoError(t, err)
	assert.Equal(t, "/caldav/principals/"+userID.String()+"/", path)
}

func TestCalendarHomeSetPath_NoUserInCtx_Returns401(t *testing.T) {
	b := &CalDAVBackend{registry: emptyRegistry()}

	_, err := b.CalendarHomeSetPath(context.Background())

	assert.Equal(t, 401, webdavStatusCode(t, err))
}

func TestCalendarHomeSetPath_AuthenticatedUser_ReturnsCalendarsPath(t *testing.T) {
	b := &CalDAVBackend{registry: emptyRegistry()}
	userID := uuid.New()
	ctx := CtxWithUser(context.Background(), userID)

	path, err := b.CalendarHomeSetPath(ctx)

	require.NoError(t, err)
	assert.Equal(t, "/caldav/principals/"+userID.String()+"/calendars/", path)
}

// ============================================================================
// CreateCalendar
// ============================================================================

func TestCreateCalendar_AlwaysForbidden(t *testing.T) {
	b := &CalDAVBackend{registry: emptyRegistry()}

	err := b.CreateCalendar(context.Background(), &caldav.Calendar{Name: "New"})

	assert.Equal(t, 403, webdavStatusCode(t, err))
}

// ============================================================================
// calendarClient() error propagation
// ============================================================================

func TestListCalendars_ServiceNotRegistered_ReturnsError(t *testing.T) {
	b := NewCalDAVBackend(emptyRegistry(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())

	_, err := b.ListCalendars(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestListCalendars_ServiceUnavailable_Returns503(t *testing.T) {
	b := NewCalDAVBackend(registryWithWorkService(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())

	_, err := b.ListCalendars(ctx)

	assert.Equal(t, 503, webdavStatusCode(t, err))
}

func TestGetCalendar_InvalidPath_Returns400(t *testing.T) {
	b := NewCalDAVBackend(emptyRegistry(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())

	_, err := b.GetCalendar(ctx, "/caldav/principals/u1/calendars/not-a-uuid/")

	assert.Equal(t, 400, webdavStatusCode(t, err))
}

func TestGetCalendar_ServiceUnavailable_Returns503(t *testing.T) {
	b := NewCalDAVBackend(registryWithWorkService(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())
	path := "/caldav/principals/u1/calendars/" + uuid.New().String() + "/"

	_, err := b.GetCalendar(ctx, path)

	assert.Equal(t, 503, webdavStatusCode(t, err))
}

func TestGetCalendarObject_InvalidPath_Returns400(t *testing.T) {
	b := NewCalDAVBackend(emptyRegistry(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())

	_, err := b.GetCalendarObject(ctx, "///", nil)

	assert.Equal(t, 400, webdavStatusCode(t, err))
}

func TestGetCalendarObject_ServiceUnavailable_Returns503(t *testing.T) {
	b := NewCalDAVBackend(registryWithWorkService(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())
	path := "/caldav/principals/u1/calendars/" + uuid.New().String() + "/event-1.ics"

	_, err := b.GetCalendarObject(ctx, path, nil)

	assert.Equal(t, 503, webdavStatusCode(t, err))
}

func TestListCalendarObjects_ServiceUnavailable_Returns503(t *testing.T) {
	b := NewCalDAVBackend(registryWithWorkService(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())
	path := "/caldav/principals/u1/calendars/" + uuid.New().String() + "/"

	_, err := b.ListCalendarObjects(ctx, path, nil)

	assert.Equal(t, 503, webdavStatusCode(t, err))
}

func TestQueryCalendarObjects_ServiceUnavailable_Returns503(t *testing.T) {
	b := NewCalDAVBackend(registryWithWorkService(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())
	path := "/caldav/principals/u1/calendars/" + uuid.New().String() + "/"

	_, err := b.QueryCalendarObjects(ctx, path, nil)

	assert.Equal(t, 503, webdavStatusCode(t, err))
}

func TestPutCalendarObject_InvalidPath_Returns400(t *testing.T) {
	b := NewCalDAVBackend(emptyRegistry(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())

	_, err := b.PutCalendarObject(ctx, "///", nil, nil)

	assert.Equal(t, 400, webdavStatusCode(t, err))
}

func TestDeleteCalendarObject_InvalidPath_Returns400(t *testing.T) {
	b := NewCalDAVBackend(emptyRegistry(), nil, nil)
	ctx := CtxWithUser(context.Background(), uuid.New())

	err := b.DeleteCalendarObject(ctx, "///")

	assert.Equal(t, 400, webdavStatusCode(t, err))
}

// ============================================================================
// queryTimeRange
// ============================================================================

func TestQueryTimeRange_NilQuery_ReturnsWideDefaultWindow(t *testing.T) {
	start, end := queryTimeRange(nil)

	now := time.Now()
	assert.WithinDuration(t, now.AddDate(-2, 0, 0), start, time.Minute)
	assert.WithinDuration(t, now.AddDate(2, 0, 0), end, time.Minute)
}

func TestQueryTimeRange_EmptyCompFilter_ReturnsWideDefaultWindow(t *testing.T) {
	start, end := queryTimeRange(&caldav.CalendarQuery{})

	now := time.Now()
	assert.WithinDuration(t, now.AddDate(-2, 0, 0), start, time.Minute)
	assert.WithinDuration(t, now.AddDate(2, 0, 0), end, time.Minute)
}

func TestQueryTimeRange_TopLevelCompFilterRange_Used(t *testing.T) {
	want := caldav.CalendarQuery{
		CompFilter: caldav.CompFilter{
			Start: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	start, end := queryTimeRange(&want)

	assert.True(t, want.CompFilter.Start.Equal(start))
	assert.True(t, want.CompFilter.End.Equal(end))
}

func TestQueryTimeRange_NestedVEventFilter_OverridesTopLevel(t *testing.T) {
	topStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	topEnd := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	veventStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	veventEnd := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	query := &caldav.CalendarQuery{
		CompFilter: caldav.CompFilter{
			Start: topStart,
			End:   topEnd,
			Comps: []caldav.CompFilter{
				{Name: "VEVENT", Start: veventStart, End: veventEnd},
			},
		},
	}

	start, end := queryTimeRange(query)

	assert.True(t, veventStart.Equal(start), "expected nested VEVENT start to win over top-level")
	assert.True(t, veventEnd.Equal(end), "expected nested VEVENT end to win over top-level")
}

func TestQueryTimeRange_NestedNonVEventFilter_Ignored(t *testing.T) {
	topStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	topEnd := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	query := &caldav.CalendarQuery{
		CompFilter: caldav.CompFilter{
			Start: topStart,
			End:   topEnd,
			Comps: []caldav.CompFilter{
				{Name: "VTODO", Start: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
	}

	start, end := queryTimeRange(query)

	assert.True(t, topStart.Equal(start), "a VTODO filter must not affect the VEVENT time range")
	assert.True(t, topEnd.Equal(end))
}

func TestQueryTimeRange_NestedVEventFilter_PartialOverride_KeepsTopLevelOtherBound(t *testing.T) {
	topStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	topEnd := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	veventStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	query := &caldav.CalendarQuery{
		CompFilter: caldav.CompFilter{
			Start: topStart,
			End:   topEnd,
			Comps: []caldav.CompFilter{
				{Name: "VEVENT", Start: veventStart}, // End left zero
			},
		},
	}

	start, end := queryTimeRange(query)

	assert.True(t, veventStart.Equal(start))
	assert.True(t, topEnd.Equal(end), "a zero VEVENT end must not blank out the top-level end")
}
