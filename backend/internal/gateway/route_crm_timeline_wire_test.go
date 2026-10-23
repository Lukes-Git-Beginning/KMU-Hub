package gateway

// Timeline of a contact against the desktop client (api/hooks/useTimeline.ts):
// the path /api/v1/crm/contacts/{id}/timeline, `offset` as a row offset, flat
// metadata and an events array that is never null. A real gRPC server on
// loopback stands in for the CRM service.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

type timelineStub struct {
	crmv1.UnimplementedCRMServiceServer
	got    *crmv1.GetContactTimelineRequest
	events []*crmv1.TimelineEvent
}

func (s *timelineStub) GetContactTimeline(_ context.Context, r *crmv1.GetContactTimelineRequest) (*crmv1.GetContactTimelineResponse, error) {
	s.got = r
	return &crmv1.GetContactTimelineResponse{Events: s.events, Total: int32(len(s.events))}, nil
}

func newTimelineRouter(t *testing.T, stub *timelineStub) http.Handler {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	crmv1.RegisterCRMServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("crm", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	NewCRMRoutes(reg, NewCRMExtRoutes(reg)).RegisterRoutes(r, guardTestAuth)
	return r
}

func getTimeline(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "contacts:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestContactTimeline_DesktopPathOffsetAndFlatMetadata(t *testing.T) {
	desc := "Rückruf vereinbart"
	stub := &timelineStub{events: []*crmv1.TimelineEvent{{
		Id: "e-1", EventType: "activity", OccurredAt: "2026-10-01T09:00:00Z", Title: "Anruf", Description: &desc,
		Metadata: &crmv1.TimelineEventMetadata{Data: map[string]string{"activity_type": `"call"`, "completed": `true`, "raw": `not json`}},
	}}}
	h := newTimelineRouter(t, stub)
	id := uuid.New().String()

	rec := getTimeline(t, h, "/api/v1/crm/contacts/"+id+"/timeline?offset=40&limit=20")
	assertStatus(t, rec, http.StatusOK)

	assert.Equal(t, id, stub.got.ContactId)
	assert.EqualValues(t, 3, stub.got.Page, "offset 40 at limit 20 is the third page")
	assert.EqualValues(t, 20, stub.got.PageSize)

	var body struct {
		Events []map[string]any `json:"events"`
		Total  int              `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	require.Len(t, body.Events, 1)
	assert.Equal(t, 1, body.Total)
	assert.Equal(t, map[string]any{"activity_type": "call", "completed": true, "raw": "not json"}, body.Events[0]["metadata"])
	assert.Equal(t, "Rückruf vereinbart", body.Events[0]["description"])
}

func TestContactTimeline_EmptyIsArrayAndOldPathStillServed(t *testing.T) {
	stub := &timelineStub{}
	h := newTimelineRouter(t, stub)
	id := uuid.New().String()

	for _, path := range []string{
		"/api/v1/contacts/" + id + "/timeline",
		"/api/v1/crm/contacts/" + id + "/timeline",
	} {
		rec := getTimeline(t, h, path)
		assertStatus(t, rec, http.StatusOK)
		assert.JSONEq(t, `{"events":[],"total":0}`, rec.Body.String(), path)
		assert.EqualValues(t, 1, stub.got.Page, "no offset means the first page")
		assert.EqualValues(t, 20, stub.got.PageSize)
	}
}

func TestContactTimeline_AliasKeepsThePermissionGuard(t *testing.T) {
	h := newTimelineRouter(t, &timelineStub{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/crm/contacts/"+uuid.New().String()+"/timeline", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "deals:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
}
