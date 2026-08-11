package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/server/response"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// ============================================================================
// HandleStartRecording
//
// startRecordingRequest carries no consent field -- consent is set afterwards
// via HandleSetRecordingConsent, once participants have seen the WS prompt
// (see the "Push recording.started WS event" comment in HandleStartRecording).
// The only local validation is on the two optional UUID fields, so that is
// the error path tested here instead of a "consent status" that does not
// exist on this request type.
// ============================================================================

func TestHandleStartRecording_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/start", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleStartRecording(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleStartRecording_InvalidJSON(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/start", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleStartRecording(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleStartRecording_InvalidCallID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/start",
		jsonBody(t, map[string]interface{}{"call_id": "not-a-uuid"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleStartRecording(rec, req)
	assertValidationError(t, rec, "call_id")
}

func TestHandleStartRecording_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/start",
		jsonBody(t, map[string]interface{}{"meeting_id": "not-a-uuid"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleStartRecording(rec, req)
	assertValidationError(t, rec, "meeting_id")
}

func TestHandleStartRecording_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/start",
		jsonBody(t, map[string]interface{}{"meeting_id": uuid.New().String()}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleStartRecording(rec, req)
	// registryWithService points at an unreachable dummy address: the request
	// clears validation and reaches the RPC layer, which then fails with
	// Unavailable -- same HTTP status as the pre-flight check above, but via
	// a different code path (proves the validation gate lets a valid body through).
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleStopRecording
// ============================================================================

func TestHandleStopRecording_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/stop", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleStopRecording(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleStopRecording_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/not-a-uuid/stop", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleStopRecording(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleStopRecording_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/stop", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleStopRecording(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSetRecordingConsent
// ============================================================================

func TestHandleSetRecordingConsent_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/consent",
		jsonBody(t, map[string]interface{}{"consented": true}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetRecordingConsent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetRecordingConsent_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/not-a-uuid/consent",
		jsonBody(t, map[string]interface{}{"consented": true}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSetRecordingConsent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// TestHandleSetRecordingConsent_InvalidConsentedType covers the closest thing
// to an "invalid consent status" this request type can produce: `consented`
// has no validate tag (a bare bool, zero value false is a legitimate answer),
// so the only rejectable case is a JSON value that is not a bool at all.
func TestHandleSetRecordingConsent_InvalidConsentedType(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/consent",
		jsonBody(t, map[string]interface{}{"consented": "yes"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetRecordingConsent(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetRecordingConsent_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/consent",
		jsonBody(t, map[string]interface{}{"consented": true}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetRecordingConsent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetRecordingConsent
// ============================================================================

func TestHandleGetRecordingConsent_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/consent", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingConsent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetRecordingConsent_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/not-a-uuid/consent", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetRecordingConsent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetRecordingConsent_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/consent", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingConsent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListRecordings
// ============================================================================

func TestHandleListRecordings_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListRecordings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListRecordings_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings?call_id="+uuid.New().String(), nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListRecordings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestProtoListRecordings_WireShape locks the shape the frontend reads for
// ListRecordingsResponse. The gateway's protojson marshaler runs with
// EmitUnpopulated=false (internal/server/response/response.go), so an empty
// repeated field is omitted from the body entirely rather than serialised as
// [] -- the same tolerant contract TestProtoMeetingOccurrences_WireShape
// already locks in for ListMeetingOccurrencesResponse.Items. The one shape
// that must never happen is an explicit "recordings":null, which would force
// every frontend consumer to null-check instead of just mapping over [].
func TestProtoListRecordings_WireShape(t *testing.T) {
	rec := httptest.NewRecorder()
	response.Proto(rec, http.StatusOK, &videov1.ListRecordingsResponse{
		Recordings: []*videov1.Recording{},
		Total:      0,
	})
	var sink map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &sink); err != nil {
		t.Fatalf("empty response is not valid JSON: %v", err)
	}
	if recordings, ok := sink["recordings"]; ok && recordings == nil {
		t.Errorf("recordings serialised as null, want [] or omitted; body: %s", rec.Body.String())
	}
}

// ============================================================================
// HandleDeleteRecording
// ============================================================================

func TestHandleDeleteRecording_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/video/recordings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteRecording(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteRecording_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/video/recordings/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteRecording(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteRecording_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/video/recordings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteRecording(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetRecordingConsents (plural -- consent snapshot + live rows, routed
// through the "work" registry connection via getVideoRecordingTagClient)
// ============================================================================

func TestHandleGetRecordingConsents_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/consents", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingConsents(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetRecordingConsents_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/not-a-uuid/consents", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetRecordingConsents(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetRecordingConsents_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/consents", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingConsents(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleTagRecordingWithConsents
// ============================================================================

func TestHandleTagRecordingWithConsents_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/tag-consents",
		jsonBody(t, map[string]interface{}{"snapshot": []map[string]string{{"user_id": "u1", "display_name": "Ada"}}}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleTagRecordingWithConsents(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleTagRecordingWithConsents_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/not-a-uuid/tag-consents",
		jsonBody(t, map[string]interface{}{"snapshot": []map[string]string{{"user_id": "u1", "display_name": "Ada"}}}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleTagRecordingWithConsents(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleTagRecordingWithConsents_EmptySnapshot(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/tag-consents",
		jsonBody(t, map[string]interface{}{"snapshot": []map[string]string{}}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleTagRecordingWithConsents(rec, req)
	assertValidationError(t, rec, "snapshot")
}

func TestHandleTagRecordingWithConsents_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/tag-consents",
		jsonBody(t, map[string]interface{}{"snapshot": []map[string]string{{"user_id": "u1", "display_name": "Ada"}}}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleTagRecordingWithConsents(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
