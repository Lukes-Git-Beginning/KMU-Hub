package gateway

// route_auth_idempotency_redact_test.go belongs to
// harden-idempotency-skip-response-body-for-one-time-secrets (BACKLOG.yml).
// Four auth handlers answer with a plaintext token that exists nowhere else
// (invitation tokens are stored hashed). middleware.Idempotency copies a
// response into idempotency_keys for 24 h unless the handler flags it with
// middleware.RedactIdempotentResponse; each test pins that flag per handler.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/middleware"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

const redactSecretToken = "plaintext-one-time-token"

type secretAuthServer struct {
	authv1.UnimplementedAuthServiceServer
	inviteToken string
}

func (s *secretAuthServer) InviteAdminUser(context.Context, *authv1.InviteAdminUserRequest) (*authv1.AdminUserResponse, error) {
	return &authv1.AdminUserResponse{User: &authv1.AdminUserInfo{Id: "u1"}, InviteToken: s.inviteToken}, nil
}

func (s *secretAuthServer) ResendAdminUserInvite(context.Context, *authv1.ResendAdminUserInviteRequest) (*authv1.AdminUserResponse, error) {
	return &authv1.AdminUserResponse{User: &authv1.AdminUserInfo{Id: "u1"}, InviteToken: s.inviteToken}, nil
}

func (s *secretAuthServer) UpdateAdminUser(context.Context, *authv1.UpdateAdminUserRequest) (*authv1.AdminUserResponse, error) {
	return &authv1.AdminUserResponse{User: &authv1.AdminUserInfo{Id: "u1"}}, nil
}

func (s *secretAuthServer) ProvisionTenant(context.Context, *authv1.ProvisionTenantRequest) (*authv1.ProvisionTenantResponse, error) {
	return &authv1.ProvisionTenantResponse{Tenant: &authv1.TenantInfo{Id: "t1"}, InvitationToken: redactSecretToken}, nil
}

func (s *secretAuthServer) CreateInvitation(context.Context, *authv1.CreateInvitationRequest) (*authv1.CreateInvitationResponse, error) {
	return &authv1.CreateInvitationResponse{Invitation: &authv1.InvitationInfo{Id: "i1"}, Token: redactSecretToken}, nil
}

func newSecretAuthRoutes(t *testing.T) *AuthRoutes {
	t.Helper()
	gs := grpc.NewServer()
	authv1.RegisterAuthServiceServer(gs, &secretAuthServer{inviteToken: redactSecretToken})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = gs.Serve(l) }()
	t.Cleanup(gs.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("auth", l.Addr().String())
	t.Cleanup(reg.Close)
	return NewAuthRoutes(reg)
}

func TestAuthRoutes_OneTimeSecretResponses_AreFlaggedForIdempotencyRedaction(t *testing.T) {
	routes := newSecretAuthRoutes(t)
	const roleID = "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name       string
		req        *http.Request
		handler    http.HandlerFunc
		wantStatus int
		wantMarked bool
	}{
		{
			name: "admin invite",
			req: httptest.NewRequest("POST", "/api/v1/admin/users/invite",
				strings.NewReader(`{"email":"a@b.test","roles":["`+roleID+`"]}`)),
			handler: routes.HandleInviteAdminUser, wantStatus: http.StatusCreated, wantMarked: true,
		},
		{
			name: "admin resend invite",
			req: withChiURLParam(httptest.NewRequest("POST", "/api/v1/admin/users/x/resend-invite", nil),
				"id", roleID),
			handler: routes.HandleResendAdminUserInvite, wantStatus: http.StatusOK, wantMarked: true,
		},
		{
			name: "provision tenant",
			req: httptest.NewRequest("POST", "/api/v1/tenants",
				strings.NewReader(`{"name":"Acme","admin_email":"a@b.test"}`)),
			handler: routes.HandleProvisionTenant, wantStatus: http.StatusCreated, wantMarked: true,
		},
		{
			name: "create invitation",
			req: withUserID(httptest.NewRequest("POST", "/api/v1/invitations",
				strings.NewReader(`{"email":"a@b.test","role":"member"}`)), "u0"),
			handler: routes.HandleCreateInvitation, wantStatus: http.StatusCreated, wantMarked: true,
		},
		{
			// No token in the response → nothing to hide, replay should stay useful.
			name: "admin user update",
			req: withChiURLParam(httptest.NewRequest("PATCH", "/api/v1/admin/users/x",
				strings.NewReader(`{"status":"active"}`)), "id", roleID),
			handler: routes.HandleUpdateAdminUser, wantStatus: http.StatusOK, wantMarked: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler(rec, tt.req)
			assertStatus(t, rec, tt.wantStatus)
			if got := rec.Header().Get(middleware.RedactIdempotencyHeader) != ""; got != tt.wantMarked {
				t.Fatalf("redact marker set = %v, want %v", got, tt.wantMarked)
			}
			if tt.wantMarked && !strings.Contains(rec.Body.String(), redactSecretToken) {
				t.Fatalf("client must still receive the token, body = %s", rec.Body.String())
			}
		})
	}
}
