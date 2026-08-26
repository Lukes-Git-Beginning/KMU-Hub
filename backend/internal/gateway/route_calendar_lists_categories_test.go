package gateway

// route_calendar_lists_categories_test.go covers the eleven route_calendar.go
// handlers that had zero test coverage as of the
// cov-gateway-calendar-lists-and-categories backlog unit (2026-08-24):
// calendar listing/browsing, calendar members, event categories, event
// attendees, holidays, calendar preferences and task deadlines.
//
// Data-exposure findings, answered by reading the server/service/repository
// layers (not by a new test here -- this package has no fake
// CalendarServiceClient, so a gateway unit test can only prove the handler
// forwards the caller's own user_id/tenant scope, never what the RPC layer
// does with it):
//
//   - HandleListBrowsableCalendars ("browsable" = calendars owned by someone
//     else): internal/work/calendar/postgres_repository.go ListBrowsable only
//     SELECTs calendar-level metadata (id, tenant_id, name, description,
//     calendar_type, color, owner_id, is_default, timezone, timestamps) --
//     no events, no event titles. A shared calendar's name/description is
//     shown to any tenant member with "calendars:read", by design (that is
//     the point of a discoverable "shared" calendar type); no event content
//     leaks. Not a finding.
//
//   - HandleListEventAttendees: internal/server/calendar_grpc.go
//     ListEventAttendees -> internal/work/event/service.go ListAttendees only
//     calls repo.GetByID(eventID, tenantID) before listing attendees -- it
//     never checks that the caller is an attendee, the event's creator, or
//     even a member of the event's calendar. Any tenant user holding the
//     coarse tenant-wide "calendars:read" permission can list the attendees
//     of ANY event in the tenant. This is not a regression introduced by this
//     handler: GetEvent (calendar_grpc.go:461, eventService.Get) has the
//     exact same shape -- tenant-scoped existence check only, no per-calendar
//     ACL -- so any user who can already read event details before this unit
//     could already reach the same information via GET /events/{id}. Compare
//     SetReminders (event/service.go:504), which DOES add
//     requireCalendarEditPermission for non-owners; the read path has no
//     equivalent. Consistent, pre-existing coarse-permission architecture
//     (documented for calendars:read across this module already), not a new
//     deviation -- no fix-unit raised, but flagged here since the backlog
//     scope asked the question directly.
//
//   - HandleListTaskDeadlinesInRange: internal/work/event/postgres_repository.go
//     ListTaskDeadlinesInRange scopes by t.assignee_id = userID OR
//     project_members.user_id = userID (own tasks / own projects only, no
//     other user's tasks leak) and carries no explicit tenant_id filter in
//     SQL -- but the `tasks` table has RLS enabled since Migration 000120
//     (`CALL enable_tenant_rls('tasks')`), consistent with the
//     already-accepted ADR-006 architecture (RLS instead of an app-level
//     filter, same pattern verified for document search in Iteration 34). No
//     fix-unit raised.
//
// Consistent with the rest of this package's ReachesRPC convention: a dummy
// localhost:0 connection makes getXClient succeed but the RPC call itself
// fail, surfacing as 503. Tests named *_ReachesRPC assert exactly that --
// they document "the handler cleared its own validation and forwarded the
// request", not what a real CalendarService would answer.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// ============================================================================
// HandleListCalendars
// ============================================================================

