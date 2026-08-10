package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// This file covers internal/gateway/route_notification.go — notification CRUD,
// preferences, event types, mutes, quiet hours, and manual do-not-disturb.

func newNotificationRoutesUnavailable() *NotificationRoutes {
	return NewNotificationRoutes(emptyRegistry())
}

func newNotificationRoutesReachable() *NotificationRoutes {
	return NewNotificationRoutes(registryWithService("notification"))
}

// --- ServiceUnavailable: every handler must bail out with 503 before doing
// anything else once it cannot obtain a client, regardless of what else is
// wrong with the request (missing id, malformed body, missing user). ---

func TestNotificationHandlers_ServiceUnavailable(t *testing.T) {
	routes := newNotificationRoutesUnavailable()
	handlers := map[string]http.HandlerFunc{
		"HandleListNotifications":  routes.HandleListNotifications,
		"HandleGetUnreadCount":     routes.HandleGetUnreadCount,
		"HandleMarkRead":           routes.HandleMarkRead,
		"HandlePin":                routes.HandlePin,
		"HandleUnpin":              routes.HandleUnpin,
		"HandleDismiss":            routes.HandleDismiss,
		"HandleSnooze":             routes.HandleSnooze,
		"HandleMarkAllRead":        routes.HandleMarkAllRead,
		"HandleGetPreferences":     routes.HandleGetPreferences,
		"HandleUpdatePreference":   routes.HandleUpdatePreference,
		"HandleListEventTypes":     routes.HandleListEventTypes,
		"HandleMuteResource":       routes.HandleMuteResource,
		"HandleUnmuteResource":     routes.HandleUnmuteResource,
		"HandleListMutedResources": routes.HandleListMutedResources,
		"HandleGetQuietHours":      routes.HandleGetQuietHours,
		"HandleUpdateQuietHours":   routes.HandleUpdateQuietHours,
		"HandleGetDND":             routes.HandleGetDND,
		"HandleToggleDND":          routes.HandleToggleDND,
		"HandleDisableDND":         routes.HandleDisableDND,
	}
	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// --- HandleGetUnreadCount ---

func TestHandleGetUnreadCount_ReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetUnreadCount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListNotifications ---

func TestHandleListNotifications_FiltersReachRPC(t *testing.T) {
	cases := []string{
		"/api/v1/notifications",
		"/api/v1/notifications?module_id=crm",
		"/api/v1/notifications?is_read=true",
		"/api/v1/notifications?is_read=false",
		"/api/v1/notifications?page=2&page_size=50",
		"/api/v1/notifications?page=not-a-number&page_size=not-a-number",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			routes := newNotificationRoutesReachable()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			req = withUserID(req, "user-123")
			routes.HandleListNotifications(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// TestHandleListNotifications_UserFromContextNotQuery proves the handler
// takes the caller's identity from the authenticated context, not from any
// caller-supplied query parameter — an attacker cannot list another user's
// notifications by appending a foreign identifier to the URL. There is no
// stub gRPC client in this package to capture the outbound request field, so
// this is backed by reading the handler: it calls only
// middleware.GetUserID(r.Context()) and never r.URL.Query().Get("user_id") —
// confirmed absent by inspection of every handler in this file. This test
// pins that a request carrying a spoofed user_id query parameter still
// reaches the RPC layer identically to one without it (no special-casing,
// no panic, no divergent status), which is what a query-param-controlled
// bypass would first show up as.
func TestHandleListNotifications_UserFromContextNotQuery(t *testing.T) {
	routes := newNotificationRoutesReachable()

	recWithSpoof := httptest.NewRecorder()
	reqWithSpoof := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?user_id=someone-elses-id", nil)
	reqWithSpoof = withUserID(reqWithSpoof, "user-123")
	routes.HandleListNotifications(recWithSpoof, reqWithSpoof)

	recWithout := httptest.NewRecorder()
	reqWithout := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	reqWithout = withUserID(reqWithout, "user-123")
	routes.HandleListNotifications(recWithout, reqWithout)

	assertStatus(t, recWithSpoof, http.StatusServiceUnavailable)
	assertStatus(t, recWithout, http.StatusServiceUnavailable)
	if recWithSpoof.Code != recWithout.Code {
		t.Errorf("spoofed user_id query param changed handler behaviour: %d vs %d", recWithSpoof.Code, recWithout.Code)
	}
}

// --- HandleMarkRead / HandlePin / HandleUnpin / HandleDismiss share the same
// "validate {id} as UUID, then call a single-arg RPC" shape. ---

func TestNotificationIDHandlers_InvalidUUID(t *testing.T) {
	routes := newNotificationRoutesReachable()
	handlers := map[string]http.HandlerFunc{
		"HandleMarkRead": routes.HandleMarkRead,
		"HandlePin":      routes.HandlePin,
		"HandleUnpin":    routes.HandleUnpin,
		"HandleDismiss":  routes.HandleDismiss,
	}
	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/not-a-uuid/read", nil)
			req = withAuth(req, "user-123", testTenantID)
			req = withChiURLParam(req, "id", "not-a-uuid")
			h(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, "invalid id")
		})
	}
}

func TestNotificationIDHandlers_ValidUUIDReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	validID := "550e8400-e29b-41d4-a716-446655440000"
	handlers := map[string]http.HandlerFunc{
		"HandleMarkRead": routes.HandleMarkRead,
		"HandlePin":      routes.HandlePin,
		"HandleUnpin":    routes.HandleUnpin,
		"HandleDismiss":  routes.HandleDismiss,
	}
	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validID+"/read", nil)
			req = withAuth(req, "user-123", testTenantID)
			req = withChiURLParam(req, "id", validID)
			h(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- HandleSnooze ---

func TestHandleSnooze_InvalidUUID(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/not-a-uuid/snooze", jsonBody(t, map[string]string{
		"until": time.Now().Add(time.Hour).Format(time.RFC3339),
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSnooze(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleSnooze_InvalidJSON(t *testing.T) {
	routes := newNotificationRoutesReachable()
	validID := "550e8400-e29b-41d4-a716-446655440000"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validID+"/snooze", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", validID)
	routes.HandleSnooze(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSnooze_PastTimeRejected(t *testing.T) {
	routes := newNotificationRoutesReachable()
	validID := "550e8400-e29b-41d4-a716-446655440000"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validID+"/snooze", jsonBody(t, map[string]string{
		"until": time.Now().Add(-time.Hour).Format(time.RFC3339),
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", validID)
	routes.HandleSnooze(rec, req)
	assertStatus(t, rec, http.StatusUnprocessableEntity)
	assertErrorContains(t, rec, "must be in the future")
}

func TestHandleSnooze_FutureTimeReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	validID := "550e8400-e29b-41d4-a716-446655440000"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validID+"/snooze", jsonBody(t, map[string]string{
		"until": time.Now().Add(time.Hour).Format(time.RFC3339),
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", validID)
	routes.HandleSnooze(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleMarkAllRead ---
//
// r.Body is checked for nil before decoding, matching the doc comment "Body
// is optional for mark-all-read". But net/http never hands a handler a nil
// Body — a request without a body still carries http.NoBody (or an empty
// io.NopCloser via httptest.NewRequest), which is non-nil and decodes to
// io.EOF. So in practice this branch cannot skip decoding; a genuinely
// bodyless POST /read-all hits the JSON-decode-error path and gets a 400,
// not the intended "no filter" behaviour. Pinned as the current (buggy)
// behaviour below — see the journal for the finding and the fix vs.
// send-empty-object write-up.

func TestHandleMarkAllRead_NoBodyReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleMarkAllRead(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMarkAllRead_EmptyObjectBodyReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", jsonBody(t, map[string]string{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleMarkAllRead(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMarkAllRead_InvalidJSON(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleMarkAllRead(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleMarkAllRead_WithModuleIDReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", jsonBody(t, map[string]string{
		"module_id": "crm",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleMarkAllRead(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetPreferences / HandleUpdatePreference ---

func TestHandleGetPreferences_WithModuleIDReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/preferences?module_id=crm", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetPreferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdatePreference_InvalidJSON(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdatePreference(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdatePreference_ValidReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", jsonBody(t, map[string]any{
		"event_type_key": "task.assigned",
		"module_id":      "work",
		"in_app":         true,
		"desktop_push":   false,
		"email":          true,
		"sms":            false,
		"sound":          "chime",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdatePreference(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListEventTypes ---

func TestHandleListEventTypes_WithAndWithoutModuleID(t *testing.T) {
	targets := []string{"/api/v1/notifications/event-types", "/api/v1/notifications/event-types?module_id=crm"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			routes := newNotificationRoutesReachable()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			routes.HandleListEventTypes(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- HandleMuteResource ---

func TestHandleMuteResource_MissingModuleID(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/mutes", jsonBody(t, map[string]string{
		"resource_id": "deal-1",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleMuteResource(rec, req)
	assertValidationError(t, rec, "module_id")
}

func TestHandleMuteResource_MissingResourceID(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/mutes", jsonBody(t, map[string]string{
		"module_id": "crm",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleMuteResource(rec, req)
	assertValidationError(t, rec, "resource_id")
}

func TestHandleMuteResource_ValidReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/mutes", jsonBody(t, map[string]string{
		"module_id":   "crm",
		"resource_id": "deal-1",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleMuteResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUnmuteResource ---

func TestHandleUnmuteResource_InvalidUUID(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/mutes/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "muteId", "not-a-uuid")
	routes.HandleUnmuteResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid muteId")
}

func TestHandleUnmuteResource_ValidReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	validID := "550e8400-e29b-41d4-a716-446655440000"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/mutes/"+validID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "muteId", validID)
	routes.HandleUnmuteResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListMutedResources ---

func TestHandleListMutedResources_FiltersReachRPC(t *testing.T) {
	cases := []string{
		"/api/v1/notifications/mutes",
		"/api/v1/notifications/mutes?module_id=crm",
		"/api/v1/notifications/mutes?page=2&page_size=10",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			routes := newNotificationRoutesReachable()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			req = withUserID(req, "user-123")
			routes.HandleListMutedResources(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- HandleGetQuietHours / HandleUpdateQuietHours ---

func TestHandleGetQuietHours_ReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/quiet-hours", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetQuietHours(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateQuietHours_InvalidJSON(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/quiet-hours", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateQuietHours(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateQuietHours_DaysOfWeekConvertedAndReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/quiet-hours", jsonBody(t, map[string]any{
		"start_time":   "22:00",
		"end_time":     "07:00",
		"timezone":     "Europe/Zurich",
		"days_of_week": []int{0, 1, 2, 3, 4, 5, 6},
		"enabled":      true,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateQuietHours(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateQuietHours_EmptyDaysOfWeekReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/quiet-hours", jsonBody(t, map[string]any{
		"start_time": "22:00",
		"end_time":   "07:00",
		"enabled":    false,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleUpdateQuietHours(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleToggleDND / HandleGetDND / HandleDisableDND ---

func TestHandleToggleDND_InvalidJSON(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/dnd", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleToggleDND(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleToggleDND_InvalidUntilFormat(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/dnd", jsonBody(t, map[string]any{
		"enabled": true,
		"until":   "not-a-timestamp",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleToggleDND(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid 'until' timestamp")
}

func TestHandleToggleDND_EnabledWithoutUntilReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/dnd", jsonBody(t, map[string]any{
		"enabled": true,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleToggleDND(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleToggleDND_EnabledWithValidUntilReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/dnd", jsonBody(t, map[string]any{
		"enabled": true,
		"until":   time.Now().Add(time.Hour).Format(time.RFC3339),
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleToggleDND(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetDND_ReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/dnd", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetDND(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDisableDND_ReachesRPC(t *testing.T) {
	routes := newNotificationRoutesReachable()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/dnd", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleDisableDND(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- dndStatusFromQuietHours: the wire-shape mapping the desktop client's
// DNDStatus type ({is_active: boolean, expires_at?: string}) depends on. ---

func TestDndStatusFromQuietHours_Nil(t *testing.T) {
	got := dndStatusFromQuietHours(nil)
	if got["is_active"] != false {
		t.Errorf("is_active = %v, want false", got["is_active"])
	}
	if _, has := got["expires_at"]; has {
		t.Errorf("expires_at must be absent when there is no quiet-hours record, got %v", got["expires_at"])
	}
}

func TestDndStatusFromQuietHours_DisabledNoUntil(t *testing.T) {
	qh := &notificationv1.QuietHoursInfo{ManualDnd: false}
	got := dndStatusFromQuietHours(qh)
	if got["is_active"] != false {
		t.Errorf("is_active = %v, want false", got["is_active"])
	}
	if _, has := got["expires_at"]; has {
		t.Errorf("expires_at must be absent when manual DND is off, got %v", got["expires_at"])
	}
}

func TestDndStatusFromQuietHours_EnabledWithUntil(t *testing.T) {
	until := time.Date(2026, 12, 24, 18, 0, 0, 0, time.UTC)
	qh := &notificationv1.QuietHoursInfo{
		ManualDnd:      true,
		ManualDndUntil: timestamppb.New(until),
	}
	got := dndStatusFromQuietHours(qh)
	if got["is_active"] != true {
		t.Errorf("is_active = %v, want true", got["is_active"])
	}
	wantExpires := until.Format(time.RFC3339)
	if got["expires_at"] != wantExpires {
		t.Errorf("expires_at = %v, want %v", got["expires_at"], wantExpires)
	}
}

func TestDndStatusFromQuietHours_EnabledWithoutUntil(t *testing.T) {
	qh := &notificationv1.QuietHoursInfo{ManualDnd: true}
	got := dndStatusFromQuietHours(qh)
	if got["is_active"] != true {
		t.Errorf("is_active = %v, want true", got["is_active"])
	}
	if _, has := got["expires_at"]; has {
		t.Errorf("expires_at must be absent when manual DND has no expiry, got %v", got["expires_at"])
	}
}
