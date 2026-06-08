package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatRoutes_ServiceName(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	if routes.ServiceName() != "chat" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "chat")
	}
}

// --- Channels ---

func TestHandleCreateChannel_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateChannel)
}

func TestHandleCreateChannel_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleCreateChannel)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

func TestHandleCreateChannel_InvalidJSON(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateChannel_MissingName(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels", jsonBody(t, map[string]interface{}{
		"description": "test",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "name")
}

func TestHandleGetChannel_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "user-123")
	routes.HandleGetChannel(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetChannel_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	withAuthRequired(routes.HandleGetChannel)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetChannel_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleGetChannel(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListChannels_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels", nil)
	req = withUserID(req, "user-123")
	routes.HandleListChannels(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListChannels_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/channels", nil)
	withAuthRequired(routes.HandleListChannels)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- Messages ---

func TestHandleSendMessage_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/123/messages", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "user-123")
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSendMessage_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/123/messages", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	withAuthRequired(routes.HandleSendMessage)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSendMessage_InvalidJSON(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/123/messages", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "user-123")
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateMessage_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/messages/123", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "user-123")
	routes.HandleUpdateMessage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteMessage_InvalidUUID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/messages/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleDeleteMessage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- DMs ---

func TestHandleGetOrCreateDM_ServiceUnavailable(t *testing.T) {
	routes := NewChatRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/dm", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleGetOrCreateDM(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetOrCreateDM_NoUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/dm", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleGetOrCreateDM)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- Validation error tests ---

func TestHandleSendMessage_MissingContent(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/123/messages", jsonBody(t, map[string]interface{}{
		"mention_everyone": false,
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "content")
}

func TestHandleGetOrCreateDM_MissingOtherUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/dm", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleGetOrCreateDM(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "other_user_id")
}

func TestHandleGetOrCreateDM_InvalidOtherUserID(t *testing.T) {
	routes := NewChatRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/channels/dm", jsonBody(t, map[string]interface{}{
		"other_user_id": "not-a-uuid",
	}))
	req = withUserID(req, "user-123")
	routes.HandleGetOrCreateDM(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "other_user_id")
}
