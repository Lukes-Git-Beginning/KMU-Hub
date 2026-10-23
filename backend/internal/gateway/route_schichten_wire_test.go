package gateway

// Schichten responses as SchichtenPage reads them: a shift without assignments answers
// `{assignments:[]}`, a template at a full hour keeps `start_minute: 0`, and the week bounds
// `from`/`to` reach the service. A loopback gRPC server plays the schichten service.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
)

const testSchichtenShiftID = "77777777-7777-7777-7777-777777777777"

type schichtenWireStub struct {
	schichtenv1.UnimplementedSchichtenServiceServer
	listed *schichtenv1.ListShiftsRequest
}

func (s *schichtenWireStub) ListShifts(_ context.Context, r *schichtenv1.ListShiftsRequest) (*schichtenv1.ListShiftsResponse, error) {
	s.listed = r
	return &schichtenv1.ListShiftsResponse{}, nil
}

func (s *schichtenWireStub) ListAssignments(context.Context, *schichtenv1.ListAssignmentsRequest) (*schichtenv1.ListAssignmentsResponse, error) {
	return &schichtenv1.ListAssignmentsResponse{}, nil
}

func (s *schichtenWireStub) ListTemplates(context.Context, *schichtenv1.ListTemplatesRequest) (*schichtenv1.ListTemplatesResponse, error) {
	return &schichtenv1.ListTemplatesResponse{Templates: []*schichtenv1.ShiftTemplate{
		{Id: "t1", Name: "Frühschicht", DurationMinutes: 480},
	}, Total: 1}, nil
}

func (s *schichtenWireStub) ApplyTemplate(context.Context, *schichtenv1.ApplyTemplateRequest) (*schichtenv1.ApplyTemplateResponse, error) {
	return &schichtenv1.ApplyTemplateResponse{}, nil
}

func (s *schichtenWireStub) PublishShifts(context.Context, *schichtenv1.PublishShiftsRequest) (*schichtenv1.PublishShiftsResponse, error) {
	return &schichtenv1.PublishShiftsResponse{}, nil
}

func (s *schichtenWireStub) ListSwapRequests(context.Context, *schichtenv1.ListSwapRequestsRequest) (*schichtenv1.ListSwapRequestsResponse, error) {
	return &schichtenv1.ListSwapRequestsResponse{}, nil
}

func (s *schichtenWireStub) GetShiftStats(context.Context, *schichtenv1.GetShiftStatsRequest) (*schichtenv1.ShiftStatsResponse, error) {
	return &schichtenv1.ShiftStatsResponse{}, nil
}

func newSchichtenWireRoutes(t *testing.T) (*SchichtenRoutes, *schichtenWireStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &schichtenWireStub{}
	srv := grpc.NewServer()
	schichtenv1.RegisterSchichtenServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("schichten", lis.Addr().String())
	t.Cleanup(reg.Close)
	return newSchichtenRoutes(reg), stub
}

func schichtenRequest(t *testing.T, target string, idParam string) *http.Request {
	t.Helper()
	req := withAuth(httptest.NewRequest(http.MethodGet, target, nil), testHRUserID, testTenantID)
	if idParam != "" {
		req = withChiURLParam(req, "id", idParam)
	}
	return req
}

func TestSchichtenWire_EmptyListsAreArrays(t *testing.T) {
	routes, _ := newSchichtenWireRoutes(t)

	cases := []struct {
		name string
		key  string
		call func(http.ResponseWriter)
	}{
		{"assignments of a shift without any", "assignments", func(w http.ResponseWriter) {
			routes.HandleListAssignments(w, schichtenRequest(t, "/x", testSchichtenShiftID))
		}},
		{"shifts", "shifts", func(w http.ResponseWriter) {
			routes.HandleListShifts(w, schichtenRequest(t, "/x", ""))
		}},
		{"swap requests", "swap_requests", func(w http.ResponseWriter) {
			routes.HandleListSwapRequests(w, schichtenRequest(t, "/x", ""))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.call(rec)
			assertStatus(t, rec, http.StatusOK)
			list, ok := decodeBody(t, rec)[c.key].([]any)
			require.True(t, ok, rec.Body.String())
			assert.Empty(t, list)
		})
	}
}

func TestSchichtenWire_ZeroValuesStayInTheBody(t *testing.T) {
	routes, _ := newSchichtenWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListTemplates(rec, schichtenRequest(t, "/x", ""))
	assertStatus(t, rec, http.StatusOK)
	tpl := decodeBody(t, rec)["templates"].([]any)[0].(map[string]any)
	for _, k := range []string{"start_hour", "start_minute", "day_of_week"} {
		assert.EqualValues(t, 0, tpl[k], "%s must be present: %s", k, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	routes.HandleApplyTemplate(rec, applyTemplateRequestFor(t))
	assertStatus(t, rec, http.StatusCreated)
	body := decodeBody(t, rec)
	assert.EqualValues(t, 0, body["created_count"], rec.Body.String())
	assert.Equal(t, []any{}, body["shifts"], rec.Body.String())

	rec = httptest.NewRecorder()
	req := withAuth(httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, map[string]any{
		"from": "2026-10-19T00:00:00Z", "to": "2026-10-25T23:59:59Z",
	})), testHRUserID, testTenantID)
	routes.HandlePublishShifts(rec, req)
	assertStatus(t, rec, http.StatusOK)
	assert.EqualValues(t, 0, decodeBody(t, rec)["published_count"], rec.Body.String())
}

func applyTemplateRequestFor(t *testing.T) *http.Request {
	t.Helper()
	req := withAuth(httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, map[string]any{
		"range_start": "2026-10-19T00:00:00Z", "range_end": "2026-10-25T23:59:59Z",
	})), testHRUserID, testTenantID)
	return withChiURLParam(req, "id", testSchichtenShiftID)
}

func TestSchichtenWire_ListShiftsPassesTheWeekBounds(t *testing.T) {
	routes, stub := newSchichtenWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListShifts(rec, schichtenRequest(t, "/x?from=2026-10-19T00:00:00Z&to=2026-10-25T23:59:59Z", ""))
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.listed)
	assert.Equal(t, "2026-10-19T00:00:00Z", stub.listed.GetFrom().AsTime().Format("2006-01-02T15:04:05Z07:00"))
	assert.Equal(t, "2026-10-25T23:59:59Z", stub.listed.GetTo().AsTime().Format("2006-01-02T15:04:05Z07:00"))

	rec = httptest.NewRecorder()
	routes.HandleListShifts(rec, schichtenRequest(t, "/x", ""))
	assertStatus(t, rec, http.StatusOK)
	assert.Nil(t, stub.listed.From, "no bound unless the client sent one")
	assert.Nil(t, stub.listed.To)

	rec = httptest.NewRecorder()
	routes.HandleListShifts(rec, schichtenRequest(t, "/x?from=yesterday", ""))
	assertStatus(t, rec, http.StatusBadRequest)
}
