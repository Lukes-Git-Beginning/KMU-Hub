package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// route_video_breakout_cohost_test.go covers the breakout-room, co-host/
// moderation, and plain-call handlers of route_video.go — the second of
// three units splitting the file (see route_video_test.go for the first).
//
// None of these handlers extract tenant/user IDs via an explicit gateway
// check (no getTenantID/GetTenantID call in the handler bodies) — the caller
// identity travels to the service purely via the outbound gRPC interceptor,
// so there is no "missing tenant" 401 case to test here, unlike the HR
// handlers. Authorization (host/co-host only) is enforced entirely in
// internal/work/meeting/service.go and is already covered there
// (TestPromoteCoHost_NonOrganizerDenied, TestDemoteCoHost_NonOrganizerDenied,
// TestMuteMeetingParticipant_NonHostDenied, TestCreateBreakoutRooms_HostOnly,
// TestAssignBreakoutParticipant_NonHost) — a gateway test can only observe
// that a gRPC error is mapped to the right HTTP status, never body content,
// since no fake VideoServiceClient exists in this package.
//
// HandleJoinBreakoutRoom takes no room ID from the request at all: the
// service resolves the caller's assignment by (meetingID, callerID), so a
// participant cannot address a room from a foreign meeting even in
// principle (internal/work/meeting/service.go JoinBreakoutRoom, verified by
// reading the source — there is no cross-meeting room-ID parameter to
// misuse).

// ============================================================================
// Call handlers
// ============================================================================

