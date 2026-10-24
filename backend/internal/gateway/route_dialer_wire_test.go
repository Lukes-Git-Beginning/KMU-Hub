package gateway

// Dialer REST bodies as the desktop sends them. A loopback gRPC server plays the dialer service
// and records what the gateway forwards.

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

	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

type dialerWireStub struct {
	dialerv1.UnimplementedDialerServiceServer
	loggedOutcome *dialerv1.LogCallOutcomeRequest
}

func (s *dialerWireStub) LogCallOutcome(_ context.Context, req *dialerv1.LogCallOutcomeRequest) (*dialerv1.DialerCallSession, error) {
	s.loggedOutcome = req
	return &dialerv1.DialerCallSession{Id: req.GetDialerCallSessionId()}, nil
}

func newDialerWireRoutes(t *testing.T) (*DialerRoutes, *dialerWireStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &dialerWireStub{}
	srv := grpc.NewServer()
	dialerv1.RegisterDialerServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("dialer", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewDialerRoutes(reg), stub
}

func logOutcomeWith(t *testing.T, routes *DialerRoutes, callbackAt string) *httptest.ResponseRecorder {
	t.Helper()
	const sessionID = "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dialer/calls/"+sessionID+"/outcome", jsonBody(t, map[string]any{
		"outcome_id":  "660e8400-e29b-41d4-a716-446655440000",
		"callback_at": callbackAt,
	}))
	req = withChiURLParam(withAuth(req, testHRUserID, testTenantID), "id", sessionID)
	rec := httptest.NewRecorder()
	routes.HandleLogCallOutcome(rec, req)
	return rec
}

func TestDialerWire_CallbackAt(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want time.Time
	}{
		// WrapUpPanel sends an <input type="datetime-local"> value: wall clock, no seconds, no zone.
		// 2026-10-26 is CET (UTC+1), the clocks went back on 25.10.
		{"datetime-local", "2026-10-26T10:00", time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)},
		{"datetime-local with seconds", "2026-10-26T10:00:30", time.Date(2026, 10, 26, 9, 0, 30, 0, time.UTC)},
		{"RFC3339 with offset", "2026-10-26T10:00:00+02:00", time.Date(2026, 10, 26, 8, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			routes, stub := newDialerWireRoutes(t)
			rec := logOutcomeWith(t, routes, c.in)
			assertStatus(t, rec, http.StatusOK)
			require.NotNil(t, stub.loggedOutcome)
			assert.True(t, c.want.Equal(stub.loggedOutcome.GetCallbackAt().AsTime()),
				"got %s, want %s", stub.loggedOutcome.GetCallbackAt().AsTime(), c.want)
		})
	}
}

func TestDialerWire_CallbackAtGarbageIs400(t *testing.T) {
	routes, stub := newDialerWireRoutes(t)
	rec := logOutcomeWith(t, routes, "next tuesday")
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid callback_at")
	assert.Nil(t, stub.loggedOutcome, "a rejected body must not reach the dialer service")
}
