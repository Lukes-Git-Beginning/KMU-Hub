package gateway

// Partial failures of the calendar, video and notification routes as the desktop reads them:
// the booking-page list keeps `pages` when empty, the editor can remove a logo, the meeting
// list honours limit/offset, meeting notes keep `content`, mutes carry resource_type/muted_at,
// and the calendar preference key matches the spec. A loopback gRPC server plays the services.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

type kommunikationStub struct {
	calv1.UnimplementedCalendarServiceServer
	videov1.UnimplementedVideoServiceServer
	notificationv1.UnimplementedNotificationServiceServer

	gotBookingUpdate *calv1.UpdateBookingPageRequest
	gotPrefs         *calv1.UpdateCalendarPreferencesRequest
	gotMeetings      *videov1.ListMeetingsRequest
	gotMute          *notificationv1.MuteResourceRequest
}

var kommunikationTime = timestamppb.New(time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC))

func (s *kommunikationStub) ListBookingPages(context.Context, *calv1.ListBookingPagesRequest) (*calv1.ListBookingPagesResponse, error) {
	return &calv1.ListBookingPagesResponse{}, nil
}

func (s *kommunikationStub) UpdateBookingPage(_ context.Context, req *calv1.UpdateBookingPageRequest) (*calv1.UpdateBookingPageResponse, error) {
	s.gotBookingUpdate = req
	return &calv1.UpdateBookingPageResponse{Page: &calv1.BookingPageProto{Id: req.GetId()}}, nil
}

func (s *kommunikationStub) UpdateCalendarPreferences(_ context.Context, req *calv1.UpdateCalendarPreferencesRequest) (*calv1.UpdateCalendarPreferencesResponse, error) {
	s.gotPrefs = req
	return &calv1.UpdateCalendarPreferencesResponse{}, nil
}

func (s *kommunikationStub) ListMeetings(_ context.Context, req *videov1.ListMeetingsRequest) (*videov1.ListMeetingsResponse, error) {
	s.gotMeetings = req
	return &videov1.ListMeetingsResponse{}, nil
}

func (s *kommunikationStub) GetMeetingNotes(_ context.Context, req *videov1.GetMeetingNotesRequest) (*videov1.MeetingNotes, error) {
	return &videov1.MeetingNotes{MeetingId: req.GetMeetingId(), AuthorId: req.GetUserId()}, nil
}

func (s *kommunikationStub) GetPreviousMeetingNotes(_ context.Context, req *videov1.GetPreviousMeetingNotesRequest) (*videov1.MeetingNotes, error) {
	return &videov1.MeetingNotes{MeetingId: req.GetCurrentMeetingId(), AuthorId: req.GetUserId()}, nil
}

func (s *kommunikationStub) MuteResource(_ context.Context, req *notificationv1.MuteResourceRequest) (*notificationv1.MuteResourceResponse, error) {
	s.gotMute = req
	return &notificationv1.MuteResourceResponse{Mute: &notificationv1.MuteInfo{
		Id: "m1", UserId: req.GetUserId(), ModuleId: req.GetModuleId(), ResourceId: req.GetResourceId(), CreatedAt: kommunikationTime,
	}}, nil
}

func (s *kommunikationStub) ListMutedResources(context.Context, *notificationv1.ListMutedResourcesRequest) (*notificationv1.ListMutedResourcesResponse, error) {
	return &notificationv1.ListMutedResourcesResponse{
		Mutes: []*notificationv1.MuteInfo{{Id: "m1", ModuleId: "channel", ResourceId: "c1", CreatedAt: kommunikationTime}},
		Total: 1,
	}, nil
}

func newKommunikationRegistry(t *testing.T) (*ServiceRegistry, *kommunikationStub) {
	t.Helper()
	stub := &kommunikationStub{}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	calv1.RegisterCalendarServiceServer(srv, stub)
	videov1.RegisterVideoServiceServer(srv, stub)
	notificationv1.RegisterNotificationServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("work", lis.Addr().String())
	reg.Register("notification", lis.Addr().String())
	t.Cleanup(reg.Close)
	return reg, stub
}