func TestHandleJoinCall_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	callID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/calls/"+callID+"/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", callID)
	routes.HandleJoinCall(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleJoinCall_InvalidUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/calls/not-a-uuid/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleJoinCall(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleJoinCall_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	callID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/calls/"+callID+"/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", callID)
	routes.HandleJoinCall(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleEndCall_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	callID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/calls/"+callID+"/end", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", callID)
	routes.HandleEndCall(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleEndCall_InvalidUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/calls/not-a-uuid/end", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleEndCall(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleEndCall_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	callID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/video/calls/"+callID+"/end", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", callID)
	routes.HandleEndCall(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetCall_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	callID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/calls/"+callID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", callID)
	routes.HandleGetCall(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetCall_InvalidUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/calls/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCall(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetCall_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	callID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/calls/"+callID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", callID)
	routes.HandleGetCall(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListActiveCalls_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/calls", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListActiveCalls(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListActiveCalls_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/video/calls", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListActiveCalls(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Co-host / moderation handlers
// ============================================================================

func TestHandlePromoteCoHost_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/cohosts",
		strings.NewReader(`{"user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandlePromoteCoHost(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandlePromoteCoHost_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/cohosts",
		strings.NewReader(`{"user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandlePromoteCoHost(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlePromoteCoHost_MissingUserID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/cohosts",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandlePromoteCoHost(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestHandlePromoteCoHost_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/cohosts",
		strings.NewReader(`{"user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandlePromoteCoHost(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDemoteCoHost_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	targetUserID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/"+meetingID+"/cohosts/"+targetUserID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	req = withChiURLParam(req, "userId", targetUserID)
	routes.HandleDemoteCoHost(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDemoteCoHost_InvalidTargetUserID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/"+meetingID+"/cohosts/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	req = withChiURLParam(req, "userId", "not-a-uuid")
	routes.HandleDemoteCoHost(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDemoteCoHost_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	targetUserID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/"+meetingID+"/cohosts/"+targetUserID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	req = withChiURLParam(req, "userId", targetUserID)
	routes.HandleDemoteCoHost(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCoHosts_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/cohosts", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListCoHosts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCoHosts_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid/cohosts", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleListCoHosts(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListCoHosts_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/cohosts", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListCoHosts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMuteMeetingParticipant_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/moderation/mute",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleMuteMeetingParticipant(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMuteMeetingParticipant_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/moderation/mute",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleMuteMeetingParticipant(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleMuteMeetingParticipant_MissingTargetUserID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/moderation/mute",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleMuteMeetingParticipant(rec, req)
	assertValidationError(t, rec, "target_user_id")
}

func TestHandleMuteMeetingParticipant_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/moderation/mute",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleMuteMeetingParticipant(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Breakout room handlers
// ============================================================================

func TestHandleCreateBreakoutRooms_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms",
		strings.NewReader(`{"count":2}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCreateBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateBreakoutRooms_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/breakout-rooms",
		strings.NewReader(`{"count":2}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateBreakoutRooms_CountOutOfBounds(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms",
		strings.NewReader(`{"count":21}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCreateBreakoutRooms(rec, req)
	assertValidationError(t, rec, "count")
}

func TestHandleCreateBreakoutRooms_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms",
		strings.NewReader(`{"count":3,"labels":["A","B","C"]}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCreateBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBreakoutRooms_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/breakout-rooms", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleListBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBreakoutRooms_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid/breakout-rooms", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListBreakoutRooms_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/breakout-rooms", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleListBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAssignBreakoutParticipant_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/assign",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleAssignBreakoutParticipant(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAssignBreakoutParticipant_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/breakout-rooms/assign",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAssignBreakoutParticipant(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleAssignBreakoutParticipant_MissingTargetUserID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/assign",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleAssignBreakoutParticipant(rec, req)
	assertValidationError(t, rec, "target_user_id")
}

func TestHandleAssignBreakoutParticipant_InvalidBreakoutRoomID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/assign",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`","breakout_room_id":"not-a-uuid"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleAssignBreakoutParticipant(rec, req)
	assertValidationError(t, rec, "breakout_room_id")
}

func TestHandleAssignBreakoutParticipant_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/assign",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`","breakout_room_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleAssignBreakoutParticipant(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// HandleJoinBreakoutRoom takes no body and no room ID — the service resolves
// the caller's own assignment for the meeting. See file-level comment.
func TestHandleJoinBreakoutRoom_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleJoinBreakoutRoom(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleJoinBreakoutRoom_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/breakout-rooms/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleJoinBreakoutRoom(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleJoinBreakoutRoom_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/join", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleJoinBreakoutRoom(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetBreakoutAssignment_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/breakout-rooms/assignment", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGetBreakoutAssignment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetBreakoutAssignment_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid/breakout-rooms/assignment", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetBreakoutAssignment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetBreakoutAssignment_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/breakout-rooms/assignment", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGetBreakoutAssignment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleReturnToMainRoom_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/return",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleReturnToMainRoom(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleReturnToMainRoom_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/breakout-rooms/return",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleReturnToMainRoom(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleReturnToMainRoom_InvalidTargetUserID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/return",
		strings.NewReader(`{"target_user_id":"not-a-uuid"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleReturnToMainRoom(rec, req)
	assertValidationError(t, rec, "target_user_id")
}

// TestHandleReturnToMainRoom_ReachesRPC_SelfReturn covers the no-target-user
// case (a participant returning themselves): targetUserID stays "" and the
// handler still reaches the RPC.
func TestHandleReturnToMainRoom_ReachesRPC_SelfReturn(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/return",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleReturnToMainRoom(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleReturnToMainRoom_ReachesRPC_ForceReturn covers the host/co-host
// path (force-returning a named participant); the service enforces the
// host/co-host check (TestReturnToMainRoom-equivalent authz lives in
// internal/work/meeting/service.go), the gateway only forwards it.
func TestHandleReturnToMainRoom_ReachesRPC_ForceReturn(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/return",
		strings.NewReader(`{"target_user_id":"`+uuid.New().String()+`"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleReturnToMainRoom(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCloseBreakoutRooms_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/close", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCloseBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCloseBreakoutRooms_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/breakout-rooms/close", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCloseBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCloseBreakoutRooms_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/breakout-rooms/close", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCloseBreakoutRooms(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