func TestHandleListCalendars_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListCalendars(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCalendars_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListCalendars(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCalendars_IncludeHidden_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars?include_hidden=true", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListCalendars(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListBrowsableCalendars
// ============================================================================

func TestHandleListBrowsableCalendars_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/browse", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListBrowsableCalendars(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBrowsableCalendars_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/browse", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListBrowsableCalendars(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBrowsableCalendars_WithSearch_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/browse?search=team", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListBrowsableCalendars(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListCalendarMembers
// ============================================================================

// TestHandleListCalendarMembers_InvalidCalendarIDUUID_ReachesRPC documents
// that, unlike HandleGetCalendar/HandleUpdateCalendar/HandleDeleteCalendar in
// the same struct, this handler reads "id" via raw chi.URLParam with no
// validateUUIDParam call -- the same pre-existing gap already catalogued for
// the membership-mutation handlers in route_calendar_membership_test.go.
func TestHandleListCalendarMembers_InvalidCalendarIDUUID_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars/not-a-uuid/members", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListCalendarMembers(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCalendarMembers_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars/"+calID+"/members", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleListCalendarMembers(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCalendarMembers_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars/"+calID+"/members", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleListCalendarMembers(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListEventAttendees
// ============================================================================

func TestHandleListEventAttendees_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+id+"/attendees", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListEventAttendees(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEventAttendees_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/not-a-uuid/attendees", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListEventAttendees(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListEventAttendees_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+id+"/attendees", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListEventAttendees(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListEventCategories / HandleCreateEventCategory / HandleDeleteEventCategory
// ============================================================================

func TestHandleListEventCategories_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/categories", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventCategories(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEventCategories_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/categories", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventCategories(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateEventCategory_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/categories", jsonBody(t, map[string]interface{}{
		"name": "Urlaub",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateEventCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateEventCategory_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/categories", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateEventCategory(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateEventCategory_MissingName(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/categories", jsonBody(t, map[string]interface{}{
		"color": "#ff0000",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateEventCategory(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateEventCategory_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/categories", jsonBody(t, map[string]interface{}{
		"name": "Urlaub",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateEventCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteEventCategory_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/categories/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteEventCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteEventCategory_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/categories/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteEventCategory(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteEventCategory_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/categories/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteEventCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListHolidays
// ============================================================================

func TestHandleListHolidays_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays?country_code=DE", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListHolidays_MissingCountryCode(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "country_code query parameter is required")
}

func TestHandleListHolidays_InvalidStartFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays?country_code=DE&start=not-a-date", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start time format")
}

func TestHandleListHolidays_InvalidEndFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays?country_code=DE&start=2026-01-01T00:00:00Z&end=not-a-date", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid end time format")
}

func TestHandleListHolidays_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays?country_code=DE", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListHolidays_WithSubdivisionAndRange_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/holidays?country_code=DE&subdivision_code=RP&start=2026-01-01T00:00:00Z&end=2026-12-31T00:00:00Z", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListHolidays(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetCalendarPreferences / HandleUpdateCalendarPreferences
// ============================================================================

func TestHandleGetCalendarPreferences_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/preferences", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleGetCalendarPreferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetCalendarPreferences_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/preferences", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleGetCalendarPreferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCalendarPreferences_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/preferences", jsonBody(t, map[string]interface{}{
		"default_view": "week",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleUpdateCalendarPreferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCalendarPreferences_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/preferences", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleUpdateCalendarPreferences(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleUpdateCalendarPreferences_EmptyBody_ReachesRPC documents that
// updateCalPreferencesRequest has no `validate:"required"` field at all --
// every field is an optional pointer -- so an empty JSON object `{}` is a
// valid request that is forwarded to the RPC layer unchanged.
func TestHandleUpdateCalendarPreferences_EmptyBody_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/preferences", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleUpdateCalendarPreferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCalendarPreferences_AllFields_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/preferences", jsonBody(t, map[string]interface{}{
		"default_view":                     "month",
		"week_days":                        5,
		"default_reminder_minutes":         15,
		"default_all_day_reminder_minutes": 60,
		"subdivision_code":                 "BY",
		"show_task_deadlines":              true,
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleUpdateCalendarPreferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListTaskDeadlinesInRange
// ============================================================================

func TestHandleListTaskDeadlinesInRange_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/task-deadlines?start=2026-01-01T00:00:00Z&end=2026-01-31T00:00:00Z", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListTaskDeadlinesInRange(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTaskDeadlinesInRange_MissingStart(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/task-deadlines?end=2026-01-31T00:00:00Z", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListTaskDeadlinesInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start and end query parameters are required")
}

func TestHandleListTaskDeadlinesInRange_MissingEnd(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/task-deadlines?start=2026-01-01T00:00:00Z", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListTaskDeadlinesInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start and end query parameters are required")
}

func TestHandleListTaskDeadlinesInRange_InvalidStartFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/task-deadlines?start=not-a-date&end=2026-01-31T00:00:00Z", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListTaskDeadlinesInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start time format")
}

func TestHandleListTaskDeadlinesInRange_InvalidEndFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/task-deadlines?start=2026-01-01T00:00:00Z&end=not-a-date", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListTaskDeadlinesInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid end time format")
}

func TestHandleListTaskDeadlinesInRange_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/task-deadlines?start=2026-01-01T00:00:00Z&end=2026-01-31T00:00:00Z", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListTaskDeadlinesInRange(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
