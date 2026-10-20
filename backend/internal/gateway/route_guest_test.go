package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
	"github.com/kmuhub/kmuhub/internal/chat/guest"
)

// newGuestRoutes creates a GuestRoutes with a nil guestService.
// This is sufficient for tests that only exercise validation/decode
// (which fail before the service is called).
func newGuestRoutes(registry *ServiceRegistry) *GuestRoutes {
	return NewGuestRoutes(nil, registry)
}

// --- HandleCreateSession ---

func TestHandleCreateSession_InvalidJSON(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", invalidJSON())
	routes.HandleCreateSession(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateSession_MissingChannelID(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"display_name": "Alice",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "channel_id")
}

func TestHandleCreateSession_InvalidChannelIDFormat(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"channel_id":   "not-a-uuid",
		"display_name": "Alice",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "channel_id")
}

func TestHandleCreateSession_MissingDisplayName(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"channel_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "display_name")
}

func TestHandleCreateSession_InvalidEmailFormat(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"channel_id":   "550e8400-e29b-41d4-a716-446655440000",
		"display_name": "Alice",
		"email":        "not-an-email",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "email")
}

// --- HandleValidateToken ---

func TestHandleValidateToken_InvalidJSON(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions/validate", invalidJSON())
	routes.HandleValidateToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleValidateToken_MissingToken(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions/validate", jsonBody(t, map[string]interface{}{}))
	routes.HandleValidateToken(rec, req)
	assertValidationError(t, rec, "token")
}

// --- HandleSendMessage (guest) ---
// Note: "send message" also exists on chat routes; these tests are prefixed
// with "Guest" to avoid redeclaration within the gateway test package.

func TestGuestHandleSendMessage_NoSession(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/channels/xxx/messages", invalidJSON())
	// Without a session in context the handler returns 401 before body decode.
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestGuestHandleSendMessage_NoSession_EmptyBody(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/channels/xxx/messages", jsonBody(t, map[string]interface{}{}))
	// Handler guards on session before body decode — verifying 401 not validation error.
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- HandleSendMessage (guest) against a real chat gRPC server ---

// guestChatServer records the SendMessageRequest and applies the guard
// ChatGRPCServer.SendMessage opens with: tenant_id is parsed unconditionally.
type guestChatServer struct {
	chatv1.UnimplementedChatServiceServer
	got chan *chatv1.SendMessageRequest
}

func (s *guestChatServer) SendMessage(_ context.Context, in *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	s.got <- in
	if _, err := uuid.Parse(in.TenantId); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	return &chatv1.SendMessageResponse{Message: &chatv1.MessageInfo{Id: uuid.New().String(), Content: in.Content}}, nil
}

// TestGuestHandleSendMessage_CarriesSessionTenant guards the fix for guest
// sends that always failed with InvalidArgument: the request never carried a
// tenant_id. A guest has no JWT, so the tenant comes from the guest session.
func TestGuestHandleSendMessage_CarriesSessionTenant(t *testing.T) {
	srv := &guestChatServer{got: make(chan *chatv1.SendMessageRequest, 1)}
	gs := grpc.NewServer()
	chatv1.RegisterChatServiceServer(gs, srv)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = gs.Serve(l) }()
	t.Cleanup(gs.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("chat", l.Addr().String())
	t.Cleanup(reg.Close)
	routes := NewGuestRoutes(guest.NewService(nil), reg)

	session := &guest.GuestSession{ID: uuid.New(), TenantID: uuid.New(), ChannelID: uuid.New()}

	r := chi.NewRouter()
	r.Post("/api/v1/guest/channels/{channelID}/messages", routes.HandleSendMessage)

	req := httptest.NewRequest("POST", "/api/v1/guest/channels/"+session.ChannelID.String()+"/messages",
		jsonBody(t, map[string]interface{}{"content": "hallo"}))
	req = req.WithContext(context.WithValue(req.Context(), guestSessionContextKey{}, session))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusCreated)
	in := <-srv.got
	if in.TenantId != session.TenantID.String() {
		t.Errorf("tenant_id = %q, want session tenant %q", in.TenantId, session.TenantID)
	}
	if in.GuestSessionId == nil || *in.GuestSessionId != session.ID.String() {
		t.Errorf("guest_session_id = %v, want %s", in.GuestSessionId, session.ID)
	}
}
