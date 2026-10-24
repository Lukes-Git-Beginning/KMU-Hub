package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// The desktop books and cancels resources under /resources/bookings
// (useResources.ts); the bare /bookings paths predate it. Both must reach the
// same RPCs, and /resources/{id} must keep meaning "the resource".

type bookingStub struct {
	calv1.UnimplementedCalendarServiceServer
	booked          *calv1.BookResourceRequest
	cancelledID     string
	deletedResource string
}

func (s *bookingStub) BookResource(_ context.Context, r *calv1.BookResourceRequest) (*calv1.BookResourceResponse, error) {
	s.booked = r
	return &calv1.BookResourceResponse{Booking: &calv1.ResourceBookingProto{Id: "b-1", ResourceId: r.ResourceId}}, nil
}

func (s *bookingStub) CancelBooking(_ context.Context, r *calv1.CancelBookingRequest) (*calv1.CancelBookingResponse, error) {
	s.cancelledID = r.Id
	return &calv1.CancelBookingResponse{}, nil
}

func (s *bookingStub) DeleteResource(_ context.Context, r *calv1.DeleteResourceRequest) (*calv1.DeleteResourceResponse, error) {
	s.deletedResource = r.Id
	return &calv1.DeleteResourceResponse{}, nil
}

func newBookingRouter(t *testing.T) (chi.Router, *bookingStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &bookingStub{}
	srv := grpc.NewServer()
	calv1.RegisterCalendarServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("work", lis.Addr().String())
	r := chi.NewRouter()
	NewCalendarRoutes(reg).RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r, stub
}

func TestResourceBookingRoutes_BothPrefixesReachTheRPCs(t *testing.T) {
	for _, prefix := range []string{"/api/v1/calendar/resources/bookings", "/api/v1/calendar/bookings"} {
		t.Run(prefix, func(t *testing.T) {
			router, stub := newBookingRouter(t)
			resourceID, eventID, bookingID := uuid.NewString(), uuid.NewString(), uuid.NewString()
			start := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)

			req := httptest.NewRequest(http.MethodPost, prefix, jsonBody(t, map[string]string{
				"resource_id": resourceID,
				"event_id":    eventID,
				"start_time":  start.Format(time.RFC3339),
				"end_time":    start.Add(time.Hour).Format(time.RFC3339),
			}))
			req.Header.Set("Content-Type", "application/json")
			req = withPermissions(withAuth(req, uuid.NewString(), testTenantID), "resources:read")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusCreated)
			require.NotNil(t, stub.booked)
			assert.Equal(t, resourceID, stub.booked.ResourceId)
			assert.Equal(t, eventID, stub.booked.EventId)

			req = httptest.NewRequest(http.MethodDelete, prefix+"/"+bookingID, nil)
			req = withPermissions(withAuth(req, uuid.NewString(), testTenantID), "resources:read")
			rec = httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusOK)
			assert.Equal(t, bookingID, stub.cancelledID)
			assert.Empty(t, stub.deletedResource, "a booking cancel must not hit the resource delete")
		})
	}
}

func TestResourceBookingRoutes_ResourceDeleteStillDeletesTheResource(t *testing.T) {
	router, stub := newBookingRouter(t)
	resourceID := uuid.NewString()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/resources/"+resourceID, nil)
	req = withPermissions(withAuth(req, uuid.NewString(), testTenantID), "resources:delete")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, resourceID, stub.deletedResource)
	assert.Empty(t, stub.cancelledID, "a resource delete must not hit the booking cancel")
}

func TestResourceBookingRoutes_RequireResourcesRead(t *testing.T) {
	router, _ := newBookingRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/resources/bookings/"+uuid.NewString(), nil)
	req = withPermissions(withAuth(req, uuid.NewString(), testTenantID), "calendars:read")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}
