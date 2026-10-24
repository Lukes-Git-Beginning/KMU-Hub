package gateway

// GET /hr/time/team lists the whole tenant only when the caller's grant says "all". The gateway
// resolves the scope from the token and hands it to the HR service in GetTeamTimeReq.Scope; a real
// gRPC server on loopback plays the service and records what arrived.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

type teamTimeStub struct {
	hrv1.UnimplementedHRServiceServer
	got *hrv1.GetTeamTimeReq
}

func (s *teamTimeStub) GetTeamTime(_ context.Context, r *hrv1.GetTeamTimeReq) (*hrv1.GetTeamTimeResp, error) {
	s.got = r
	return &hrv1.GetTeamTimeResp{}, nil
}

func TestHandleGetTeamTime_PassesTheResolvedScope(t *testing.T) {
	cases := []struct {
		name   string
		scopes map[string]string
		want   string
	}{
		{"admin (both keys at all)", map[string]string{"hr:read": auth.ScopeAll, "zeiterfassung:team:view": auth.ScopeAll}, auth.ScopeAll},
		{"manager after the seed (team:view at team)", map[string]string{"zeiterfassung:team:view": auth.ScopeTeam}, auth.ScopeTeam},
		{"token minted before the scopes claim", nil, auth.ScopeAll},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &teamTimeStub{}
			routes := NewHRRoutes(registryFor(t, "biz", func(s *grpc.Server) { hrv1.RegisterHRServiceServer(s, stub) }), nil)

			req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/team?week_start=2026-08-17", nil), leaveCallerID, testTenantID)
			if tc.scopes != nil {
				req = withScopes(req, tc.scopes)
			}
			rec := httptest.NewRecorder()
			routes.HandleGetTeamTime(rec, req)

			assertStatus(t, rec, http.StatusOK)
			require.NotNil(t, stub.got)
			assert.Equal(t, tc.want, stub.got.GetScope())
			assert.Equal(t, "2026-08-17", stub.got.GetWeekStart())
		})
	}
}

// narrowestScopeAny feeds GetTeamTimeReq.Scope. The narrower grant has to win, and an
// unrecognised value must never read as wide.
func TestNarrowestScopeAny(t *testing.T) {
	pairs := [][2]string{{"hr", "read"}, {"zeiterfassung:team", "view"}}
	cases := []struct {
		name   string
		scopes map[string]string
		want   string
	}{
		{"unassigned key does not widen a narrowed one", map[string]string{"hr:read": auth.ScopeOwn}, auth.ScopeOwn},
		{"the narrower of two assigned scopes wins", map[string]string{"hr:read": auth.ScopeAll, "zeiterfassung:team:view": auth.ScopeTeam}, auth.ScopeTeam},
		{"an unknown scope value is treated as the narrowest", map[string]string{"zeiterfassung:team:view": "bogus"}, auth.ScopeOwn},
		{"no scopes claim resolves to all", nil, auth.ScopeAll},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.scopes != nil {
				ctx = context.WithValue(ctx, middleware.UserScopesKey, tc.scopes)
			}
			assert.Equal(t, tc.want, narrowestScopeAny(ctx, pairs...))
		})
	}
}
