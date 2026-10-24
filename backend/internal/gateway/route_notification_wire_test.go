package gateway

// Wire shape of the notification settings routes against the desktop types
// (api/notification-client.ts QuietHours, NotificationCenter.tsx preferences):
// quiet hours speak `days`/`is_active` (0=Sunday) next to the service's
// `days_of_week`/`enabled` (ISO), false booleans are written out, and a PUT
// /preferences forwards only the fields the client sent. A real gRPC server on
// loopback stands in for the notification service.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

type notificationWireStub struct {
	notificationv1.UnimplementedNotificationServiceServer
	quietHours *notificationv1.QuietHoursInfo
	updateQH   *notificationv1.UpdateQuietHoursRequest
	updatePref *notificationv1.UpdateNotificationPreferenceRequest
}

func (s *notificationWireStub) GetQuietHours(context.Context, *notificationv1.GetQuietHoursRequest) (*notificationv1.GetQuietHoursResponse, error) {
	if s.quietHours == nil {
		return &notificationv1.GetQuietHoursResponse{QuietHours: &notificationv1.QuietHoursInfo{}}, nil
	}
	return &notificationv1.GetQuietHoursResponse{QuietHours: s.quietHours}, nil
}

func (s *notificationWireStub) UpdateQuietHours(_ context.Context, r *notificationv1.UpdateQuietHoursRequest) (*notificationv1.UpdateQuietHoursResponse, error) {
	s.updateQH = r
	return &notificationv1.UpdateQuietHoursResponse{QuietHours: &notificationv1.QuietHoursInfo{
		StartTime: r.StartTime, EndTime: r.EndTime, Timezone: r.Timezone,
		DaysOfWeek: r.DaysOfWeek, Enabled: r.Enabled,
	}}, nil
}

func (s *notificationWireStub) GetNotificationPreferences(context.Context, *notificationv1.GetNotificationPreferencesRequest) (*notificationv1.GetNotificationPreferencesResponse, error) {
	key := "chat.mention"
	return &notificationv1.GetNotificationPreferencesResponse{Preferences: []*notificationv1.NotificationPreferenceInfo{
		{Id: "p-1", EventTypeKey: &key}, // every delivery flag false
	}}, nil
}

func (s *notificationWireStub) UpdateNotificationPreference(_ context.Context, r *notificationv1.UpdateNotificationPreferenceRequest) (*notificationv1.UpdateNotificationPreferenceResponse, error) {
	s.updatePref = r
	return &notificationv1.UpdateNotificationPreferenceResponse{Preference: &notificationv1.NotificationPreferenceInfo{Id: "p-1"}}, nil
}

func (s *notificationWireStub) ListEventTypes(context.Context, *notificationv1.ListEventTypesRequest) (*notificationv1.ListEventTypesResponse, error) {
	return &notificationv1.ListEventTypesResponse{EventTypes: []*notificationv1.EventTypeInfo{
		{Id: "e-1", EventKey: "chat.mention", DisplayName: "Mention"}, // defaults all false
	}}, nil
}

func newNotificationWireRouter(t *testing.T) (http.Handler, *notificationWireStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &notificationWireStub{}
	srv := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("notification", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	NewNotificationRoutes(reg).RegisterRoutes(r, guardTestAuth)
	return r, stub
}

func doNotification(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, jsonBody(t, body))
	}
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withRoles(req, "admin")
	req = withPermissions(req, "notifications:read", "notifications:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestQuietHours_WeekdayMappingBothWays(t *testing.T) {
	// 0=Sunday (desktop) <-> 7=Sunday (service); 1..6 are identical.
	iso, ok := desktopDaysToISO([]int{0, 1, 5, 6})
	require.True(t, ok)
	assert.Equal(t, []int32{7, 1, 5, 6}, iso)
	assert.Equal(t, []int{0, 1, 5, 6}, isoDaysToDesktop(iso))

	_, ok = desktopDaysToISO([]int{7})
	assert.False(t, ok, "7 is not a desktop weekday")
	_, ok = desktopDaysToISO([]int{-1})
	assert.False(t, ok)
}