func kommunikationRecord(t *testing.T, h http.HandlerFunc, method, target string, body any, want int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if body != nil {
		req = httptest.NewRequest(method, target, jsonBody(t, body))
	}
	req = withAuth(req, testHRUserID, testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	rec := httptest.NewRecorder()
	h(rec, req)
	assertStatus(t, rec, want)
	return rec
}

func kommunikationCall(t *testing.T, h http.HandlerFunc, method, target string, body any, want int) map[string]any {
	t.Helper()
	return decodeBody(t, kommunikationRecord(t, h, method, target, body, want))
}

func TestKommunikationWire_EmptyBookingPageListKeepsPagesArray(t *testing.T) {
	reg, _ := newKommunikationRegistry(t)
	br := NewBookingRoutes(reg, nil)

	got := kommunikationCall(t, br.HandleListBookingPages, http.MethodGet, "/x", nil, http.StatusOK)
	assert.Equal(t, []any{}, got["pages"])
}

func TestKommunikationWire_BookingPageLogoCanBeRemoved(t *testing.T) {
	reg, stub := newKommunikationRegistry(t)
	br := NewBookingRoutes(reg, nil)

	for name, body := range map[string]map[string]any{
		"null":         {"logo_url": nil},
		"empty string": {"logo_url": ""},
	} {
		stub.gotBookingUpdate = nil
		kommunikationCall(t, br.HandleUpdateBookingPage, http.MethodPut, "/x", body, http.StatusOK)
		require.NotNil(t, stub.gotBookingUpdate, name)
		require.NotNil(t, stub.gotBookingUpdate.LogoUrl, "%s must reach the RPC as a clear", name)
		assert.Equal(t, "", stub.gotBookingUpdate.GetLogoUrl(), name)
	}

	stub.gotBookingUpdate = nil
	kommunikationCall(t, br.HandleUpdateBookingPage, http.MethodPut, "/x", map[string]any{"company_name": "Acme"}, http.StatusOK)
	assert.Nil(t, stub.gotBookingUpdate.LogoUrl, "an omitted logo_url must leave the logo alone")

	stub.gotBookingUpdate = nil
	kommunikationCall(t, br.HandleUpdateBookingPage, http.MethodPut, "/x", map[string]any{"logo_url": "https://x.test/l.png"}, http.StatusOK)
	assert.Equal(t, "https://x.test/l.png", stub.gotBookingUpdate.GetLogoUrl())
}

func TestKommunikationWire_MeetingListHonoursLimitAndOffset(t *testing.T) {
	reg, stub := newKommunikationRegistry(t)
	vr := NewVideoRoutes(reg, "k", "s")

	kommunikationRecord(t, vr.HandleListMeetings, http.MethodGet, "/x", nil, http.StatusOK)
	assert.EqualValues(t, 100, stub.gotMeetings.GetPageSize(), "no filter asks for the whole list, not 20")
	assert.EqualValues(t, 1, stub.gotMeetings.GetPage())

	kommunikationRecord(t, vr.HandleListMeetings, http.MethodGet, "/x?limit=25&offset=50", nil, http.StatusOK)
	assert.EqualValues(t, 25, stub.gotMeetings.GetPageSize())
	assert.EqualValues(t, 3, stub.gotMeetings.GetPage())

	kommunikationRecord(t, vr.HandleListMeetings, http.MethodGet, "/x?limit=25&offset=10", nil, http.StatusBadRequest)
}

func TestKommunikationWire_MeetingWithoutNotesStillCarriesContent(t *testing.T) {
	reg, _ := newKommunikationRegistry(t)
	vr := NewVideoRoutes(reg, "k", "s")

	for name, h := range map[string]http.HandlerFunc{"notes": vr.HandleGetMeetingNotes, "previous": vr.HandleGetPreviousMeetingNotes} {
		got := kommunikationCall(t, h, http.MethodGet, "/x", nil, http.StatusOK)
		content, present := got["content"]
		assert.True(t, present, "%s: content must be a member", name)
		assert.Equal(t, "", content, name)
	}
}

func TestKommunikationWire_MutesCarryResourceTypeAndMutedAt(t *testing.T) {
	reg, stub := newKommunikationRegistry(t)
	nr := NewNotificationRoutes(reg)

	list := kommunikationCall(t, nr.HandleListMutedResources, http.MethodGet, "/x", nil, http.StatusOK)
	mutes, ok := list["mutes"].([]any)
	require.True(t, ok)
	require.Len(t, mutes, 1)
	m := mutes[0].(map[string]any)
	assert.Equal(t, "channel", m["resource_type"])
	assert.Equal(t, "c1", m["resource_id"])
	assert.Equal(t, "2026-10-24T09:30:00Z", m["muted_at"])
	assert.EqualValues(t, 1, list["total"])

	// The desktop posts resource_type, not module_id.
	created := kommunikationCall(t, nr.HandleMuteResource, http.MethodPost, "/x",
		map[string]any{"resource_type": "thread", "resource_id": "t9"}, http.StatusCreated)
	assert.Equal(t, "thread", stub.gotMute.GetModuleId())
	mute := created["mute"].(map[string]any)
	assert.Equal(t, "thread", mute["resource_type"])
	assert.Equal(t, "2026-10-24T09:30:00Z", mute["muted_at"])

	kommunikationCall(t, nr.HandleMuteResource, http.MethodPost, "/x",
		map[string]any{"module_id": "crm", "resource_id": "t9"}, http.StatusCreated)
	assert.Equal(t, "crm", stub.gotMute.GetModuleId(), "the earlier module_id spelling still works")

	kommunikationCall(t, nr.HandleMuteResource, http.MethodPost, "/x",
		map[string]any{"resource_id": "t9"}, http.StatusBadRequest)
}

func TestKommunikationWire_CalendarPreferenceKeyMatchesSpec(t *testing.T) {
	reg, stub := newKommunikationRegistry(t)
	cr := NewCalendarRoutes(reg)

	kommunikationCall(t, cr.HandleUpdateCalendarPreferences, http.MethodPut, "/x",
		map[string]any{"default_allday_reminder_minutes": 45}, http.StatusOK)
	require.NotNil(t, stub.gotPrefs.DefaultAlldayReminderMinutes)
	assert.EqualValues(t, 45, stub.gotPrefs.GetDefaultAlldayReminderMinutes())

	stub.gotPrefs = nil
	kommunikationCall(t, cr.HandleUpdateCalendarPreferences, http.MethodPut, "/x",
		map[string]any{"default_all_day_reminder_minutes": 60}, http.StatusOK)
	assert.EqualValues(t, 60, stub.gotPrefs.GetDefaultAlldayReminderMinutes(), "the earlier spelling still works")
}
