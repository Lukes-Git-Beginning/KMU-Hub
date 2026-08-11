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
// HandleGetMeeting
// ============================================================================

func TestHandleGetMeeting_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetMeeting_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetMeeting(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetMeeting_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateMeeting
// ============================================================================

func TestHandleUpdateMeeting_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+id, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateMeeting_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateMeeting(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateMeeting_InvalidJSON(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+id, invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateMeeting(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleUpdateMeeting_InvalidScheduledStartFormat covers the manual
// parseTimestamp check that runs after decodeAndValidate: ScheduledStart has
// no validate tag of its own (it is optional), so an unparseable value is
// only caught by this handler-local branch, not by the validation layer.
func TestHandleUpdateMeeting_InvalidScheduledStartFormat(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+id,
		jsonBody(t, map[string]interface{}{"scheduled_start": "gestern"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateMeeting(rec, req)
	assertErrorContains(t, rec, "invalid scheduled_start format")
}

func TestHandleUpdateMeeting_InvalidScheduledEndFormat(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+id,
		jsonBody(t, map[string]interface{}{"scheduled_end": "morgen"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateMeeting(rec, req)
	assertErrorContains(t, rec, "invalid scheduled_end format")
}

func TestHandleUpdateMeeting_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+id,
		jsonBody(t, map[string]interface{}{"title": "Renamed"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteMeeting
// ============================================================================

func TestHandleDeleteMeeting_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteMeeting_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteMeeting(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteMeeting_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListMeetings
// ============================================================================

func TestHandleListMeetings_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListMeetings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListMeetings_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings?status=in_progress&start_after=2026-09-01T00:00:00Z", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListMeetings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListMeetings_UnknownStatusIgnored documents that an unrecognised
// status filter value is silently dropped (StatusFilter stays nil) instead of
// rejected -- the statusMap lookup in HandleListMeetings has no else branch.
// The request still reaches the RPC layer rather than failing validation.
func TestHandleListMeetings_UnknownStatusIgnored(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings?status=not_a_real_status", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListMeetings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestProtoListMeetings_WireShape locks the bare-array shape HandleListMeetings
// promises the frontend (useMeetings maps the response directly): an empty
// result must serialise as [], never null, because response.ProtoList always
// builds a non-nil zero-length slice before encoding.
func TestProtoListMeetings_WireShape(t *testing.T) {
	rec := httptest.NewRecorder()
	response.ProtoList(rec, http.StatusOK, []*videov1.Meeting{})
	if body := rec.Body.String(); body != "[]\n" {
		t.Errorf("empty meeting list serialised as %q, want \"[]\\n\"", body)
	}
	var sink []json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &sink); err != nil {
		t.Fatalf("empty response is not valid JSON array: %v", err)
	}
}

// ============================================================================
// HandleStartMeeting
//
// The meeting-status transition itself (e.g. rejecting a second start on an
// already in-progress meeting) is enforced by the Video service, not the
// gateway -- there is no bufconn stub for VideoServiceClient in this package
// (same constraint documented for the recording handlers in
// route_video_recording_test.go), so the closest available error path here
// is the RPC-layer failure once the request has cleared local validation.
// ============================================================================

func TestHandleStartMeeting_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/start", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleStartMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleStartMeeting_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/start", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleStartMeeting(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleStartMeeting_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/start", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleStartMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleJoinMeeting
// ============================================================================

func TestHandleJoinMeeting_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleJoinMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleJoinMeeting_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleJoinMeeting(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleJoinMeeting_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleJoinMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleEndMeeting
// ============================================================================

func TestHandleEndMeeting_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/end", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleEndMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleEndMeeting_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/end", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleEndMeeting(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleEndMeeting_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/end", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleEndMeeting(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSetMeetingLock
// ============================================================================

func TestHandleSetMeetingLock_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/lock",
		jsonBody(t, map[string]interface{}{"locked": true}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleSetMeetingLock(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetMeetingLock_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/lock",
		jsonBody(t, map[string]interface{}{"locked": true}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleSetMeetingLock(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// TestHandleSetMeetingLock_InvalidLockedType covers the closest thing to an
// "invalid lock state" this request type can produce: Locked is a bare bool
// with no validate tag (false is a legitimate unlock request), so only a
// non-bool JSON value is rejectable.
func TestHandleSetMeetingLock_InvalidLockedType(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/lock",
		jsonBody(t, map[string]interface{}{"locked": "yes"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleSetMeetingLock(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetMeetingLock_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/lock",
		jsonBody(t, map[string]interface{}{"locked": false}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleSetMeetingLock(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleMuteAllMeetingParticipants
// ============================================================================

func TestHandleMuteAllMeetingParticipants_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/moderation/mute-all", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleMuteAllMeetingParticipants(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMuteAllMeetingParticipants_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/moderation/mute-all", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleMuteAllMeetingParticipants(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleMuteAllMeetingParticipants_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/moderation/mute-all", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleMuteAllMeetingParticipants(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleRemoveMeetingParticipant
// ============================================================================

func TestHandleRemoveMeetingParticipant_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/moderation/kick",
		jsonBody(t, map[string]interface{}{"target_user_id": uuid.New().String()}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleRemoveMeetingParticipant(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRemoveMeetingParticipant_InvalidIDUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/moderation/kick",
		jsonBody(t, map[string]interface{}{"target_user_id": uuid.New().String()}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleRemoveMeetingParticipant(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleRemoveMeetingParticipant_MissingTargetUserID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/moderation/kick",
		jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleRemoveMeetingParticipant(rec, req)
	assertValidationError(t, rec, "target_user_id")
}

func TestHandleRemoveMeetingParticipant_InvalidTargetUserID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/moderation/kick",
		jsonBody(t, map[string]interface{}{"target_user_id": "not-a-uuid"}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleRemoveMeetingParticipant(rec, req)
	assertValidationError(t, rec, "target_user_id")
}

func TestHandleRemoveMeetingParticipant_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+id+"/moderation/kick",
		jsonBody(t, map[string]interface{}{"target_user_id": uuid.New().String()}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", id)
	routes.HandleRemoveMeetingParticipant(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