func TestQuietHours_PutStoresDesktopFields(t *testing.T) {
	h, stub := newNotificationWireRouter(t)

	rec := doNotification(t, h, http.MethodPut, "/api/v1/notifications/quiet-hours", map[string]any{
		"start_time": "22:00", "end_time": "07:00", "days": []int{0, 1, 2}, "is_active": true,
	})
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.updateQH)
	assert.True(t, stub.updateQH.Enabled, "is_active must reach the service as enabled")
	assert.Equal(t, []int32{7, 1, 2}, stub.updateQH.DaysOfWeek, "Sunday 0 becomes ISO 7")
	assert.Equal(t, "Europe/Berlin", stub.updateQH.Timezone, "the desktop form sends no timezone")

	qh := decodeObject(t, rec)["quiet_hours"].(map[string]any)
	assert.Equal(t, true, qh["is_active"])
	assert.Equal(t, []any{float64(0), float64(1), float64(2)}, qh["days"], "response speaks the desktop weekday mask")
	assert.Equal(t, []any{float64(7), float64(1), float64(2)}, qh["days_of_week"])
}

func TestQuietHours_PutKeepsServiceVocabulary(t *testing.T) {
	h, stub := newNotificationWireRouter(t)

	rec := doNotification(t, h, http.MethodPut, "/api/v1/notifications/quiet-hours", map[string]any{
		"start_time": "18:00", "end_time": "08:00", "timezone": "Europe/Vienna",
		"days_of_week": []int{1, 7}, "enabled": true,
	})
	assertStatus(t, rec, http.StatusOK)
	assert.True(t, stub.updateQH.Enabled)
	assert.Equal(t, []int32{1, 7}, stub.updateQH.DaysOfWeek)
	assert.Equal(t, "Europe/Vienna", stub.updateQH.Timezone)

	rec = doNotification(t, h, http.MethodPut, "/api/v1/notifications/quiet-hours", map[string]any{
		"start_time": "18:00", "end_time": "08:00",
	})
	assertStatus(t, rec, http.StatusOK)
	assert.False(t, stub.updateQH.Enabled, "no switch sent means off")
	assert.Equal(t, []int32{1, 2, 3, 4, 5, 6, 7}, stub.updateQH.DaysOfWeek, "no weekday sent means every day")

	rec = doNotification(t, h, http.MethodPut, "/api/v1/notifications/quiet-hours", map[string]any{
		"start_time": "18:00", "end_time": "08:00", "days": []int{9},
	})
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestQuietHours_PartialPutKeepsTheStoredSchedule(t *testing.T) {
	h, stub := newNotificationWireRouter(t)
	stub.quietHours = &notificationv1.QuietHoursInfo{
		StartTime: "20:00", EndTime: "06:00", Timezone: "Europe/Zurich",
		DaysOfWeek: []int32{7, 6}, Enabled: true,
	}

	// Only the times: weekdays, switch and timezone stay.
	rec := doNotification(t, h, http.MethodPut, "/api/v1/notifications/quiet-hours", map[string]any{
		"start_time": "21:00", "end_time": "05:00",
	})
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, "21:00", stub.updateQH.StartTime)
	assert.Equal(t, "05:00", stub.updateQH.EndTime)
	assert.Equal(t, "Europe/Zurich", stub.updateQH.Timezone)
	assert.Equal(t, []int32{7, 6}, stub.updateQH.DaysOfWeek)
	assert.True(t, stub.updateQH.Enabled)

	// Only the switch: times, weekdays and timezone stay.
	rec = doNotification(t, h, http.MethodPut, "/api/v1/notifications/quiet-hours", map[string]any{"is_active": false})
	assertStatus(t, rec, http.StatusOK)
	assert.False(t, stub.updateQH.Enabled)
	assert.Equal(t, "20:00", stub.updateQH.StartTime)
	assert.Equal(t, "06:00", stub.updateQH.EndTime)
	assert.Equal(t, "Europe/Zurich", stub.updateQH.Timezone)
	assert.Equal(t, []int32{7, 6}, stub.updateQH.DaysOfWeek)

	// Only the weekdays.
	rec = doNotification(t, h, http.MethodPut, "/api/v1/notifications/quiet-hours", map[string]any{"days": []int{1, 2}})
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, []int32{1, 2}, stub.updateQH.DaysOfWeek)
	assert.True(t, stub.updateQH.Enabled)
	assert.Equal(t, "20:00", stub.updateQH.StartTime)
}

