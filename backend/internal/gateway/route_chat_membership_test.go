package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the channel membership and role-management handler group in
// route_chat.go: HandleJoinChannel, HandleLeaveChannel,
// HandleGetChannelMembers, HandleUpdateMemberRole, HandleArchiveChannel and
// HandleDeleteChannel. route_chat_test.go already covers channel CRUD,
// messages and DMs; this file fills the membership gap.

const (
	testChatChannelID    = "550e8400-e29b-41d4-a716-446655440000"
	testChatTargetUserID = "660e8400-e29b-41d4-a716-446655440001"
)

// --- HandleJoinChannel ---

func TestHandleJoinChannel_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleJoinChannel)
}

func TestHandleJoinChannel_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/"+testChatChannelID+"/join", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	withAuthRequired(routes.HandleJoinChannel)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleJoinChannel_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/bad/join", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleJoinChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleJoinChannel_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/"+testChatChannelID+"/join", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleJoinChannel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleLeaveChannel ---

func TestHandleLeaveChannel_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleLeaveChannel)
}

func TestHandleLeaveChannel_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/"+testChatChannelID+"/leave", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	withAuthRequired(routes.HandleLeaveChannel)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleLeaveChannel_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/bad/leave", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleLeaveChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleLeaveChannel_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/"+testChatChannelID+"/leave", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleLeaveChannel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetChannelMembers ---

func TestHandleGetChannelMembers_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetChannelMembers)
}

func TestHandleGetChannelMembers_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/bad/members", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleGetChannelMembers(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetChannelMembers_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/"+testChatChannelID+"/members", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleGetChannelMembers(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateMemberRole ---

func TestHandleUpdateMemberRole_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateMemberRole)
}

func TestHandleUpdateMemberRole_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID+"/members/"+testChatTargetUserID+"/role",
		jsonBody(t, map[string]interface{}{"role": "admin"}))
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withChiURLParam(req, "userId", testChatTargetUserID)
	routes.HandleUpdateMemberRole(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

func TestHandleUpdateMemberRole_InvalidChannelID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/bad/members/"+testChatTargetUserID+"/role",
		jsonBody(t, map[string]interface{}{"role": "admin"}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "userId", testChatTargetUserID)
	req = withUserID(req, "requester-123")
	routes.HandleUpdateMemberRole(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateMemberRole_InvalidTargetUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID+"/members/bad/role",
		jsonBody(t, map[string]interface{}{"role": "admin"}))
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withChiURLParam(req, "userId", "not-a-uuid")
	req = withUserID(req, "requester-123")
	routes.HandleUpdateMemberRole(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid userId")
}

func TestHandleUpdateMemberRole_InvalidJSON(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID+"/members/"+testChatTargetUserID+"/role", invalidJSON())
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withChiURLParam(req, "userId", testChatTargetUserID)
	req = withUserID(req, "requester-123")
	routes.HandleUpdateMemberRole(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateMemberRole_UnknownRole(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID+"/members/"+testChatTargetUserID+"/role",
		jsonBody(t, map[string]interface{}{"role": "superadmin"}))
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withChiURLParam(req, "userId", testChatTargetUserID)
	req = withUserID(req, "requester-123")
	routes.HandleUpdateMemberRole(rec, req)
	assertValidationError(t, rec, "role")
}

func TestHandleUpdateMemberRole_MissingRole(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID+"/members/"+testChatTargetUserID+"/role",
		jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withChiURLParam(req, "userId", testChatTargetUserID)
	req = withUserID(req, "requester-123")
	routes.HandleUpdateMemberRole(rec, req)
	assertValidationError(t, rec, "role")
}

func TestHandleUpdateMemberRole_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/channels/"+testChatChannelID+"/members/"+testChatTargetUserID+"/role",
		jsonBody(t, map[string]interface{}{"role": "admin"}))
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withChiURLParam(req, "userId", testChatTargetUserID)
	req = withUserID(req, "requester-123")
	routes.HandleUpdateMemberRole(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleArchiveChannel ---

func TestHandleArchiveChannel_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleArchiveChannel)
}

func TestHandleArchiveChannel_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/bad/archive", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleArchiveChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleArchiveChannel_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/"+testChatChannelID+"/archive", nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleArchiveChannel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteChannel ---

func TestHandleDeleteChannel_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteChannel)
}

func TestHandleDeleteChannel_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/channels/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleDeleteChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteChannel_ReachesRPC(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/channels/"+testChatChannelID, nil)
	req = withChiURLParam(req, "id", testChatChannelID)
	req = withUserID(req, "user-123")
	routes.HandleDeleteChannel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
