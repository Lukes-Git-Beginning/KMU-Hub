package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// ============================================================================
// HandleGetRecordingStatus
// ============================================================================

func TestHandleGetRecordingStatus_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/status", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetRecordingStatus_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/not-a-uuid/status", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetRecordingStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetRecordingStatus_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/status", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetRecordingDownloadURL
//
// ACL (participant-only) and presign expiry (1h, service.go GetRecordingDownloadURL)
// live in the service layer, not the handler -- the handler is pure
// parse/call/respond, so only the transport-level cases are testable here.
// ============================================================================

func TestHandleGetRecordingDownloadURL_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/download", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingDownloadURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetRecordingDownloadURL_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/not-a-uuid/download", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetRecordingDownloadURL(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetRecordingDownloadURL_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/recordings/"+id+"/download", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetRecordingDownloadURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateRecordingMetadata
//
// updateRecordingMetadataRequest fields are all optional pointers (patch
// semantics) -- there is no `validate:"required"` tag to trigger a 400 on a
// missing field, so InvalidJSON is the only local-validation case besides the
// UUID path param.
// ============================================================================

func TestHandleUpdateRecordingMetadata_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/video/recordings/"+id+"/metadata",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateRecordingMetadata(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateRecordingMetadata_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/video/recordings/not-a-uuid/metadata",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateRecordingMetadata(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateRecordingMetadata_InvalidJSON(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/video/recordings/"+id+"/metadata", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateRecordingMetadata(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateRecordingMetadata_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/video/recordings/"+id+"/metadata",
		jsonBody(t, map[string]interface{}{"status": "completed", "duration_seconds": 120}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateRecordingMetadata(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCleanupExpiredRecording
//
// Admin/cron endpoint (route_video.go:135, RequirePermission("recordings",
// "admin") applied at the router level, not inside the handler -- so it is
// not exercisable from a direct handler call in this package).
// ============================================================================

func TestHandleCleanupExpiredRecording_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/cleanup", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleCleanupExpiredRecording(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCleanupExpiredRecording_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/not-a-uuid/cleanup", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCleanupExpiredRecording(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCleanupExpiredRecording_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/recordings/"+id+"/cleanup", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleCleanupExpiredRecording(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListRecordingsByMeeting
// ============================================================================

func TestHandleListRecordingsByMeeting_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/meetings/"+meetingID+"/recordings", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListRecordingsByMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListRecordingsByMeeting_InvalidMeetingIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/meetings/not-a-uuid/recordings", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleListRecordingsByMeeting(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListRecordingsByMeeting_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/meetings/"+meetingID+"/recordings?page=2&page_size=10", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListRecordingsByMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