func TestQuietHours_GetWithAndWithoutRecord(t *testing.T) {
	h, stub := newNotificationWireRouter(t)

	rec := doNotification(t, h, http.MethodGet, "/api/v1/notifications/quiet-hours", nil)
	assertStatus(t, rec, http.StatusOK)
	qh := decodeObject(t, rec)["quiet_hours"].(map[string]any)
	assert.Equal(t, "22:00", qh["start_time"], "no record: the form starts from the defaults")
	assert.Equal(t, "Europe/Berlin", qh["timezone"])
	assert.Equal(t, false, qh["is_active"], "no record: the schedule is off, not on")
	assert.Len(t, qh["days"], 7)

	stub.quietHours = &notificationv1.QuietHoursInfo{
		StartTime: "20:00", EndTime: "06:00", Timezone: "Europe/Berlin",
		DaysOfWeek: []int32{7, 6}, Enabled: true,
	}
	rec = doNotification(t, h, http.MethodGet, "/api/v1/notifications/quiet-hours", nil)
	assertStatus(t, rec, http.StatusOK)
	qh = decodeObject(t, rec)["quiet_hours"].(map[string]any)
	assert.Equal(t, true, qh["is_active"])
	assert.Equal(t, true, qh["enabled"])
	assert.Equal(t, []any{float64(0), float64(6)}, qh["days"], "ISO Sunday 7 reads back as 0")
}

func TestPreferences_FalseValuesAreWrittenOut(t *testing.T) {
	h, _ := newNotificationWireRouter(t)

	rec := doNotification(t, h, http.MethodGet, "/api/v1/notifications/preferences", nil)
	assertStatus(t, rec, http.StatusOK)
	prefs := decodeObject(t, rec)["preferences"].([]any)
	require.Len(t, prefs, 1)
	pref := prefs[0].(map[string]any)
	for _, k := range []string{"in_app", "desktop_push", "email", "sms"} {
		v, present := pref[k]
		assert.True(t, present, "%s must be present when false", k)
		assert.Equal(t, false, v)
	}
	assert.Equal(t, "", pref["sound"])

	rec = doNotification(t, h, http.MethodGet, "/api/v1/notifications/event-types", nil)
	assertStatus(t, rec, http.StatusOK)
	ets := decodeObject(t, rec)["event_types"].([]any)
	require.Len(t, ets, 1)
	et := ets[0].(map[string]any)
	for _, k := range []string{"default_in_app", "default_desktop_push"} {
		v, present := et[k]
		assert.True(t, present, "%s must be present when false", k)
		assert.Equal(t, false, v)
	}
}

func TestPreferences_PutForwardsOnlySentFields(t *testing.T) {
	h, stub := newNotificationWireRouter(t)

	rec := doNotification(t, h, http.MethodPut, "/api/v1/notifications/preferences", map[string]any{
		"event_type_key": "chat.mention", "in_app": false, "desktop_push": true,
	})
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.updatePref)
	require.NotNil(t, stub.updatePref.InApp)
	assert.False(t, *stub.updatePref.InApp, "an explicit false is a value, not 'unset'")
	require.NotNil(t, stub.updatePref.DesktopPush)
	assert.True(t, *stub.updatePref.DesktopPush)
	assert.Nil(t, stub.updatePref.Email, "unsent email must reach the service as unset")
	assert.Nil(t, stub.updatePref.Sms)
	assert.Nil(t, stub.updatePref.Sound)

	pref := decodeObject(t, rec)["preference"].(map[string]any)
	assert.Equal(t, false, pref["in_app"])
}
