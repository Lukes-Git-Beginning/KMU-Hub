package gateway

import (
	"net/http/httptest"
	"testing"
)

// CalendarRoutes.ServiceName() uses "work" (shared gRPC connection with Work service).

func TestCalendarRoutes_ServiceName(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	if routes.ServiceName() != "work" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "work")
	}
}

// --- HandleCreateCalendar ---

func TestHandleCreateCalendar_MissingName(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/calendars", jsonBody(t, map[string]interface{}{
		"calendar_type": "personal",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateCalendar(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateCalendar_MissingCalendarType(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/calendars", jsonBody(t, map[string]interface{}{
		"name": "Team Calendar",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateCalendar(rec, req)
	assertValidationError(t, rec, "calendar_type")
}

func TestHandleCreateCalendar_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/calendars", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateCalendar(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleCreateEvent ---

func TestHandleCreateEvent_MissingTitle(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/events", jsonBody(t, map[string]interface{}{
		"calendar_id": "550e8400-e29b-41d4-a716-446655440000",
		"start_time":  "2026-06-10T10:00:00Z",
		"end_time":    "2026-06-10T11:00:00Z",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateEvent(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateEvent_MissingCalendarID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/events", jsonBody(t, map[string]interface{}{
		"title":      "Team Meeting",
		"start_time": "2026-06-10T10:00:00Z",
		"end_time":   "2026-06-10T11:00:00Z",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateEvent(rec, req)
	assertValidationError(t, rec, "calendar_id")
}

func TestHandleCreateEvent_InvalidCalendarIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/events", jsonBody(t, map[string]interface{}{
		"title":       "Team Meeting",
		"calendar_id": "not-a-uuid",
		"start_time":  "2026-06-10T10:00:00Z",
		"end_time":    "2026-06-10T11:00:00Z",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateEvent(rec, req)
	assertValidationError(t, rec, "calendar_id")
}

// --- HandleSeedHolidays ---

func TestHandleSeedHolidays_YearZero(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/holidays/seed", jsonBody(t, map[string]interface{}{
		"year":         0,
		"country_code": "DE",
	}))
	req = withUserID(req, "user-123")
	routes.HandleSeedHolidays(rec, req)
	assertValidationError(t, rec, "year")
}

func TestHandleSeedHolidays_MissingCountryCode(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/holidays/seed", jsonBody(t, map[string]interface{}{
		"year": 2026,
	}))
	req = withUserID(req, "user-123")
	routes.HandleSeedHolidays(rec, req)
	assertValidationError(t, rec, "country_code")
}

// --- HandleRSVPToEvent ---

func TestHandleRSVPToEvent_InvalidStatus(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/events/550e8400-e29b-41d4-a716-446655440000/rsvp",
		jsonBody(t, map[string]interface{}{
			"status": "maybe", // not in oneof
		}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRSVPToEvent(rec, req)
	assertValidationError(t, rec, "status")
}

func TestHandleRSVPToEvent_MissingStatus(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/calendar/events/550e8400-e29b-41d4-a716-446655440000/rsvp",
		jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRSVPToEvent(rec, req)
	assertValidationError(t, rec, "status")
}

// --- HandleUpdateRecurringEvent ---

func TestHandleUpdateRecurringEvent_InvalidScope(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/calendar/events/550e8400-e29b-41d4-a716-446655440000/recurring",
		jsonBody(t, map[string]interface{}{
			"scope":         "invalid_scope",
			"original_date": "2026-06-10",
		}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateRecurringEvent(rec, req)
	assertValidationError(t, rec, "scope")
}
